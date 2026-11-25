package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIPhase1ArticleDateMoveArchiveAndStarFilter(t *testing.T) {
	fb := &fakeBackend{
		sites: []store.Site{
			{ID: "site-1", Name: "Primary Site", About: "Source"},
			{ID: "site-2", Name: "Second Site", About: "Target"},
			{ID: "site-arch", Name: "Archived Site", Archived: true},
		},
		articles: []store.Article{
			{
				ID:       "art-1",
				SiteID:   "site-1",
				Title:    "First Article",
				Content:  "Body",
				Created:  time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Modified: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				Starred:  true,
				StarType: store.StarTypeTodo,
			},
		},
	}

	app := New(fb)
	app.selectSite("site-1")

	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// 1. Archived sites section toggle expands archived site.
	if err := tt.Click("Archived Site"); err == nil {
		t.Fatal("archived site should be collapsed initially")
	}
	if err := tt.Click("已归档站点"); err != nil {
		t.Fatalf("expected toggle-archived-sites button: %v", err)
	}
	tt.Frame()
	if err := tt.Click("Archived Site"); err != nil {
		t.Fatalf("expected Archived Site to be visible after expanding: %v", err)
	}
	app.selectSite("site-1")
	tt.Frame()

	// 2. Article Settings updates custom Created date.
	app.openArticleSettings(fb.articles[0])
	tt.Frame()
	app.articleSettingsDate = "2024-08-15 09:30"
	if err := tt.Click("保存设置"); err != nil {
		t.Fatalf("expected 保存设置 button in Article Settings: %v", err)
	}
	tt.Frame()
	wantLocal, _ := time.ParseInLocation(articleDateInputLayout, "2024-08-15 09:30", time.Local)
	if !fb.articles[0].Created.Equal(wantLocal.UTC()) {
		t.Fatalf("expected updated date %v, got %v", wantLocal.UTC(), fb.articles[0].Created)
	}

	// 3. Multi-type star filter matches Todo and excludes Heart.
	app.listFilter = listFilterStarred
	app.starTypeFilter = store.StarTypeTodo
	if !app.matchesSiteFilter(fb.articles[0]) {
		t.Fatal("expected Todo-starred article to match Todo filter")
	}
	app.starTypeFilter = store.StarTypeHeart
	if app.matchesSiteFilter(fb.articles[0]) {
		t.Fatal("expected Todo-starred article not to match Heart filter")
	}
	app.listFilter = listFilterAll
	app.starTypeFilter = -1

	// 4. Move Article sheet moves article to Second Site.
	app.openMoveArticleSheet(fb.articles[0])
	tt.Frame()
	if err := tt.Click("移动"); err != nil {
		t.Fatalf("expected 移动 button in Move Article sheet: %v", err)
	}
	tt.Frame()
	if fb.articles[0].SiteID != "site-2" {
		t.Fatalf("expected article moved to site-2, got %s", fb.articles[0].SiteID)
	}
}
