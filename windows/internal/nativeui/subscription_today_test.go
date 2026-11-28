package nativeui

import (
	"slices"
	"testing"
	"time"

	"windows/internal/feeds"
	"windows/internal/store"
)

func TestSubscriptionTodayUsesLocalCalendarDate(t *testing.T) {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	next := start.AddDate(0, 0, 1)
	midday := start.Add(12 * time.Hour)
	_, offset := start.Zone()
	sourceZone := time.FixedZone("source", -8*60*60)
	inDay, outDay := start.Add(time.Hour), next.Add(time.Hour)
	if offset < 0 {
		sourceZone = time.FixedZone("source", 8*60*60)
		inDay, outDay = next.Add(-time.Hour), start.Add(-time.Hour)
	}
	gmt := time.FixedZone("GMT", 0)
	items := []feeds.Item{
		{ID: "rss-gmt", Published: midday.In(gmt).Format(time.RFC1123)},
		{ID: "rss-offset", Published: midday.In(sourceZone).Format(time.RFC1123Z)},
		{ID: "offset-into-today", Published: inDay.In(sourceZone).Format(time.RFC3339)},
		{ID: "offset-out-of-today", Published: outDay.In(sourceZone).Format(time.RFC3339)},
		{ID: "local-rfc3339", Published: midday.Format(time.RFC3339)},
		{ID: "legacy-local-date", Published: now.Format(time.DateOnly)},
		{ID: "invalid", Published: now.Format(time.DateOnly) + "-invalid"},
		{ID: "missing"},
	}
	a := App{
		mode:          viewToday,
		subscriptions: []store.Subscription{{URL: "https://example.com/feed", Feed: feeds.Feed{Items: items}}},
	}

	count := a.totalTodayCount()
	var keys []string
	for _, entry := range a.visibleSubscriptionItems() {
		keys = append(keys, entry.effective)
	}
	if time.Now().Format(time.DateOnly) != now.Format(time.DateOnly) {
		t.Skip("the local date changed during the check")
	}
	slices.Sort(keys)
	want := []string{"legacy-local-date", "local-rfc3339", "offset-into-today", "rss-gmt", "rss-offset"}
	if count != len(want) {
		t.Errorf("Today count = %d, want %d", count, len(want))
	}
	if !slices.Equal(keys, want) {
		t.Errorf("Today list = %q, want %q", keys, want)
	}
}
