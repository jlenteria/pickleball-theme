// Package pickleball is the registry of themes this plugin can brand itself
// as.
//
// A Theme carries everything both the admin dashboard and the captive
// portal need to render identically for it: display identity, the accent
// palette that recolors Bootstrap on both surfaces, and the bundled default
// logo. Admin and portal always resolve the CURRENTLY SELECTED theme from
// the same settings.Settings.ThemeID (see the settings package), so the two
// surfaces can never disagree about which theme is active.
//
// Only Pickleball ships today. Add a new theme by appending an entry to
// registry -- nothing else in this plugin needs to change to make it
// selectable on the settings page.
package pickleball

// Theme describes one selectable theme.
type Theme struct {
	// ID is the stable identifier persisted in settings.Settings.ThemeID.
	// Never change an existing ID once shipped -- a saved config referencing
	// a since-renamed ID would silently fall back to Default().
	ID string
	// Name is the theme's display name, used as the default branding name
	// when the admin hasn't set a custom club/venue name.
	Name string
	// TagLine is a short phrase shown under the logo on the portal home card.
	TagLine string
	// Icon is a bootstrap-icons class (e.g. "bi-circle-fill") used where a
	// small inline icon stands in for the logo.
	Icon string
	// DefaultLogo is a path under resources/assets/public, served as the
	// bundled logo until the admin uploads a custom one.
	DefaultLogo string

	// Primary/PrimaryRGB/PrimaryDark/PrimarySoft/Accent/AccentRGB are the
	// accent palette. Primary recolors Bootstrap's `primary` slot (buttons,
	// links, focus rings) on both surfaces -- see resources/assets/shared/css.
	// PrimaryRGB/AccentRGB are the same colors as bare "r, g, b" triples,
	// for rgba()/--bs-*-rgb consumers that can't derive channels from a hex.
	Primary     string
	PrimaryRGB  string
	PrimaryDark string
	PrimarySoft string
	Accent      string
	AccentRGB   string
}

// registry is every theme this plugin can render, in display order.
var registry = []Theme{
	{
		ID:          "pickleball",
		Name:        "Pickleball",
		TagLine:     "Dink. Drive. Connect.",
		Icon:        "bi-circle-fill",
		DefaultLogo: "images/pickleball-logo.svg",

		// A teal court/paddle primary with a pickleball-yellow accent --
		// sampled to sit well on both a white admin dashboard and a dark
		// portal backdrop.
		Primary:     "#0E7C7B",
		PrimaryRGB:  "14, 124, 123",
		PrimaryDark: "#0A5958",
		PrimarySoft: "#E3F5F4",
		Accent:      "#C6E92B",
		AccentRGB:   "198, 233, 43",
	},
}

// All returns every selectable theme, in display order.
func All() []Theme {
	return registry
}

// Default is the theme a fresh install starts on.
func Default() Theme {
	return registry[0]
}

// Get resolves a theme by ID, falling back to Default() for an empty or
// unrecognized ID -- e.g. one saved by a build that shipped a theme this
// one no longer carries.
func Get(id string) Theme {
	for _, t := range registry {
		if t.ID == id {
			return t
		}
	}
	return Default()
}
