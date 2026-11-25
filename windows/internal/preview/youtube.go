package preview

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	ytVideoIDPattern      = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	standaloneYTLinkRegex = regexp.MustCompile(`(?i)<p>\s*(?:<a\s+href="([^"]+)"[^>]*>[^<]*</a>|(https?://[^\s<]+))\s*</p>`)
)

// ExtractYouTubeID returns the validated 11-character YouTube video ID if raw
// is a youtube.com/watch, youtu.be, or youtube.com/shorts URL.
func ExtractYouTubeID(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	var id string
	switch host {
	case "youtu.be":
		id = strings.Trim(u.Path, "/")
		if idx := strings.IndexByte(id, '/'); idx >= 0 {
			id = id[:idx]
		}
	case "youtube.com", "www.youtube.com", "m.youtube.com":
		switch {
		case u.Path == "/watch":
			id = u.Query().Get("v")
		case strings.HasPrefix(u.Path, "/shorts/"):
			id = strings.TrimPrefix(u.Path, "/shorts/")
			if idx := strings.IndexByte(id, '/'); idx >= 0 {
				id = id[:idx]
			}
		case strings.HasPrefix(u.Path, "/embed/"):
			id = strings.TrimPrefix(u.Path, "/embed/")
			if idx := strings.IndexByte(id, '/'); idx >= 0 {
				id = id[:idx]
			}
		}
	default:
		return "", false
	}
	if !ytVideoIDPattern.MatchString(id) {
		return "", false
	}
	return id, true
}

// EmbedYouTubeLinks converts standalone paragraph YouTube URLs into responsive
// privacy-enhanced iframe embeds.
func EmbedYouTubeLinks(renderedHTML string) string {
	return standaloneYTLinkRegex.ReplaceAllStringFunc(renderedHTML, func(match string) string {
		subs := standaloneYTLinkRegex.FindStringSubmatch(match)
		rawURL := ""
		if len(subs) > 1 && subs[1] != "" {
			rawURL = subs[1]
		} else if len(subs) > 2 {
			rawURL = subs[2]
		}
		id, ok := ExtractYouTubeID(rawURL)
		if !ok {
			return match
		}
		return fmt.Sprintf(
			`<div class="planet-youtube-embed" style="position:relative;padding-bottom:56.25%%;height:0;overflow:hidden;border-radius:8px;margin:16px 0;"><iframe src="https://www.youtube-nocookie.com/embed/%s" title="YouTube video player" style="position:absolute;top:0;left:0;width:100%%;height:100%%;border:0;" loading="lazy" referrerpolicy="strict-origin-when-cross-origin" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe></div>`,
			id,
		)
	})
}
