package google

import (
	"context"
	"time"

	calendarapplication "opentoggl/backend/apps/backend/internal/calendar/application"
	"opentoggl/backend/apps/backend/internal/calendar/domain"
)

// Provider adapts OAuthConfig + Client to calendarapplication.Provider.
type Provider struct {
	OAuth    OAuthConfig
	Calendar Client
}

func NewProvider(oauth OAuthConfig, calendar Client) Provider {
	return Provider{OAuth: oauth, Calendar: calendar}
}

func (provider Provider) Configured() bool {
	return provider.OAuth.Configured()
}

func (provider Provider) AuthCodeURL(state string, redirectURL string) string {
	return provider.OAuth.AuthCodeURL(state, redirectURL)
}

func (provider Provider) ExchangeCode(ctx context.Context, code string, redirectURL string) (calendarapplication.ProviderToken, error) {
	token, err := provider.OAuth.ExchangeCode(ctx, code, redirectURL)
	if err != nil {
		return calendarapplication.ProviderToken{}, err
	}
	return toApplicationToken(token), nil
}

func (provider Provider) RefreshAccessToken(ctx context.Context, refreshToken string) (calendarapplication.ProviderToken, error) {
	token, err := provider.OAuth.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		return calendarapplication.ProviderToken{}, err
	}
	return toApplicationToken(token), nil
}

func (provider Provider) FetchAccountEmail(ctx context.Context, accessToken string) (string, error) {
	return provider.OAuth.FetchAccountEmail(ctx, accessToken)
}

func (provider Provider) ListCalendars(ctx context.Context, accessToken string) ([]domain.RemoteCalendar, error) {
	return provider.Calendar.ListCalendars(ctx, accessToken)
}

func (provider Provider) ListEvents(ctx context.Context, accessToken string, calendarExternalID string, start, end time.Time) ([]domain.Event, error) {
	return provider.Calendar.ListEvents(ctx, accessToken, calendarExternalID, start, end)
}

func toApplicationToken(token Token) calendarapplication.ProviderToken {
	return calendarapplication.ProviderToken{
		AccessToken:   token.AccessToken,
		RefreshToken:  token.RefreshToken,
		ExpiresAt:     token.ExpiresAt,
		HasWriteScope: token.HasWriteScope,
	}
}
