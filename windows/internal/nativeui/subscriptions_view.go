package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobemd "github.com/lobehub/lobe-mygo/pkg/markdown"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"golang.org/x/net/html"
	"windows/internal/feeds"
	"windows/internal/store"
)

type subItemEntry struct {
	subURL    string
	subTitle  string
	item      feeds.Item
	effective string
}

func subscriptionTitle(s store.Subscription) string {
	if strings.TrimSpace(s.CustomTitle) != "" {
		return s.CustomTitle
	}
	if strings.TrimSpace(s.Feed.Title) != "" {
		return s.Feed.Title
	}
	return s.URL
}

func feedItemKey(item feeds.Item) string {
	if item.Key != "" {
		return item.Key
	}
	if item.ID != "" {
		return item.ID
	}
	if item.Link != "" {
		return item.Link
	}
	return item.Title
}

func (a *App) totalTodayCount() int {
	today := time.Now().Format("2006-01-02")
	n := 0
	for _, s := range a.subscriptions {
		for _, item := range s.Feed.Items {
			if strings.HasPrefix(item.Published, today) {
				n++
			}
		}
	}
	return n
}

func (a *App) totalUnreadCount() int {
	n := 0
	for _, s := range a.subscriptions {
		for _, item := range s.Feed.Items {
			if !item.Read {
				n++
			}
		}
	}
	return n
}

func (a *App) totalStarredCount() int {
	n := 0
	for _, s := range a.subscriptions {
		for _, item := range s.Feed.Items {
			if item.Starred {
				n++
			}
		}
	}
	return n
}

func (a *App) openSubscriptionView(mode viewMode, subURL string) {
	a.mode = mode
	a.editing = false
	a.selectedSubURL = subURL
	a.selectedSubItemSubURL = ""
	a.selectedSubItemKey = ""
	a.subSearchQuery = ""
	a.listSearchOpen = false
	a.listFilter = listFilterAll
	a.starTypeFilter = -1
	a.clearStatus()
	items := a.visibleSubscriptionItems()
	if len(items) > 0 {
		a.selectSubItem(items[0].subURL, items[0].effective)
	}
}

func (a *App) visibleSubscriptionItems() []subItemEntry {
	query := strings.ToLower(strings.TrimSpace(a.subSearchQuery))
	var out []subItemEntry
	for _, s := range a.subscriptions {
		if a.mode == viewSubscription && s.URL != a.selectedSubURL {
			continue
		}
		stitle := subscriptionTitle(s)
		for _, item := range s.Feed.Items {
			if a.mode == viewToday && !strings.HasPrefix(item.Published, time.Now().Format("2006-01-02")) {
				continue
			}
			if a.mode == viewUnread && item.Read {
				continue
			}
			if a.mode == viewStarred {
				if !item.Starred {
					continue
				}
				if a.starTypeFilter >= 0 && item.StarType != a.starTypeFilter {
					continue
				}
			}
			if a.listFilter == listFilterUnread && item.Read {
				continue
			}
			if a.listFilter == listFilterStarred {
				if !item.Starred {
					continue
				}
				if a.starTypeFilter >= 0 && item.StarType != a.starTypeFilter {
					continue
				}
			}
			if query != "" {
				hay := strings.ToLower(item.Title + " " + stitle)
				if !strings.Contains(hay, query) {
					continue
				}
			}
			out = append(out, subItemEntry{
				subURL:    s.URL,
				subTitle:  stitle,
				item:      item,
				effective: feedItemKey(item),
			})
		}
	}
	return out
}

func (a *App) currentSubItem() (*subItemEntry, bool) {
	for _, s := range a.subscriptions {
		if s.URL != a.selectedSubItemSubURL {
			continue
		}
		for _, item := range s.Feed.Items {
			k := feedItemKey(item)
			if k == a.selectedSubItemKey {
				e := subItemEntry{subURL: s.URL, subTitle: subscriptionTitle(s), item: item, effective: k}
				return &e, true
			}
		}
	}
	return nil, false
}

func (a *App) currentSubscription() *store.Subscription {
	for i := range a.subscriptions {
		if a.subscriptions[i].URL == a.selectedSubURL {
			return &a.subscriptions[i]
		}
	}
	return nil
}

func (a *App) selectSubItem(subURL, itemKey string) {
	a.selectedSubItemSubURL = subURL
	a.selectedSubItemKey = itemKey
	a.subReaderScroll.Y = 0
	if cur, ok := a.currentSubItem(); ok && !cur.item.Read && a.backend != nil {
		if err := a.backend.SetSubscriptionRead(subURL, itemKey, true); err == nil {
			a.Reload()
		}
	}
}

