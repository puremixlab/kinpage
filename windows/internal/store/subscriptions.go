package store

import (
	"errors"
	"strings"
	"time"

	"windows/internal/feeds"
	"windows/internal/siteimage"
)

type Subscription struct {
	URL             string     `json:"url"`
	CustomTitle     string     `json:"customTitle,omitempty"`
	Feed            feeds.Feed `json:"feed"`
	CID             string     `json:"cid,omitempty"`
	PinnedCID       string     `json:"pinnedCid,omitempty"`
	LastUpdated     *time.Time `json:"lastUpdated,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
	OfflinePinned   bool       `json:"offlinePinned,omitempty"`
	OfflineSnapshot string     `json:"offlineSnapshot,omitempty"`
}

func (s *Store) ListSubscriptions() ([]Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]Subscription{}, s.data.Subscriptions...)
	for i := range result {
		result[i].Feed.Items = append([]feeds.Item{}, result[i].Feed.Items...)
		for j := range result[i].Feed.Items {
			result[i].Feed.Items[j].Key = feeds.ItemKey(result[i].Feed.Items[j])
			result[i].Feed.Items[j].Attachments = append([]string(nil), result[i].Feed.Items[j].Attachments...)
			result[i].Feed.Items[j].Media = append([]feeds.RemoteMedia(nil), result[i].Feed.Items[j].Media...)
			if assets := result[i].Feed.Items[j].OfflineMedia; assets != nil {
				copyAssets := make(map[string]feeds.OfflineAsset, len(assets))
				for key, value := range assets {
					copyAssets[key] = value
				}
				result[i].Feed.Items[j].OfflineMedia = copyAssets
			}
		}
	}
	return result, nil
}

func (s *Store) RemoveSubscription(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Subscriptions = make([]Subscription, 0, len(s.data.Subscriptions))
	found := false
	for _, sub := range s.data.Subscriptions {
		if sub.URL == address {
			found = true
			continue
		}
		next.Subscriptions = append(next.Subscriptions, sub)
	}
	if !found {
		return errors.New("订阅不存在")
	}
	return s.write(next)
}

func (s *Store) SaveSubscription(address string, feed feeds.Feed) (Subscription, error) {
	return s.saveSubscription(address, feed, nil, "", nil)
}

func (s *Store) SavePlanetSubscription(address string, feed feeds.Feed, cid, pinnedCID string) (Subscription, error) {
	if cid == "" {
		return Subscription{}, errors.New("站点 CID 无效")
	}
	return s.saveSubscription(address, feed, &cid, pinnedCID, nil)
}

func (s *Store) SaveOfflineSubscription(address string, feed feeds.Feed, snapshot string) (Subscription, error) {
	if snapshot == "" {
		return Subscription{}, errors.New("离线缓存标识无效")
	}
	return s.saveSubscription(address, feed, nil, "", &snapshot)
}

func (s *Store) saveSubscription(address string, feed feeds.Feed, cid *string, pinnedCID string, offlineSnapshot *string) (Subscription, error) {
	if feed.IconPNG != "" {
		if _, _, err := siteimage.DecodePNG(feed.IconPNG); err != nil {
			feed.IconPNG = ""
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	feed.Items = append([]feeds.Item{}, feed.Items...)
	for i := range feed.Items {
		feed.Items[i].Key = feeds.ItemKey(feed.Items[i])
		feed.Items[i].Starred = false
		feed.Items[i].Read = false
	}
	now := time.Now().UTC()
	subscription := Subscription{URL: address, Feed: feed, LastUpdated: &now}
	if offlineSnapshot != nil {
		subscription.OfflinePinned = true
		subscription.OfflineSnapshot = *offlineSnapshot
	}
	if cid != nil {
		subscription.CID = *cid
		subscription.PinnedCID = pinnedCID
	}
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	found := false
	for i, item := range next.Subscriptions {
		if item.URL == address {
			subscription.CustomTitle = item.CustomTitle
			if subscription.Feed.IconPNG == "" && item.Feed.IconPNG != "" {
				subscription.Feed.IconPNG = item.Feed.IconPNG
			}
			if offlineSnapshot == nil {
				subscription.OfflinePinned = item.OfflinePinned
				subscription.OfflineSnapshot = item.OfflineSnapshot
			}
			if cid == nil {
				subscription.CID = item.CID
				subscription.PinnedCID = item.PinnedCID
			} else if pinnedCID == "" {
				subscription.PinnedCID = item.PinnedCID
			}
			seen := make(map[string]bool)
			identity := feeds.ItemKey
			stars := map[string]bool{}
			reads := map[string]bool{}
			for _, old := range item.Feed.Items {
				stars[identity(old)] = old.Starred
				reads[identity(old)] = old.Read
			}
			for i := range subscription.Feed.Items {
				subscription.Feed.Items[i].Starred = stars[identity(subscription.Feed.Items[i])]
				subscription.Feed.Items[i].Read = reads[identity(subscription.Feed.Items[i])]
			}

			for _, entry := range subscription.Feed.Items {
				if key := identity(entry); key != "" {
					seen[key] = true
				}
			}
			for _, entry := range item.Feed.Items {
				key := identity(entry)
				if key == "" || !seen[key] {
					subscription.Feed.Items = append(subscription.Feed.Items, entry)
					if key != "" {
						seen[key] = true
					}
				}
			}
			next.Subscriptions[i] = subscription
			found = true
			break
		}
	}
	if !found {
		next.Subscriptions = append(next.Subscriptions, subscription)
	}
	if err := s.write(next); err != nil {
		return Subscription{}, err
	}
	return subscription, nil
}

// UpdateSubscriptionTitle saves a user-chosen display name for an existing subscription.
func (s *Store) UpdateSubscriptionTitle(address, customTitle string) (Subscription, error) {
	customTitle = strings.TrimSpace(customTitle)
	if len([]rune(customTitle)) > 200 {
		return Subscription{}, errors.New("订阅名称不能超过 200 个字符")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	for i := range next.Subscriptions {
		if next.Subscriptions[i].URL != address {
			continue
		}
		next.Subscriptions[i].CustomTitle = customTitle
		if err := s.write(next); err != nil {
			return Subscription{}, err
		}
		return next.Subscriptions[i], nil
	}
	return Subscription{}, errors.New("订阅不存在")
}

// ReplaceSubscription updates a subscription's address, feed content and display name in place.
func (s *Store) ReplaceSubscription(oldAddress, newAddress string, feed feeds.Feed, cid, pinnedCID, customTitle string) (Subscription, error) {
	customTitle = strings.TrimSpace(customTitle)
	if len([]rune(customTitle)) > 200 {
		return Subscription{}, errors.New("订阅名称不能超过 200 个字符")
	}
	if newAddress == "" {
		return Subscription{}, errors.New("订阅地址无效")
	}
	if feed.IconPNG != "" {
		if _, _, err := siteimage.DecodePNG(feed.IconPNG); err != nil {
			feed.IconPNG = ""
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldIndex := -1
	for i, item := range s.data.Subscriptions {
		if item.URL == oldAddress {
			oldIndex = i
		} else if item.URL == newAddress {
			return Subscription{}, errors.New("该订阅地址已存在")
		}
	}
	if oldIndex < 0 {
		return Subscription{}, errors.New("订阅不存在")
	}
	old := s.data.Subscriptions[oldIndex]
	feed.Items = append([]feeds.Item{}, feed.Items...)
	stars := map[string]bool{}
	reads := map[string]bool{}
	for _, entry := range old.Feed.Items {
		key := feeds.ItemKey(entry)
		stars[key] = entry.Starred
		reads[key] = entry.Read
	}
	for i := range feed.Items {
		key := feeds.ItemKey(feed.Items[i])
		feed.Items[i].Key = key
		feed.Items[i].Starred = stars[key]
		feed.Items[i].Read = reads[key]
	}
	if feed.IconPNG == "" && old.Feed.IconPNG != "" {
		feed.IconPNG = old.Feed.IconPNG
	}
	now := time.Now().UTC()
	updated := Subscription{
		URL:         newAddress,
		CustomTitle: customTitle,
		Feed:        feed,
		CID:         cid,
		PinnedCID:   pinnedCID,
		LastUpdated: &now,
	}
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	next.Subscriptions[oldIndex] = updated
	if err := s.write(next); err != nil {
		return Subscription{}, err
	}
	return updated, nil
}

// SetSubscriptionPinned records only a pin for the current immutable revision.
func (s *Store) SetSubscriptionPinned(address, cid string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	for i := range next.Subscriptions {
		sub := &next.Subscriptions[i]
		if sub.URL == address {
			if cid == "" || sub.CID != cid {
				return errors.New("站点版本已改变，请重新固定")
			}
			sub.PinnedCID = cid
			return s.write(next)
		}
	}
	return errors.New("订阅不存在")
}

func (s *Store) SetSubscriptionError(address, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	if chars := []rune(message); len(chars) > 1000 {
		message = string(chars[:1000])
	}
	for i := range next.Subscriptions {
		if next.Subscriptions[i].URL == address {
			next.Subscriptions[i].LastError = message
			return s.write(next)
		}
	}
	return errors.New("订阅不存在")
}

// SetSubscriptionIcon saves a late icon only for the feed revision that supplied it.
func (s *Store) SetSubscriptionIcon(address string, updatedAt time.Time, icon string) (bool, error) {
	if _, _, err := siteimage.DecodePNG(icon); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, sub := range s.data.Subscriptions {
		if sub.URL != address || sub.LastUpdated == nil || !sub.LastUpdated.Equal(updatedAt) || sub.Feed.IconPNG == icon {
			continue
		}
		next := s.data
		next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
		next.Subscriptions[i].Feed.IconPNG = icon
		if err := s.write(next); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// SetSubscriptionStarred persists the state without replacing refreshed content.
func (s *Store) SetSubscriptionStarred(address, key string, starred bool) error {
	return s.SetSubscriptionStar(address, key, starred, StarTypeDefault)
}

// SetSubscriptionRead changes local reading state without replacing feed data.
func (s *Store) SetSubscriptionRead(address, key string, read bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return errors.New("文章标识无效")
	}
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	for i, sub := range next.Subscriptions {
		if sub.URL != address {
			continue
		}
		sub.Feed.Items = append([]feeds.Item{}, sub.Feed.Items...)
		for j := range sub.Feed.Items {
			if feeds.ItemKey(sub.Feed.Items[j]) != key {
				continue
			}
			if sub.Feed.Items[j].Read == read {
				return nil
			}
			sub.Feed.Items[j].Read = read
			sub.Feed.Items[j].Key = key
			next.Subscriptions[i] = sub
			return s.write(next)
		}
		return errors.New("订阅文章不存在")
	}
	return errors.New("订阅不存在")
}

// MarkSubscriptionRead marks the current cached items in one atomic write.
func (s *Store) MarkSubscriptionRead(address string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Subscriptions = append([]Subscription{}, s.data.Subscriptions...)
	for i, sub := range next.Subscriptions {
		if sub.URL != address {
			continue
		}
		sub.Feed.Items = append([]feeds.Item{}, sub.Feed.Items...)
		changed := false
		for j := range sub.Feed.Items {
			if !sub.Feed.Items[j].Read {
				sub.Feed.Items[j].Read = true
				changed = true
			}
		}
		if !changed {
			return nil
		}
		next.Subscriptions[i] = sub
		return s.write(next)
	}
	return errors.New("订阅不存在")
}

// SetSubscriptionStar sets or clears a subscription item's star and star type.
func (s *Store) SetSubscriptionStar(address, key string, starred bool, starType int) error {
	if !ValidStarType(starType) {
		return errors.New("星标类型无效")
	}
	if !starred {
		starType = StarTypeDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if key == "" {
		return errors.New("文章标识无效")
	}
	next := s.data
	next.Subscriptions = append([]Subscription(nil), s.data.Subscriptions...)
	for i, sub := range next.Subscriptions {
		if sub.URL == address {
			sub.Feed.Items = append([]feeds.Item(nil), sub.Feed.Items...)
			for j := range sub.Feed.Items {
				if feeds.ItemKey(sub.Feed.Items[j]) == key {
					sub.Feed.Items[j].Starred = starred
					sub.Feed.Items[j].StarType = starType
					sub.Feed.Items[j].Key = key
					next.Subscriptions[i] = sub
					return s.write(next)
				}
			}
			return errors.New("订阅文章不存在")
		}
	}
	return errors.New("订阅不存在")
}
