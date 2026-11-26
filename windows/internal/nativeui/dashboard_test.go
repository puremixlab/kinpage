package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIDashboardHeroDraftSiteFilterAndCompose(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "技术日志", Created: now},
			{ID: "s2", Name: "生活随笔", Created: now},
		},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "Go 渲染实践", Content: "内容正文", Created: now, Modified: now},
			{ID: "a2", SiteID: "s2", Title: "周末咖啡", Content: "内容正文", Created: now, Modified: now},
		},
		drafts: []store.Draft{
			{ID: "d1", SiteID: "s1", Title: "未完稿件", Content: "草稿正文内容", Created: now, Modified: now},
		},
		activity: []store.DailyActivity{
			{Date: now.AddDate(0, 0, -2).Format("2006-01-02"), Articles: 2, Drafts: 1},
			{Date: now.Format("2006-01-02"), Articles: 1, Drafts: 2},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1360, 900)
	tt.Frame()

	// Click the elevated hero manuscript draft sheet directly.
	if err := tt.Click("未完稿件  ·  技术日志"); err != nil {
		t.Fatalf("expected hero draft sheet to be clickable: %v", err)
	}
	if !app.editing || app.activeDraftID != "d1" || app.selectedSiteID != "s1" {
		t.Fatalf("expected draft d1 opened in editor, editing=%v draft=%q site=%q", app.editing, app.activeDraftID, app.selectedSiteID)
	}

	// Return to overview and filter recent articles by site s2.
	if err := tt.Click("写作台"); err != nil {
		t.Fatalf("failed to return to overview: %v", err)
	}
	app.articleFilterSiteID = "s2"
	tt.Frame()
	if err := tt.Click("周末咖啡  ·  生活随笔"); err != nil {
		t.Fatalf("expected filtered article to be clickable: %v", err)
	}
	if app.selectedSiteID != "s2" || app.selectedReadID != "a2" {
		t.Fatalf("expected opened article a2 in site s2, got site=%q art=%q", app.selectedSiteID, app.selectedReadID)
	}

	// Return to overview and trigger multi-site compose picker from unified toolbar.
	if err := tt.Click("写作台"); err != nil {
		t.Fatalf("failed to return to overview: %v", err)
	}
	if err := tt.Click("写文章"); err != nil {
		t.Fatalf("failed to click compose on overview: %v", err)
	}
	if !app.composePickerOpen {
		t.Fatal("expected multi-site compose picker modal to open")
	}

	// Pick site s1 in modal.
	if err := tt.Click("技术日志"); err != nil {
		t.Fatalf("failed to pick site in compose modal: %v", err)
	}
	if app.composePickerOpen || !app.editing || app.selectedSiteID != "s1" {
		t.Fatalf("expected editor open for s1, modal=%v editing=%v site=%q", app.composePickerOpen, app.editing, app.selectedSiteID)
	}
}

func TestDashboardResponsiveColumnsAndWordCount(t *testing.T) {
	if got := dashboardCardColumns(600); got != 2 {
		t.Fatalf("dashboardCardColumns(600) = %d, want 2", got)
	}
	if got := dashboardCardColumns(980); got != 3 {
		t.Fatalf("dashboardCardColumns(980) = %d, want 3", got)
	}
	if got := formatWordCount(0); got != "暂无正文" {
		t.Fatalf("formatWordCount(0) = %q", got)
	}
	if got := formatWordCount(640); got != "640 字" {
		t.Fatalf("formatWordCount(640) = %q", got)
	}
	if got := formatWordCount(1280); got != "1,280 字" {
		t.Fatalf("formatWordCount(1280) = %q", got)
	}
}
