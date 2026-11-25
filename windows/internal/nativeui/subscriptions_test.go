package nativeui

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/feeds"
	"windows/internal/store"
)

func TestNativeUISubscriptionsFollowStarEditAndUnfollow(t *testing.T) {
	backend := &fakeBackend{}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	tt.Frame()

	// Follow a new subscription.
	if err := tt.Click("+ 关注"); err != nil {
		t.Fatalf("failed to open follow modal: %v", err)
	}
	app.followAddress = "https://example.com/feed.xml"
	tt.Frame()
	if err := tt.Click("关注"); err != nil {
		t.Fatalf("failed to submit follow modal: %v", err)
	}
	if len(backend.subscriptions) != 1 || app.mode != viewSubscription {
		t.Fatalf("expected subscription followed and active, subs=%d mode=%v", len(backend.subscriptions), app.mode)
	}

	// Opening the subscription auto-selected and marked the first item read.
	if !backend.subscriptions[0].Feed.Items[0].Read {
		t.Fatal("expected first subscription item to be marked read on open")
	}

	// Star the item and verify starred view lists it.
	if err := tt.Click("☆ 星标"); err != nil {
		t.Fatalf("failed to star subscription item: %v", err)
	}
	if !backend.subscriptions[0].Feed.Items[0].Starred {
		t.Fatal("expected subscription item starred")
	}

	// Edit subscription custom title.
	if err := tt.Click("编辑"); err != nil {
		t.Fatalf("failed to open edit subscription modal: %v", err)
	}
	app.editSubTitle = "精选技术周刊"
	tt.Frame()
	if err := tt.Click("保存订阅"); err != nil {
		t.Fatalf("failed to save subscription edit: %v", err)
	}
	if backend.subscriptions[0].CustomTitle != "精选技术周刊" {
		t.Fatalf("expected custom title updated, got %q", backend.subscriptions[0].CustomTitle)
	}

	// Unfollow subscription with confirmation dialog.
	if err := tt.Click("取消关注"); err != nil {
		t.Fatalf("failed to open unfollow confirmation: %v", err)
	}
	if !app.unfollowOpen {
		t.Fatal("expected unfollow confirmation dialog open")
	}
	if err := tt.Click("取消关注"); err != nil {
		t.Fatalf("failed to confirm unfollow: %v", err)
	}
	if len(backend.subscriptions) != 0 {
		t.Fatalf("expected 0 subscriptions after unfollow, got %d", len(backend.subscriptions))
	}
}

func TestNativeUIGlobalSearchAndRelatedArticles(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "我的博客", Created: now}},
		articles: []store.Article{{
			ID: "a1", SiteID: "s1", Title: "MyGo 原生渲染指南", Content: "深入 Direct3D 渲染",
			Created: now, Modified: now,
		}},
		subscriptions: []store.Subscription{{
			URL: "https://planet.example/rss",
			Feed: feeds.Feed{
				Title: "外部观察",
				Items: []feeds.Item{{
					Key:     "ext-1",
					Title:   "原生桌面架构演进",
					Content: "<h1>标题</h1><p>告别 <strong>WebView</strong></p>",
				}},
			},
		}},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	tt.Frame()

	// Global search for subscription item.
	if err := tt.Click("搜索文章"); err != nil {
		t.Fatalf("failed to open global search modal: %v", err)
	}
	app.searchQuery = "原生"
	tt.Frame()
	if err := tt.Click("搜索"); err != nil {
		t.Fatalf("failed to trigger global search: %v", err)
	}
	if len(app.searchResults) != 2 {
		t.Fatalf("expected 2 search results, got %d", len(app.searchResults))
	}

	if err := tt.Click("原生桌面架构演进  ·  外部观察"); err != nil {
		t.Fatalf("failed to jump to subscription search result: %v", err)
	}
	if app.mode != viewSubscription || app.selectedSubItemKey != "ext-1" {
		t.Fatalf("expected jumped to subscription item ext-1, mode=%v key=%q", app.mode, app.selectedSubItemKey)
	}

	// Open related articles from subscription reader and jump to own article.
	if err := tt.Click("相关文章"); err != nil {
		t.Fatalf("failed to open related articles modal: %v", err)
	}
	if !app.relatedModalOpen || len(app.relatedResults) != 1 {
		t.Fatalf("expected 1 related article, open=%v count=%d", app.relatedModalOpen, len(app.relatedResults))
	}
	if err := tt.Click("MyGo 原生渲染指南  ·  我的博客"); err != nil {
		t.Fatalf("failed to jump to related article: %v", err)
	}
	if app.mode != viewSite || app.selectedReadID != "a1" {
		t.Fatalf("expected jumped to own article a1, mode=%v id=%q", app.mode, app.selectedReadID)
	}
}

func TestHTMLToReadableMarkdown(t *testing.T) {
	md := htmlToReadableMarkdown(`<h2>标题</h2><p>段落 <strong>加粗</strong> 与 <a href="https://example.com">链接</a></p><ul><li>条目一</li></ul><script>alert(1)</script>`)
	for _, want := range []string{"## 标题", "**加粗**", "[链接](https://example.com)", "- 条目一"} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in converted markdown:\n%s", want, md)
		}
	}
	if strings.Contains(md, "alert(1)") {
		t.Fatalf("expected script content stripped:\n%s", md)
	}
}
