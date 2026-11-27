package store

import (
	"testing"
	"time"
)

func TestComputeWritingBadgesEmpty(t *testing.T) {
	badges := ComputeWritingBadges(nil, DashboardSnapshot{})
	if len(badges) != 4 {
		t.Fatalf("expected 4 badges, got %d", len(badges))
	}
	for _, b := range badges {
		if b.Unlocked || b.Value != 0 || b.Tier != 0 || b.NextGoal <= 0 {
			t.Fatalf("expected locked initial badge %+v", b)
		}
	}
}

func TestComputeWritingBadgesMilestonesAndStreak(t *testing.T) {
	now := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	activity := []DailyActivity{
		{Date: "2026-11-10", Articles: 1},
		{Date: "2026-11-12", Articles: 0, Drafts: 0}, // zero activity ignored
		{Date: "2026-11-20", Articles: 1},
		{Date: "2026-11-21", Drafts: 2},
		{Date: "2026-11-22", Articles: 1, Drafts: 1},
		{Date: "2026-11-22", Articles: 1}, // duplicate date deduplicated
		{Date: "2026-11-25", Articles: 1},
		{Date: "2026-11-26", Articles: 1},
		{Date: "2026-11-27", Articles: 2},
		{Date: "2026-11-28", Drafts: 1},
	}
	snapshot := DashboardSnapshot{
		SiteCount:    2,
		ArticleCount: 12,
		Sites: []DashboardSite{
			{ID: "s1", Name: "Site 1", HasPublished: true},
			{ID: "s2", Name: "Site 2", LastPublished: &now},
		},
	}

	badges := ComputeWritingBadges(activity, snapshot)
	byKind := map[WritingBadgeKind]WritingBadge{}
	for _, b := range badges {
		byKind[b.Kind] = b
	}

	if b := byKind[BadgeKindActiveDays]; !b.Unlocked || b.Value != 8 || b.Tier != 2 || b.NextGoal != 30 {
		t.Fatalf("unexpected active_days badge: %+v", b)
	}
	if b := byKind[BadgeKindStreak]; !b.Unlocked || b.Value != 4 || b.Tier != 1 || b.NextGoal != 7 {
		t.Fatalf("unexpected streak badge: %+v", b)
	}
	if b := byKind[BadgeKindPublished]; !b.Unlocked || b.Value != 12 || b.Tier != 2 || b.NextGoal != 20 {
		t.Fatalf("unexpected published badge: %+v", b)
	}
	if b := byKind[BadgeKindPlanet]; !b.Unlocked || b.Value != 2 || b.Tier != 1 || b.NextGoal != 3 {
		t.Fatalf("unexpected planet badge: %+v", b)
	}
}
