package navs

import (
	"net/http"

	sdkapi "sdk/api"
)

// SetAdminNavs registers this plugin's settings entry under the admin
// Themes category. Always shown, regardless of whether this plugin is the
// currently active admin/portal theme -- an operator can configure the
// theme ahead of switching to it.
func SetAdminNavs(api sdkapi.IPluginApi) {
	api.Http().Navs().AdminNavsFactory(func(r *http.Request) []sdkapi.AdminNavItemOpt {
		return []sdkapi.AdminNavItemOpt{
			{
				Category:  sdkapi.NavCategoryThemes,
				Label:     api.Translate("label", "Pickleball Theme"),
				RouteName: "admin:pickleball-theme:settings",
				Icon:      "<svg width='22' height='22' viewBox='0 0 22 22' aria-hidden='true' focusable='false'><circle cx='11' cy='11' r='9' fill='currentColor' opacity='0.9'/><circle cx='8' cy='7' r='1.1' fill='currentColor'/><circle cx='14.5' cy='8.5' r='1.1' fill='currentColor'/><circle cx='9' cy='14.5' r='1.1' fill='currentColor'/><circle cx='15' cy='14' r='1.1' fill='currentColor'/></svg>",
				Order:     5000,
				Keywords: []string{
					api.Translate("label", "theme"),
					api.Translate("label", "pickleball"),
					api.Translate("label", "branding"),
				},
			},
		}
	})
}
