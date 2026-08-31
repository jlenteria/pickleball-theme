package theme

import (
	"fmt"

	"com.flarego.pickleball-theme/app/pickleball"
)

// accentStyleTag builds a complete inline <style>...</style> element that
// re-declares this theme's brand CSS custom properties (see
// resources/assets/shared/css/brand.css) for the CURRENTLY SELECTED theme.
// Every admin and portal page render emits it, right after the theme's own
// stylesheet, so an admin switching the active theme repaints both surfaces
// on the very next page load -- no CSS rebuild, no plugin restart.
//
// It returns the WHOLE tag, not just its body, and every layout renders it
// via a top-level `@templ.Raw(...)` call rather than a literal <style> tag
// wrapping a templ expression. HTML treats <style> (like <script>) as a raw
// text element -- its content is never parsed as further markup, templ
// expressions included -- so `<style>@templ.Raw(x)</style>` renders the
// literal text "@templ.Raw(x)" instead of evaluating it. Building the tag
// here and Raw-ing it as one opaque block is the same pattern core's own
// activation page uses for the identical reason.
//
// Every value comes from the pickleball.Theme registry (never user input),
// so building it as a plain fmt.Sprintf string and rendering it via
// templ.Raw is safe.
func accentStyleTag(t pickleball.Theme) string {
	return fmt.Sprintf(
		`<style>:root{--brand-primary:%s;--brand-primary-rgb:%s;--brand-primary-dark:%s;--brand-primary-soft:%s;--brand-accent:%s;--brand-accent-rgb:%s;--brand-gradient:linear-gradient(135deg,%s 0%%,%s 100%%);}</style>`,
		t.Primary, t.PrimaryRGB, t.PrimaryDark, t.PrimarySoft, t.Accent, t.AccentRGB, t.Primary, t.PrimaryDark,
	)
}
