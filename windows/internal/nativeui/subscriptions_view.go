package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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
			if a.mode == viewUnread && item.Read {
				continue
			}
			if a.mode == viewStarred && !item.Starred {
				continue
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
	ui.Split(c.Key("sub-split"), &a.postsPanelWidth,
		func() { a.viewSubscriptionItemsList(c) },
		func() { a.viewSubscriptionReader(c) },
	).SplitReset(280).Grow(1).MinHeight(0).AlignSelf(ui.Stretch)
}

func (a *App) viewSubscriptionItemsList(c *ui.Context) {
	t := c.Theme()
	heading := "订阅阅读"
	switch a.mode {
	case viewUnread:
		heading = fmt.Sprintf("未读 (%d)", a.totalUnreadCount())
	case viewStarred:
		heading = fmt.Sprintf("星标 (%d)", a.totalStarredCount())
	case viewSubscription:
		if sub := a.currentSubscription(); sub != nil {
			heading = subscriptionTitle(*sub)
		}
	}

	items := a.visibleSubscriptionItems()

	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Column(c).Padding(12).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, heading).Bold().FontSize(14)
			})
			if a.mode == viewSubscription {
				if sub := a.currentSubscription(); sub != nil {
					subURL := sub.URL
					ui.Row(c).Gap(6).Wrap().Children(func() {
						if ui.Button(c.Key("sub-refresh"), "刷新").Clicked() {
							if _, err := a.backend.RefreshSubscription(subURL); err != nil {
								a.setError("刷新订阅失败：" + err.Error())
							} else {
								a.Reload()
								a.setStatus("已刷新订阅")
							}
						}
						if ui.Button(c.Key("sub-mark-all"), "全部已读").Clicked() {
							if err := a.backend.MarkSubscriptionRead(subURL); err != nil {
								a.setError("标记已读失败：" + err.Error())
							} else {
								a.Reload()
								a.setStatus("已全部标记为已读")
							}
						}
						if ui.Button(c.Key("sub-edit"), "编辑").Clicked() {
							a.editSubOldURL = sub.URL
							a.editSubNewURL = sub.URL
							a.editSubTitle = sub.CustomTitle
							a.editSubErr = ""
							a.editSubOpen = true
						}
						if ui.Button(c.Key("sub-unfollow"), "取消关注").Clicked() {
							a.unfollowURL = sub.URL
							a.unfollowTitle = subscriptionTitle(*sub)
							a.unfollowOpen = true
						}
					})
				}
			}
			ui.SearchField(c.Key("sub-search"), &a.subSearchQuery).Placeholder("筛选订阅文章…").AlignSelf(ui.Stretch)
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(10).Gap(4).Children(func() {
				if len(items) == 0 {
					ui.Text(c, "暂无符合条件的订阅内容").FontSize(12).TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				}
				for _, entry := range items {
					subURL, key := entry.subURL, entry.effective
					prefix := ""
					if !entry.item.Read {
						prefix += "● "
					}
					if entry.item.Starred {
						prefix += "★ "
					}
					label := prefix + fallbackTitle(entry.item.Title)
					selected := a.selectedSubItemSubURL == subURL && a.selectedSubItemKey == key
					btn := ui.Button(c.Key("subitem-"+subURL+"-"+key), label).AlignSelf(ui.Stretch)
					if selected {
						btn.Background(t.Accent).TextColor(t.AccentText)
					}
					if btn.Clicked() {
						a.selectSubItem(subURL, key)
					}
				}
			})
		})
	})
}

func (a *App) viewSubscriptionReader(c *ui.Context) {
	t := c.Theme()
	cur, ok := a.currentSubItem()
	if !ok {
		ui.Column(c).Fill().Center().Gap(8).Children(func() {
			ui.Text(c, "从左侧选择一篇订阅文章开始阅读").FontSize(13).TextColor(t.TextMuted)
		})
		return
	}

	subURL, key := cur.subURL, cur.effective
	body := cur.item.Content
	if strings.TrimSpace(body) == "" {
		body = cur.item.Description
	}
	mdBody := htmlToReadableMarkdown(body)

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).PaddingX(16).PaddingY(10).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(t.Surface).Shrink(0).Children(func() {
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, fallbackTitle(cur.item.Title)).Bold().FontSize(15)
				meta := cur.subTitle
				if cur.item.Published != "" {
					meta += "  ·  " + cur.item.Published
				}
				ui.Text(c, meta).FontSize(11).TextColor(t.TextMuted)
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				starLabel := "☆ 星标"
				if cur.item.Starred {
					starLabel = "★ 取消星标"
				}
				if ui.Button(c.Key("sub-star"), starLabel).Clicked() {
					if err := a.backend.StarSubscription(subURL, key, !cur.item.Starred); err != nil {
						a.setError("更新星标失败：" + err.Error())
					} else {
						a.Reload()
					}
				}
				readLabel := "标为未读"
				if !cur.item.Read {
					readLabel = "标为已读"
				}
				if ui.Button(c.Key("sub-toggle-read"), readLabel).Clicked() {
					if err := a.backend.SetSubscriptionRead(subURL, key, !cur.item.Read); err != nil {
						a.setError("更新阅读状态失败：" + err.Error())
					} else {
						a.Reload()
					}
				}
				if ui.Button(c.Key("sub-related"), "相关文章").Clicked() {
					a.openRelatedArticles("", "", subURL, key, cur.item.Title)
				}
			})
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		ui.Scroll(c).TrackScroll(&a.subReaderScroll).Fill().Background(t.Background).Children(func() {
			ui.Column(c).Padding(24).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				if cur.item.Link != "" {
					ui.Text(c, "原文链接："+cur.item.Link).FontSize(11).TextColor(t.TextMuted)
				}
				if strings.TrimSpace(mdBody) == "" {
					ui.Text(c, "该订阅条目没有正文内容。").FontSize(13).TextColor(t.TextMuted)
				} else {
					markdown.View(c, mdBody, &a.markdownOpts)
				}
			})
		})
	})
}

