package nativeui

import (
	"path/filepath"
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

func TestDashboardBadgesWallHideAndPersistence(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "山野通信", Created: now, LastPublished: &now},
		},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "第一篇", Content: "正文", Created: now, Modified: now},
		},
		activity: []store.DailyActivity{
			{Date: now.AddDate(0, 0, -1).Format("2006-01-02"), Articles: 1},
			{Date: now.Format("2006-01-02"), Articles: 1},
		},
	}

	prefsFile := filepath.Join(t.TempDir(), "nativeui-layout.json")
	app := New(backend)
	app.EnablePrefsPersistence(prefsFile)
	tt := ui.NewTester(app.View, 1360, 900)
	tt.Frame()

	// Real pointer drag: moving updates live drag offset without persisting mid-drag.
	badgeRect, ok := tt.Find("累计写作天数")
	if !ok {
		t.Fatal("expected active days badge to be rendered on dashboard")
	}
	cx, cy := badgeRect.X+badgeRect.W/2, badgeRect.Y+badgeRect.H/2
	tt.Press(cx, cy)
	tt.Move(cx+80, cy+12)
	if app.badgeDragOffsets["active_days"] == nil || app.badgeDragOffsets["active_days"].X <= 10 {
		t.Fatalf("expected live drag offset to follow pointer, got %+v", app.badgeDragOffsets["active_days"])
	}
	if len(app.badgePositions) != 0 {
		t.Fatalf("unfinished drag must not persist badgePositions, got %+v", app.badgePositions)
	}
	// Canceling press restores original position.
	tt.CancelPress()
	tt.Advance(600 * time.Millisecond)
	if len(app.badgePositions) != 0 {
		t.Fatalf("canceling press should keep badgePositions empty, got %+v", app.badgePositions)
	}

	// Complete drag and drop: persists position without falsely opening Achievement Wall.
	tt.Press(cx, cy)
	tt.Move(cx+90, cy+10)
	tt.Release(cx+90, cy+10)
	tt.Frame()
	if app.badgeWallOpen {
		t.Fatal("dropping a dragged badge must not open the achievement wall")
	}
	savedPos, ok := app.badgePositions["active_days"]
	if !ok || savedPos.X < 60 {
		t.Fatalf("expected dragged active_days position to be saved, got saved=%+v (ok=%v) offset=%+v rect=%+v", savedPos, ok, app.badgeDragOffsets["active_days"], badgeRect)
	}

	// Clicking the moved badge still opens the Achievement Wall sheet, and Escape closes it.
	if err := tt.Click("累计写作天数"); err != nil {
		t.Fatalf("expected moved active days badge to be clickable: %v", err)
	}
	if !app.badgeWallOpen {
		t.Fatal("expected badge wall sheet to open after clicking badge")
	}

	// Reset layout button restores default positions.
	if err := tt.Click("重置排布"); err != nil {
		t.Fatalf("expected reset layout button in badge wall: %v", err)
	}
	if len(app.badgePositions) != 0 {
		t.Fatalf("expected custom badgePositions cleared after reset, got %+v", app.badgePositions)
	}
	tt.Key(0, ui.KeyEscape)
	if app.badgeWallOpen {
		t.Fatal("expected Escape to close the achievement wall")
	}

	// Drag again, then collapse via × button, and verify both persist across reload.
	tt.Advance(600 * time.Millisecond)
	badgeRect2, _ := tt.Find("累计写作天数")
	cx2, cy2 := badgeRect2.X+badgeRect2.W/2, badgeRect2.Y+badgeRect2.H/2
	tt.Press(cx2, cy2)
	tt.Move(cx2+70, cy2+6)
	tt.Release(cx2+70, cy2+6)
	tt.Frame()
	persistedPos := app.badgePositions["active_days"]

	if err := tt.Click("收起徽章"); err != nil {
		t.Fatalf("expected hide badges button to work: %v", err)
	}
	if !app.badgesHidden {
		t.Fatal("expected badgesHidden=true after clicking hide button")
	}
	tt.Frame()

	// Reload into a fresh App and verify both badgePositions and badgesHidden persisted.
	app2 := New(backend)
	app2.EnablePrefsPersistence(prefsFile)
	if !app2.badgesHidden {
		t.Fatal("expected badgesHidden=true to persist across reload")
	}
	gotPos, ok := app2.badgePositions["active_days"]
	if !ok || !badgePointsNear(gotPos, persistedPos) {
		t.Fatalf("expected persisted active_days position near %+v, got %+v (ok=%v)", persistedPos, gotPos, ok)
	}

	// Clicking the compact trophy pill restores badges when hidden.
	tt2 := ui.NewTester(app2.View, 1360, 900)
	tt2.Frame()
	if err := tt2.Click("写作徽章 (3/4)"); err != nil {
		t.Fatalf("expected compact badge pill to be clickable: %v", err)
	}
	if app2.badgesHidden {
		t.Fatal("expected clicking compact badge pill to restore floating badges")
	}

	// Switch heatmap between Day / Week / Month below the Trophy and verify persistence.
	if err := tt2.Click("周"); err != nil {
		t.Fatalf("expected weekly heatmap switcher to be clickable: %v", err)
	}
	if app2.activityMode != activityModeWeekly {
		t.Fatalf("expected activityModeWeekly, got %d", app2.activityMode)
	}
	if err := tt2.Click("月"); err != nil {
		t.Fatalf("expected monthly heatmap switcher to be clickable: %v", err)
	}
	if app2.activityMode != activityModeMonthly {
		t.Fatalf("expected activityModeMonthly, got %d", app2.activityMode)
	}
	tt2.Frame()

	app3 := New(backend)
	app3.EnablePrefsPersistence(prefsFile)
	if app3.activityMode != activityModeMonthly {
		t.Fatalf("expected persisted activityModeMonthly, got %d", app3.activityMode)
	}
}
