// Package application implements the one-way "connect a calendar, browse its
// events, turn one into a time entry" flow described in
// [../doc.go](../doc.go). It never writes anything back to the provider.
package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"opentoggl/backend/apps/backend/internal/calendar/domain"

	"github.com/samber/lo"
)

// ProviderToken is the access/refresh token pair returned by a provider's
// OAuth token endpoint, decoupled from any specific provider SDK type.
type ProviderToken struct {
	AccessToken   string
	RefreshToken  string
	ExpiresAt     time.Time
	HasWriteScope bool
}

// Provider is implemented per calendar provider (Google today) for the OAuth
// handshake and the read-only calendar/event listing calls. redirectURL is
// threaded through per-call rather than fixed at construction because it is
// derived from the instance's runtime-configurable site URL (see
// routeHandlers.samlBaseURL, reused for this), not a static setting; Google
// requires the exact same redirect_uri on both the auth request and the code
// exchange, but not on a refresh.
type Provider interface {
	Configured() bool
	AuthCodeURL(state string, redirectURL string) string
	ExchangeCode(ctx context.Context, code string, redirectURL string) (ProviderToken, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (ProviderToken, error)
	FetchAccountEmail(ctx context.Context, accessToken string) (string, error)
	ListCalendars(ctx context.Context, accessToken string) ([]domain.RemoteCalendar, error)
	ListEvents(ctx context.Context, accessToken string, calendarExternalID string, start, end time.Time) ([]domain.Event, error)
}

// Repository persists integrations and their calendar selection state.
type Repository interface {
	UpsertIntegration(ctx context.Context, userID int64, provider, email, accessToken, refreshToken string, tokenExpiresAt *time.Time, hasWriteScope bool, scopes []string) (domain.Integration, error)
	ListIntegrationsByUserID(ctx context.Context, userID int64) ([]domain.Integration, error)
	GetIntegration(ctx context.Context, userID int64, integrationID int64) (domain.Integration, bool, error)
	UpdateTokens(ctx context.Context, integrationID int64, accessToken string, tokenExpiresAt *time.Time) error
	MarkIntegrationError(ctx context.Context, integrationID int64, errorStatus string) error
	SoftDeleteIntegration(ctx context.Context, userID int64, integrationID int64) error
	ReplaceCalendars(ctx context.Context, integrationID int64, remote []domain.RemoteCalendar) error
	ListCalendars(ctx context.Context, integrationID int64) ([]domain.Calendar, error)
	SetCalendarSelected(ctx context.Context, integrationID int64, calendarID int64, selected bool) (domain.Calendar, error)
	ListSelectedCalendarsForUser(ctx context.Context, userID int64) ([]domain.SelectedCalendar, error)
}

// PendingState is the short-lived, single-use OAuth CSRF state cached
// between BeginSetup and CompleteCallback.
type PendingState struct {
	UserID   int64
	ReturnTo string
}

// StateCache is the minimal cache contract this service needs (the same
// platform-wide cache the SAML login flow uses for its own pending state).
type StateCache interface {
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
	Get(ctx context.Context, key string, dest any) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

const stateTTL = 10 * time.Minute

// Service implements the calendar integration use cases. Providers is keyed
// by provider name ("google") so a future Outlook provider can be added
// without changing callers.
type Service struct {
	repo      Repository
	providers map[string]Provider
	cache     StateCache
}

func NewService(repo Repository, cache StateCache, providers map[string]Provider) *Service {
	return &Service{repo: repo, providers: providers, cache: cache}
}

func (service *Service) provider(name string) (Provider, error) {
	provider, ok := service.providers[name]
	if !ok {
		return nil, domain.ErrUnsupportedProvider
	}
	if !provider.Configured() {
		return nil, domain.ErrNotConfigured
	}
	return provider, nil
}

func (service *Service) ListIntegrations(ctx context.Context, userID int64) ([]domain.Integration, error) {
	return service.repo.ListIntegrationsByUserID(ctx, userID)
}

// BeginSetup returns the URL to redirect the browser to for the given
// provider's consent screen, having stashed a CSRF state token that
// CompleteCallback will validate. redirectURL must be the exact callback URL
// this deployment is registered under with the provider.
func (service *Service) BeginSetup(ctx context.Context, userID int64, providerName string, returnTo string, redirectURL string) (string, error) {
	oauthProvider, err := service.provider(providerName)
	if err != nil {
		return "", err
	}

	state, err := randomState()
	if err != nil {
		return "", fmt.Errorf("generate calendar oauth state: %w", err)
	}
	if err := service.cache.Set(ctx, stateKey(providerName, state), PendingState{UserID: userID, ReturnTo: returnTo}, stateTTL); err != nil {
		return "", fmt.Errorf("store calendar oauth state: %w", err)
	}

	return oauthProvider.AuthCodeURL(state, redirectURL), nil
}

// CompleteCallback exchanges the authorization code for tokens, resolves the
// connected account's email, upserts the integration, and seeds its initial
// calendar list. It returns the ReturnTo path the caller should redirect to.
// redirectURL must match the one BeginSetup used for this same flow.
func (service *Service) CompleteCallback(ctx context.Context, providerName string, code string, state string, redirectURL string) (string, error) {
	oauthProvider, err := service.provider(providerName)
	if err != nil {
		return "", err
	}

	var pending PendingState
	key := stateKey(providerName, state)
	found, err := service.cache.Get(ctx, key, &pending)
	if err != nil {
		return "", fmt.Errorf("read calendar oauth state: %w", err)
	}
	if !found {
		return "", domain.ErrInvalidState
	}
	_ = service.cache.Del(ctx, key)

	token, err := oauthProvider.ExchangeCode(ctx, code, redirectURL)
	if err != nil {
		return "", fmt.Errorf("exchange calendar oauth code: %w", err)
	}

	email, err := oauthProvider.FetchAccountEmail(ctx, token.AccessToken)
	if err != nil {
		return "", fmt.Errorf("fetch calendar account email: %w", err)
	}

	integration, err := service.repo.UpsertIntegration(
		ctx, pending.UserID, providerName, email,
		token.AccessToken, token.RefreshToken, lo.ToPtr(token.ExpiresAt),
		token.HasWriteScope, []string{domain.ProviderGoogle + ":calendar.readonly"},
	)
	if err != nil {
		return "", fmt.Errorf("save calendar integration: %w", err)
	}

	remoteCalendars, err := oauthProvider.ListCalendars(ctx, token.AccessToken)
	if err != nil {
		return "", fmt.Errorf("list calendars after connect: %w", err)
	}
	if err := service.repo.ReplaceCalendars(ctx, integration.ID, remoteCalendars); err != nil {
		return "", fmt.Errorf("seed calendars after connect: %w", err)
	}

	return pending.ReturnTo, nil
}

func (service *Service) DisconnectIntegration(ctx context.Context, userID int64, integrationID int64) error {
	return service.repo.SoftDeleteIntegration(ctx, userID, integrationID)
}

func (service *Service) ListCalendars(ctx context.Context, userID int64, integrationID int64) ([]domain.Calendar, error) {
	if _, found, err := service.repo.GetIntegration(ctx, userID, integrationID); err != nil {
		return nil, err
	} else if !found {
		return nil, domain.ErrIntegrationNotFound
	}
	return service.repo.ListCalendars(ctx, integrationID)
}

func (service *Service) SetCalendarSelected(ctx context.Context, userID int64, integrationID int64, calendarID int64, selected bool) (domain.Calendar, error) {
	if _, found, err := service.repo.GetIntegration(ctx, userID, integrationID); err != nil {
		return domain.Calendar{}, err
	} else if !found {
		return domain.Calendar{}, domain.ErrIntegrationNotFound
	}
	return service.repo.SetCalendarSelected(ctx, integrationID, calendarID, selected)
}

// ListEvents fetches events live from every selected calendar across all of
// the user's connected integrations, refreshing an expired access token
// on the fly. A single calendar's failure (e.g. a revoked grant) is recorded
// on its integration but does not fail the whole request.
func (service *Service) ListEvents(ctx context.Context, userID int64, start, end time.Time) ([]domain.Event, error) {
	selectedCalendars, err := service.repo.ListSelectedCalendarsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	events := make([]domain.Event, 0)
	for _, selected := range selectedCalendars {
		oauthProvider, err := service.provider(selected.Provider)
		if err != nil {
			continue
		}

		accessToken := selected.AccessToken
		if selected.TokenExpiresAt != nil && time.Now().After(*selected.TokenExpiresAt) {
			refreshed, err := oauthProvider.RefreshAccessToken(ctx, selected.RefreshToken)
			if err != nil {
				_ = service.repo.MarkIntegrationError(ctx, selected.IntegrationID, "calendar access has expired or been revoked")
				continue
			}
			accessToken = refreshed.AccessToken
			_ = service.repo.UpdateTokens(ctx, selected.IntegrationID, accessToken, lo.ToPtr(refreshed.ExpiresAt))
		}

		calendarEvents, err := oauthProvider.ListEvents(ctx, accessToken, selected.Calendar.ExternalID, start, end)
		if err != nil {
			_ = service.repo.MarkIntegrationError(ctx, selected.IntegrationID, "error fetching events from the provider")
			continue
		}
		for i := range calendarEvents {
			calendarEvents[i].CalendarID = selected.Calendar.ID
		}
		events = append(events, calendarEvents...)
	}
	return events, nil
}

func stateKey(providerName, state string) string {
	return "calendar:oauth:" + providerName + ":" + state
}

func randomState() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
