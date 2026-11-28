package nativeui

import (
	"context"

	"windows/internal/feeds"
	"windows/internal/store"
)

// SubscriptionUpdate contains saved feed data and an optional late icon update.
// FinishIcon returns the new PNG only when it has been saved for this revision.
type SubscriptionUpdate struct {
	Subscription store.Subscription
	FinishIcon   feeds.IconLoader
}

type subscriptionTaskKind uint8

const (
	followTask subscriptionTaskKind = iota
	editSubscriptionTask
	refreshSubscriptionTask
	followFeaturedTask
	unfollowTask
)

type subscriptionTask struct {
	kind   subscriptionTaskKind
	cancel context.CancelFunc
	done   chan func()
}

// SetBackgroundContext binds background operations to the app lifetime and
// provides a callback that can request a window redraw from a worker.
func (a *App) SetBackgroundContext(ctx context.Context, invalidate func()) {
	a.backgroundContext = ctx
	a.invalidateWindow = invalidate
}

func (a *App) taskContext() context.Context {
	if a.backgroundContext != nil {
		return a.backgroundContext
	}
	return context.Background()
}

// run performs the backend work and returns the UI update for the next frame.
func (a *App) startSubscriptionTask(kind subscriptionTaskKind, run func(context.Context) func()) bool {
	if a.backend == nil || a.subscriptionTask != nil {
		return false
	}
	parent := a.taskContext()
	ctx, cancel := context.WithCancel(parent)
	task := &subscriptionTask{kind: kind, cancel: cancel, done: make(chan func(), 1)}
	a.subscriptionTask = task
	invalidate := a.invalidateWindow
	go func() {
		task.done <- run(ctx)
		if invalidate != nil && parent.Err() == nil {
			invalidate()
		}
	}()
	return true
}

func (a *App) subscriptionTaskIs(kind subscriptionTaskKind) bool {
	return a.subscriptionTask != nil && a.subscriptionTask.kind == kind
}

func (a *App) cancelSubscriptionTask() {
	if a.subscriptionTask != nil {
		a.subscriptionTask.cancel()
		a.subscriptionTask = nil
	}
}

func (a *App) finishSubscriptionTask() {
	task := a.subscriptionTask
	if task == nil {
		return
	}
	select {
	case finish := <-task.done:
		a.cancelSubscriptionTask()
		finish()
	default:
	}
}

func (a *App) reloadSubscriptions() {
	if subs, err := a.backend.Subscriptions(); err == nil {
		a.subscriptions = subs
	}
}

func (a *App) loadSubscriptionIcon(load feeds.IconLoader) {
	if load == nil {
		return
	}
	ctx, invalidate := a.taskContext(), a.invalidateWindow
	go func() {
		if load(ctx) != "" && ctx.Err() == nil {
			a.MarkSubscriptionsDirty()
			if invalidate != nil {
				invalidate()
			}
		}
	}()
}

// MarkSubscriptionsDirty asks the next frame to read the saved subscriptions.
// It is safe for callers on a maintenance goroutine.
func (a *App) MarkSubscriptionsDirty() {
	a.subscriptionsDirty.Store(true)
}

func (a *App) refreshSubscription(address string) {
	backend := a.backend
	if a.startSubscriptionTask(refreshSubscriptionTask, func(ctx context.Context) func() {
		update, err := backend.RefreshSubscriptionContext(ctx, address)
		return func() {
			a.reloadSubscriptions()
			if err != nil {
				a.setError(a.tf("status.refreshSubErr", err.Error()))
				return
			}
			a.loadSubscriptionIcon(update.FinishIcon)
			a.setStatus(a.t("status.refreshedSub"))
		}
	}) {
		a.setStatus(a.t("status.refreshingSub"))
	}
}

func (a *App) unfollowSubscription(address string) {
	if a.subscriptionTaskIs(unfollowTask) {
		return
	}
	// An ongoing refresh must give way to an explicit removal. Unfollow also
	// runs off-thread because the backend waits for any in-flight commit.
	a.cancelSubscriptionTask()
	backend := a.backend
	a.startSubscriptionTask(unfollowTask, func(ctx context.Context) func() {
		err := backend.Unfollow(address)
		return func() {
			a.reloadSubscriptions()
			if err != nil {
				a.setError(a.tf("status.unfollowErr", err.Error()))
				return
			}
			if a.selectedSubURL == address {
				a.selectedSubURL = ""
				a.mode = viewOverview
			}
			a.setStatus(a.t("status.unfollowed"))
		}
	})
}
