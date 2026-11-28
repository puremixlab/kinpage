package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/feeds"
	"windows/internal/follow"
	"windows/internal/nativeui"
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
	return completeSubscription(l.FollowContext(context.Background(), address))
}

func (l *Library) FollowContext(ctx context.Context, address string) (nativeui.SubscriptionUpdate, error) {
	return l.follow(ctx, address, false, time.Time{})
}

// RefreshSubscription only refreshes an existing subscription. A refresh
// queued before removal must not recreate the removed subscription.
func (l *Library) RefreshSubscription(address string) (store.Subscription, error) {
	return completeSubscription(l.RefreshSubscriptionContext(context.Background(), address))
}

func (l *Library) RefreshSubscriptionContext(ctx context.Context, address string) (nativeui.SubscriptionUpdate, error) {
	return l.follow(ctx, address, true, time.Time{})
}

func (l *Library) follow(ctx context.Context, address string, requireExisting bool, staleBefore time.Time) (nativeui.SubscriptionUpdate, error) {
	address = strings.TrimSpace(address)
	s, err := l.open()
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	if isHTTPAddress(address) {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
		if err := ctx.Err(); err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		if requireExisting {
			if err := requireSubscription(s, address, staleBefore); err != nil {
				return nativeui.SubscriptionUpdate{}, err
			}
		}
		feed, loadIcon, err := feeds.FetchContent(ctx, address)
		if err != nil {
			return nativeui.SubscriptionUpdate{}, recordSubscriptionError(ctx, s, address, err)
		}
		subs, e := s.ListSubscriptions()
		if e != nil {
			return nativeui.SubscriptionUpdate{}, e
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
				return nativeui.SubscriptionUpdate{}, e
			}
			saved, snapshot, e := offline.PinWithPreviousContext(ctx, root, address, feed, previous)
			if e != nil {
				return nativeui.SubscriptionUpdate{}, recordSubscriptionError(ctx, s, address, e)
			}
			if err := ctx.Err(); err != nil {
				_ = offline.Discard(root, snapshot)
				return nativeui.SubscriptionUpdate{}, err
			}
			result, err := s.SaveOfflineSubscription(address, saved, snapshot)
			if err != nil {
				_ = offline.Discard(root, snapshot)
			}
			return subscriptionUpdate(s, result, loadIcon), err
		}
		if err := ctx.Err(); err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		sub, err := s.SaveSubscription(address, feed)
		return subscriptionUpdate(s, sub, loadIcon), err
	}
	name, err := follow.Normalize(address)
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	if requireExisting {
		if err := requireSubscription(s, name, staleBefore); err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		if !staleBefore.IsZero() && s.IPFSDisabled() {
			return nativeui.SubscriptionUpdate{}, errSubscriptionNotDue
		}
	}
	if _, _, err = l.ensureNode(true); err != nil {
		return nativeui.SubscriptionUpdate{}, recordSubscriptionError(ctx, s, name, err)
	}
	pinned := false
	subs, err := s.ListSubscriptions()
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	for _, sub := range subs {
		if sub.URL == name && sub.PinnedCID != "" {
			pinned = true
			break
		}
	}
	result, err := follow.FetchContent(ctx, l.node, name, pinned)
	if err != nil {
		return nativeui.SubscriptionUpdate{}, recordSubscriptionError(ctx, s, name, err)
	}
	if err := ctx.Err(); err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	pinnedCID := ""
	if pinned {
		pinnedCID = result.CID
	}
	sub, err := s.SavePlanetSubscription(name, result.Feed, result.CID, pinnedCID)
	return subscriptionUpdate(s, sub, result.LoadIcon), err
}

func recordSubscriptionError(ctx context.Context, s *store.Store, address string, err error) error {
	if cancelled := ctx.Err(); cancelled != nil {
		return cancelled
	}
	_ = s.SetSubscriptionError(address, err.Error())
	return err
}

var (
	errSubscriptionMissing = errors.New("订阅不存在")
	errSubscriptionNotDue  = errors.New("订阅暂不需要自动刷新")
)

func requireSubscription(s *store.Store, address string, staleBefore time.Time) error {
	subs, err := s.ListSubscriptions()
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.URL == address {
			if !staleBefore.IsZero() && sub.LastUpdated != nil && sub.LastUpdated.After(staleBefore) {
				return errSubscriptionNotDue
			}
			return nil
		}
	}
	return errSubscriptionMissing
}

