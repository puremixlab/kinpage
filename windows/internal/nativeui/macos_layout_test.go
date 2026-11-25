package nativeui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/feeds"
	"windows/internal/store"
)

func TestNativeUIMacOSContextMenusAndFilters(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "主站", Created: now}},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "普通文章", Content: "正文一", ArticleType: 0, Created: now, Modified: now},
			{ID: "a2", SiteID: "s1", Title: "关于页面", Content: "正文二", ArticleType: 1, Created: now, Modified: now},
		},
		subscriptions: []store.Subscription{{
			URL: "https://example.com/rss",
			Feed: feeds.Feed{
				Title: "外部周刊",
				Items: []feeds.Item{{Key: "k1", Title: "本周精选", Content: "<p>内容</p>"}},
			},
		}},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1360, 860)
	tt.Frame()

	// Sidebar site context menu -> 站点设置…
	if err := tt.RightClick("主站"); err != nil {
		t.Fatalf("failed to right-click site in sidebar: %v", err)
	}
	if err := tt.ChooseMenuItem("站点设置…"); err != nil {
		t.Fatalf("failed to choose 站点设置… from context menu: %v", err)
	}
	if !app.siteSettings.open {
		t.Fatal("expected site settings to open from sidebar context menu")
	}
	app.siteSettings.open = false
	tt.Frame()

	// Select site and test filter menu -> 独立页面.
	if err := tt.Click("主站"); err != nil {
		t.Fatalf("failed to select site: %v", err)
	}
	if err := tt.Click("筛选文章"); err != nil {
		t.Fatalf("failed to open filter menu: %v", err)
	}
	if err := tt.ChooseMenuItem("独立页面"); err != nil {
		t.Fatalf("failed to choose 独立页面 filter: %v", err)
	}
	if app.listFilter != listFilterPages {
		t.Fatalf("expected listFilterPages, got %v", app.listFilter)
	}
	if !tt.HasText("1 篇文章 · 已筛选") {
		t.Fatal("expected filtered subtitle '1 篇文章 · 已筛选' showing non-page article filtered out")
	}
	if !tt.HasText("关于页面") {
		t.Fatal("expected page article visible under 独立页面 filter")
	}

	// Reset filter to 全部 and test article context menu -> 置顶文章.
	if err := tt.Click("筛选文章"); err != nil {
		t.Fatalf("failed to reopen filter menu: %v", err)
	}
	if err := tt.ChooseMenuItem("全部"); err != nil {
		t.Fatalf("failed to reset filter: %v", err)
	}
	if err := tt.RightClick("关于页面"); err != nil {
		t.Fatalf("failed to right-click article row: %v", err)
	}
	if err := tt.ChooseMenuItem("置顶文章"); err != nil {
		t.Fatalf("failed to pin article from context menu: %v", err)
	}
	if app.articles[0].ID != "a2" || app.articles[0].Pinned == nil {
		t.Fatalf("expected a2 pinned first, got %+v", app.articles[0])
	}

	// Toggle inline list search via filter menu.
	if err := tt.Click("筛选文章"); err != nil {
		t.Fatalf("failed to open filter menu for search toggle: %v", err)
	}
	if err := tt.ChooseMenuItem("在列表中筛选…"); err != nil {
		t.Fatalf("failed to enable list search: %v", err)
	}
	if !app.listSearchOpen {
		t.Fatal("expected listSearchOpen to be true")
	}

	// Subscription context menu -> 添加星标.
	if err := tt.Click("外部周刊"); err != nil {
		t.Fatalf("failed to select subscription: %v", err)
	}
	if err := tt.RightClick("本周精选"); err != nil {
		t.Fatalf("failed to right-click subscription item: %v", err)
	}
	if err := tt.ChooseMenuItem("添加星标"); err != nil {
		t.Fatalf("failed to star subscription item from context menu: %v", err)
	}
	if !backend.subscriptions[0].Feed.Items[0].Starred {
		t.Fatal("expected subscription item starred via context menu")
	}
}

func TestNativeUILayoutPrefsPersistence(t *testing.T) {
	dir := t.TempDir()
	prefsFile := filepath.Join(dir, "nativeui-layout.json")

	backend := &fakeBackend{}
	app1 := New(backend)
	app1.EnablePrefsPersistence(prefsFile)
	app1.sidebarWidth = 260
	app1.postsPanelWidth = 310
	app1.sidebarCollapsed = true
	app1.postsCollapsed = true

	tt := ui.NewTester(app1.View, 1280, 820)
	tt.Frame()

	app2 := New(backend)
	app2.EnablePrefsPersistence(prefsFile)
	if app2.sidebarWidth != 260 || app2.postsPanelWidth != 310 || !app2.sidebarCollapsed || !app2.postsCollapsed {
		t.Fatalf("expected persisted prefs restored, got sidebar=%v posts=%v collapsed=%v/%v",
			app2.sidebarWidth, app2.postsPanelWidth, app2.sidebarCollapsed, app2.postsCollapsed)
	}
}
