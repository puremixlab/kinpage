package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIDashboardActivityModesAndTrendRanges(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "主站", Created: now}},
		activity: []store.DailyActivity{
			{Date: now.AddDate(0, 0, -2).Format("2006-01-02"), Articles: 2, Drafts: 1},
			{Date: now.Format("2006-01-02"), Articles: 1, Drafts: 2},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 900)
	tt.Frame()

	if err := tt.Click("按周"); err != nil {
		t.Fatalf("failed to switch heatmap to weekly: %v", err)
	}
	if app.activityMode != activityModeWeekly {
		t.Fatalf("expected weekly mode, got %d", app.activityMode)
	}

	if err := tt.Click("累计"); err != nil {
		t.Fatalf("failed to switch heatmap to cumulative: %v", err)
	}
	if app.activityMode != activityModeCumulative {
		t.Fatalf("expected cumulative mode, got %d", app.activityMode)
	}

	if err := tt.Click("90 天"); err != nil {
		t.Fatalf("failed to switch trend range to 90d: %v", err)
	}
	if app.trendRange != trendRange90d {
		t.Fatalf("expected 90d trend range, got %d", app.trendRange)
	}

	if err := tt.Click("1 年"); err != nil {
		t.Fatalf("failed to switch trend range to 1y: %v", err)
	}
	if app.trendRange != trendRange1y {
		t.Fatalf("expected 1y trend range, got %d", app.trendRange)
	}
}

func TestNativeUIDashboardSiteFilterAndMultiSiteCompose(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "技术日志", Created: now},
			{ID: "s2", Name: "生活随笔", Created: now},
		},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "Go 渲染实践", Content: "内容", Created: now, Modified: now},
			{ID: "a2", SiteID: "s2", Title: "周末咖啡", Content: "内容", Created: now, Modified: now},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 960)
	tt.Frame()

	// Filter recent articles by site s2.
	app.articleFilterSiteID = "s2"
	tt.Frame()
	if err := tt.Click("周末咖啡  ·  生活随笔"); err != nil {
		t.Fatalf("expected filtered article to be clickable: %v", err)
	}
	if app.selectedSiteID != "s2" || app.selectedReadID != "a2" {
		t.Fatalf("expected opened article a2 in site s2, got site=%q art=%q", app.selectedSiteID, app.selectedReadID)
	}

	// Return to overview and trigger multi-site compose picker.
	if err := tt.Click("总览"); err != nil {
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
