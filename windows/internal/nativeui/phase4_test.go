package nativeui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/feeds"
	"windows/internal/store"
)

func TestNativeUIPhase4SelectionRestoreReorderInternalLinksAndFeatured(t *testing.T) {
	now := time.Now()
	fb := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "Alpha", IPNS: "k51alpha", Domain: "alpha.eth", Created: now},
			{ID: "s2", Name: "Beta", IPNS: "k51beta", Created: now},
		},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "First", Slug: "first-post", Created: now, Modified: now},
			{ID: "a2", SiteID: "s1", Title: "About", Slug: "about", Created: now, Modified: now},
		},
		subscriptions: []store.Subscription{
			{URL: "planetable.eth", Feed: feeds.Feed{Title: "Planetable", Items: []feeds.Item{{Key: "k1", Title: "Release"}}}},
		},
	}

	prefsPath := filepath.Join(t.TempDir(), "nativeui-layout.json")
	app1 := New(fb)
	app1.EnablePrefsPersistence(prefsPath)
	app1.selectSite("s1")
	app1.openArticleForRead("s1", "a2")
	app1.moveSiteOrder("s2", -1)
	app1.maybeSaveLayoutPrefs()

	// 1. Fresh App instance restores last site, article, and sidebar order.
	app2 := New(fb)
	app2.EnablePrefsPersistence(prefsPath)
	if app2.mode != viewSite || app2.selectedSiteID != "s1" || app2.selectedReadID != "a2" {
		t.Fatalf("expected restored selection (site s1, art a2), got mode=%v site=%q art=%q", app2.mode, app2.selectedSiteID, app2.selectedReadID)
	}
	ordered := app2.orderedSites()
	if len(ordered) != 2 || ordered[0].ID != "s2" {
		t.Fatalf("expected reordered sites [s2, s1], got %+v", ordered)
	}

	// 2. Internal links jump in-app to relative slug, planet:// IPNS, and subscribed ENS.
	if !app2.NavigateInternalLink("/first-post/") || app2.selectedReadID != "a1" {
		t.Fatalf("expected relative slug link to select a1, got %q", app2.selectedReadID)
	}
	if !app2.NavigateInternalLink("planet://k51beta") || app2.selectedSiteID != "s2" {
		t.Fatalf("expected planet:// IPNS link to select s2, got %q", app2.selectedSiteID)
	}
	if !app2.NavigateInternalLink("https://planetable.eth.limo/") || app2.mode != viewSubscription || app2.selectedSubURL != "planetable.eth" {
		t.Fatalf("expected subscribed ENS link to open subscription, got mode=%v sub=%q", app2.mode, app2.selectedSubURL)
	}

	// 3. Empty workspace shows Follow Featured button and follows curated feeds.
	emptyFB := &fakeBackend{}
	emptyApp := New(emptyFB)
	tt := ui.NewTester(emptyApp.View, 1280, 820)
	tt.Frame()
	if err := tt.Click("关注精选站点"); err != nil {
		t.Fatalf("expected 关注精选站点 button in empty workspace: %v", err)
	}
	tt.Frame()
	if len(emptyFB.subscriptions) != len(featuredPlanetAddresses) {
		t.Fatalf("expected %d featured subscriptions, got %d", len(featuredPlanetAddresses), len(emptyFB.subscriptions))
	}
}
