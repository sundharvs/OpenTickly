package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"opentoggl/backend/apps/backend/internal/calendar/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// UpsertIntegration inserts a new integration or, if the user already has a
// non-deleted integration with this provider, revives/updates it with the
// freshly obtained tokens.
func (repo *Repository) UpsertIntegration(
	ctx context.Context,
	userID int64,
	provider string,
	email string,
	accessToken string,
	refreshToken string,
	tokenExpiresAt *time.Time,
	hasWriteScope bool,
	scopes []string,
) (domain.Integration, error) {
	row := repo.pool.QueryRow(ctx, `
		insert into calendar_integrations (
			user_id, provider, email, access_token, refresh_token,
			token_expires_at, has_write_scope, scopes, deleted_at
		)
		values ($1, $2, $3, $4, $5, $6, $7, $8, null)
		on conflict (user_id, provider) where deleted_at is null
		do update set
			email = excluded.email,
			access_token = excluded.access_token,
			refresh_token = excluded.refresh_token,
			token_expires_at = excluded.token_expires_at,
			has_write_scope = excluded.has_write_scope,
			scopes = excluded.scopes,
			error_status = '',
			updated_at = now()
		returning id, user_id, provider, email, access_token, refresh_token,
			token_expires_at, has_write_scope, error_status, scopes, created_at, updated_at
	`, userID, provider, email, accessToken, refreshToken, tokenExpiresAt, hasWriteScope, scopes)

	integration, err := scanIntegration(row)
	if err != nil {
		return domain.Integration{}, fmt.Errorf("upsert calendar integration for user %d: %w", userID, err)
	}
	return integration, nil
}

