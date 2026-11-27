package nativeui

import (
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

type badgePos struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type layoutPrefs struct {
	Locale           string              `json:"locale,omitempty"`
	SidebarWidth     float32             `json:"sidebarWidth"`
	PostsPanelWidth  float32             `json:"postsPanelWidth"`
	SidebarCollapsed bool                `json:"sidebarCollapsed"`
	PostsCollapsed   bool                `json:"postsCollapsed"`
	LastMode         int                 `json:"lastMode,omitempty"`
	LastSiteID       string              `json:"lastSiteId,omitempty"`
	LastArticleID    string              `json:"lastArticleId,omitempty"`
	LastSubURL       string              `json:"lastSubUrl,omitempty"`
	LastSubItemKey   string              `json:"lastSubItemKey,omitempty"`
	SiteOrder        []string            `json:"siteOrder,omitempty"`
	SubOrder         []string            `json:"subOrder,omitempty"`
	BadgesHidden     bool                `json:"badgesHidden,omitempty"`
	BadgePositions   map[string]badgePos `json:"badgePositions,omitempty"`
}

func layoutPrefsEqual(a, b layoutPrefs) bool {
	return a.Locale == b.Locale &&
		a.SidebarWidth == b.SidebarWidth &&
		a.PostsPanelWidth == b.PostsPanelWidth &&
		a.SidebarCollapsed == b.SidebarCollapsed &&
		a.PostsCollapsed == b.PostsCollapsed &&
		a.LastMode == b.LastMode &&
		a.LastSiteID == b.LastSiteID &&
		a.LastArticleID == b.LastArticleID &&
		a.LastSubURL == b.LastSubURL &&
		a.LastSubItemKey == b.LastSubItemKey &&
		slices.Equal(a.SiteOrder, b.SiteOrder) &&
		slices.Equal(a.SubOrder, b.SubOrder) &&
		a.BadgesHidden == b.BadgesHidden &&
		maps.Equal(a.BadgePositions, b.BadgePositions)
}

// EnablePrefsPersistence loads saved split widths, panel collapse states,
// sidebar ordering, and last active selection from path (if present), and
// persists subsequent changes whenever state updates.
func (a *App) EnablePrefsPersistence(path string) {
	a.prefsPath = path
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.lastSavedPrefs = a.currentLayoutPrefs()
		return
	}
	var p layoutPrefs
	if err := json.Unmarshal(data, &p); err != nil {
		a.lastSavedPrefs = a.currentLayoutPrefs()
		return
	}
	if p.Locale != "" {
		a.locale = NormalizeLocale(p.Locale)
		a.syncEditorLocale()
	}
	if p.SidebarWidth >= 220 && p.SidebarWidth <= 420 {
		a.sidebarWidth = p.SidebarWidth
	}
	if p.PostsPanelWidth >= 240 && p.PostsPanelWidth <= 520 {
		a.postsPanelWidth = p.PostsPanelWidth
	}
	a.sidebarCollapsed = p.SidebarCollapsed
	a.postsCollapsed = p.PostsCollapsed
	a.siteOrder = append([]string(nil), p.SiteOrder...)
	a.subOrder = append([]string(nil), p.SubOrder...)
	a.badgesHidden = p.BadgesHidden
	if len(p.BadgePositions) > 0 {
		a.badgePositions = make(map[string]ui.Point, len(p.BadgePositions))
		a.badgeDragOffsets = make(map[string]*ui.Point, len(p.BadgePositions))
		for id, pos := range p.BadgePositions {
			pt := ui.Point{X: pos.X, Y: pos.Y}
			a.badgePositions[id] = pt
			live := pt
			a.badgeDragOffsets[id] = &live
		}
	}
	a.restoreLastSelection(p)
	a.lastSavedPrefs = a.currentLayoutPrefs()
}

func (a *App) restoreLastSelection(p layoutPrefs) {
	switch viewMode(p.LastMode) {
	case viewSite:
		if p.LastSiteID == "" {
			return
		}
		for _, s := range a.sites {
			if s.ID == p.LastSiteID {
				a.selectSite(s.ID)
				if p.LastArticleID != "" {
					for _, art := range a.articles {
						if art.ID == p.LastArticleID {
							a.openArticleForRead(s.ID, art.ID)
							break
						}
					}
				}
				return
			}
		}
	case viewToday, viewUnread, viewStarred:
		a.openSubscriptionView(viewMode(p.LastMode), "")
		if p.LastSubURL != "" && p.LastSubItemKey != "" {
			a.selectSubItem(p.LastSubURL, p.LastSubItemKey)
		}
	case viewSubscription:
		if p.LastSubURL == "" {
			return
		}
		for _, sub := range a.subscriptions {
			if sub.URL == p.LastSubURL {
				a.openSubscriptionView(viewSubscription, sub.URL)
				if p.LastSubItemKey != "" {
					a.selectSubItem(sub.URL, p.LastSubItemKey)
				}
				return
			}
		}
	}
}

