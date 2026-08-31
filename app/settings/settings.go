// Package settings persists and resolves this theme's admin-editable
// branding: the selected theme, an optional custom club/venue name, and an
// optional custom logo.
//
// This is the ONE source both the admin dashboard and the captive portal
// read from (see app/theme/admin.go and app/theme/portal.go) -- it is what
// keeps the two surfaces aligned on the same theme. There is exactly one of
// this, so it lives in the plugin's JSON config store rather than a SQLite
// table -- see .claude/rules/flarewifi-database-conventions.md.
//
// The uploaded logo is stored as a file in the plugin storage area
// (api.Storage()), and only its stored filename is recorded here. Keeping
// the upload out of resources/assets/public is deliberate: that tree ships
// with the plugin and is replaced on every plugin update/reinstall, which
// would silently wipe the admin's custom logo. Storage survives updates, and
// an empty filename always means "fall back to the selected theme's bundled
// default logo".
package settings

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/pickleball"
)

const configKey = "branding_settings"

// Settings is the admin-editable branding.
type Settings struct {
	// ThemeID selects the active pickleball.Theme. Empty (a fresh install)
	// is resolved as pickleball.Default() by Theme().
	ThemeID string `json:"theme_id"`
	// ClubName overrides the theme's own Name as the branding shown to
	// guests and admins (e.g. "Riverside Pickleball Club"). Empty falls
	// back to the theme's Name.
	ClubName string `json:"club_name"`
	// LogoFile is the stored filename of a custom logo image in plugin
	// storage. Empty means "use the selected theme's bundled default logo".
	LogoFile       string          `json:"logo_file"`
	CourtRates     []CourtRate     `json:"court_rates"`
	BookingURL     string          `json:"booking_url"`
	Tutorials      []Tutorial      `json:"tutorials"`
	BannerFile     string          `json:"banner_file"`
	AffiliateLinks []AffiliateLink `json:"affiliate_links"`
	Game           GameSettings    `json:"game"`
}

// GameSettings controls the lightweight, client-side rally game shown to
// captive-portal guests. It is presentation-only and never changes wifi credits.
type GameSettings struct {
	Enabled     bool   `json:"enabled"`
	Title       string `json:"title"`
	TargetScore int    `json:"target_score"`
	Difficulty  string `json:"difficulty"`
}

const (
	defaultGameTitle       = "Dink Rally"
	defaultGameTargetScore = 7
	defaultGameDifficulty  = "medium"
)

const (
	maxClubNameRunes  = 80
	maxGameTitleRunes = 60
	maxCourtRates     = 100
	maxTutorials      = 50
	maxAffiliateLinks = 100
)

// CourtRate is a guest-facing pickleball court/session price configured by
// the venue operator.
type CourtRate struct {
	Name        string  `json:"name"`
	DurationMin int     `json:"duration_min"`
	Price       float64 `json:"price"`
}

// Tutorial is a guest-facing pickleball learning resource configured by the
// venue operator.
type Tutorial struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	VideoFile string `json:"video_file"`
}

// AffiliateLink is a venue-curated recommendation for pickleball gear or
// services, optionally accompanied by a preview image.
type AffiliateLink struct {
	Title     string `json:"title"`
	Category  string `json:"category"`
	URL       string `json:"url"`
	ImageURL  string `json:"image_url"`
	ImageFile string `json:"image_file"`
}

func AffiliateLinksWithImages(api sdkapi.IPluginApi, s Settings) []AffiliateLink {
	if len(s.AffiliateLinks) == 0 {
		return nil
	}

	links := make([]AffiliateLink, 0, len(s.AffiliateLinks))
	for _, link := range s.AffiliateLinks {
		// Skip invalid links
		if link.Title == "" || link.URL == "" {
			continue
		}

		// Resolve uploaded image file to URL
		if link.ImageFile != "" && api.Storage().Exists(link.ImageFile) {
			link.ImageURL = api.Storage().UrlFor(link.ImageFile)
		}

		links = append(links, link)
	}
	return links
}

// TutorialsWithVideos resolves uploaded tutorial video URLs.
func TutorialsWithVideos(api sdkapi.IPluginApi, s Settings) []Tutorial {
	if len(s.Tutorials) == 0 {
		return nil
	}

	tutorials := make([]Tutorial, 0, len(s.Tutorials))
	for _, tutorial := range s.Tutorials {
		// Skip invalid tutorials
		if tutorial.Title == "" {
			continue
		}

		// Resolve uploaded video file to URL
		if tutorial.VideoFile != "" && api.Storage().Exists(tutorial.VideoFile) {
			tutorial.URL = api.Storage().UrlFor(tutorial.VideoFile)
		}

		tutorials = append(tutorials, tutorial)
	}
	return tutorials
}

