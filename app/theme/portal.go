package theme

import (
	"fmt"
	"net/http"
	"strings"

	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/settings"
	"com.flarego.pickleball-theme/resources/views/portal"
)

// authPageSuffixes are pages that share this portal layout but belong to an
// ADMIN account rather than a client device, so the connection banner is
// meaningless on them. Matching on suffix rather than a fixed prefix keeps
// this correct regardless of which plugin's own route prefix serves these
// pages (e.g. the admin-security plugin's forgot-password flow).
var authPageSuffixes = []string{
	"/login",
	"/send-otp",
	"/verify-otp",
	"/reset-password",
}

func isAuthPage(path string) bool {
	for _, suffix := range authPageSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// SetPortalTheme registers the Pickleball Theme captive portal: a single
// card centered on the currently selected theme's gradient backdrop -- see
// app/settings and app/pickleball.
func SetPortalTheme(api sdkapi.IPluginApi) {
	api.Themes().NewPortalTheme(sdkapi.PortalThemeOpts{
		JsFile:  "theme.js",
		CssFile: "theme.css",
		// Theme-only, cross-page Bootstrap accent remap -- loads only while
		// this theme is the active portal theme (unlike a global.css entry).
		ThemeCssFiles: []string{"theme-accent.css"},
		CssLib:        sdkapi.CssLibBootstrap5,
		PreviewImage:  "images/portal-preview.png",
		LayoutBuilder: func(w http.ResponseWriter, r *http.Request, c sdkapi.IThemeComponents) {
			cfg := settings.Get(api)
			theme := settings.Theme(cfg)

			// Deliberately resolves NO client device here -- this builder runs
			// on every portal page render, and a real connection check shells
			// out to `nft` through a shared, 30s-ceiling worker queue. A
			// banner the SSE refresh corrects a moment later is not worth
			// putting on the render path of every page; the layout emits an
			// empty slot instead and htmx's `load` trigger fills it from the
			// status-bar endpoint that does this work.
			data := portal.PortalLayoutData{
				API:           api,
				Components:    c,
				BrandName:     settings.DisplayName(cfg, theme),
				ThemeID:       theme.ID,
				AccentStyle:   accentStyleTag(theme),
				ShowStatusBar: !isAuthPage(r.URL.Path),
			}
			if err := portal.PortalLayout(data).Render(r.Context(), w); err != nil {
				fmt.Fprintf(w, "<p>Error rendering portal layout: %s</p>", err.Error())
			}
		},
		LoginPageFactory: func(w http.ResponseWriter, r *http.Request, data sdkapi.LoginPageData) sdkapi.ViewPage {
			cfg := settings.Get(api)
			theme := settings.Theme(cfg)
			csrfHTML := api.Http().Helpers().CsrfHtmlTag(r)
			return sdkapi.ViewPage{
				PageContent: portal.PortalLoginPage(api, csrfHTML, settings.DisplayName(cfg, theme), settings.LogoURL(api, cfg, theme), data),
			}
		},
		IndexPageFactory: func(w http.ResponseWriter, r *http.Request) sdkapi.ViewPage {
			cfg := settings.Get(api)
			theme := settings.Theme(cfg)
			navs := api.Http().Navs().GetPortalItems(r)

			indexData := portal.PortalIndexData{
				Navs:         navs,
				IsOnline:     api.Machine().IsOnline(),
				BrandName:    settings.DisplayName(cfg, theme),
				ThemeTagLine: theme.TagLine,
				LogoURL:      settings.LogoURL(api, cfg, theme),
				BannerURL:    settings.BannerURL(api, cfg),
				CourtRates:   cfg.CourtRates,
				BookingURL:   cfg.BookingURL,
				Tutorials:    settings.TutorialsWithVideos(api, cfg),
				Game:         cfg.Game,
			}

			clnt, err := api.Http().GetClientDevice(r)
			if err != nil {
				api.Logger().Error(fmt.Sprintf("pickleball-theme: get client device error: %v", err))
				page := portal.PortalIndexPage(api, indexData)
				return sdkapi.ViewPage{PageContent: page}
			}

			indexData.DeviceMac = clnt.MacAddr()
			indexData.DeviceIP = clnt.IpAddr()

			// IsConnectedCached, not IsConnected: the latter is a real
			// firewall round trip that must not sit on the render path --
			// see the LayoutBuilder comment above.
			isConnected := clnt.Session().IsConnectedCached()
			indexData.IsSessionRunning = isConnected
			if isConnected {
				indexData.Client = clnt
				session := clnt.Session().SessionData()
				indexData.Session = session

				// Never fail the whole page over a reporting query -- fall
				// back to the active session's own totals on error.
				summary, sErr := clnt.SessionSummary(r.Context())
				if sErr != nil {
					summary = sdkapi.SessionSummary{
						IsConnected: isConnected,
						IsPaused:    session.IsPaused,
						UpdatedAt:   session.UpdatedAt,
					}
					if session.EnforcesTime() {
						summary.TimeSecs = session.TimeSecs
					}
					if session.EnforcesData() {
						summary.DataMb = session.DataMb
					}
				}
				indexData.Summary = summary
			}

			page := portal.PortalIndexPage(api, indexData)
			return sdkapi.ViewPage{PageContent: page}
		},
	})
}
