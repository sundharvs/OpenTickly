// Package google implements the pieces of Google's OAuth2 + Calendar REST
// APIs this integration needs, using only the standard library. A dedicated
// oauth2 client library was deliberately not added: the flow here is a single
// well-documented authorization-code exchange plus refresh, and keeping it in
// net/http avoids a new module dependency for something this small.
package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL    = "https://oauth2.googleapis.com/token"
	userInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

	// Scope grants read-only calendar access plus the account email, matching
	// the one-way "import only" scope of this integration.
	Scope = "https://www.googleapis.com/auth/calendar.readonly https://www.googleapis.com/auth/userinfo.email"
)

// OAuthConfig holds the instance-wide Google OAuth client credentials
// (configured via OPENTOGGL_GOOGLE_CALENDAR_CLIENT_ID/SECRET). The redirect
// URL is not stored here — it is derived per-request from the instance's
// runtime-configurable site URL and passed into AuthCodeURL/ExchangeCode.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

func (cfg OAuthConfig) Configured() bool {
	return cfg.ClientID != "" && cfg.ClientSecret != ""
}

func (cfg OAuthConfig) httpClient() *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return http.DefaultClient
}

// Token is the result of an authorization-code exchange or a refresh.
// RefreshToken is only populated on the initial exchange — Google omits it
// from refresh responses, so callers must keep the one already on file.
type Token struct {
	AccessToken   string
	RefreshToken  string
	ExpiresAt     time.Time
	Scope         string
	HasWriteScope bool
}

// AuthCodeURL builds the consent-screen URL the browser should be redirected
// to, with the given opaque CSRF state echoed back on callback.
func (cfg OAuthConfig) AuthCodeURL(state string, redirectURL string) string {
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", redirectURL)
	values.Set("response_type", "code")
	values.Set("scope", Scope)
	values.Set("state", state)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	values.Set("include_granted_scopes", "true")
	return authURL + "?" + values.Encode()
}

// ExchangeCode trades an authorization code from the OAuth callback for an
// access + refresh token pair. redirectURL must match the one AuthCodeURL
// used to start this same flow.
func (cfg OAuthConfig) ExchangeCode(ctx context.Context, code string, redirectURL string) (Token, error) {
	values := url.Values{}
	values.Set("code", code)
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	values.Set("redirect_uri", redirectURL)
	values.Set("grant_type", "authorization_code")
	return cfg.requestToken(ctx, values)
}

// RefreshAccessToken uses a previously stored refresh token to obtain a new
// access token. The returned Token's RefreshToken is empty — reuse the
// existing one.
func (cfg OAuthConfig) RefreshAccessToken(ctx context.Context, refreshToken string) (Token, error) {
	values := url.Values{}
	values.Set("refresh_token", refreshToken)
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	values.Set("grant_type", "refresh_token")
	return cfg.requestToken(ctx, values)
}

func (cfg OAuthConfig) requestToken(ctx context.Context, values url.Values) (Token, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return Token{}, fmt.Errorf("build google token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := cfg.httpClient().Do(request)
	if err != nil {
		return Token{}, fmt.Errorf("call google token endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return Token{}, fmt.Errorf("read google token response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return Token{}, fmt.Errorf("google token endpoint returned %d: %s", response.StatusCode, string(body))
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Token{}, fmt.Errorf("decode google token response: %w", err)
	}

	return Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second),
		Scope:        payload.Scope,
		HasWriteScope: strings.Contains(payload.Scope, "auth/calendar") &&
			!strings.Contains(payload.Scope, "auth/calendar.readonly"),
	}, nil
}

// FetchAccountEmail resolves the email address of the account that just
// authorized this integration, using the access token from ExchangeCode.
func (cfg OAuthConfig) FetchAccountEmail(ctx context.Context, accessToken string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return "", fmt.Errorf("build google userinfo request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := cfg.httpClient().Do(request)
	if err != nil {
		return "", fmt.Errorf("call google userinfo endpoint: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read google userinfo response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google userinfo endpoint returned %d: %s", response.StatusCode, string(body))
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("decode google userinfo response: %w", err)
	}
	return payload.Email, nil
}
