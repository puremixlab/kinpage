package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
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

func (a *App) viewPostsPanel(c *ui.Context) {
	pal := a.pal
	site := a.currentSite()
	if site == nil {
		return
	}
	siteID := site.ID
	query := strings.ToLower(strings.TrimSpace(a.postSearchQuery))

	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		// Full-height Column 2 integrated 44px header (macOS navigationTitle + navigationSubtitle).
		a.viewListPaneHeader(c, fallbackTitle(site.Name), fmt.Sprintf("%d 篇文章", len(a.articles)+len(a.drafts)))

		// Live filter SearchBar using lobe-mygo pkg/field.
		ui.Box(c).Padding(8, 10, 4, 10).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			field.SearchBar(c.Key("posts-filter"), &a.postSearchQuery, field.SearchBarProps{
				Placeholder: "筛选本站文章…",
				Variant:     field.InputFilled,
				Size:        theme.Small,
			})
		})

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(8).PaddingY(6).Gap(0).Children(func() {
				if len(a.drafts) > 0 {
					for _, d := range a.drafts {
						draft := d
						title := fallbackTitle(draft.Title)
						if query != "" && !strings.Contains(strings.ToLower(title), query) {
							continue
						}
						selected := a.editing && a.activeDraftID == draft.ID
						restBg, hoverBg := ui.Transparent, pal.hover
						titleColor := pal.ink
						excerptColor := pal.secondary
						metaColor := pal.muted
						if selected {
							restBg, hoverBg = pal.listSelected, pal.listSelected
							titleColor = ui.Hex("#ffffff")
							excerptColor = pal.listSelMuted
							metaColor = pal.listSelMuted
						}
						snippet := excerptText(draft.Content, 56)
						if interactiveRow(c, "draft-"+draft.ID, "✎ "+title, restBg, hoverBg, pal.active).
							Radius(7).Padding(10, 12, 10, 14).
							Children(func() {
								ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
									ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(titleColor).MaxLines(2).Ellipsis("…")
									if snippet != "" {
										ui.Text(c, snippet).Font(macPlanetFont).FontSize(12).TextColor(excerptColor).MaxLines(2).Ellipsis("…")
									}
									ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
										if selected {
											ui.Text(c, "草稿").Font(macPlanetFont).FontSize(11).TextColor(metaColor)
										} else {
											display.Tag(c, "草稿", display.TagProps{Color: display.TagInfo, Size: theme.Small})
										}
										ui.Text(c, formatSlashDate(draft.Modified)).Font(macPlanetFont).FontSize(11).TextColor(metaColor)
									})
								})
							}).Clicked() {
							a.guardUnsaved(func() { a.openDraft(draft) })
						}
						if !selected {
							ui.Box(c).Padding(0, 12, 0, 14).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
						}
					}
				}

				if len(a.articles) == 0 && len(a.drafts) == 0 {
					ui.Box(c).PaddingY(32).AlignSelf(ui.Stretch).Children(func() {
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.FileText(),
							Title:       "暂无文章",
							Description: "点击上方编辑图标开始创作",
						})
					})
				}
				for _, art := range a.articles {
					artID := art.ID
					title := fallbackTitle(art.Title)
					if query != "" && !strings.Contains(strings.ToLower(title), query) {
						continue
					}
					selected := !a.editing && a.selectedReadID == artID
					restBg, hoverBg := ui.Transparent, pal.hover
					titleColor := pal.ink
					excerptColor := pal.secondary
					metaColor := pal.muted
					if selected {
						restBg, hoverBg = pal.listSelected, pal.listSelected
						titleColor = ui.Hex("#ffffff")
						excerptColor = pal.listSelMuted
						metaColor = pal.listSelMuted
					}
					displayTitle := title
					if art.Pinned != nil {
						displayTitle = "📌 " + displayTitle
					}
					snippet := excerptText(art.Content, 56)
					if interactiveRow(c, "article-"+artID, displayTitle, restBg, hoverBg, pal.active).
						Radius(7).Padding(10, 12, 10, 14).
						Children(func() {
							ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, displayTitle).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(titleColor).MaxLines(2).Ellipsis("…")
								if snippet != "" {
									ui.Text(c, snippet).Font(macPlanetFont).FontSize(12).TextColor(excerptColor).MaxLines(2).Ellipsis("…")
								}
								ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, formatSlashDate(art.Modified)).Font(macPlanetFont).FontSize(11).TextColor(metaColor).NoWrap()
								})
							})
						}).Clicked() {
						a.guardUnsaved(func() { a.openArticleForRead(siteID, artID) })
					}
					if !selected {
						ui.Box(c).Padding(0, 12, 0, 14).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
					}
				}
			})
		})
	})
}