func (repo *Repository) ListIntegrationsByUserID(ctx context.Context, userID int64) ([]domain.Integration, error) {
	rows, err := repo.pool.Query(ctx, `
		select id, user_id, provider, email, access_token, refresh_token,
			token_expires_at, has_write_scope, error_status, scopes, created_at, updated_at
		from calendar_integrations
		where user_id = $1 and deleted_at is null
		order by created_at asc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list calendar integrations for user %d: %w", userID, err)
	}
	defer rows.Close()

	integrations := make([]domain.Integration, 0)
	for rows.Next() {
		integration, err := scanIntegration(rows)
		if err != nil {
			return nil, fmt.Errorf("scan calendar integration for user %d: %w", userID, err)
		}
		integrations = append(integrations, integration)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calendar integrations for user %d: %w", userID, err)
	}
	return integrations, nil
}

func (repo *Repository) GetIntegration(ctx context.Context, userID int64, integrationID int64) (domain.Integration, bool, error) {
	row := repo.pool.QueryRow(ctx, `
		select id, user_id, provider, email, access_token, refresh_token,
			token_expires_at, has_write_scope, error_status, scopes, created_at, updated_at
		from calendar_integrations
		where id = $1 and user_id = $2 and deleted_at is null
	`, integrationID, userID)

	integration, err := scanIntegration(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Integration{}, false, nil
		}
		return domain.Integration{}, false, fmt.Errorf("get calendar integration %d: %w", integrationID, err)
	}
	return integration, true, nil
}

func (repo *Repository) UpdateTokens(
	ctx context.Context,
	integrationID int64,
	accessToken string,
	tokenExpiresAt *time.Time,
) error {
	if _, err := repo.pool.Exec(ctx, `
		update calendar_integrations
		set access_token = $2, token_expires_at = $3, error_status = '', updated_at = now()
		where id = $1
	`, integrationID, accessToken, tokenExpiresAt); err != nil {
		return fmt.Errorf("update calendar integration %d tokens: %w", integrationID, err)
	}
	return nil
}

func (repo *Repository) MarkIntegrationError(ctx context.Context, integrationID int64, errorStatus string) error {
	if _, err := repo.pool.Exec(ctx, `
		update calendar_integrations
		set error_status = $2, updated_at = now()
		where id = $1
	`, integrationID, errorStatus); err != nil {
		return fmt.Errorf("mark calendar integration %d error: %w", integrationID, err)
	}
	return nil
}

func (repo *Repository) SoftDeleteIntegration(ctx context.Context, userID int64, integrationID int64) error {
	if _, err := repo.pool.Exec(ctx, `
		update calendar_integrations
		set deleted_at = now(), updated_at = now()
		where id = $1 and user_id = $2 and deleted_at is null
	`, integrationID, userID); err != nil {
		return fmt.Errorf("disconnect calendar integration %d: %w", integrationID, err)
	}
	return nil
}

// ReplaceCalendars reconciles the locally stored calendar list for an
// integration against what the provider currently reports. New calendars are
// inserted (the primary one pre-selected); existing calendars keep their
// stored `selected` value untouched.
func (repo *Repository) ReplaceCalendars(ctx context.Context, integrationID int64, remote []domain.RemoteCalendar) error {
	for _, calendar := range remote {
		if _, err := repo.pool.Exec(ctx, `
			insert into calendar_integration_calendars (
				calendar_integration_id, external_id, name, background_color, foreground_color, selected
			)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (calendar_integration_id, external_id)
			do update set
				name = excluded.name,
				background_color = excluded.background_color,
				foreground_color = excluded.foreground_color,
				updated_at = now()
		`, integrationID, calendar.ExternalID, calendar.Name, calendar.BackgroundColor, calendar.ForegroundColor, calendar.Primary); err != nil {
			return fmt.Errorf("upsert calendar %s for integration %d: %w", calendar.ExternalID, integrationID, err)
		}
	}
	return nil
}

func (repo *Repository) ListCalendars(ctx context.Context, integrationID int64) ([]domain.Calendar, error) {
	rows, err := repo.pool.Query(ctx, `
		select id, calendar_integration_id, external_id, name, background_color, foreground_color, selected, created_at, updated_at
		from calendar_integration_calendars
		where calendar_integration_id = $1
		order by name asc
	`, integrationID)
	if err != nil {
		return nil, fmt.Errorf("list calendars for integration %d: %w", integrationID, err)
	}
	defer rows.Close()

	calendars := make([]domain.Calendar, 0)
	for rows.Next() {
		calendar, err := scanCalendar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan calendar for integration %d: %w", integrationID, err)
		}
		calendars = append(calendars, calendar)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate calendars for integration %d: %w", integrationID, err)
	}
	return calendars, nil
}

func (repo *Repository) SetCalendarSelected(ctx context.Context, integrationID int64, calendarID int64, selected bool) (domain.Calendar, error) {
	row := repo.pool.QueryRow(ctx, `
		update calendar_integration_calendars
		set selected = $3, updated_at = now()
		where id = $2 and calendar_integration_id = $1
		returning id, calendar_integration_id, external_id, name, background_color, foreground_color, selected, created_at, updated_at
	`, integrationID, calendarID, selected)

	calendar, err := scanCalendar(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Calendar{}, domain.ErrCalendarNotFound
		}
		return domain.Calendar{}, fmt.Errorf("set calendar %d selected: %w", calendarID, err)
	}
	return calendar, nil
}

// ListSelectedCalendarsForUser returns every selected calendar across all of
// the user's non-deleted integrations, joined with enough integration data to
// call the provider on the user's behalf.
func (repo *Repository) ListSelectedCalendarsForUser(ctx context.Context, userID int64) ([]domain.SelectedCalendar, error) {
	rows, err := repo.pool.Query(ctx, `
		select
			c.id, c.calendar_integration_id, c.external_id, c.name,
			c.background_color, c.foreground_color, c.selected, c.created_at, c.updated_at,
			i.provider, i.access_token, i.refresh_token, i.token_expires_at
		from calendar_integration_calendars c
		join calendar_integrations i on i.id = c.calendar_integration_id
		where i.user_id = $1 and i.deleted_at is null and c.selected
		order by c.name asc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list selected calendars for user %d: %w", userID, err)
	}
	defer rows.Close()

	selected := make([]domain.SelectedCalendar, 0)
	for rows.Next() {
		var (
			calendar       domain.Calendar
			provider       string
			accessToken    string
			refreshToken   string
			tokenExpiresAt *time.Time
		)
		if err := rows.Scan(
			&calendar.ID, &calendar.IntegrationID, &calendar.ExternalID, &calendar.Name,
			&calendar.BackgroundColor, &calendar.ForegroundColor, &calendar.Selected, &calendar.CreatedAt, &calendar.UpdatedAt,
			&provider, &accessToken, &refreshToken, &tokenExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan selected calendar for user %d: %w", userID, err)
		}
		selected = append(selected, domain.SelectedCalendar{
			Calendar:       calendar,
			IntegrationID:  calendar.IntegrationID,
			Provider:       provider,
			AccessToken:    accessToken,
			RefreshToken:   refreshToken,
			TokenExpiresAt: tokenExpiresAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate selected calendars for user %d: %w", userID, err)
	}
	return selected, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanIntegration(row scannable) (domain.Integration, error) {
	var (
		integration    domain.Integration
		tokenExpiresAt *time.Time
	)
	if err := row.Scan(
		&integration.ID, &integration.UserID, &integration.Provider, &integration.Email,
		&integration.AccessToken, &integration.RefreshToken, &tokenExpiresAt,
		&integration.HasWriteScope, &integration.ErrorStatus, &integration.Scopes,
		&integration.CreatedAt, &integration.UpdatedAt,
	); err != nil {
		return domain.Integration{}, err
	}
	integration.TokenExpiresAt = tokenExpiresAt
	return integration, nil
}

func scanCalendar(row scannable) (domain.Calendar, error) {
	var calendar domain.Calendar
	if err := row.Scan(
		&calendar.ID, &calendar.IntegrationID, &calendar.ExternalID, &calendar.Name,
		&calendar.BackgroundColor, &calendar.ForegroundColor, &calendar.Selected,
		&calendar.CreatedAt, &calendar.UpdatedAt,
	); err != nil {
		return domain.Calendar{}, err
	}
	return calendar, nil
}
