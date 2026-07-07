package publicapi

import (
	"errors"
	"net/http"
	"time"

	calendarapplication "opentoggl/backend/apps/backend/internal/calendar/application"
	"opentoggl/backend/apps/backend/internal/calendar/domain"
	publictrackapi "opentoggl/backend/apps/backend/internal/http/generated/publictrack"
	identityapplication "opentoggl/backend/apps/backend/internal/identity/application"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
)

// ScopeAuthorizer resolves the current public-track API caller. routeHandlers
// in bootstrap implements this already for every other public-api handler.
type ScopeAuthorizer interface {
	RequirePublicTrackUser(ctx echo.Context) (*identityapplication.UserSnapshot, error)
}

type Handler struct {
	calendarApp *calendarapplication.Service
	scope       ScopeAuthorizer
	redirectURL func(echo.Context) string
}

// NewHandler builds the calendar transport handler. redirectURL returns the
// exact OAuth callback URL for the current request (derived from the
// instance's runtime-configurable site URL, e.g.
// `<site_url>/integrations/calendar/callback/google`) — it must match
// whatever the raw callback route (registered outside this OpenAPI surface,
// mirroring /auth/sso/resolve) computes for the same request.
func NewHandler(calendarApp *calendarapplication.Service, scope ScopeAuthorizer, redirectURL func(echo.Context) string) *Handler {
	return &Handler{calendarApp: calendarApp, scope: scope, redirectURL: redirectURL}
}

func (handler *Handler) GetIntegrationsCalendar(ctx echo.Context) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}

	integrations, err := handler.calendarApp.ListIntegrations(ctx.Request().Context(), user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error").SetInternal(err)
	}

	response := make([]publictrackapi.GithubComTogglTogglApiInternalModelsIntegration, 0, len(integrations))
	for _, integration := range integrations {
		response = append(response, integrationToAPI(integration))
	}
	return ctx.JSON(http.StatusOK, response)
}

func (handler *Handler) GetIntegrationsCalendarSetup(
	ctx echo.Context,
	params publictrackapi.GetIntegrationsCalendarSetupParams,
) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}

	returnTo := "/profile"
	if params.ReturnTo != nil && *params.ReturnTo != "" {
		returnTo = *params.ReturnTo
	}

	authURL, err := handler.calendarApp.BeginSetup(ctx.Request().Context(), user.ID, params.Provider, returnTo, handler.redirectURL(ctx))
	if err != nil {
		return calendarHTTPError(err)
	}
	return ctx.Redirect(http.StatusFound, authURL)
}

func (handler *Handler) DeleteIntegrationsCalendarIntegrationId(ctx echo.Context, integrationId int) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}
	if err := handler.calendarApp.DisconnectIntegration(ctx.Request().Context(), user.ID, int64(integrationId)); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error").SetInternal(err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (handler *Handler) GetIntegrationsCalendarIntegrationIdCalendars(
	ctx echo.Context,
	integrationId int,
	params publictrackapi.GetIntegrationsCalendarIntegrationIdCalendarsParams,
) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}

	calendars, err := handler.calendarApp.ListCalendars(ctx.Request().Context(), user.ID, int64(integrationId))
	if err != nil {
		return calendarHTTPError(err)
	}
	if params.Selected != nil {
		calendars = filterCalendarsBySelected(calendars, *params.Selected)
	}

	response := make([]publictrackapi.GithubComTogglTogglApiInternalModelsCalendar, 0, len(calendars))
	for _, calendar := range calendars {
		response = append(response, calendarToAPI(calendar))
	}
	return ctx.JSON(http.StatusOK, publictrackapi.HandlercalendarCalendarsResponse{
		Calendars: &response,
	})
}

func (handler *Handler) PatchIntegrationsCalendarIntegrationIdCalendarsCalendarId(
	ctx echo.Context,
	integrationId int,
	calendarId int,
) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}

	var payload publictrackapi.HandlercalendarPatchCalendar
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Bad Request")
	}
	if payload.Selected == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "the payload should not be empty")
	}

	calendar, err := handler.calendarApp.SetCalendarSelected(
		ctx.Request().Context(), user.ID, int64(integrationId), int64(calendarId), *payload.Selected,
	)
	if err != nil {
		return calendarHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, []publictrackapi.GithubComTogglTogglApiInternalModelsCalendar{calendarToAPI(calendar)})
}

