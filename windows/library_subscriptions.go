package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/feeds"
	"windows/internal/follow"
	"windows/internal/offline"
	"windows/internal/preview"
	"windows/internal/store"
)

func (l *Library) SetSubscriptionStar(address, key string, starred bool, starType int) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.SetSubscriptionStar(address, key, starred, starType)
}

func (l *Library) Subscriptions() ([]store.Subscription, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.ListSubscriptions()
}

// Follow adds or refreshes either a web feed or a Planet/IPNS/ENS site.
func (l *Library) Follow(address string) (store.Subscription, error) {
	return l.follow(address, false)
}

// RefreshSubscription only refreshes an existing subscription. A refresh
// queued before removal must not recreate the removed subscription.
func (l *Library) RefreshSubscription(address string) (store.Subscription, error) {
	return l.follow(address, true)
}

func (l *Library) follow(address string, requireExisting bool) (store.Subscription, error) {
	address = strings.TrimSpace(address)
	s, err := l.open()
	if err != nil {
		return store.Subscription{}, err
	}
	if strings.HasPrefix(strings.ToLower(address), "http://") || strings.HasPrefix(strings.ToLower(address), "https://") {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
		if requireExisting {
			if err := requireSubscription(s, address); err != nil {
				return store.Subscription{}, err
			}
		}
		feed, err := feeds.Fetch(address)
		if err != nil {
			_ = s.SetSubscriptionError(address, err.Error())
			return store.Subscription{}, err
		}
		subs, e := s.ListSubscriptions()
		if e != nil {
			return store.Subscription{}, e
		}
		pinned := false
		var previous feeds.Feed
		for _, sub := range subs {
			if sub.URL == address && sub.OfflinePinned {
				pinned = true
				previous = sub.Feed
				break
			}
		}
		if pinned {
			root, e := mygo.App.Path(mygo.PathUserData)
			if e != nil {
				return store.Subscription{}, e
			}
			saved, snapshot, e := offline.PinWithPrevious(root, address, feed, previous)
			if e != nil {
				_ = s.SetSubscriptionError(address, e.Error())
				return store.Subscription{}, e
			}
			result, e := s.SaveOfflineSubscription(address, saved, snapshot)
			if e != nil {
				_ = offline.Discard(root, snapshot)
			}
			return result, e
		}
		return s.SaveSubscription(address, feed)
	}
	name, err := follow.Normalize(address)
	if err != nil {
		return store.Subscription{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if requireExisting {
		if err := requireSubscription(s, name); err != nil {
			return store.Subscription{}, err
		}
	}
	if _, _, err = l.ensureNode(true); err != nil {
		_ = s.SetSubscriptionError(name, err.Error())
		return store.Subscription{}, err
	}
	pinned := false
	subs, err := s.ListSubscriptions()
	if err != nil {
		return store.Subscription{}, err
	}
	for _, sub := range subs {
		if sub.URL == name && sub.PinnedCID != "" {
			pinned = true
			break
		}
	}
	result, err := follow.Fetch(l.node, name, pinned)
	if err != nil {
		_ = s.SetSubscriptionError(name, err.Error())
		return store.Subscription{}, err
	}
	pinnedCID := ""
	if pinned {
		pinnedCID = result.CID
	}
	return s.SavePlanetSubscription(name, result.Feed, result.CID, pinnedCID)
}

func requireSubscription(s *store.Store, address string) error {
	subs, err := s.ListSubscriptions()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.URL == address {
			return nil
		}
	}
	return errors.New("订阅不存在")
}

func isHTTPAddress(address string) bool {
	lower := strings.ToLower(address)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// UpdateSubscription edits an existing subscription's display name and/or address.
func (l *Library) UpdateSubscription(oldAddress, newAddress, customTitle string) (store.Subscription, error) {
	oldAddress = strings.TrimSpace(oldAddress)
	newAddress = strings.TrimSpace(newAddress)
	if oldAddress == "" {
		return store.Subscription{}, errors.New("订阅不存在")
	}
	if newAddress == "" {
		return store.Subscription{}, errors.New("请输入订阅地址")
	}
	oldIsHTTP := isHTTPAddress(oldAddress)
	if !oldIsHTTP {
		normalized, err := follow.Normalize(oldAddress)
		if err != nil {
			return store.Subscription{}, err
		}
		oldAddress = normalized
	}
	newIsHTTP := isHTTPAddress(newAddress)
	if !newIsHTTP {
		normalized, err := follow.Normalize(newAddress)
		if err != nil {
			return store.Subscription{}, err
		}
		newAddress = normalized
	}
	s, err := l.open()
	if err != nil {
		return store.Subscription{}, err
	}
	if newAddress == oldAddress {
		if oldIsHTTP {
			l.feedMu.Lock()
			defer l.feedMu.Unlock()
		} else {
			l.publicMu.Lock()
			defer l.publicMu.Unlock()
		}
		return s.UpdateSubscriptionTitle(oldAddress, customTitle)
	}
	if !oldIsHTTP || !newIsHTTP {
		l.publicMu.Lock()
		defer l.publicMu.Unlock()
	}
	if oldIsHTTP || newIsHTTP {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return store.Subscription{}, err
	}
	found := false
	for _, sub := range subs {
		if sub.URL == oldAddress {
			found = true
		} else if sub.URL == newAddress {
			return store.Subscription{}, errors.New("该订阅地址已存在")
		}
	}
	if !found {
		return store.Subscription{}, errors.New("订阅不存在")
	}
	if newIsHTTP {
		feed, err := feeds.Fetch(newAddress)
		if err != nil {
			return store.Subscription{}, err
		}
		return s.ReplaceSubscription(oldAddress, newAddress, feed, "", "", customTitle)
	}
	if _, _, err = l.ensureNode(true); err != nil {
		return store.Subscription{}, err
	}
	result, err := follow.Fetch(l.node, newAddress, false)
	if err != nil {
		return store.Subscription{}, err
	}
	return s.ReplaceSubscription(oldAddress, newAddress, result.Feed, result.CID, "", customTitle)
}

// Unfollow removes the subscription and its saved item state. Shared IPFS pins
// and cached files are not deleted by this operation.
func (l *Library) Unfollow(address string) error {
	address = strings.TrimSpace(address)
	s, err := l.open()
	if err != nil {
		return err
	}
	if strings.HasPrefix(strings.ToLower(address), "http://") || strings.HasPrefix(strings.ToLower(address), "https://") {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
		return s.RemoveSubscription(address)
	}
	address, err = follow.Normalize(address)
	if err != nil {
		return err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	return s.RemoveSubscription(address)
}

// PinSubscription stores the current Planet revision in Kubo for offline use.
func (l *Library) PinSubscription(address string) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	if strings.HasPrefix(strings.ToLower(address), "http://") || strings.HasPrefix(strings.ToLower(address), "https://") {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
		subs, e := s.ListSubscriptions()
		if e != nil {
			return e
		}
		var selected *store.Subscription
		for i := range subs {
			if subs[i].URL == address {
				selected = &subs[i]
				break
			}
		}
		if selected == nil {
			return errors.New("订阅不存在")
		}
		root, e := mygo.App.Path(mygo.PathUserData)
		if e != nil {
			return e
		}
		saved, snapshot, e := offline.PinWithPrevious(root, address, selected.Feed, selected.Feed)
		if e != nil {
			return e
		}
		_, e = s.SaveOfflineSubscription(address, saved, snapshot)
		if e != nil {
			_ = offline.Discard(root, snapshot)
		}
		return e
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	subs, err := s.ListSubscriptions()
	if err != nil {
		return err
	}
	cid := ""
	for _, sub := range subs {
		if sub.URL == address {
			cid = sub.CID
			break
		}
	}
	if cid == "" {
		return errors.New("此订阅没有可固定的 IPFS 内容")
	}
	if _, _, err = l.ensureNode(true); err != nil {
		return err
	}
	if err = l.node.Pin(cid); err != nil {
		return err
	}
	return s.SetSubscriptionPinned(address, cid)
}

func (l *Library) StarSubscription(address, key string, starred bool) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.SetSubscriptionStarred(address, key, starred)
}

func (l *Library) SetSubscriptionRead(address, key string, read bool) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.SetSubscriptionRead(address, key, read)
}

func (l *Library) MarkSubscriptionRead(address string) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.MarkSubscriptionRead(address)
}

