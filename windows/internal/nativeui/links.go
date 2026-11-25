package nativeui

import (
	"net/url"
	"strings"
)

// NavigateInternalLink intercepts planet:// links, relative article links
// (UUID or slug), or URLs matching a local site or subscribed Planet and
// switches selection directly inside the app. Returns true if handled.
func (a *App) NavigateInternalLink(rawURL string) bool {
	target := strings.TrimSpace(rawURL)
	if target == "" {
		return false
	}

	// 1. planet://<ipns-or-domain>
	if strings.HasPrefix(strings.ToLower(target), "planet://") {
		addr := strings.Trim(target[len("planet://"):], "/")
		if addr == "" {
			return false
		}
		for _, s := range a.sites {
			if strings.EqualFold(s.IPNS, addr) || (s.Domain != "" && strings.EqualFold(s.Domain, addr)) {
				a.selectSite(s.ID)
				return true
			}
		}
		for _, sub := range a.subscriptions {
			if strings.EqualFold(sub.URL, addr) || strings.Contains(strings.ToLower(sub.URL), strings.ToLower(addr)) {
				a.openSubscriptionView(viewSubscription, sub.URL)
				return true
			}
		}
		return false
	}

	// 2. Relative article link (/slug/ or /UUID/ or slug) within current site.
	if !strings.Contains(target, "://") && a.selectedSiteID != "" {
		seg := strings.Trim(target, "/.")
		if idx := strings.IndexByte(seg, '/'); idx >= 0 {
			seg = seg[:idx]
		}
		if seg != "" {
			for _, art := range a.articles {
				if strings.EqualFold(art.ID, seg) || (art.Slug != "" && strings.EqualFold(art.Slug, seg)) {
					a.openArticleForRead(art.SiteID, art.ID)
					return true
				}
			}
		}
		return false
	}

	// 3. Full http(s) URL matching a subscribed Planet or local site domain.
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, s := range a.sites {
		if s.Domain != "" && strings.EqualFold(s.Domain, host) {
			a.selectSite(s.ID)
			return true
		}
	}
	for _, sub := range a.subscriptions {
		if strings.EqualFold(sub.URL, target) || strings.EqualFold(sub.URL, host) || strings.HasPrefix(host, strings.ToLower(sub.URL)+".") {
			a.openSubscriptionView(viewSubscription, sub.URL)
			return true
		}
	}
	return false
}