func (handler *Handler) GetIntegrationsCalendarEvents(
	ctx echo.Context,
	params publictrackapi.GetIntegrationsCalendarEventsParams,
) error {
	user, err := handler.scope.RequirePublicTrackUser(ctx)
	if err != nil {
		return err
	}

	start, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "start_date has invalid format")
	}
	end, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "end_date has invalid format")
	}
	if !start.Before(end) {
		return echo.NewHTTPError(http.StatusBadRequest, "start_date should come before end_date")
	}

	events, err := handler.calendarApp.ListEvents(ctx.Request().Context(), user.ID, start, end)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error").SetInternal(err)
	}

	response := make([]publictrackapi.GithubComTogglTogglApiInternalModelsEvent, 0, len(events))
	for _, event := range events {
		response = append(response, eventToAPI(event))
	}
	return ctx.JSON(http.StatusOK, publictrackapi.HandlercalendarEventsResponse{
		Events: &response,
	})
}

func filterCalendarsBySelected(calendars []domain.Calendar, selected bool) []domain.Calendar {
	filtered := make([]domain.Calendar, 0, len(calendars))
	for _, calendar := range calendars {
		if calendar.Selected == selected {
			filtered = append(filtered, calendar)
		}
	}
	return filtered
}

func calendarHTTPError(err error) error {
	switch {
	case errors.Is(err, domain.ErrIntegrationNotFound), errors.Is(err, domain.ErrCalendarNotFound):
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrUnsupportedProvider):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid provider name")
	case errors.Is(err, domain.ErrNotConfigured):
		return echo.NewHTTPError(http.StatusBadRequest, "calendar provider is not configured on this instance")
	case errors.Is(err, domain.ErrInvalidState):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired oauth state")
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal Server Error").SetInternal(err)
	}
}

func integrationToAPI(integration domain.Integration) publictrackapi.GithubComTogglTogglApiInternalModelsIntegration {
	return publictrackapi.GithubComTogglTogglApiInternalModelsIntegration{
		CalendarIntegrationId: lo.ToPtr(int(integration.ID)),
		Provider:              lo.ToPtr(integration.Provider),
		Email:                 lo.ToPtr(integration.Email),
		HasWriteScope:         lo.ToPtr(integration.HasWriteScope),
		ErrorStatus:           lo.ToPtr(integration.ErrorStatus),
		Scopes:                lo.ToPtr(integration.Scopes),
		CreatedAt:             lo.ToPtr(integration.CreatedAt.Format(time.RFC3339)),
	}
}

func calendarToAPI(calendar domain.Calendar) publictrackapi.GithubComTogglTogglApiInternalModelsCalendar {
	return publictrackapi.GithubComTogglTogglApiInternalModelsCalendar{
		CalendarId:            lo.ToPtr(int(calendar.ID)),
		CalendarIntegrationId: lo.ToPtr(int(calendar.IntegrationID)),
		ExternalId:            lo.ToPtr(calendar.ExternalID),
		Name:                  lo.ToPtr(calendar.Name),
		BackgroundColor:       lo.ToPtr(calendar.BackgroundColor),
		ForegroundColor:       lo.ToPtr(calendar.ForegroundColor),
		Selected:              lo.ToPtr(calendar.Selected),
		CreatedAt:             lo.ToPtr(calendar.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:             lo.ToPtr(calendar.UpdatedAt.Format(time.RFC3339)),
	}
}

func eventToAPI(event domain.Event) publictrackapi.GithubComTogglTogglApiInternalModelsEvent {
	return publictrackapi.GithubComTogglTogglApiInternalModelsEvent{
		CalendarId: lo.ToPtr(int(event.CalendarID)),
		ExternalId: lo.ToPtr(event.ExternalID),
		Title:      lo.ToPtr(event.Title),
		Provider:   lo.ToPtr(event.Provider),
		StartTime:  lo.ToPtr(event.StartTime.Format(time.RFC3339)),
		EndTime:    lo.ToPtr(event.EndTime.Format(time.RFC3339)),
		AllDay:     lo.ToPtr(event.AllDay),
		HtmlLink:   lo.ToPtr(event.HTMLLink),
		IcalUid:    lo.ToPtr(event.ICalUID),
	}
}
