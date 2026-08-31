package handlers

import (
	"fmt"
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/settings"
	"com.flarego.pickleball-theme/resources/views/portal"
)

// PortalStatusBarCtrl backs the portal layout's status-bar htmx slot.
func PortalStatusBarCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// No client device means there is no connection state to report --
		// an admin on the login or forgot-password page, which share the
		// portal layout. Render nothing rather than a wrong "Disconnected",
		// so those pages collapse the slot instead of showing a banner about
		// a device that isn't the one browsing. Debug, not Error: this is an
		// expected condition on those pages, not a failure.
		clnt, err := api.Http().GetClientDevice(r)
		if err != nil {
			api.Logger().Debug(fmt.Sprintf("pickleball-theme status-bar: no client device, omitting banner: %v", err))
			return
		}

		// IsConnectedCached, not IsConnected -- see LayoutBuilder's own
		// comment in app/theme/portal.go for the full reasoning.
		view := portal.StatusBar(api, clnt.Session().IsConnectedCached())
		view.Render(r.Context(), w)
	}
}

// CourtRatesCtrl renders the venue's public pickleball court rates.
func CourtRatesCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := settings.Get(api)
		api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
			PageContent: portal.VenueInfoPage(api, cfg.CourtRates, cfg.BookingURL, settings.TutorialsWithVideos(api, cfg), settings.AffiliateLinksWithImages(api, cfg)),
		})
	}
}

// PortalInternetStatusCtrl backs the portal index's internet-status htmx slot.
func PortalInternetStatusCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view := portal.InternetStatus(api, api.Machine().IsOnline())
		view.Render(r.Context(), w)
	}
}

// PortalSessionInfoCtrl backs the portal index's session-info htmx slot.
func PortalSessionInfoCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, summary, isRunning := getSessionInfo(api, r)
		clnt, _ := api.Http().GetClientDevice(r)

		view := portal.SessionInfo(api, clnt, session, summary, isRunning)
		if err := view.Render(r.Context(), w); err != nil {
			api.Logger().Error(fmt.Sprintf("pickleball-theme: failed to render session info: %v", err))
		}
	}
}

// getSessionInfo retrieves session data with graceful fallback
func getSessionInfo(api sdkapi.IPluginApi, r *http.Request) (sdkapi.SessionData, sdkapi.SessionSummary, bool) {
	var session sdkapi.SessionData
	var summary sdkapi.SessionSummary
	var isRunning bool

	clnt, err := api.Http().GetClientDevice(r)
	if err != nil {
		api.Logger().Debug(fmt.Sprintf("pickleball-theme session-info: no client device: %v", err))
		return session, summary, false
	}

	// IsConnectedCached, not IsConnected -- see PortalStatusBarCtrl above.
	isRunning = clnt.Session().IsConnectedCached()
	if !isRunning {
		return session, summary, false
	}

	session = clnt.Session().SessionData()

	// Never fail this partial over a reporting query -- fall back to the
	// active session's own totals on error.
	summary, sErr := clnt.SessionSummary(r.Context())
	if sErr != nil {
		api.Logger().Debug(fmt.Sprintf("pickleball-theme: session summary query failed, using session data: %v", sErr))
		summary = buildSummaryFromSession(session, isRunning)
	}

	return session, summary, isRunning
}

// buildSummaryFromSession creates a fallback summary from session data
func buildSummaryFromSession(session sdkapi.SessionData, isRunning bool) sdkapi.SessionSummary {
	summary := sdkapi.SessionSummary{
		IsConnected: isRunning,
		IsPaused:    session.IsPaused,
		UpdatedAt:   session.UpdatedAt,
	}

	if session.EnforcesTime() {
		summary.TimeSecs = session.TimeSecs
	}
	if session.EnforcesData() {
		summary.DataMb = session.DataMb
	}

	return summary
}
