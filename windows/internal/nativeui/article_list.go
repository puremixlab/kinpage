package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"windows/internal/store"
)

func (a *App) viewSiteWorkspace(c *ui.Context) {
	if a.postsCollapsed {
		a.viewDetailPane(c)
		return
	}
	ui.Split(c.Key("site-split"), &a.postsPanelWidth,
		func() { a.viewPostsPanel(c) },
		func() { a.viewDetailPane(c) },
	).SplitReset(276).Grow(1).MinHeight(0).AlignSelf(ui.Stretch)
}

func (a *App) matchesSiteFilter(art store.Article) bool {
	switch a.listFilter {
	case listFilterPages:
		return art.ArticleType == 1
	case listFilterNav:
		return art.IsIncludedInNavigation
	case listFilterPinned:
		return art.Pinned != nil
	case listFilterStarred:
		if !art.Starred {
			return false
		}
		if a.starTypeFilter >= 0 {
			return art.StarType == a.starTypeFilter
		}
		return true
	default:
		return true
	}
}

func (a *App) viewPostsPanel(c *ui.Context) {
	pal := a.pal
	site := a.currentSite()
	if site == nil {
		return
	}
	siteID := site.ID
	query := strings.ToLower(strings.TrimSpace(a.postSearchQuery))

	var filteredDrafts []store.Draft
	if a.listFilter == listFilterAll {
		for _, d := range a.drafts {
			title := fallbackTitle(d.Title)
			if query != "" && !strings.Contains(strings.ToLower(title), query) {
				continue
			}
			filteredDrafts = append(filteredDrafts, d)
		}
	}

	var filteredArticles []store.Article
	for _, art := range a.articles {
		if !a.matchesSiteFilter(art) {
			continue
		}
		title := fallbackTitle(art.Title)
		if query != "" && !strings.Contains(strings.ToLower(title), query) {
			continue
		}
		filteredArticles = append(filteredArticles, art)
	}

	totalVisible := len(filteredDrafts) + len(filteredArticles)
	subtitle := fmt.Sprintf("%d 篇文章", totalVisible)
	if a.listFilter != listFilterAll {
		subtitle += " · 已筛选"
	}

	ui.Column(c).Fill().MinWidth(240).Background(pal.canvas).Children(func() {
		// Full-height Column 2 integrated 44px header (macOS navigationTitle + navigationSubtitle + filter menu).
		a.viewListPaneHeader(c, fallbackTitle(site.Name), subtitle, false)

		// Inline list filter bar: hidden by default, toggled via Ctrl+F or filter menu.
		if a.listSearchOpen {
			ui.Row(c).Padding(8, 10, 4, 10).Gap(4).AlignItems(ui.Center).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					field.SearchBar(c.Key("posts-filter"), &a.postSearchQuery, field.SearchBarProps{
						Placeholder: "筛选本站文章…",
						Variant:     field.InputFilled,
						Size:        theme.Small,
					})
				})
				if toolbarIconBtn(c, "posts-filter-close", lucide.X(), "关闭筛选").Clicked() {
					a.listSearchOpen = false
					a.postSearchQuery = ""
				}
			})
		}

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(8).PaddingY(6).Gap(0).Children(func() {
				if len(filteredDrafts) > 0 {
					ui.Text(c, fmt.Sprintf("草稿 (%d)", len(filteredDrafts))).
						Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).Padding(2, 10, 4, 24)
					for _, d := range filteredDrafts {
						draft := d
						title := fallbackTitle(draft.Title)
						selected := a.editing && a.activeDraftID == draft.ID
						row := articleRow(c, articleRowProps{
							key:         "draft-" + draft.ID,
							ariaLabel:   "✎ " + title,
							title:       title,
							summary:     excerptText(draft.Content, 80),
							dateText:    formatRelativeDate(draft.Modified),
							selected:    selected,
							isDraft:     true,
							attachments: len(draft.Attachments),
						}, pal).ContextMenu(a.draftContextMenu(draft))
						if row.Clicked() {
							a.guardUnsaved(func() { a.openDraft(draft) })
						}
						if !selected {
							ui.Box(c).Padding(0, 12, 0, 24).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
						}
					}
					if len(filteredArticles) > 0 {
						ui.Text(c, "已发布").
							Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).Padding(10, 10, 4, 24)
					}
				}

				if totalVisible == 0 {
					ui.Box(c).PaddingY(32).AlignSelf(ui.Stretch).Children(func() {
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.FileText(),
							Title:       "暂无文章",
							Description: "点击上方编辑图标开始创作",
						})
					})
				}

				for _, art := range filteredArticles {
					item := art
					artID := item.ID
					title := fallbackTitle(item.Title)
					selected := !a.editing && a.selectedReadID == artID
					navWeight := 0
					if item.IsIncludedInNavigation {
						navWeight = item.NavigationWeight
						if navWeight <= 0 {
							navWeight = 1
						}
					}
					row := articleRow(c, articleRowProps{
						key:         "article-" + artID,
						ariaLabel:   title,
						title:       title,
						summary:     excerptText(item.Content, 80),
						dateText:    formatRelativeDate(item.Modified),
						selected:    selected,
						starred:     item.Starred,
						starType:    item.StarType,
						pinned:      item.Pinned != nil,
						isPage:      item.ArticleType == 1,
						navWeight:   navWeight,
						attachments: len(item.Attachments),
					}, pal).ContextMenu(a.articleContextMenu(item))
					if row.Clicked() {
						a.guardUnsaved(func() { a.openArticleForRead(siteID, artID) })
					}
					if !selected {
						ui.Box(c).Padding(0, 12, 0, 24).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
					}
				}
			})
		})
	})
}
