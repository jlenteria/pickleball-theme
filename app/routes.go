package app

import (
	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/handlers"
)

func SetupRoutes(api sdkapi.IPluginApi) {
	adminR := api.Http().Router().AdminRouter(nil)
	portalR := api.Http().Router().HttpRouter(nil)

	// Admin login is served on the plugin's plain router, not HttpsOnly:true
	// -- RootRouter is shared by both the plain-HTTP and HTTPS listeners, so
	// this already reaches HTTPS too whenever the page that rendered the
	// form was itself served over HTTPS. Whether the login page/form is
	// forced onto HTTPS at all is decided once, globally, by
	// middlewares.ForceHTTPS (AppConfig.ForceAdminHttps) -- this route must
	// not impose a second, unconditional HTTPS requirement on top of that.
	// GET re-renders the login form (e.g. if this URL is ever visited
	// directly); POST authenticates.
	portalR.Get("/login", handlers.AdminLoginPageCtrl(api)).Name("auth:login-page")
	portalR.Post("/login", handlers.AdminAuthenticateCtrl(api)).Name("auth:login")

	// Theme + branding settings. Rendered inside whichever admin theme is
	// active.
	adminR.Group("/pickleball-theme", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/settings", handlers.ShowSettingsCtrl(api)).Name("admin:pickleball-theme:settings")
		subrouter.Post("/settings", handlers.SaveSettingsCtrl(api)).Name("admin:pickleball-theme:settings:save")
	})

	portalR.Group("/pickleball-theme", func(subrouter sdkapi.IHttpRouterInstance) {
		subrouter.Get("/status-bar", handlers.PortalStatusBarCtrl(api)).Name("portal-theme:pickleball-theme:status-bar")
		subrouter.Get("/session-info", handlers.PortalSessionInfoCtrl(api)).Name("portal-theme:pickleball-theme:session-info")
		subrouter.Get("/internet-status", handlers.PortalInternetStatusCtrl(api)).Name("portal-theme:pickleball-theme:internet-status")
		subrouter.Get("/court-rates", handlers.CourtRatesCtrl(api)).Name("portal-theme:pickleball-theme:court-rates")
		subrouter.Post("/voucher", handlers.PortalVoucherCtrl(api)).Name("portal-theme:pickleball-theme:voucher")
	})
}
