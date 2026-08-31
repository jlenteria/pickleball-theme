package handlers

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	sdkapi "sdk/api"

	"com.flarego.pickleball-theme/app/settings"
	adminviews "com.flarego.pickleball-theme/resources/views/admin"
)

// routeSettingsIndex is the admin settings page route this controller
// renders and redirects back to.
const routeSettingsIndex = "admin:pickleball-theme:settings"

// logoStoreBase is the stored filename stem for an uploaded logo; the
// uploaded file's own extension is appended, so a fixed name replaces the
// previous upload in place instead of accumulating files in storage.
const logoStoreBase = "logo"

// bundledLogoPath is shipped with the plugin and is never stored in the
// writable plugin storage area.
const bundledLogoPath = "images/pickleball-logo.svg"

// errUnsupportedImage is returned when an uploaded file is not an allowed image type.
var errUnsupportedImage = errors.New("unsupported image type")

// allowedImageExts are the accepted logo upload extensions.
var allowedImageExts = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".svg":  true,
}

// Validation constants for form inputs
const (
	maxClubNameLen      = 80  // Maximum length for club/venue name
	maxTutorialTitleLen = 100 // Maximum length for tutorial titles
	maxAffiliateNameLen = 100 // Maximum length for affiliate recommendation titles
	maxCategoryLen      = 60  // Maximum length for affiliate categories
	maxSessionNameLen   = 80  // Maximum length for court session names
	maxImageUploadBytes = 2 << 20
	maxVideoUploadBytes = 25 << 20
	maxMultipartBytes   = 32 << 20
)

// ShowSettingsCtrl renders the theme + branding settings page inside the
// active admin theme.
func ShowSettingsCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := settings.Get(api)
		selected := settings.Theme(cfg)

		data := adminviews.SettingsData{
			ClubName:        cfg.ClubName,
			LogoURL:         settings.LogoURL(api, cfg, selected),
			HasCustomLogo:   cfg.LogoFile != "",
			SaveURL:         api.Http().Helpers().UrlForRoute(routeSettingsIndex + ":save"),
			DefaultName:     selected.Name,
			CourtRates:      cfg.CourtRates,
			BookingURL:      cfg.BookingURL,
			Tutorials:       cfg.Tutorials,
			BannerURL:       settings.BannerURL(api, cfg),
			HasCustomBanner: cfg.BannerFile != "",
			AffiliateLinks:  settings.AffiliateLinksWithImages(api, cfg),
			Game:            cfg.Game,
		}

		api.Http().Response().AdminView(w, r, sdkapi.ViewPage{
			Assets:      sdkapi.ViewAssets{CssFile: "settings.css", JsFile: "settings.js"},
			PageContent: adminviews.SettingsView(api, data),
		})
	}
}

// SaveSettingsCtrl persists the selected theme, an optional custom club
// name, and an optional uploaded logo (or resets the logo to the theme's
// bundled default when the reset checkbox is ticked).
func SaveSettingsCtrl(api sdkapi.IPluginApi) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := api.Http().Response()
		// Bound the complete multipart request as well as each individual
		// upload. This prevents oversized discarded parts from consuming
		// unbounded disk/temp-file space during ParseMultipartForm.
		r.Body = http.MaxBytesReader(w, r.Body, maxMultipartBytes)

		// 8 MiB in-memory threshold; larger parts spill to temp files. The
		// storage layer enforces the real per-file size cap on write.
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Invalid form data"), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}

		cfg := settings.Get(api)

		// Handle court rate operations
		if err := handleCourtRateOperations(api, r, &cfg); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}

		// Handle club name and booking URL
		if err := handleClubSettings(api, r, &cfg); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}
		if err := handleGameSettings(api, r, &cfg); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}

		// Handle tutorial operations
		if err := handleTutorialOperations(api, r, &cfg); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}
		// Handle affiliate link operations
		if err := handleAffiliateLinkOperations(api, r, &cfg); err != nil {
			res.FlashMsg(w, r, err.Error(), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}

		bannerName, err := resolveBannerField(api, r, cfg.BannerFile)
		if err != nil {
			res.FlashMsg(w, r, api.Translate("error", "Banner must be a PNG, JPG, WEBP, GIF, or SVG image"), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}
		cfg.BannerFile = bannerName

		name, err := resolveLogoField(api, r, cfg.LogoFile)
		if err != nil {
			if errors.Is(err, errUnsupportedImage) {
				res.FlashMsg(w, r, api.Translate("error", "Logo must be a PNG, JPG, WEBP, GIF, or SVG image"), sdkapi.FlashMsgError)
			} else {
				api.Logger().Error("pickleball-theme: failed to store logo image: " + err.Error())
				res.FlashMsg(w, r, api.Translate("error", "Unable to save the logo image"), sdkapi.FlashMsgError)
			}
			res.Redirect(w, r, routeSettingsIndex)
			return
		}
		cfg.LogoFile = name

		if err := settings.Save(api, &cfg); err != nil {
			api.Logger().Error("pickleball-theme: failed to save theme settings: " + err.Error())
			res.FlashMsg(w, r, api.Translate("error", "Unable to save theme settings"), sdkapi.FlashMsgError)
			res.Redirect(w, r, routeSettingsIndex)
			return
		}

		res.FlashMsg(w, r, api.Translate("success", "Pickleball theme settings saved"), sdkapi.FlashMsgSuccess)
		res.Redirect(w, r, routeSettingsIndex)
	}
}

func isHTTPURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func resolveBannerField(api sdkapi.IPluginApi, r *http.Request, current string) (string, error) {
	if r.FormValue("remove_banner") == "on" {
		if current != "" {
			_ = api.Storage().Delete(current)
		}
		return "", nil
	}
	file, header, err := r.FormFile("banner")
	if err != nil {
		return current, nil
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExts[ext] {
		return current, errUnsupportedImage
	}
	data, err := readUpload(file, maxImageUploadBytes)
	if err != nil || !validImageData(ext, data) {
		return current, errUnsupportedImage
	}
	newName := "banner" + ext
	if _, err := api.Storage().WriteReader(newName, bytes.NewReader(data)); err != nil {
		return current, err
	}
	if current != "" && current != newName {
		_ = api.Storage().Delete(current)
	}
	return newName, nil
}

func resolveTutorialUploadField(api sdkapi.IPluginApi, r *http.Request) (string, error) {
	file, header, err := r.FormFile("tutorial_video")
	if err != nil {
		return "", nil
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".mp4" && ext != ".webm" && ext != ".mov" {
		return "", errUnsupportedImage
	}
	data, err := readUpload(file, maxVideoUploadBytes)
	if err != nil || !validVideoData(ext, data) {
		return "", errUnsupportedImage
	}
	newName := "tutorial-video-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	if _, err := api.Storage().WriteReader(newName, bytes.NewReader(data)); err != nil {
		return "", err
	}
	return newName, nil
}

func resolveAffiliateImageField(api sdkapi.IPluginApi, r *http.Request) (string, error) {
	file, header, err := r.FormFile("affiliate_image")
	if err != nil {
		return "", nil
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExts[ext] {
		return "", errUnsupportedImage
	}
	data, err := readUpload(file, maxImageUploadBytes)
	if err != nil || !validImageData(ext, data) {
		return "", errUnsupportedImage
	}
	newName := "affiliate-image-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	if _, err := api.Storage().WriteReader(newName, bytes.NewReader(data)); err != nil {
		return "", err
	}
	return newName, nil
}

// =============================================================================
// HELPER FUNCTIONS (internal)
// =============================================================================

// handleClubSettings validates and updates club name and booking URL
func handleClubSettings(api sdkapi.IPluginApi, r *http.Request, cfg *settings.Settings) error {
	clubName := strings.TrimSpace(r.FormValue("club_name"))
	if utf8.RuneCountInString(clubName) > maxClubNameLen {
		return errors.New(api.Translate("error", "Venue name must be 80 characters or less"))
	}
	cfg.ClubName = clubName

	bookingURL := strings.TrimSpace(r.FormValue("booking_url"))
	if bookingURL != "" && !isHTTPURL(bookingURL) {
		return errors.New(api.Translate("error", "Booking link must be a valid HTTP or HTTPS URL"))
	}
	cfg.BookingURL = bookingURL

	return nil
}

func handleGameSettings(api sdkapi.IPluginApi, r *http.Request, cfg *settings.Settings) error {
	if r.FormValue("game_settings_present") == "" {
		return nil
	}
	cfg.Game.Enabled = r.FormValue("game_enabled") == "on"
	cfg.Game.Title = strings.TrimSpace(r.FormValue("game_title"))
	if utf8.RuneCountInString(cfg.Game.Title) > 60 {
		return errors.New(api.Translate("error", "Game title must be 60 characters or less"))
	}
	target, err := strconv.Atoi(r.FormValue("game_target_score"))
	if err != nil || target < 1 || target > 21 {
		return errors.New(api.Translate("error", "Target score must be between 1 and 21"))
	}
	cfg.Game.TargetScore = target
	cfg.Game.Difficulty = r.FormValue("game_difficulty")
	if cfg.Game.Difficulty != "easy" && cfg.Game.Difficulty != "medium" && cfg.Game.Difficulty != "hard" {
		return errors.New(api.Translate("error", "Choose an easy, medium, or hard game difficulty"))
	}
	return nil
}

// handleAffiliateLinkOperations processes affiliate link additions and removals with validation
func handleAffiliateLinkOperations(api sdkapi.IPluginApi, r *http.Request, cfg *settings.Settings) error {
	affiliateTitle := strings.TrimSpace(r.FormValue("affiliate_title"))
	affiliateCategory := strings.TrimSpace(r.FormValue("affiliate_category"))
	affiliateURL := strings.TrimSpace(r.FormValue("affiliate_url"))
	affiliateImageURL := strings.TrimSpace(r.FormValue("affiliate_image_url"))

	// Add new affiliate link
	if affiliateTitle != "" || affiliateCategory != "" || affiliateURL != "" || affiliateImageURL != "" {
		if affiliateTitle == "" {
			return errors.New(api.Translate("error", "Recommendation title is required"))
		}
		if utf8.RuneCountInString(affiliateTitle) > maxAffiliateNameLen {
			return errors.New(api.Translate("error", "Recommendation title must be 100 characters or less"))
		}
		if utf8.RuneCountInString(affiliateCategory) > maxCategoryLen {
			return errors.New(api.Translate("error", "Category must be 60 characters or less"))
		}
		if !isHTTPURL(affiliateURL) {
			return errors.New(api.Translate("error", "Affiliate link must be a valid HTTP or HTTPS URL"))
		}
		if affiliateImageURL != "" && !isHTTPURL(affiliateImageURL) {
			return errors.New(api.Translate("error", "Image URL must be a valid HTTP or HTTPS URL"))
		}

		imageFile, imageErr := resolveAffiliateImageField(api, r)
		if imageErr != nil {
			return errors.New(api.Translate("error", "Preview image must be a PNG, JPG, WEBP, GIF, or SVG image"))
		}
		if len(cfg.AffiliateLinks) >= 100 {
			return errors.New(api.Translate("error", "You can add up to 100 recommendations"))
		}

		cfg.AffiliateLinks = append(cfg.AffiliateLinks, settings.AffiliateLink{
			Title:     affiliateTitle,
			Category:  affiliateCategory,
			URL:       affiliateURL,
			ImageURL:  affiliateImageURL,
			ImageFile: imageFile,
		})
	}

	// Remove affiliate link
	if index, err := strconv.Atoi(r.FormValue("remove_affiliate")); err == nil {
		if index < 0 || index >= len(cfg.AffiliateLinks) {
			return errors.New(api.Translate("error", "Invalid affiliate link index"))
		}
		if file := cfg.AffiliateLinks[index].ImageFile; file != "" {
			_ = api.Storage().Delete(file)
		}
		cfg.AffiliateLinks = append(cfg.AffiliateLinks[:index], cfg.AffiliateLinks[index+1:]...)
	}

	return nil
}

// handleTutorialOperations processes tutorial additions and removals with validation
func handleTutorialOperations(api sdkapi.IPluginApi, r *http.Request, cfg *settings.Settings) error {
	tutorialTitle := strings.TrimSpace(r.FormValue("tutorial_title"))
	tutorialURL := strings.TrimSpace(r.FormValue("tutorial_url"))

	// Add new tutorial
	if tutorialTitle != "" {
		if utf8.RuneCountInString(tutorialTitle) > maxTutorialTitleLen {
			return errors.New(api.Translate("error", "Tutorial title must be 100 characters or less"))
		}

		videoFile, videoErr := resolveTutorialUploadField(api, r)
		if videoErr != nil {
			return errors.New(api.Translate("error", "Tutorial video must be an MP4, WEBM, or MOV file"))
		}
		if len(cfg.Tutorials) >= 50 {
			return errors.New(api.Translate("error", "You can add up to 50 tutorials"))
		}

		if videoFile == "" && (tutorialURL == "" || !isHTTPURL(tutorialURL)) {
			return errors.New(api.Translate("error", "Tutorials need a title and either a valid HTTP/HTTPS link or an uploaded video"))
		}

		cfg.Tutorials = append(cfg.Tutorials, settings.Tutorial{
			Title:     tutorialTitle,
			URL:       tutorialURL,
			VideoFile: videoFile,
		})
	}

	// Remove tutorial
	if index, err := strconv.Atoi(r.FormValue("remove_tutorial")); err == nil {
		if index < 0 || index >= len(cfg.Tutorials) {
			return errors.New(api.Translate("error", "Invalid tutorial index"))
		}
		if file := cfg.Tutorials[index].VideoFile; file != "" {
			_ = api.Storage().Delete(file)
		}
		cfg.Tutorials = append(cfg.Tutorials[:index], cfg.Tutorials[index+1:]...)
	}

	return nil
}

// handleCourtRateOperations processes court rate additions and removals
func handleCourtRateOperations(api sdkapi.IPluginApi, r *http.Request, cfg *settings.Settings) error {
	// Add new court rate
	if name := strings.TrimSpace(r.FormValue("court_rate_name")); name != "" {
		duration, durationErr := strconv.Atoi(r.FormValue("court_rate_duration"))
		price, priceErr := strconv.ParseFloat(r.FormValue("court_rate_price"), 64)

		if durationErr != nil || priceErr != nil {
			return errors.New(api.Translate("error", "Invalid court rate values. Duration and price must be numbers"))
		}
		if duration <= 0 {
			return errors.New(api.Translate("error", "Duration must be greater than zero"))
		}
		if price < 0 {
			return errors.New(api.Translate("error", "Price cannot be negative"))
		}
		if utf8.RuneCountInString(name) > maxSessionNameLen {
			return errors.New(api.Translate("error", "Session name must be 80 characters or less"))
		}
		if len(cfg.CourtRates) >= 100 {
			return errors.New(api.Translate("error", "You can add up to 100 court rates"))
		}

		cfg.CourtRates = append(cfg.CourtRates, settings.CourtRate{
			Name:        name,
			DurationMin: duration,
			Price:       price,
		})
	}

	// Remove court rate
	if index, err := strconv.Atoi(r.FormValue("remove_court_rate")); err == nil {
		if index < 0 || index >= len(cfg.CourtRates) {
			return errors.New(api.Translate("error", "Invalid court rate index"))
		}
		cfg.CourtRates = append(cfg.CourtRates[:index], cfg.CourtRates[index+1:]...)
	}

	return nil
}

// resolveLogoField applies the form's logo input to storage and returns the
// resulting stored filename ("" == use the selected theme's bundled
// default):
//   - reset checkbox ticked -> delete any custom file, return "".
//   - a file uploaded       -> validate + store it (deleting a stale
//     differently-named previous file), return the new filename.
//   - nothing submitted     -> return the current filename unchanged.
func resolveLogoField(api sdkapi.IPluginApi, r *http.Request, current string) (string, error) {
	if r.FormValue("remove_logo") == "on" {
		if current != "" && current != bundledLogoPath {
			if err := api.Storage().Delete(current); err != nil {
				api.Logger().Error("pickleball-theme: failed to delete logo: " + err.Error())
			}
		}
		return "", nil
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		// No file chosen: keep whatever is stored now.
		return current, nil
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExts[ext] {
		return current, errUnsupportedImage
	}
	data, err := readUpload(file, maxImageUploadBytes)
	if err != nil || !validImageData(ext, data) {
		return current, errUnsupportedImage
	}

	newName := logoStoreBase + ext
	if _, err := api.Storage().WriteReader(newName, bytes.NewReader(data)); err != nil {
		return current, err
	}

	// A previous upload with a different extension (e.g. logo.png ->
	// logo.webp) would otherwise be orphaned in storage.
	if current != "" && current != newName && current != bundledLogoPath {
		if err := api.Storage().Delete(current); err != nil {
			api.Logger().Error("pickleball-theme: failed to delete stale logo: " + err.Error())
		}
	}

	return newName, nil
}

func readUpload(file io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errors.New("uploaded file is too large")
	}
	return data, nil
}

func validImageData(ext string, data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if ext == ".svg" {
		lower := strings.ToLower(string(data))
		return strings.Contains(lower, "<svg") &&
			!strings.Contains(lower, "<script") &&
			!strings.Contains(lower, "onload=") &&
			!strings.Contains(lower, "javascript:")
	}
	mime := http.DetectContentType(data)
	if ext == ".webp" {
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	}
	allowed := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	}
	return allowed[ext] && (strings.HasPrefix(mime, "image/") || mime == "image/webp")
}

func validVideoData(ext string, data []byte) bool {
	if len(data) < 12 {
		return false
	}
	switch ext {
	case ".mp4", ".mov":
		return bytes.Contains(data[:minInt(len(data), 64)], []byte("ftyp"))
	case ".webm":
		return len(data) >= 4 && bytes.Equal(data[:4], []byte{0x1A, 0x45, 0xDF, 0xA3})
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
