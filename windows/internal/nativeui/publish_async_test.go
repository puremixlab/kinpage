package nativeui

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/publisher"
	"windows/internal/store"
)

type waitingPublishBackend struct {
	*fakeBackend
	started chan string
	release <-chan struct{}
	calls   atomic.Int32
}

func (b *waitingPublishBackend) PublishSiteContext(ctx context.Context, siteID string) (publisher.Result, error) {
	b.calls.Add(1)
	if b.started != nil {
		b.started <- siteID
	}
	if b.release != nil {
		select {
		case <-b.release:
		case <-ctx.Done():
			return publisher.Result{}, ctx.Err()
		}
	}
	return b.fakeBackend.PublishSiteContext(ctx, siteID)
}

func TestNativeUIEditorArticleSaveReportsPendingPublication(t *testing.T) {
	site := store.Site{ID: "s1", Name: "文章发布测试"}
	draft := store.Draft{ID: "d1", SiteID: site.ID, Title: "初稿", Content: "正文"}
	backend := &waitingPublishBackend{fakeBackend: &fakeBackend{
		sites:  []store.Site{site},
		drafts: []store.Draft{draft},
	}}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()
	if err := tt.Click(site.Name); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("✎ " + draft.Title); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("发布"); err != nil {
		t.Fatal(err)
	}
	if len(backend.articles) != 1 || app.editing {
		t.Fatalf("local article save did not complete: articles=%d editing=%v", len(backend.articles), app.editing)
	}
	if calls := backend.calls.Load(); calls != 0 {
		t.Fatalf("local article save requested %d site publications", calls)
	}
	if app.statusIsErr || !strings.Contains(app.statusText, "待发布") {
		t.Fatalf("local article save did not report pending publication: %q", app.statusText)
	}
}

func TestNativeUIPublishSiteClickReturnsWhileRequestIsPending(t *testing.T) {
	site := store.Site{ID: "s1", Name: "站点发布测试"}
	release := make(chan struct{})
	backend := &waitingPublishBackend{
		fakeBackend: &fakeBackend{sites: []store.Site{site}},
		started:     make(chan string, 1),
		release:     release,
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()
	if err := tt.Click(site.Name); err != nil {
		t.Fatal(err)
	}

	clickFinished := make(chan struct{})
	var clickErr error
	go func() {
		defer close(clickFinished)
		clickErr = tt.Click("发布站点")
	}()
	t.Cleanup(func() {
		close(release)
		select {
		case <-clickFinished:
		case <-time.After(5 * time.Second):
			t.Error("publish click still blocked after releasing the backend")
		}
	})

	select {
	case siteID := <-backend.started:
		if siteID != site.ID {
			t.Fatalf("publishing site %q, want %q", siteID, site.ID)
		}
	case <-time.After(5 * time.Second):
		select {
		case <-clickFinished:
			if clickErr != nil {
				t.Fatal(clickErr)
			}
		default:
		}
		t.Fatal("publish request did not reach the backend")
	}
	select {
	case <-clickFinished:
		if clickErr != nil {
			t.Fatal(clickErr)
		}
	case <-time.After(time.Second):
		t.Fatal("publish site click blocked the UI frame while the request was pending")
	}
}
