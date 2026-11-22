package store

import (
	"sort"
	"time"
)

// DailyActivity counts writing actions on one local calendar day. A record
// counts once per day: on its creation day and, when different, on its
// last-modified day.
type DailyActivity struct {
	Date     string `json:"date"`
	Articles int    `json:"articles"`
	Drafts   int    `json:"drafts"`
}

const activityDayFormat = "2006-01-02"

// WritingActivity returns one consistent, detached summary of per-day writing
// actions across the whole library. Days without activity are omitted and the
// result is sorted by date ascending. It performs no network or filesystem
// operations and never copies article or draft bodies.
func (s *Store) WritingActivity() []DailyActivity {
	return s.writingActivityIn(time.Local)
}

func (s *Store) writingActivityIn(loc *time.Location) []DailyActivity {
	s.mu.Lock()
	defer s.mu.Unlock()
	live := make(map[string]bool)
	for _, site := range s.data.Sites {
		if !site.Removed && !site.DeletionPending {
			live[site.ID] = true
		}
	}
	type dayCount struct{ articles, drafts int }
	counts := make(map[string]*dayCount)
	record := func(created, modified time.Time, isDraft bool) {
		if created.IsZero() && modified.IsZero() {
			return
		}
		days := []time.Time{created, modified}
		for i, day := range days {
			if day.IsZero() {
				days[i] = days[1-i]
			}
		}
		first, last := days[0], days[1]
		if last.Before(first) {
			last = first
		}
		first, last = first.In(loc), last.In(loc)
		bump := func(moment time.Time) {
			key := moment.Format(activityDayFormat)
			entry := counts[key]
			if entry == nil {
				entry = &dayCount{}
				counts[key] = entry
			}
			if isDraft {
				entry.drafts++
			} else {
				entry.articles++
			}
		}
		bump(first)
		if last.Format(activityDayFormat) != first.Format(activityDayFormat) {
			bump(last)
		}
	}
	for _, article := range s.data.Articles {
		if live[article.SiteID] {
			record(article.Created, article.Modified, false)
		}
	}
	for _, draft := range s.data.Drafts {
		if live[draft.SiteID] {
			record(draft.Created, draft.Modified, true)
		}
	}
	result := make([]DailyActivity, 0, len(counts))
	for key, entry := range counts {
		result = append(result, DailyActivity{Date: key, Articles: entry.articles, Drafts: entry.drafts})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })
	return result
}
