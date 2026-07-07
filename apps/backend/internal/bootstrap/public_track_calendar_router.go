package bootstrap

import (
	publictrackapi "opentoggl/backend/apps/backend/internal/http/generated/publictrack"

	"github.com/labstack/echo/v4"
)

func (server *publicTrackOpenAPIServer) GetIntegrationsCalendar(ctx echo.Context) error {
	return server.calendar.GetIntegrationsCalendar(ctx)
}

func (server *publicTrackOpenAPIServer) GetIntegrationsCalendarSetup(
	ctx echo.Context,
	params publictrackapi.GetIntegrationsCalendarSetupParams,
) error {
	return server.calendar.GetIntegrationsCalendarSetup(ctx, params)
}

func (server *publicTrackOpenAPIServer) DeleteIntegrationsCalendarIntegrationId(ctx echo.Context, integrationId int) error {
	return server.calendar.DeleteIntegrationsCalendarIntegrationId(ctx, integrationId)
}

func (server *publicTrackOpenAPIServer) GetIntegrationsCalendarIntegrationIdCalendars(
	ctx echo.Context,
	integrationId int,
	params publictrackapi.GetIntegrationsCalendarIntegrationIdCalendarsParams,
) error {
	return server.calendar.GetIntegrationsCalendarIntegrationIdCalendars(ctx, integrationId, params)
}

func (server *publicTrackOpenAPIServer) PatchIntegrationsCalendarIntegrationIdCalendarsCalendarId(
	ctx echo.Context,
	integrationId int,
	calendarId int,
) error {
	return server.calendar.PatchIntegrationsCalendarIntegrationIdCalendarsCalendarId(ctx, integrationId, calendarId)
}

func (server *publicTrackOpenAPIServer) GetIntegrationsCalendarEvents(
	ctx echo.Context,
	params publictrackapi.GetIntegrationsCalendarEventsParams,
) error {
	return server.calendar.GetIntegrationsCalendarEvents(ctx, params)
}
