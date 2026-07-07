package bootstrap

import (
	"net/http"
	"strings"

	httpapp "opentoggl/backend/apps/backend/internal/http"

	"github.com/labstack/echo/v4"
)

// calendarOAuthErrorRedirect is where a failed connect attempt lands; the SPA
// surfaces it as a toast, mirroring samlLoginErrorRedirect.
const calendarOAuthErrorRedirect = "/profile?calendar_connect_error=1"

// newCalendarRoutes registers the Google OAuth callback as a plain browser
// route, outside the public-track OpenAPI surface — Google redirects here
// directly, so this can't be a typed JSON API operation. It mirrors
// /auth/sso/resolve's status as the one sanctioned exception to routing
// everything through a generated client.
func newCalendarRoutes(handlers *routeHandlers) httpapp.RouteRegistrar {
	return func(server *echo.Echo) {
		server.GET("/integrations/calendar/callback/google", handlers.calendarOAuthCallback)
	}
}

func (handlers *routeHandlers) calendarOAuthCallback(ctx echo.Context) error {
	requestCtx := ctx.Request().Context()
	code := ctx.QueryParam("code")
	state := ctx.QueryParam("state")
	if code == "" || state == "" {
		return ctx.Redirect(http.StatusFound, calendarOAuthErrorRedirect)
	}

	returnTo, err := handlers.calendarApp.CompleteCallback(requestCtx, "google", code, state, handlers.calendarCallbackURL(ctx))
	if err != nil {
		return ctx.Redirect(http.StatusFound, calendarOAuthErrorRedirect)
	}
	if returnTo == "" {
		returnTo = "/profile"
	}
	return ctx.Redirect(http.StatusFound, returnTo)
}

// calendarCallbackURL is the exact redirect_uri Google must send the browser
// back to, derived from the same runtime-configurable site URL SAML uses.
func (handlers *routeHandlers) calendarCallbackURL(ctx echo.Context) string {
	return strings.TrimRight(handlers.samlBaseURL(ctx), "/") + "/integrations/calendar/callback/google"
}
