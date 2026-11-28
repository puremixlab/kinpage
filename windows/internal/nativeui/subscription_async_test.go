package nativeui

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

type waitingFollowBackend struct {
	*fakeBackend
	started chan string
	release chan struct{}
}

func (b *waitingFollowBackend) FollowContext(ctx context.Context, address string) (SubscriptionUpdate, error) {
	b.started <- address
	select {
	case <-ctx.Done():
		return SubscriptionUpdate{}, ctx.Err()
	case <-b.release:
		return b.fakeBackend.FollowContext(ctx, address)
	}
}

func monitorSubscriptionTasks(t *testing.T, app *App, tt *ui.Tester) func() {
	t.Helper()
	ready := make(chan struct{}, 1)
	app.SetBackgroundContext(t.Context(), func() {
		select {
		case ready <- struct{}{}:
		default:
		}
	})
	return func() {
		t.Helper()
		select {
		case <-ready:
			tt.Frame()
		case <-time.After(5 * time.Second):
			t.Fatal("subscription task did not request a redraw")
		}
	}
}

func submitDelayedFollow(t *testing.T, app *App, tt *ui.Tester, backend *waitingFollowBackend) {
	t.Helper()
	const address = "https://example.test/feed.xml"
	tt.Frame()
	if err := tt.Click("设置"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("关注站点…"); err != nil {
		t.Fatal(err)
	}
	app.followAddress = address
	tt.Frame()

	clicked := make(chan error, 1)
	go func() { clicked <- tt.Click("关注") }()

	select {
	case requested := <-backend.started:
		if requested != address {
			t.Fatalf("observed subscription request for %q, want %q", requested, address)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("follow request did not reach the backend")
	}
	select {
	case err := <-clicked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("follow click blocked the UI frame while the request was pending")
	}
}

func TestNativeUIFollowClickReturnsWhileRequestIsPending(t *testing.T) {
	backend := &waitingFollowBackend{
		fakeBackend: &fakeBackend{},
		started:     make(chan string, 2),
		release:     make(chan struct{}),
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	complete := monitorSubscriptionTasks(t, app, tt)
	submitDelayedFollow(t, app, tt, backend)
	if !app.followModalOpen || !app.subscriptionTaskIs(followTask) {
		t.Fatal("follow modal did not keep its pending state")
	}
	task := app.subscriptionTask
	app.submitFollow()
	if app.subscriptionTask != task {
		t.Fatal("duplicate submission replaced the pending task")
	}
	close(backend.release)
	complete()
	if app.followModalOpen || app.mode != viewSubscription || len(app.subscriptions) != 1 {
		t.Fatalf("follow completion did not open the subscription: mode=%v subs=%d", app.mode, len(app.subscriptions))
	}
}

func TestNativeUICancelsPendingFollowWhenModalCloses(t *testing.T) {
	backend := &waitingFollowBackend{
		fakeBackend: &fakeBackend{},
		started:     make(chan string, 1),
		release:     make(chan struct{}),
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	complete := monitorSubscriptionTasks(t, app, tt)
	submitDelayedFollow(t, app, tt, backend)
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	// The backend only returns when the task cancels its context.
	complete()
	if app.followModalOpen || app.subscriptionTask != nil || app.mode != viewOverview || len(app.subscriptions) != 0 {
		t.Fatal("cancelled follow changed the active view or subscriptions")
	}
}
