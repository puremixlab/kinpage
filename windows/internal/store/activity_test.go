package store

import (
	"reflect"
	"testing"
	"time"
)

func TestWritingActivityBucketsByLocalDayAndSortsSparseDays(t *testing.T) {
	loc := time.FixedZone("UTC8", 8*3600)
	created := time.Date(2026, 11, 5, 20, 0, 0, 0, time.UTC) // 2026-11-06 in +08:00
	modified := time.Date(2026, 11, 9, 2, 0, 0, 0, time.UTC) // 2026-11-09 in +08:00
	s := &Store{data: data{
		Sites:    []Site{{ID: "a"}},
		Articles: []Article{{ID: "1", SiteID: "a", Created: created, Modified: modified}},
		Drafts:   []Draft{{ID: "d", SiteID: "a", Created: created, Modified: created}},
	}}
	got := s.writingActivityIn(loc)
	want := []DailyActivity{
		{Date: "2026-11-06", Articles: 1, Drafts: 1},
		{Date: "2026-11-09", Articles: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	utc := s.writingActivityIn(time.UTC)
	if !reflect.DeepEqual(utc, []DailyActivity{
		{Date: "2026-11-05", Articles: 1, Drafts: 1},
		{Date: "2026-11-09", Articles: 1},
	}) {
		t.Fatalf("UTC bucketing wrong: %+v", utc)
	}
}

func TestWritingActivityCountsSameDayRecordOnceAndSkipsOutsideSites(t *testing.T) {
	day := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 11, 5, 18, 0, 0, 0, time.UTC)
	s := &Store{data: data{
		Sites: []Site{{ID: "a"}, {ID: "removed", Removed: true}, {ID: "deleting", DeletionPending: true}},
		Articles: []Article{
			{ID: "1", SiteID: "a", Created: day, Modified: evening},
			{ID: "2", SiteID: "removed", Created: day, Modified: evening},
			{ID: "3", SiteID: "deleting", Created: day},
			{ID: "4", SiteID: "missing", Created: day},
		},
		Drafts: []Draft{
			{ID: "d", SiteID: "a", Created: day, Modified: evening},
			{ID: "old", SiteID: "removed", Created: day},
		},
	}}
	got := s.writingActivityIn(time.UTC)
	want := []DailyActivity{{Date: "2026-11-05", Articles: 1, Drafts: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWritingActivityFallsBackWhenTimestampsMissing(t *testing.T) {
	day := time.Date(2026, 11, 5, 8, 0, 0, 0, time.UTC)
	s := &Store{data: data{
		Sites: []Site{{ID: "a"}},
		Articles: []Article{
			{ID: "1", SiteID: "a", Created: day},
			{ID: "2", SiteID: "a", Modified: day},
			{ID: "3", SiteID: "a"},
		},
	}}
	got := s.writingActivityIn(time.UTC)
	want := []DailyActivity{{Date: "2026-11-05", Articles: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWritingActivityEmptyStoreReturnsEmptyArray(t *testing.T) {
	got := (&Store{}).writingActivityIn(time.UTC)
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty slice, got %+v", got)
	}
}