func (a *App) viewFollowModal(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.followModalOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, "关注订阅").Bold().FontSize(16)
			ui.Text(c, "支持 RSS / Atom 链接、Planet IPNS 或 ENS 域名").FontSize(12).TextColor(t.TextMuted)
			if ui.TextInput(c.Key("follow-addr"), &a.followAddress).Placeholder("https://example.com/feed.xml 或 vitalik.eth").Label("订阅地址").AutoFocus().AlignSelf(ui.Stretch).Submitted() {
				a.submitFollow()
			}
			if a.followErr != "" {
				ui.Text(c, a.followErr).FontSize(12).TextColor(t.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("follow-cancel"), "取消").Clicked() {
					a.followModalOpen = false
				}
				if ui.PrimaryButton(c.Key("follow-submit"), "关注").Clicked() {
					a.submitFollow()
				}
			})
		})
	})
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
	t := c.Theme()
	ui.Modal(c, &a.editSubOpen, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, "编辑订阅").Bold().FontSize(16)
			ui.Form(c, func() {
				ui.Field(c, "订阅地址", func() {
					ui.TextInput(c.Key("edit-sub-url"), &a.editSubNewURL).Label("订阅地址").AlignSelf(ui.Stretch)
				})
				ui.Field(c, "自定义名称", func() {
					ui.TextInput(c.Key("edit-sub-title"), &a.editSubTitle).Placeholder("留空使用源标题").Label("自定义名称").AlignSelf(ui.Stretch)
				})
			})
			if a.editSubErr != "" {
				ui.Text(c, a.editSubErr).FontSize(12).TextColor(t.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("edit-sub-cancel"), "取消").Clicked() {
					a.editSubOpen = false
				}
				if ui.PrimaryButton(c.Key("edit-sub-save"), "保存订阅").Clicked() {
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
			})
		})
	})
}

func (a *App) viewUnfollowDialog(c *ui.Context) {
	switch ui.AlertDialog(c, &a.unfollowOpen,
		"取消关注订阅？",
		fmt.Sprintf("确定要取消关注“%s”吗？", fallbackTitle(a.unfollowTitle)),
		"取消", "取消关注") {
	case 1:
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
	case 0, ui.AlertDismissed:
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
	t := c.Theme()
	ui.Modal(c, &a.searchModalOpen, func() {
		ui.Column(c).Width(540).Gap(12).Children(func() {
			ui.Text(c, "全局混合搜索").Bold().FontSize(16)
			ui.Row(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
				if ui.TextInput(c.Key("global-search-input"), &a.searchQuery).Placeholder("搜索本站文章与订阅内容…").Label("搜索关键词").AutoFocus().Grow(1).Submitted() {
					a.runGlobalSearch()
				}
				if ui.PrimaryButton(c.Key("global-search-go"), "搜索").Clicked() {
					a.runGlobalSearch()
				}
			})
			if a.searchErr != "" {
				ui.Text(c, a.searchErr).FontSize(12).TextColor(t.Danger)
			}
			ui.Scroll(c).Height(300).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					if len(a.searchResults) == 0 {
						ui.Text(c, "输入关键词并点击搜索，可同时检索本地站点与已订阅内容。").FontSize(12).TextColor(t.TextMuted)
					}
					for i, r := range a.searchResults {
						res := r
						label := fmt.Sprintf("%s  ·  %s", fallbackTitle(res.Title), fallbackTitle(res.SourceName))
						if ui.Button(c.Key(fmt.Sprintf("search-res-%d", i)), label).AlignSelf(ui.Stretch).Clicked() {
							a.searchModalOpen = false
							a.jumpToSearchResult(res)
						}
						if strings.TrimSpace(res.Snippet) != "" {
							ui.Text(c, res.Snippet).FontSize(11).TextColor(t.TextMuted).PaddingX(6)
						}
					}
				})
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("global-search-close"), "关闭").Clicked() {
					a.searchModalOpen = false
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
	t := c.Theme()
	ui.Modal(c, &a.relatedModalOpen, func() {
		ui.Column(c).Width(480).Gap(12).Children(func() {
			ui.Text(c, "相关文章").Bold().FontSize(16)
			if a.relatedTitle != "" {
				ui.Text(c, "与“"+a.relatedTitle+"”相关的内容：").FontSize(12).TextColor(t.TextMuted)
			}
			if a.relatedErr != "" {
				ui.Text(c, a.relatedErr).FontSize(12).TextColor(t.Danger)
			}
			ui.Scroll(c).Height(260).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					if len(a.relatedResults) == 0 && a.relatedErr == "" {
						ui.Text(c, "暂无相关文章推荐").FontSize(12).TextColor(t.TextMuted)
					}
					for i, r := range a.relatedResults {
						res := r
						label := fmt.Sprintf("%s  ·  %s", fallbackTitle(res.Title), fallbackTitle(res.SourceName))
						if ui.Button(c.Key(fmt.Sprintf("related-res-%d", i)), label).AlignSelf(ui.Stretch).Clicked() {
							a.relatedModalOpen = false
							a.jumpToSearchResult(res)
						}
					}
				})
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("related-close"), "关闭").Clicked() {
					a.relatedModalOpen = false
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
