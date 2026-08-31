package admin

import (
	"fmt"
	"net/url"
	"strings"

	sdkapi "sdk/api"
)

func isYouTubeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" || host == "youtu.be"
}

func getYouTubeEmbedURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Hostname() == "youtu.be" {
		return "https://www.youtube.com/embed/" + strings.Trim(u.Path, "/")
	}
	if id := u.Query().Get("v"); id != "" {
		return "https://www.youtube.com/embed/" + id
	}
	return raw
}

// swatchStyle builds the inline background-color style for a sport color
// swatch on the portal-branding settings page.
func swatchStyle(hex string) string {
	return fmt.Sprintf("background-color: %s", hex)
}

// currentCurrency returns the machine's configured ISO currency code. Rates
// are formatted through Payments().FormatCurrency in the template so this
// label and the displayed amounts come from the same application setting.
func currentCurrency(api sdkapi.IPluginApi) string {
	cfg, err := api.Config().Application().Get()
	if err != nil || cfg.Currency == "" {
		return "USD"
	}
	return cfg.Currency
}
