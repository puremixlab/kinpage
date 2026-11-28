package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/feeds"
)

func TestAutomaticSubscriptionRefreshBacksOffAndPreservesReadingState(t *testing.T) {
	mygo.App.SetPath(mygo.PathUserData, t.TempDir())
	t.Cleanup(func() { mygo.App.SetPath(mygo.PathUserData, "") })
	var requests atomic.Int32
	var offline atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed.xml" {
			http.NotFound(w, r)
			return
		}
		requests.Add(1)
		if offline.Load() {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "<rss version=\"2.0\"><channel><title>Feed</title><item><guid>post</guid><title>Refreshed</title></item></channel></rss>")
	}))
	defer server.Close()
	lib := &Library{}
	db, err := lib.open()
	if err != nil {
		t.Fatal(err)
	}
	address := server.URL + "/feed.xml"
	saved, err := db.SaveSubscription(address, feeds.Feed{Title: "Feed", Items: []feeds.Item{{ID: "post", Title: "Cached"}}})
	if err != nil {
		t.Fatal(err)
	}
	key := saved.Feed.Items[0].Key
	if err := lib.SetSubscriptionRead(address, key, true); err != nil {
		t.Fatal(err)
	}
	if err := lib.StarSubscription(address, key, true); err != nil {
		t.Fatal(err)
	}
	retries := make(map[string]subscriptionRetry)
	changes := 0
	notify := func() { changes++ }
	failureTime := saved.LastUpdated.Add(subscriptionRefreshInterval)
	lib.refreshDueSubscriptions(t.Context(), failureTime.Add(-time.Minute), retries, notify)
	if requests.Load() != 0 || changes != 0 {
		t.Fatal("fresh subscription was fetched")
	}
	offline.Store(true)
	lib.refreshDueSubscriptions(t.Context(), failureTime, retries, notify)
	subs, err := lib.Subscriptions()
	if err != nil || len(subs) != 1 || subs[0].LastError == "" || subs[0].Feed.Items[0].Title != "Cached" {
		t.Fatalf("offline refresh lost cache or error: %+v %v", subs, err)
	}
	lib.refreshDueSubscriptions(t.Context(), failureTime.Add(5*time.Minute), retries, notify)
	if requests.Load() != 1 || changes != 1 {
		t.Fatal("refresh ignored the retry delay")
	}
	secondFailure := failureTime.Add(subscriptionRefreshInterval)
	lib.refreshDueSubscriptions(t.Context(), secondFailure, retries, notify)
	if retry := retries[address]; retry.delay != 30*time.Minute {
		t.Fatalf("second failure backoff = %v", retry.delay)
	}
	offline.Store(false)
	lib.refreshDueSubscriptions(t.Context(), secondFailure.Add(15*time.Minute), retries, notify)
	if requests.Load() != 2 {
		t.Fatal("refresh ignored the increased retry delay")
	}
	lib.refreshDueSubscriptions(t.Context(), secondFailure.Add(30*time.Minute), retries, notify)
	subs, err = lib.Subscriptions()
	if err != nil || len(subs) != 1 {
		t.Fatalf("subscription missing after retry: %+v %v", subs, err)
	}
	item := subs[0].Feed.Items[0]
	if requests.Load() != 3 || changes != 3 || len(retries) != 0 ||
		subs[0].LastError != "" || item.Title != "Refreshed" || !item.Read || !item.Starred {
		t.Fatalf("unexpected retry result or reading state: %+v %v", subs[0], retries)
	}
}