// SubscriptionPreview renders registered Planet images from the local IPFS repository.
func (l *Library) SubscriptionPreview(address, key string) (string, error) {
	l.feedMu.Lock()
	defer l.feedMu.Unlock()
	s, err := l.open()
	if err != nil {
		return "", err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return "", err
	}
	var selected *feeds.Item
	for _, sub := range subs {
		if sub.URL == address {
			for i := range sub.Feed.Items {
				item := &sub.Feed.Items[i]
				if feeds.ItemKey(*item) == key {
					selected = item
					break
				}
			}
			break
		}
	}
	if selected == nil {
		return "", errors.New("订阅文章不存在")
	}
	if selected.ContentCID == "" && selected.OfflineBase != "" {
		root, e := mygo.App.Path(mygo.PathUserData)
		if e != nil {
			return "", e
		}
		return offline.Render(root, *selected)
	}
	if selected.ContentCID == "" || selected.ArticleID == "" {
		return "", errors.New("订阅文章不存在")
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if _, _, err = l.ensureNode(true); err != nil {
		return "", err
	}
	allowed := map[string]bool{}
	for _, name := range selected.Attachments {
		allowed[name] = true
	}
	return preview.HTMLWithImages(selected.Content, func(name string) ([]byte, string, error) {
		if !allowed[name] {
			return nil, "", errors.New("附件未登记")
		}
		path := selected.ArticleID + "/" + name
		if selected.ContentPath != "" {
			path = selected.ContentPath + "/" + path
		}
		data, err := l.node.Read(selected.ContentCID, path)
		if err != nil {
			return nil, "", err
		}
		if len(data) > 8<<20 {
			return nil, "", errors.New("图片过大")
		}
		mime := http.DetectContentType(data)
		return data, mime, nil
	})
}
