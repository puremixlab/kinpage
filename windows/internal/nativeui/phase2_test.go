package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIPhase2OpenIPFSResourceExternalReloadAndRebuild(t *testing.T) {
	fb := &fakeBackend{
		sites: []store.Site{
			{ID: "s1", Name: "Primary", Created: time.Now()},
		},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "Post 1", Created: time.Now(), Modified: time.Now()},
		},
	}
	app := New(fb)
	var openedURL string
	app.openExternal = func(u string) error {
		openedURL = u
		return nil
	}

	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// 1. Open IPFS Resource sheet resolves CID and calls openExternal.
	app.openIPFSResourceModal()
	tt.Frame()
	app.openIPFSResInput = "planetable.eth"
	if err := tt.Click("在本地网关打开"); err != nil {
		t.Fatalf("expected 在本地网关打开 button: %v", err)
	}
	tt.Frame()
	if openedURL != "http://127.0.0.1:8080/ipns/planetable.eth" {
		t.Fatalf("unexpected opened gateway URL: %q", openedURL)
	}
	if app.openIPFSResOpen {
		t.Fatal("expected Open IPFS Resource sheet to close after opening")
	}

	// 2. External change flag triggers automatic reload on next frame.
	fb.sites = append(fb.sites, store.Site{ID: "s2", Name: "Externally Added", Created: time.Now()})
	app.MarkDirtyExternal()
	tt.Frame()
	if len(app.sites) != 2 {
		t.Fatalf("expected 2 sites after external dirty reload, got %d", len(app.sites))
	}

	// 3. Quick Rebuild via sidebar context menu.
	if err := tt.RightClick("Primary"); err != nil {
		t.Fatalf("failed to right-click Primary site: %v", err)
	}
	if err := tt.ChooseMenuItem("快速重建 (仅首页)"); err != nil {
		t.Fatalf("failed to choose 快速重建 (仅首页): %v", err)
	}
	if len(fb.rebuildCalls) != 1 || fb.rebuildCalls[0] != "s1:quick" {
		t.Fatalf("unexpected rebuild calls: %v", fb.rebuildCalls)
	}
}
