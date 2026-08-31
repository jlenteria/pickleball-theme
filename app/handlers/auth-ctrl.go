package handlers

import (
	"net/http"

	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/settings"
	"com.flarego.pickleball-theme/resources/views/portal"
)

// AdminLoginPageCtrl re-renders the login page. It backs the GET on the
// plugin's own plain /login route -- there for direct navigation to this
// URL (e.g. a stale bookmark); a normal login flow only ever POSTs here,
// with no GET round-trip in between. Already-authenticated admins are sent
// straight to the dashboard.
func AdminLoginPageCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := api.Http().Auth().IsAuthenticated(r); err == nil {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		cfg := settings.Get(api)
		theme := settings.Theme(cfg)
		csrfHTML := api.Http().Helpers().CsrfHtmlTag(r)
		api.Http().Response().PortalView(w, r, sdkapi.ViewPage{
			PageContent: portal.PortalLoginPage(
				api, csrfHTML,
				settings.DisplayName(cfg, theme), settings.LogoURL(api, cfg, theme),
				sdkapi.LoginPageData{LoginLinks: api.Http().Navs().GetLoginLinks(r)},
			),
		})
	}
}

// AdminAuthenticateCtrl authenticates an admin sign-in submitted from the
// captive portal login page. It is registered on the plugin's plain
// router, not HTTPS-restricted, so it works whether or not
// AppConfig.ForceAdminHttps is on -- the admin dashboard is only forced
// onto HTTPS by middlewares.ForceHTTPS when that setting is enabled, and
// this route must not impose a second, unconditional HTTPS requirement of
// its own.
func AdminAuthenticateCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			api.Http().Response().FlashMsg(w, r, api.Translate("error", "Invalid form data"), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		acct, err := api.Http().Auth().AuthenticateAdminLogin(r, username, password)
		if err != nil {
			// Do NOT html.EscapeString here -- err.Error() renders through
			// templ's own auto-escaping on the login page; pre-escaping
			// here would double-escape it.
			api.Http().Response().FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		api.Http().Auth().SignIn(w, acct)
		api.Http().Response().FlashMsg(w, r, api.Translate("info", "Logged in successfully"), sdkapi.FlashMsgSuccess)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}
