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
	heading := a.t("list.subDefaultHeading")
	switch a.mode {
	case viewToday:
		heading = a.t("nav.today")
	case viewUnread:
		heading = a.t("nav.unread")
	case viewStarred:
		heading = a.t("nav.starred")
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
	subtitle := a.tf("list.articleCount", len(items))
	if unreadCount > 0 {
		subtitle = a.tf("list.unreadTotalCount", unreadCount, len(items))
	}
	if a.listFilter != listFilterAll {
		subtitle += a.t("filter.suffix")
	}

	ui.Column(c).Fill().MinWidth(240).Background(pal.canvas).Children(func() {
		// Full-height Column 2 integrated 44px header (macOS navigationTitle + navigationSubtitle + filter menu).
		a.viewListPaneHeader(c, heading, subtitle, true)

		if a.listSearchOpen {
			ui.Row(c).Padding(8, 10, 4, 10).Gap(4).AlignItems(ui.Center).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					field.SearchBar(c.Key("sub-filter"), &a.subSearchQuery, field.SearchBarProps{
						Placeholder: a.t("list.searchSubPlaceholder"),
						Variant:     field.InputFilled,
						Size:        theme.Small,
					})
				})
				if toolbarIconBtn(c, "sub-filter-close", lucide.X(), a.t("list.closeFilter")).Clicked() {
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
							Title:       a.t("list.emptySubTitle"),
							Description: a.t("list.emptySubDesc"),
						})
					})
				}
				for _, entry := range items {
					subURL, key := entry.subURL, entry.effective
					title := a.fallbackTitle(entry.item.Title)
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
						dateText:    a.formatFeedDate(entry.item.Published),
						sourceLabel: sourceLabel,
						selected:    selected,
						starred:     entry.item.Starred,
						starType:    entry.item.StarType,
						unread:      !entry.item.Read,
						locale:      a.Locale(),
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
					Title:       a.t("reader.subEmptyTitle"),
					Description: a.t("reader.subEmptyDesc"),
				})
			})
			return
		}

		baseURL := a.subscriptionBaseURL(*cur)

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
							ui.Text(c, a.tf("reader.webPreviewPrefix", targetURL)).Font(macPlanetFont).FontSize(13).TextColor(pal.muted)
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
		mdBody := buildSubscriptionMarkdown(body, cur.item, baseURL)

		colW := a.detailColumnWidth(c, 740)
		ui.Scroll(c).TrackScroll(&a.subReaderScroll).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
				ui.Column(c).Width(colW).Gap(12).Children(func() {
					ui.Text(c, a.fallbackTitle(cur.item.Title)).Font(macPlanetFont).FontSize(28).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.ink)
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
							Title: a.t("reader.subNoContent"),
						})
					} else {
						lobemd.Markdown(c.Key("sub-md-"+cur.effective), mdBody, lobemd.MarkdownProps{
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
		Title:        a.t("dialog.follow.title"),
		Width:        440,
		PrimaryText:  a.t("dialog.follow.primary"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Text(c, a.t("dialog.follow.hint")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		if ui.TextInput(c.Key("follow-addr"), &a.followAddress).Placeholder(a.t("dialog.follow.placeholder")).Label(a.t("dialog.follow.label")).AutoFocus().AlignSelf(ui.Stretch).Submitted() {
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
		a.followErr = a.t("dialog.follow.addrRequired")
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
	a.setStatus(a.tf("status.followed", subscriptionTitle(sub)))
}

func (a *App) viewEditSubscriptionModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.editSubOpen, macSheetProps{
		Title:        a.t("dialog.editSub.title"),
		Width:        460,
		PrimaryText:  a.t("dialog.editSub.primary"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("dialog.editSub.addrField"), "", pal, func() {
				ui.TextInput(c.Key("edit-sub-url"), &a.editSubNewURL).Label(a.t("dialog.editSub.addrField")).AlignSelf(ui.Stretch)
			})
			macField(c, a.t("dialog.editSub.customTitleField"), "", pal, func() {
				ui.TextInput(c.Key("edit-sub-title"), &a.editSubTitle).Placeholder(a.t("dialog.editSub.customTitlePlaceholder")).Label(a.t("dialog.editSub.customTitleField")).AlignSelf(ui.Stretch)
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
		a.setStatus(a.tf("status.subUpdated", subscriptionTitle(sub)))
	}
}

func (a *App) viewUnfollowDialog(c *ui.Context) {
	if macConfirmSheet(c, &a.unfollowOpen,
		a.t("dialog.unfollow.title"),
		a.tf("dialog.unfollow.message", a.fallbackTitle(a.unfollowTitle)),
		a.t("common.cancel"), a.t("dialog.unfollow.confirm"), true, a.pal) {
		if a.backend != nil && a.unfollowURL != "" {
			if err := a.backend.Unfollow(a.unfollowURL); err != nil {
				a.setError(a.tf("status.unfollowErr", err.Error()))
			} else {
				if a.selectedSubURL == a.unfollowURL {
					a.selectedSubURL = ""
					a.mode = viewOverview
				}
				a.Reload()
				a.setStatus(a.t("status.unfollowed"))
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
		Title:        a.t("dialog.search.title"),
		Width:        560,
		CancelText:   a.t("common.close"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			if ui.TextInput(c.Key("global-search-input"), &a.searchQuery).Placeholder(a.t("dialog.search.placeholder")).Label(a.t("dialog.search.label")).AutoFocus().Grow(1).Submitted() {
				a.runGlobalSearch()
			}
			if macPrimaryButtonSmall(c, "global-search-go", a.t("common.search"), pal) {
				a.runGlobalSearch()
			}
		})
		if a.searchErr != "" {
			ui.Text(c, a.searchErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(300).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(a.searchResults) == 0 {
					ui.Text(c, a.t("dialog.search.hint")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for i, r := range a.searchResults {
					res := r
					label := fmt.Sprintf("%s  ·  %s", a.fallbackTitle(res.Title), a.fallbackTitle(res.SourceName))
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
		Title:        a.t("dialog.related.title"),
		Width:        500,
		CancelText:   a.t("common.close"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		if a.relatedTitle != "" {
			ui.Text(c, a.tf("dialog.related.prompt", a.relatedTitle)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		}
		if a.relatedErr != "" {
			ui.Text(c, a.relatedErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(260).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Children(func() {
				if len(a.relatedResults) == 0 && a.relatedErr == "" {
					ui.Text(c, a.t("dialog.related.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for i, r := range a.relatedResults {
					res := r
					label := fmt.Sprintf("%s  ·  %s", a.fallbackTitle(res.Title), a.fallbackTitle(res.SourceName))
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

func formatMarkdownImage(alt, rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}
	cleanAlt := strings.Join(strings.Fields(alt), " ")
	cleanAlt = strings.ReplaceAll(cleanAlt, "]", "\\]")
	if strings.ContainsAny(u, " \t\r\n()") {
		return "![" + cleanAlt + "](<" + u + ">)"
	}
	return "![" + cleanAlt + "](" + u + ")"
}

func firstSrcsetURL(srcset string) string {
	for _, entry := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(entry))
		if len(fields) > 0 && fields[0] != "" {
			return fields[0]
		}
	}
	return ""
}

func nodeContainsImage(n *html.Node) bool {
	if n == nil {
		return false
	}
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)
		if tag == "img" || tag == "picture" {
			return true
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if nodeContainsImage(ch) {
			return true
		}
	}
	return false
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
	extractImgURL := func(n *html.Node, pictureFallback string) (string, string) {
		src := ""
		lazySrc := ""
		srcset := ""
		alt := ""
		for _, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			val := strings.TrimSpace(attr.Val)
			if val == "" {
				continue
			}
			switch key {
			case "src":
				if !strings.HasPrefix(strings.ToLower(val), "data:image/svg+xml") {
					src = val
				}
			case "data-src", "data-original", "data-lazy-src", "data-actualsrc", "data-url":
				if lazySrc == "" {
					lazySrc = val
				}
			case "srcset", "data-srcset":
				if srcset == "" {
					srcset = val
				}
			case "alt":
				alt = val
			}
		}
		// Prefer lazy-load attributes when src is missing or a tiny data: placeholder.
		chosen := src
		if lazySrc != "" && (chosen == "" || strings.HasPrefix(strings.ToLower(chosen), "data:")) {
			chosen = lazySrc
		}
		if chosen == "" && srcset != "" {
			chosen = firstSrcsetURL(srcset)
		}
		if chosen == "" && pictureFallback != "" {
			chosen = pictureFallback
		}
		return chosen, alt
	}
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
			case "picture":
				sourceFallback := ""
				var imgNode *html.Node
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					if ch.Type != html.ElementNode {
						continue
					}
					chTag := strings.ToLower(ch.Data)
					if chTag == "source" && sourceFallback == "" {
						for _, attr := range ch.Attr {
							k := strings.ToLower(attr.Key)
							if k == "srcset" || k == "data-srcset" || k == "src" {
								if u := firstSrcsetURL(attr.Val); u != "" {
									sourceFallback = u
									break
								}
							}
						}
					} else if chTag == "img" && imgNode == nil {
						imgNode = ch
					}
				}
				if imgNode != nil {
					chosen, alt := extractImgURL(imgNode, sourceFallback)
					if chosen != "" {
						resolved := resolveRelativeURL(chosen, baseURL)
						out.WriteString("\n\n" + formatMarkdownImage(alt, resolved) + "\n\n")
					}
				} else if sourceFallback != "" {
					resolved := resolveRelativeURL(sourceFallback, baseURL)
					out.WriteString("\n\n" + formatMarkdownImage("", resolved) + "\n\n")
				}
				return
			case "img":
				chosen, alt := extractImgURL(n, "")
				if chosen != "" {
					resolved := resolveRelativeURL(chosen, baseURL)
					out.WriteString("\n\n" + formatMarkdownImage(alt, resolved) + "\n\n")
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
				// RSS/Atom feeds frequently wrap images inside <a href="..."><img src="..."></a>.
				// Preserve nested images by walking children normally instead of dropping non-text nodes.
				if nodeContainsImage(n) {
					for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
						walk(ch)
					}
					return
				}
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

// subscriptionBaseURL computes the best HTTP(S) base URL for resolving relative
// image/link references in a subscription item, supporting both normal web feeds
// and Planet/IPFS/ENS subscriptions (via local Kubo gateway or public gateway).
func (a *App) subscriptionBaseURL(cur subItemEntry) string {
	if cid := strings.TrimSpace(cur.item.ContentCID); cid != "" {
		gw := "https://eth.sucks"
		if a.ipfsModal.status.Running && strings.TrimSpace(a.ipfsModal.status.Gateway) != "" {
			gw = strings.TrimRight(strings.TrimSpace(a.ipfsModal.status.Gateway), "/")
		}
		parts := []string{gw, "ipfs", cid}
		if p := strings.Trim(strings.TrimSpace(cur.item.ContentPath), "/"); p != "" {
			parts = append(parts, p)
		}
		if id := strings.Trim(strings.TrimSpace(cur.item.ArticleID), "/"); id != "" {
			parts = append(parts, id)
		}
		return strings.Join(parts, "/") + "/"
	}
	if link := strings.TrimSpace(cur.item.Link); strings.HasPrefix(strings.ToLower(link), "http://") || strings.HasPrefix(strings.ToLower(link), "https://") {
		return link
	}
	if sub := strings.TrimSpace(cur.subURL); strings.HasPrefix(strings.ToLower(sub), "http://") || strings.HasPrefix(strings.ToLower(sub), "https://") {
		return sub
	}
	return strings.TrimSpace(cur.item.Link)
}

func isSubImageFilename(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// promoteMarkdownImages moves linked images ([![alt](img)](href)) and inline
// images mixed with text onto standalone paragraphs because mygo's Markdown
// renderer only loads and paints images in image-only paragraphs.
func promoteMarkdownImages(md string) string {
	if !strings.Contains(md, "![") {
		return md
	}
	lines := strings.Split(md, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if inFence || !strings.Contains(line, "![") {
			out = append(out, line)
			continue
		}
		out = append(out, promoteLineMarkdownImages(line))
	}
	return strings.Join(out, "\n")
}

func promoteLineMarkdownImages(line string) string {
	var b strings.Builder
	i := 0
	inCode := false
	for i < len(line) {
		if line[i] == '`' {
			inCode = !inCode
			b.WriteByte(line[i])
			i++
			continue
		}
		if inCode {
			b.WriteByte(line[i])
			i++
			continue
		}
		// Match linked image: [![alt](dest)](href)
		if line[i] == '[' && i+2 < len(line) && line[i+1] == '!' && line[i+2] == '[' {
			if imgEnd, ok := matchMarkdownImageEnd(line, i+1); ok && imgEnd < len(line) && line[imgEnd] == ']' {
				if imgEnd+1 < len(line) && line[imgEnd+1] == '(' {
					if linkEnd, ok2 := matchParenEnd(line, imgEnd+1); ok2 {
						imgExpr := line[i+1 : imgEnd]
						b.WriteString("\n\n" + imgExpr + "\n\n")
						i = linkEnd
						continue
					}
				}
			}
		}
		// Match inline image: ![alt](dest)
		if line[i] == '!' && i+1 < len(line) && line[i+1] == '[' {
			if imgEnd, ok := matchMarkdownImageEnd(line, i); ok {
				imgExpr := line[i:imgEnd]
				// Only add paragraph breaks if there is non-whitespace text on the same line.
				if strings.TrimSpace(line) != imgExpr {
					b.WriteString("\n\n" + imgExpr + "\n\n")
				} else {
					b.WriteString(imgExpr)
				}
				i = imgEnd
				continue
			}
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

func matchMarkdownImageEnd(s string, start int) (int, bool) {
	if start+1 >= len(s) || s[start] != '!' || s[start+1] != '[' {
		return 0, false
	}
	j := start + 2
	for j < len(s) {
		if s[j] == '\\' && j+1 < len(s) {
			j += 2
			continue
		}
		if s[j] == ']' {
			break
		}
		j++
	}
	if j >= len(s) || s[j] != ']' || j+1 >= len(s) || s[j+1] != '(' {
		return 0, false
	}
	return matchParenEnd(s, j+1)
}

func matchParenEnd(s string, openPos int) (int, bool) {
	if openPos >= len(s) || s[openPos] != '(' {
		return 0, false
	}
	j := openPos + 1
	if j < len(s) && s[j] == '<' {
		for j < len(s) && s[j] != '>' {
			j++
		}
		if j >= len(s) {
			return 0, false
		}
		j++ // past '>'
		for j < len(s) && s[j] != ')' {
			j++
		}
		if j < len(s) && s[j] == ')' {
			return j + 1, true
		}
		return 0, false
	}
	depth := 1
	for j < len(s) {
		if s[j] == '\\' && j+1 < len(s) {
			j += 2
			continue
		}
		if s[j] == '(' {
			depth++
		} else if s[j] == ')' {
			depth--
			if depth == 0 {
				return j + 1, true
			}
		}
		j++
	}
	return 0, false
}

// buildSubscriptionMarkdown converts HTML (if present) to readable Markdown and
// appends unreferenced Planet image attachments or RSS media image enclosures.
func buildSubscriptionMarkdown(body string, item feeds.Item, baseURL string) string {
	md := promoteMarkdownImages(htmlToReadableMarkdownWithBase(body, baseURL))
	var extras []string
	for _, name := range item.Attachments {
		name = strings.TrimSpace(name)
		if name == "" || !isSubImageFilename(name) {
			continue
		}
		resolved := resolveRelativeURL(name, baseURL)
		if strings.Contains(md, name) || strings.Contains(md, resolved) {
			continue
		}
		extras = append(extras, formatMarkdownImage(name, resolved))
	}
	for _, m := range item.Media {
		u := strings.TrimSpace(m.URL)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(m.Type)), "image/") && !isSubImageFilename(u) {
			continue
		}
		resolved := resolveRelativeURL(u, baseURL)
		if strings.Contains(md, u) || strings.Contains(md, resolved) {
			continue
		}
		extras = append(extras, formatMarkdownImage("", resolved))
	}
	if len(extras) > 0 {
		if strings.TrimSpace(md) != "" {
			md += "\n\n"
		}
		md += strings.Join(extras, "\n\n")
	}
	return md
}