func isHTTPAddress(address string) bool {
	lower := strings.ToLower(address)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// UpdateSubscription edits an existing subscription's display name and/or address.
func (l *Library) UpdateSubscription(oldAddress, newAddress, customTitle string) (store.Subscription, error) {
	return completeSubscription(l.UpdateSubscriptionContext(context.Background(), oldAddress, newAddress, customTitle))
}

func (l *Library) UpdateSubscriptionContext(ctx context.Context, oldAddress, newAddress, customTitle string) (nativeui.SubscriptionUpdate, error) {
	oldAddress = strings.TrimSpace(oldAddress)
	newAddress = strings.TrimSpace(newAddress)
	if oldAddress == "" {
		return nativeui.SubscriptionUpdate{}, errors.New("订阅不存在")
	}
	if newAddress == "" {
		return nativeui.SubscriptionUpdate{}, errors.New("请输入订阅地址")
	}
	oldIsHTTP := isHTTPAddress(oldAddress)
	if !oldIsHTTP {
		normalized, err := follow.Normalize(oldAddress)
		if err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		oldAddress = normalized
	}
	newIsHTTP := isHTTPAddress(newAddress)
	if !newIsHTTP {
		normalized, err := follow.Normalize(newAddress)
		if err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		newAddress = normalized
	}
	s, err := l.open()
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	if newAddress == oldAddress {
		if oldIsHTTP {
			l.feedMu.Lock()
			defer l.feedMu.Unlock()
		} else {
			l.publicMu.Lock()
			defer l.publicMu.Unlock()
		}
		if err := ctx.Err(); err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		sub, err := s.UpdateSubscriptionTitle(oldAddress, customTitle)
		return nativeui.SubscriptionUpdate{Subscription: sub}, err
	}
	if !oldIsHTTP || !newIsHTTP {
		l.publicMu.Lock()
		defer l.publicMu.Unlock()
	}
	if oldIsHTTP || newIsHTTP {
		l.feedMu.Lock()
		defer l.feedMu.Unlock()
	}
	if err := ctx.Err(); err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	found := false
	for _, sub := range subs {
		if sub.URL == oldAddress {
			found = true
		} else if sub.URL == newAddress {
			return nativeui.SubscriptionUpdate{}, errors.New("该订阅地址已存在")
		}
	}
	if !found {
		return nativeui.SubscriptionUpdate{}, errors.New("订阅不存在")
	}
	if newIsHTTP {
		feed, loadIcon, err := feeds.FetchContent(ctx, newAddress)
		if err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		if err := ctx.Err(); err != nil {
			return nativeui.SubscriptionUpdate{}, err
		}
		sub, err := s.ReplaceSubscription(oldAddress, newAddress, feed, "", "", customTitle)
		return subscriptionUpdate(s, sub, loadIcon), err
	}
	if _, _, err = l.ensureNode(true); err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	result, err := follow.FetchContent(ctx, l.node, newAddress, false)
	if err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	if err := ctx.Err(); err != nil {
		return nativeui.SubscriptionUpdate{}, err
	}
	sub, err := s.ReplaceSubscription(oldAddress, newAddress, result.Feed, result.CID, "", customTitle)
	return subscriptionUpdate(s, sub, result.LoadIcon), err
}

func subscriptionUpdate(s *store.Store, sub store.Subscription, loadIcon feeds.IconLoader) nativeui.SubscriptionUpdate {
	update := nativeui.SubscriptionUpdate{Subscription: sub}
	if loadIcon == nil || sub.LastUpdated == nil {
		return update
	}
	updatedAt := *sub.LastUpdated
	update.FinishIcon = func(ctx context.Context) string {
		icon := loadIcon(ctx)
		if icon == "" || ctx.Err() != nil {
			return ""
		}
		changed, err := s.SetSubscriptionIcon(sub.URL, updatedAt, icon)
		if err != nil || !changed {
			return ""
		}
		return icon
	}
	return update
}

func completeSubscription(update nativeui.SubscriptionUpdate, err error) (store.Subscription, error) {
	sub := update.Subscription
	if err == nil && update.FinishIcon != nil {
		if icon := update.FinishIcon(context.Background()); icon != "" {
			sub.Feed.IconPNG = icon
		}
	}
	return sub, err
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