func (a *App) viewSubscriptionWorkspace(c *ui.Context) {
	if a.postsCollapsed {
		a.viewSubscriptionReader(c)
		return
	}
	ui.Split(c.Key("sub-split"), &a.postsPanelWidth,
		func() { a.viewSubscriptionItemsList(c) },
		func() { a.viewSubscriptionReader(c) },
	).SplitReset(276).Grow(1).MinHeight(0).AlignSelf(ui.Stretch)
}

func (a *App) viewSubscriptionItemsList(c *ui.Context) {
	pal := a.pal
	heading := "订阅阅读"
	switch a.mode {
	case viewToday:
		heading = "今天"
	case viewUnread:
		heading = "未读"
	case viewStarred:
		heading = "收藏"
	case viewSubscription:
		if sub := a.currentSubscription(); sub != nil {
			heading = subscriptionTitle(*sub)
		}
	}

	items := a.visibleSubscriptionItems()
	unreadCount := 0
	for _, entry := range items {
		if !entry.item.Read {
			unreadCount++
		}
	}
	subtitle := fmt.Sprintf("%d 篇文章", len(items))
	if unreadCount > 0 {
		subtitle = fmt.Sprintf("%d 未读 · 共 %d 篇", unreadCount, len(items))
	}
	if a.listFilter != listFilterAll {
		subtitle += " · 已筛选"
	}

	ui.Column(c).Fill().MinWidth(240).Background(pal.canvas).Children(func() {
		// Full-height Column 2 integrated 44px header (macOS navigationTitle + navigationSubtitle + filter menu).
		a.viewListPaneHeader(c, heading, subtitle, true)

		if a.listSearchOpen {
			ui.Row(c).Padding(8, 10, 4, 10).Gap(4).AlignItems(ui.Center).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					field.SearchBar(c.Key("sub-filter"), &a.subSearchQuery, field.SearchBarProps{
						Placeholder: "筛选订阅条目…",
						Variant:     field.InputFilled,
						Size:        theme.Small,
					})
				})
				if toolbarIconBtn(c, "sub-filter-close", lucide.X(), "关闭筛选").Clicked() {
					a.listSearchOpen = false
					a.subSearchQuery = ""
				}
			})
		}

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(8).PaddingY(6).Gap(0).Children(func() {
				if len(items) == 0 {
					ui.Box(c).PaddingY(32).AlignSelf(ui.Stretch).Children(func() {
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.Inbox(),
							Title:       "暂无订阅内容",
							Description: "尝试切换分类或刷新订阅源",
						})
					})
				}
				for _, entry := range items {
					subURL, key := entry.subURL, entry.effective
					title := fallbackTitle(entry.item.Title)
					selected := a.selectedSubItemSubURL == subURL && a.selectedSubItemKey == key
					rawBody := entry.item.Content
					if strings.TrimSpace(rawBody) == "" {
						rawBody = entry.item.Description
					}
					sourceLabel := ""
					if a.mode != viewSubscription {
						sourceLabel = entry.subTitle
					}
					row := articleRow(c, articleRowProps{
						key:         "subitem-" + subURL + "-" + key,
						ariaLabel:   title,
						title:       title,
						summary:     excerptText(htmlToReadableMarkdown(rawBody), 80),
						dateText:    formatFeedDate(entry.item.Published),
						sourceLabel: sourceLabel,
						selected:    selected,
						starred:     entry.item.Starred,
						starType:    entry.item.StarType,
						unread:      !entry.item.Read,
					}, pal).ContextMenu(a.subItemContextMenu(entry))
					if row.Clicked() {
						a.selectSubItem(subURL, key)
					}
					if !selected {
						ui.Box(c).Padding(0, 12, 0, 24).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
					}
				}
			})
		})
	})
}

