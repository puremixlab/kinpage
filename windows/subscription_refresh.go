package main

import (
	"context"
	"errors"
	"log"
	"time"

	"windows/internal/feeds"
)

const (
	subscriptionRefreshInterval = 15 * time.Minute
	subscriptionMaxBackoff      = 2 * time.Hour
)

type subscriptionRetry struct {
	delay time.Duration
	at    time.Time
}

func (l *Library) maintainSubscriptions(ctx context.Context, changed func()) {
	retries := make(map[string]subscriptionRetry)
	ticker := time.NewTicker(subscriptionRefreshInterval)
	defer ticker.Stop()
	for {
		l.refreshDueSubscriptions(ctx, time.Now(), retries, changed)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (l *Library) refreshDueSubscriptions(ctx context.Context, now time.Time, retries map[string]subscriptionRetry, changed func()) {
	subs, err := l.Subscriptions()
	if err != nil {
		log.Printf("automatic subscription refresh: %v", err)
		return
	}
	staleBefore := now.Add(-subscriptionRefreshInterval)
	present := make(map[string]bool, len(subs))
	var icons []feeds.IconLoader
	for _, sub := range subs {
		if ctx.Err() != nil {
			return
		}
		present[sub.URL] = true
		if sub.LastUpdated != nil && sub.LastUpdated.After(staleBefore) {
			delete(retries, sub.URL)
			continue
		}
		retry := retries[sub.URL]
		if now.Before(retry.at) {
			continue
		}
		update, err := l.follow(ctx, sub.URL, true, staleBefore)
		switch {
		case ctx.Err() != nil:
			return
		case errors.Is(err, errSubscriptionMissing) || errors.Is(err, errSubscriptionNotDue):
			delete(retries, sub.URL)
			continue
		case err != nil:
			delay := min(max(retry.delay*2, subscriptionRefreshInterval), subscriptionMaxBackoff)
			retries[sub.URL] = subscriptionRetry{delay: delay, at: now.Add(delay)}
		default:
			delete(retries, sub.URL)
			if update.FinishIcon != nil && update.Subscription.Feed.IconPNG == "" {
				icons = append(icons, update.FinishIcon)
			}
		}
		changed()
	}
	for address := range retries {
		if !present[address] {
			delete(retries, address)
		}
	}
	// Load missing avatars after the feed content for all due sources is saved.
	for _, load := range icons {
		if ctx.Err() != nil {
			return
		}
		if load(ctx) != "" {
			changed()
		}
	}
}