func (a *App) currentLayoutPrefs() layoutPrefs {
	var badgePosMap map[string]badgePos
	if len(a.badgePositions) > 0 {
		badgePosMap = make(map[string]badgePos, len(a.badgePositions))
		for id, pt := range a.badgePositions {
			badgePosMap[id] = badgePos{
				X: float32(math.Round(float64(pt.X))),
				Y: float32(math.Round(float64(pt.Y))),
			}
		}
	}
	return layoutPrefs{
		Locale:           string(a.Locale()),
		SidebarWidth:     float32(math.Round(float64(a.sidebarWidth))),
		PostsPanelWidth:  float32(math.Round(float64(a.postsPanelWidth))),
		SidebarCollapsed: a.sidebarCollapsed,
		PostsCollapsed:   a.postsCollapsed,
		LastMode:         int(a.mode),
		LastSiteID:       a.selectedSiteID,
		LastArticleID:    a.selectedReadID,
		LastSubURL:       a.selectedSubURL,
		LastSubItemKey:   a.selectedSubItemKey,
		SiteOrder:        append([]string(nil), a.siteOrder...),
		SubOrder:         append([]string(nil), a.subOrder...),
		BadgesHidden:     a.badgesHidden,
		BadgePositions:   badgePosMap,
	}
}

func (a *App) maybeSaveLayoutPrefs() {
	if a.prefsPath == "" {
		return
	}
	cur := a.currentLayoutPrefs()
	if layoutPrefsEqual(cur, a.lastSavedPrefs) {
		return
	}
	a.lastSavedPrefs = cur
	if err := os.MkdirAll(filepath.Dir(a.prefsPath), 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(cur, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(a.prefsPath, data, 0o600)
}

func (a *App) orderedSites() []store.Site {
	if len(a.siteOrder) == 0 {
		return a.sites
	}
	rank := make(map[string]int, len(a.siteOrder))
	for i, id := range a.siteOrder {
		rank[id] = i
	}
	out := append([]store.Site(nil), a.sites...)
	slices.SortStableFunc(out, func(x, y store.Site) int {
		rx, okX := rank[x.ID]
		ry, okY := rank[y.ID]
		if !okX {
			rx = len(a.siteOrder)
		}
		if !okY {
			ry = len(a.siteOrder)
		}
		return rx - ry
	})
	return out
}

func (a *App) orderedSubscriptions() []store.Subscription {
	if len(a.subOrder) == 0 {
		return a.subscriptions
	}
	rank := make(map[string]int, len(a.subOrder))
	for i, u := range a.subOrder {
		rank[u] = i
	}
	out := append([]store.Subscription(nil), a.subscriptions...)
	slices.SortStableFunc(out, func(x, y store.Subscription) int {
		rx, okX := rank[x.URL]
		ry, okY := rank[y.URL]
		if !okX {
			rx = len(a.subOrder)
		}
		if !okY {
			ry = len(a.subOrder)
		}
		return rx - ry
	})
	return out
}

func (a *App) moveSiteOrder(siteID string, delta int) {
	ordered := a.orderedSites()
	var ids []string
	for _, s := range ordered {
		ids = append(ids, s.ID)
	}
	idx := slices.Index(ids, siteID)
	if idx < 0 {
		return
	}
	target := idx + delta
	if target < 0 || target >= len(ids) {
		return
	}
	ids[idx], ids[target] = ids[target], ids[idx]
	a.siteOrder = ids
}

func (a *App) moveSubOrder(subURL string, delta int) {
	ordered := a.orderedSubscriptions()
	var urls []string
	for _, s := range ordered {
		urls = append(urls, s.URL)
	}
	idx := slices.Index(urls, subURL)
	if idx < 0 {
		return
	}
	target := idx + delta
	if target < 0 || target >= len(urls) {
		return
	}
	urls[idx], urls[target] = urls[target], urls[idx]
	a.subOrder = urls
}