func (a *App) viewSubscriptionReader(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		a.viewDetailToolbar(c, true)
		cur, ok := a.currentSubItem()
		if !ok {
			ui.Column(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Background(pal.canvas).Center().Children(func() {
				display.Empty(c, display.EmptyProps{
					Icon:        lucide.BookOpen(),
					Title:       "选择一篇订阅文章",
					Description: "从左侧列表点击任意条目即可开始阅读",
				})
			})
			return
		}

		baseURL := strings.TrimSpace(cur.item.Link)
		if baseURL == "" {
			baseURL = cur.subURL
		}

		// Optional embedded WebView original webpage mode (matching macOS Planet).
		if a.subShowWebView && a.newWebView != nil && strings.TrimSpace(cur.item.Link) != "" {
			if a.subWebView == nil {
				if wv, err := a.newWebView(); err == nil && wv != nil {
					a.subWebView = wv
				} else {
					a.subShowWebView = false
				}
			}
			if a.subWebView != nil {
				targetURL := strings.TrimSpace(cur.item.Link)
				if a.subWebViewURL != targetURL {
					a.subWebViewURL = targetURL
					_ = a.subWebView.LoadURL(targetURL)
				}
				ui.Box(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
					if a.subWebView.Native() != nil {
						ui.WebView(c, a.subWebView.Native()).Fill()
					} else {
						ui.Column(c).Fill().Center().Children(func() {
							ui.Text(c, "原网页预览："+targetURL).Font(macPlanetFont).FontSize(13).TextColor(pal.muted)
						})
					}
				})
				return
			}
		}

		body := cur.item.Content
		if strings.TrimSpace(body) == "" {
			body = cur.item.Description
		}
		mdBody := htmlToReadableMarkdownWithBase(body, baseURL)

		ui.Scroll(c).TrackScroll(&a.subReaderScroll).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(740).Gap(12).Children(func() {
					ui.Text(c, fallbackTitle(cur.item.Title)).Font(macPlanetFont).FontSize(28).FontWeight(700).LetterSpacing(-0.02).TextColor(pal.ink)
					meta := cur.subTitle
					if cur.item.Published != "" {
						meta += "   ·   " + cur.item.Published
					}
					ui.Text(c, meta).Font(macPlanetFont).FontSize(12.5).TextColor(pal.muted)
					if cur.item.Link != "" {
						ui.Text(c, cur.item.Link).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
					}
					hairlineH(c, pal)
					if strings.TrimSpace(mdBody) == "" {
						display.Empty(c, display.EmptyProps{
							Icon:  lucide.FileText(),
							Title: "该订阅条目没有正文内容",
						})
					} else {
						lobemd.Markdown(c, mdBody, lobemd.MarkdownProps{
							FontSize:              15,
							LineHeight:            1.8,
							HeaderMultiple:        0.65,
							MarginMultiple:        1.2,
							FullFeaturedCodeBlock: true,
							LoadImage:             a.markdownImageLoader(baseURL, ""),
						})
					}
				})
			})
		})
	})
}

func (a *App) viewFollowModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.followModalOpen, macSheetProps{
		Title:       "关注订阅",
		Width:       440,
		PrimaryText: "关注",
		CancelText:  "取消",
	}, pal, func() {
		ui.Text(c, "支持 RSS / Atom 链接、Planet IPNS 或 ENS 域名").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		if ui.TextInput(c.Key("follow-addr"), &a.followAddress).Placeholder("https://example.com/feed.xml 或 vitalik.eth").Label("订阅地址").AutoFocus().AlignSelf(ui.Stretch).Submitted() {
			a.submitFollow()
		}
		if a.followErr != "" {
			ui.Text(c, a.followErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		a.submitFollow()
	}
}

func (a *App) submitFollow() {
	addr := strings.TrimSpace(a.followAddress)
	if addr == "" {
		a.followErr = "请输入订阅地址"
		return
	}
	sub, err := a.backend.Follow(addr)
	if err != nil {
		a.followErr = err.Error()
		return
	}
	a.followModalOpen = false
	a.followErr = ""
	a.Reload()
	a.openSubscriptionView(viewSubscription, sub.URL)
	a.setStatus("已关注：" + subscriptionTitle(sub))
}

func (a *App) viewEditSubscriptionModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.editSubOpen, macSheetProps{
		Title:       "编辑订阅",
		Width:       460,
		PrimaryText: "保存订阅",
		CancelText:  "取消",
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, "订阅地址", "", pal, func() {
				ui.TextInput(c.Key("edit-sub-url"), &a.editSubNewURL).Label("订阅地址").AlignSelf(ui.Stretch)
			})
			macField(c, "自定义名称", "", pal, func() {
				ui.TextInput(c.Key("edit-sub-title"), &a.editSubTitle).Placeholder("留空使用源标题").Label("自定义名称").AlignSelf(ui.Stretch)
			})
		})
		if a.editSubErr != "" {
			ui.Text(c, a.editSubErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		sub, err := a.backend.UpdateSubscription(a.editSubOldURL, strings.TrimSpace(a.editSubNewURL), strings.TrimSpace(a.editSubTitle))
		if err != nil {
			a.editSubErr = err.Error()
			return
		}
		a.editSubOpen = false
		a.selectedSubURL = sub.URL
		a.Reload()
		a.setStatus("已更新订阅：" + subscriptionTitle(sub))
	}
}

