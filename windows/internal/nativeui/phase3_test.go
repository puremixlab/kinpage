package nativeui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIPhase3InteractiveTodoAndQuickPostDiscard(t *testing.T) {
	fb := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "Primary", Created: time.Now()},
		},
		articles: []store.Article{
			{
				ID:       "a1",
				SiteID:   "s1",
				Title:    "Tasks",
				Content:  "- [ ] Buy milk\n- [x] Walk dog",
				Created:  time.Now(),
				Modified: time.Now(),
			},
		},
	}
	app := New(fb)
	app.selectSite("s1")

	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// 1. Clicking unchecked To-Do in reader toggles and saves it.
	if err := tt.Click("Buy milk"); err != nil {
		t.Fatalf("expected interactive todo checkbox 'Buy milk': %v", err)
	}
	tt.Frame()
	if !strings.Contains(fb.articles[0].Content, "- [x] Buy milk") {
		t.Fatalf("expected article todo to be toggled, got: %s", fb.articles[0].Content)
	}

	// 2. QuickPost with unsaved text asks for discard confirmation on cancel.
	app.openQuickPost()
	tt.Frame()
	app.quickPostContent = "Unsaved thought"
	if err := tt.Click("取消"); err != nil {
		t.Fatalf("expected 取消 button in QuickPost: %v", err)
	}
	tt.Frame()
	if !app.quickPostDiscardConfirm || !app.quickPostOpen {
		t.Fatal("expected discard confirmation sheet when closing QuickPost with unsaved text")
	}
	if err := tt.Click("放弃"); err != nil {
		t.Fatalf("expected 放弃 button in discard confirmation: %v", err)
	}
	tt.Frame()
	if app.quickPostOpen {
		t.Fatal("expected QuickPost to close after confirming discard")
	}
}