// BannerURL resolves the uploaded venue banner, falling back to the bundled
// Pickleball banner when no custom image is configured.
func BannerURL(api sdkapi.IPluginApi, s Settings) string {
	if s.BannerFile != "" && api.Storage().Exists(s.BannerFile) {
		return api.Storage().UrlFor(s.BannerFile)
	}
	return api.Http().Helpers().PublicPath("images/pickleball-banner.svg")
}

// Get returns the saved settings, or a zero-value Settings (all defaults)
// when nothing has been saved yet or the stored blob is unreadable.
func Get(api sdkapi.IPluginApi) Settings {
	var s Settings

	b, err := api.Config().Plugin().Read(configKey)
	if err != nil {
		// No saved settings yet - return defaults, including an enabled game.
		s.Game.Enabled = true
		return sanitizeSettings(s)
	}

	if err := json.Unmarshal(b, &s); err != nil {
		api.Logger().Error("pickleball-theme: failed to unmarshal settings, using defaults: " + err.Error())
		return sanitizeSettings(Settings{Game: GameSettings{Enabled: true}})
	}
	var raw struct {
		Game *json.RawMessage `json:"game"`
	}
	if json.Unmarshal(b, &raw) == nil && raw.Game == nil {
		s.Game.Enabled = true
	}

	// Validate and sanitize loaded settings
	s = sanitizeSettings(s)

	return s
}

// sanitizeSettings validates and cleans up loaded settings
func sanitizeSettings(s Settings) Settings {
	if strings.TrimSpace(s.Game.Title) == "" {
		s.Game.Title = defaultGameTitle
	}
	if utf8.RuneCountInString(s.Game.Title) > maxGameTitleRunes {
		s.Game.Title = truncateRunes(s.Game.Title, maxGameTitleRunes)
	}
	if s.Game.TargetScore < 1 || s.Game.TargetScore > 21 {
		s.Game.TargetScore = defaultGameTargetScore
	}
	if s.Game.Difficulty != "easy" && s.Game.Difficulty != "medium" && s.Game.Difficulty != "hard" {
		s.Game.Difficulty = defaultGameDifficulty
	}
	// Ensure reasonable limits on arrays to prevent memory issues
	if len(s.CourtRates) > maxCourtRates {
		s.CourtRates = s.CourtRates[:maxCourtRates]
	}
	if len(s.Tutorials) > maxTutorials {
		s.Tutorials = s.Tutorials[:maxTutorials]
	}
	if len(s.AffiliateLinks) > maxAffiliateLinks {
		s.AffiliateLinks = s.AffiliateLinks[:maxAffiliateLinks]
	}
	if utf8.RuneCountInString(s.ClubName) > maxClubNameRunes {
		s.ClubName = truncateRunes(s.ClubName, maxClubNameRunes)
	}

	return s
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// Save persists the given settings.
func Save(api sdkapi.IPluginApi, s *Settings) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return api.Config().Plugin().Write(configKey, b)
}

// Theme resolves the currently active theme, falling back to
// pickleball.Default() for a fresh or unrecognized ThemeID.
func Theme(s Settings) pickleball.Theme {
	return pickleball.Get(s.ThemeID)
}

// DisplayName resolves the branding name shown to guests and admins: the
// admin's own club/venue name when set, otherwise the active theme's Name.
func DisplayName(s Settings, theme pickleball.Theme) string {
	if name := strings.TrimSpace(s.ClubName); name != "" {
		return name
	}
	return theme.Name
}

// LogoURL resolves the logo image URL: the uploaded custom logo when
// present, otherwise the active theme's bundled default. The Exists check
// means a config entry left pointing at a file that was removed out from
// under it degrades to the theme default rather than rendering a broken
// image.
func LogoURL(api sdkapi.IPluginApi, s Settings, theme pickleball.Theme) string {
	if s.LogoFile != "" && api.Storage().Exists(s.LogoFile) {
		return api.Storage().UrlFor(s.LogoFile)
	}
	return api.Http().Helpers().PublicPath(theme.DefaultLogo)
}