func (a *App) viewUnfollowDialog(c *ui.Context) {
	if macConfirmSheet(c, &a.unfollowOpen,
		"取消关注订阅？",
		fmt.Sprintf("确定要取消关注“%s”吗？", fallbackTitle(a.unfollowTitle)),
		"取消", "取消关注", true, a.pal) {
		if a.backend != nil && a.unfollowURL != "" {
			if err := a.backend.Unfollow(a.unfollowURL); err != nil {
				a.setError("取消关注失败：" + err.Error())
			} else {
				if a.selectedSubURL == a.unfollowURL {
					a.selectedSubURL = ""
					a.mode = viewOverview
				}
				a.Reload()
				a.setStatus("已取消关注")
			}
		}
		a.unfollowURL = ""
		a.unfollowTitle = ""
	} else if !a.unfollowOpen {
		a.unfollowURL = ""
		a.unfollowTitle = ""
	}
}

func (a *App) openGlobalSearch() {
	a.searchModalOpen = true
	a.searchErr = ""
}

func (a *App) runGlobalSearch() {
	q := strings.TrimSpace(a.searchQuery)
	if q == "" {
		a.searchResults = nil
		a.searchErr = ""
		return
	}
	res, err := a.backend.SearchArticles(q)
	if err != nil {
		a.searchErr = err.Error()
		a.searchResults = nil
		return
	}
	a.searchErr = ""
	a.searchResults = res
}

func (a *App) viewGlobalSearchModal(c *ui.Context) {
	pal := a.pal
	macSheet(c, &a.searchModalOpen, macSheetProps{
		Title:      "搜索文章",
		Width:      560,
		CancelText: "关闭",
	}, pal, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			if ui.TextInput(c.Key("global-search-input"), &a.searchQuery).Placeholder("搜索本站文章与订阅内容…").Label("搜索关键词").AutoFocus().Grow(1).Submitted() {
				a.runGlobalSearch()
			}
			if macPrimaryButtonSmall(c, "global-search-go", "搜索", pal) {
				a.runGlobalSearch()
			}
		})
		if a.searchErr != "" {
			ui.Text(c, a.searchErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(300).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(a.searchResults) == 0 {
					ui.Text(c, "输入关键词并点击搜索，可同时检索本地站点与已订阅内容。").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for i, r := range a.searchResults {
					res := r
					label := fmt.Sprintf("%s  ·  %s", fallbackTitle(res.Title), fallbackTitle(res.SourceName))
					if interactiveRow(c, fmt.Sprintf("search-res-%d", i), label, pal.subtle, pal.hover, pal.active).
						Radius(6).Padding(8, 10, 8, 10).
						Children(func() {
							ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, label).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink).MaxLines(1).Ellipsis("…")
								if strings.TrimSpace(res.Snippet) != "" {
									ui.Text(c, res.Snippet).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted).MaxLines(2).Ellipsis("…")
								}
							})
						}).Clicked() {
						a.searchModalOpen = false
						a.jumpToSearchResult(res)
					}
				}
			})
		})
	})
}

func (a *App) openRelatedArticles(siteID, articleID, subURL, itemKey, title string) {
	a.relatedTitle = title
	a.relatedErr = ""
	a.relatedResults = nil
	a.relatedModalOpen = true
	if a.backend == nil {
		return
	}
	res, err := a.backend.RelatedArticles(siteID, articleID, subURL, itemKey)
	if err != nil {
		a.relatedErr = err.Error()
		return
	}
	a.relatedResults = res
}

func (a *App) viewRelatedArticlesModal(c *ui.Context) {
	pal := a.pal
	macSheet(c, &a.relatedModalOpen, macSheetProps{
		Title:      "相关文章",
		Width:      500,
		CancelText: "关闭",
	}, pal, func() {
		if a.relatedTitle != "" {
			ui.Text(c, "与“"+a.relatedTitle+"”相关的内容：").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		}
		if a.relatedErr != "" {
			ui.Text(c, a.relatedErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(260).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(a.relatedResults) == 0 && a.relatedErr == "" {
					ui.Text(c, "暂无相关文章推荐").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for i, r := range a.relatedResults {
					res := r
					label := fmt.Sprintf("%s  ·  %s", fallbackTitle(res.Title), fallbackTitle(res.SourceName))
					if interactiveRow(c, fmt.Sprintf("related-res-%d", i), label, pal.subtle, pal.hover, pal.active).
						Radius(6).Padding(8, 10, 8, 10).
						Children(func() {
							ui.Text(c, label).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink).MaxLines(1).Ellipsis("…")
						}).Clicked() {
						a.relatedModalOpen = false
						a.jumpToSearchResult(res)
					}
				}
			})
		})
	})
}

