package portal

import (
	"net/url"
	"strings"
)

func isDirectVideoURL(raw string) bool {
	path := strings.ToLower(raw)
	return strings.HasSuffix(path, ".mp4") || strings.HasSuffix(path, ".webm") || strings.HasSuffix(path, ".mov")
}

// embedTutorialURL converts common YouTube sharing links into embeddable URLs.
// Other validated HTTP(S) links are returned unchanged for iframe embedding.
func embedTutorialURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	host := strings.ToLower(u.Hostname())
	if host == "youtu.be" {
		return "https://www.youtube.com/embed/" + strings.Trim(u.Path, "/")
	}
	if host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" {
		if id := u.Query().Get("v"); id != "" {
			return "https://www.youtube.com/embed/" + id
		}
	}
	return raw
}
