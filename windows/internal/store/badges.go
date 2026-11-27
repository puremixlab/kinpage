package store

import (
	"sort"
	"time"
)

// WritingBadgeKind identifies one of the local writing achievement families.
type WritingBadgeKind string

const (
	BadgeKindActiveDays WritingBadgeKind = "active_days"
	BadgeKindStreak     WritingBadgeKind = "streak"
	BadgeKindPublished  WritingBadgeKind = "published"
	BadgeKindPlanet     WritingBadgeKind = "planet"
)

// WritingBadge represents a locally computed writing achievement and its progress.
type WritingBadge struct {
	ID          string           `json:"id"`
	Kind        WritingBadgeKind `json:"kind"`
	Unlocked    bool             `json:"unlocked"`
	Value       int              `json:"value"`
	Tier        int              `json:"tier"`
	CurrentGoal int              `json:"currentGoal"`
	NextGoal    int              `json:"nextGoal"`
}

var (
	activeDaysTiers = []int{1, 7, 30, 100, 365}
	streakTiers     = []int{2, 7, 14, 30, 100}
	publishedTiers  = []int{1, 5, 20, 50, 100}
	planetTiers     = []int{1, 3, 5, 10}
)

// ComputeWritingBadges derives local writing achievements from daily activity
// and the dashboard metadata snapshot. It performs no I/O and returns badges in
// a deterministic order.
func ComputeWritingBadges(activity []DailyActivity, snapshot DashboardSnapshot) []WritingBadge {
	activeDates := collectActiveDates(activity)
	activeDays := len(activeDates)
	streak := computeBestStreak(activeDates)

	publishedSites := 0
	for _, s := range snapshot.Sites {
		if s.HasPublished || s.LastPublished != nil {
			publishedSites++
		}
	}

	return []WritingBadge{
		buildBadge("active_days", BadgeKindActiveDays, activeDays, activeDaysTiers),
		buildBadge("streak", BadgeKindStreak, streak, streakTiers),
		buildBadge("published", BadgeKindPublished, snapshot.ArticleCount, publishedTiers),
		buildBadge("planet", BadgeKindPlanet, publishedSites, planetTiers),
	}
}

func buildBadge(id string, kind WritingBadgeKind, value int, tiers []int) WritingBadge {
	if value < 0 {
		value = 0
	}
	b := WritingBadge{
		ID:    id,
		Kind:  kind,
		Value: value,
	}
	for i, goal := range tiers {
		if value >= goal {
			b.Unlocked = true
			b.Tier = i + 1
			b.CurrentGoal = goal
		} else {
			b.NextGoal = goal
			break
		}
	}
	return b
}

func collectActiveDates(activity []DailyActivity) []time.Time {
	seen := make(map[string]struct{}, len(activity))
	dates := make([]time.Time, 0, len(activity))
	for _, item := range activity {
		if item.Articles+item.Drafts <= 0 {
			continue
		}
		if _, ok := seen[item.Date]; ok {
			continue
		}
		parsed, err := time.Parse(activityDayFormat, item.Date)
		if err != nil {
			continue
		}
		seen[item.Date] = struct{}{}
		dates = append(dates, parsed)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func computeBestStreak(sortedDates []time.Time) int {
	if len(sortedDates) == 0 {
		return 0
	}
	best := 1
	run := 1
	for i := 1; i < len(sortedDates); i++ {
		prev := sortedDates[i-1]
		curr := sortedDates[i]
		// Compare calendar dates in UTC (parsed from YYYY-MM-DD) to avoid DST drift.
		if prev.AddDate(0, 0, 1).Equal(curr) {
			run++
			if run > best {
				best = run
			}
		} else {
			run = 1
		}
	}
	return best
}