func (a *App) jumpToSearchResult(res store.SearchResult) {
	if res.SiteID != "" && res.ArticleID != "" {
		a.guardUnsaved(func() { a.openArticleForRead(res.SiteID, res.ArticleID) })
		return
	}
	if res.SubscriptionURL != "" && res.ItemKey != "" {
		a.guardUnsaved(func() {
			a.mode = viewSubscription
			a.selectedSubURL = res.SubscriptionURL
			a.selectSubItem(res.SubscriptionURL, res.ItemKey)
		})
	}
}

// htmlToReadableMarkdown converts simple RSS/Atom HTML fragments into clean
// Markdown suitable for native ui/markdown.View rendering without a WebView.
func htmlToReadableMarkdown(raw string) string {
	return htmlToReadableMarkdownWithBase(raw, "")
}

func htmlToReadableMarkdownWithBase(raw, baseURL string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.Contains(trimmed, "<") {
		return trimmed
	}
	doc, err := html.Parse(strings.NewReader(trimmed))
	if err != nil {
		return trimmed
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			out.WriteString(n.Data)
		case html.ElementNode:
			tag := strings.ToLower(n.Data)
			switch tag {
			case "script", "style", "noscript":
				return
			case "br":
				out.WriteString("\n")
				return
			case "img":
				src := ""
				alt := ""
				for _, attr := range n.Attr {
					key := strings.ToLower(attr.Key)
					val := strings.TrimSpace(attr.Val)
					switch key {
					case "src":
						if val != "" && !strings.HasPrefix(strings.ToLower(val), "data:image/svg+xml") {
							src = val
						}
					case "data-src", "data-original", "data-lazy-src":
						if src == "" && val != "" {
							src = val
						}
					case "alt":
						alt = val
					}
				}
				if src != "" {
					resolved := resolveRelativeURL(src, baseURL)
					alt = strings.ReplaceAll(alt, "]", "\\]")
					out.WriteString("\n\n![" + alt + "](" + resolved + ")\n\n")
				}
				return
			case "figure":
				out.WriteString("\n")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n")
				return
			case "figcaption":
				var capText strings.Builder
				var collectCap func(*html.Node)
				collectCap = func(c *html.Node) {
					if c.Type == html.TextNode {
						capText.WriteString(c.Data)
					}
					for ch := c.FirstChild; ch != nil; ch = ch.NextSibling {
						collectCap(ch)
					}
				}
				collectCap(n)
				if s := strings.TrimSpace(capText.String()); s != "" {
					out.WriteString("\n*" + s + "*\n\n")
				}
				return
			case "h1", "h2", "h3", "h4":
				level := int(tag[1] - '0')
				out.WriteString("\n" + strings.Repeat("#", level) + " ")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n\n")
				return
			case "p", "div", "section", "article", "blockquote":
				if tag == "blockquote" {
					out.WriteString("\n> ")
				} else {
					out.WriteString("\n")
				}
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n\n")
				return
			case "li":
				out.WriteString("\n- ")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				return
			case "strong", "b":
				out.WriteString("**")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("**")
				return
			case "em", "i":
				out.WriteString("*")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("*")
				return
			case "code":
				out.WriteString("`")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("`")
				return
			case "a":
				href := ""
				for _, attr := range n.Attr {
					if strings.EqualFold(attr.Key, "href") {
						href = strings.TrimSpace(attr.Val)
						break
					}
				}
				var label strings.Builder
				var collect func(*html.Node)
				collect = func(c *html.Node) {
					if c.Type == html.TextNode {
						label.WriteString(c.Data)
					}
					for ch := c.FirstChild; ch != nil; ch = ch.NextSibling {
						collect(ch)
					}
				}
				collect(n)
				text := strings.TrimSpace(label.String())
				if href != "" && text != "" {
					out.WriteString("[" + text + "](" + href + ")")
				} else if text != "" {
					out.WriteString(text)
				}
				return
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	return strings.TrimSpace(out.String())
}
