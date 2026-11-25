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

	if err := tt.Click("每周"); err != nil {
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
		drafts: []store.Draft{
			{ID: "d1", SiteID: "s1", Title: "未完稿件", Content: "草稿正文", Created: now, Modified: now},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 960)
	tt.Frame()

	// Filter to drafts only via the bookmark shelf and open the draft card.
	if err := tt.Click("草稿 · 1"); err != nil {
		t.Fatalf("expected drafts filter chip to be clickable: %v", err)
	}
	if app.articleFilterSiteID != dashboardFilterDrafts {
		t.Fatalf("expected drafts filter active, got %q", app.articleFilterSiteID)
	}
	if err := tt.Click("未完稿件  ·  技术日志"); err != nil {
		t.Fatalf("expected draft card to be clickable: %v", err)
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

	// Return to overview and trigger multi-site compose picker.
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

func TestDashboardResponsiveColumns(t *testing.T) {
	cases := []struct {
		width float32
		want  int
	}{
		{480, 2},
		{640, 3},
		{980, 4},
		{1200, 5},
		{1600, 6},
	}
	for _, tc := range cases {
		if got := dashboardCardColumns(tc.width); got != tc.want {
			t.Fatalf("dashboardCardColumns(%v) = %d, want %d", tc.width, got, tc.want)
		}
	}
}
