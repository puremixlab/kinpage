package nativeui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/store"
)

const (
	activityModeDaily      = 0
	activityModeWeekly     = 1
	activityModeCumulative = 2
)

var featuredPlanetAddresses = []string{
	"planetable.eth",
	"vitalik.eth",
}

func (a *App) followFeaturedPlanets() {
	if a.backend == nil {
		return
	}
	followed := 0
	var lastErr error
	for _, addr := range featuredPlanetAddresses {
		if _, err := a.backend.Follow(addr); err != nil {
			lastErr = err
		} else {
			followed++
		}
	}
	a.Reload()
	if followed > 0 {
		a.setStatus(fmt.Sprintf("已关注 %d 个精选站点", followed))
	} else if lastErr != nil {
		a.setError("关注精选站点失败：" + lastErr.Error())
	}
}

func (a *App) handleDashboardCompose() {
	switch len(a.sites) {
	case 0:
		a.createSiteOpen = true
	case 1:
		a.startNewDraft(a.sites[0].ID)
	default:
		a.composePickerOpen = true
	}
}

func (a *App) writingStatsSummary() (last30 int, streak int) {
	counts := make(map[string]int, len(a.activity))
	for _, item := range a.activity {
		counts[item.Date] = item.Articles + item.Drafts
	}
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	for i := range 30 {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		last30 += counts[key]
	}
	for i := 0; i < 365; i++ {
		key := today.AddDate(0, 0, -i).Format("2006-01-02")
		if counts[key] > 0 {
			streak++
		} else if i > 0 {
			break
		}
	}
	return last30, streak
}

// paperDeskPalette returns tactile desk + paper-sheet colors tailored for
// the monochrome Linear workbench + physical paper card aesthetic.
func paperDeskColors(pal macPalette) (deskBg, paperBg, paperHover, paperEdge, paperShadow ui.Color) {
	if pal.dark {
		return ui.Hex("#161618"), ui.Hex("#202124"), ui.Hex("#26282c"),
			ui.RGBA(255, 255, 255, 0.085), ui.RGBA(0, 0, 0, 0.34)
	}
	return ui.Hex("#f5f5f4"), ui.Hex("#ffffff"), ui.Hex("#fcfcfb"),
		ui.RGBA(15, 23, 42, 0.09), ui.RGBA(15, 23, 42, 0.055)
}

func (a *App) viewDashboard(c *ui.Context) {
	pal := a.pal
	deskBg, paperBg, _, paperEdge, paperShadow := paperDeskColors(pal)

	ui.Scroll(c).TrackScroll(&a.dashboardScroll).Grow(1).MinHeight(0).Background(deskBg).Children(func() {
		ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(40).PaddingY(24).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(840).Gap(20).Children(func() {
				// Title-less compact utility row: understated stats on left, monochrome actions on right.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, fmt.Sprintf("%d 站点  ·  %d 篇文章  ·  %d 篇草稿",
						a.dashboard.SiteCount, a.dashboard.ArticleCount, a.dashboard.DraftCount)).
						Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
					ui.Row(c).AlignItems(ui.Center).Gap(8).Shrink(0).Children(func() {
						if len(a.sites) > 0 {
							if macSecondaryButton(c, "dash-hero-quick", "快捷发文", pal) {
								a.openQuickPost()
							}
						}
						btn := ui.ButtonBase(c.Key("dash-hero-compose")).Label("写文章").
							Height(28).PaddingX(14).Radius(6).Background(pal.ink).
							Center().Cursor(ui.CursorPointer).Transition(rowHoverTransition)
						if btn.Hovered() {
							btn.Background(pal.ink.Alpha(0.86))
						}
						btn.Children(func() {
							ui.Text(c, "写文章").Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.canvas).NoWrap()
						})
						if btn.Clicked() {
							a.handleDashboardCompose()
						}
					})
				})

				if a.dashboard.SiteCount == 0 {
					ui.Column(c).AlignSelf(ui.Stretch).Padding(40).Radius(8).
						Background(paperBg).Border(0.5, paperEdge).Shadow(0, 2, 6, 0, paperShadow).Children(func() {
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.Globe(),
							Title:       "还没有创建站点",
							Description: "创建一个站点后即可开始本地写作、管理草稿与发布到 IPFS。",
							Action: func() {
								ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
									if macPrimaryButton(c, "dash-create-first", "创建第一个站点", pal) {
										a.createSiteOpen = true
									}
									if len(a.subscriptions) == 0 {
										if macSecondaryButton(c, "dash-follow-featured", "关注精选站点", pal) {
											a.followFeaturedPlanets()
										}
									}
								})
							},
						})
					})
					return
				}

				// 1. Sleek full-width monochrome activity strip.
				a.viewLinearActivityStrip(c)

				// 2. Drafts rendered as tactile memo paper sheets.
				a.viewDashboardDrafts(c)

				// 3. Site bookmark shelf (unified filter + status dots) + tactile paper-sheet article cards.
				a.viewDashboardArticles(c)

				// Minimal quiet footer.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).PaddingY(2).Children(func() {
					ui.Text(c, "所有草稿与站点均保存在本地").Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
					if ghostIconButton(c, "dash-refresh", "刷新数据", "刷新数据", pal) {
						a.Reload()
						a.setStatus("已刷新数据")
					}
				})
			})
		})
	})
}

func (a *App) viewLinearActivityStrip(c *ui.Context) {
	pal := a.pal
	_, paperBg, _, paperEdge, paperShadow := paperDeskColors(pal)
	_, streak := a.writingStatsSummary()
	counts := make(map[string]int, len(a.activity))
	totalActions := 0
	for _, item := range a.activity {
		n := item.Articles + item.Drafts
		counts[item.Date] = n
		totalActions += n
	}

	// Faithful Option B sleek horizontal activity strip: left summary + compact
	// mode switch, right-aligned 36-week monochrome heatmap in one compact row.
	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(16).
		Padding(14, 18, 14, 18).Background(paperBg).Radius(8).
		Border(0.5, paperEdge).Shadow(0, 1, 4, 0, paperShadow).Children(func() {
		ui.Column(c).Gap(6).Shrink(0).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, fmt.Sprintf("近半年写作 %d 次", totalActions)).Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.ink).NoWrap()
				streakText := "今天也可以写点什么"
				if streak > 0 {
					streakText = fmt.Sprintf("连续 %d 天保持记录", streak)
				}
				ui.Text(c, streakText).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted).NoWrap()
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				modes := []struct {
					id    int
					label string
				}{
					{activityModeDaily, "按日"},
					{activityModeWeekly, "按周"},
					{activityModeCumulative, "累计"},
				}
				for _, m := range modes {
					active := a.activityMode == m.id
					col := pal.muted
					if active {
						col = pal.ink
					}
					btn := ui.ButtonBase(c.Key(fmt.Sprintf("act-mode-%d", m.id))).Label(m.label).
						Padding(1, 0, 1, 0).Cursor(ui.CursorPointer)
					btn.Children(func() {
						t := ui.Text(c, m.label).Font(macPlanetFont).FontSize(11).TextColor(col).NoWrap()
						if active {
							t.FontWeight(600)
						}
					})
					if btn.Clicked() {
						a.activityMode = m.id
					}
				}
			})
		})
		a.viewHeatmapGrid(c, counts)
	})
}

func (a *App) viewDashboardDrafts(c *ui.Context) {
	if len(a.dashboard.RecentDrafts) == 0 {
		return
	}
	pal := a.pal
	_, paperBg, paperHover, paperEdge, paperShadow := paperDeskColors(pal)

	ui.Column(c).AlignSelf(ui.Stretch).Gap(8).Children(func() {
		ui.Text(c, fmt.Sprintf("草稿 · %d", len(a.dashboard.RecentDrafts))).
			Font(macPlanetFont).FontSize(11.5).FontWeight(600).TextColor(pal.muted)
		for _, d := range a.dashboard.RecentDrafts {
			siteID, draftID := d.SiteID, d.ID
			title := fallbackTitle(d.Title)
			siteName := fallbackTitle(d.SiteName)
			label := fmt.Sprintf("%s  ·  %s", title, siteName)

			card := interactiveRow(c, "dash-draft-"+draftID, label, paperBg, paperHover, pal.active).
				Radius(6).Padding(0, 0, 0, 0).Border(0.5, paperEdge).Shadow(0, 2, 5, 0, paperShadow)
			card.Children(func() {
				ui.Row(c).AlignItems(ui.Stretch).Grow(1).MinWidth(0).Children(func() {
					// Subtle left binding spine like a physical notepad sheet.
					ui.Box(c).Width(3).Background(pal.ink.Alpha(0.28)).Shrink(0)
					ui.Column(c).Padding(12, 16, 12, 14).Gap(5).Grow(1).MinWidth(0).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(7).Grow(1).MinWidth(0).Children(func() {
								ui.Icon(c, lucide.PenLine()).Size(13, 13).TextColor(pal.secondary).Shrink(0)
								ui.Text(c, title).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).MaxLines(1).Ellipsis("…").Grow(1)
							})
							ui.Text(c, formatShortDate(d.Modified)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted).Shrink(0).NoWrap()
						})
						ui.Text(c, siteName+" · 本地草稿").Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted).NoWrap()
					})
				})
			})
			if card.Clicked() {
				a.openDraftByID(siteID, draftID)
			}
		}
	})
}

func (a *App) viewDashboardArticles(c *ui.Context) {
	pal := a.pal
	_, paperBg, paperHover, paperEdge, paperShadow := paperDeskColors(pal)
	if a.articleFilterSiteID != "" {
		found := false
		for _, s := range a.dashboard.Sites {
			if s.ID == a.articleFilterSiteID {
				found = true
				break
			}
		}
		if !found {
			a.articleFilterSiteID = ""
		}
	}

	ui.Column(c).AlignSelf(ui.Stretch).Gap(10).Children(func() {
		// Unified Site Bookmark Shelf: displays sites with avatar & publish-status dot
		// while acting as the article filter bar (eliminating redundant site blocks).
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
				a.siteBookmarkChip(c, "art-filter-all", "所有站点", nil, -1, ui.Transparent, a.articleFilterSiteID == "", func() {
					a.articleFilterSiteID = ""
				})
				for i, s := range a.dashboard.Sites {
					idx, sid := i, s.ID
					dotColor := pal.muted
					if s.PublishPending {
						dotColor = pal.amber
					} else if s.HasPublished {
						dotColor = pal.green
					}
					var fullSite store.Site
					fullSite.Name = s.Name
					for _, candidate := range a.sites {
						if candidate.ID == sid {
							fullSite = candidate
							break
						}
					}
					fs := fullSite
					a.siteBookmarkChip(c, "art-filter-"+sid, fallbackTitle(s.Name), &fs, idx, dotColor, a.articleFilterSiteID == sid, func() {
						if a.articleFilterSiteID == sid {
							a.selectSite(sid)
						} else {
							a.articleFilterSiteID = sid
						}
					})
				}
			})
			if macIconButton(c, "dash-sites-add", iconPlus, 26, 13, pal.muted, "新建站点") {
				a.newSiteName = ""
				a.newSiteAbout = ""
				a.createSiteError = ""
				a.createSiteOpen = true
			}
		})

		// Tactile paper-sheet article cards.
		shown := 0
		for _, art := range a.dashboard.RecentArticles {
			if a.articleFilterSiteID != "" && art.SiteID != a.articleFilterSiteID {
				continue
			}
			shown++
			siteID, artID := art.SiteID, art.ID
			title := fallbackTitle(art.Title)
			siteName := fallbackTitle(art.SiteName)
			label := fmt.Sprintf("%s  ·  %s", title, siteName)

			card := interactiveRow(c, "dash-art-"+artID, label, paperBg, paperHover, pal.active).
				Padding(13, 16, 13, 16).Radius(6).
				Border(0.5, paperEdge).Shadow(0, 2, 6, 0, paperShadow)
			card.Children(func() {
				ui.Column(c).Gap(6).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, title).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).MaxLines(1).Ellipsis("…").Grow(1)
						ui.Text(c, formatShortDate(art.Modified)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted).Shrink(0).NoWrap()
					})
					ui.Row(c).AlignItems(ui.Center).Gap(7).AlignSelf(ui.Stretch).Children(func() {
						ui.Icon(c, lucide.FileText()).Size(12, 12).TextColor(pal.muted).Shrink(0)
						ui.Text(c, siteName).Font(macPlanetFont).FontSize(11.5).TextColor(pal.secondary).NoWrap()
					})
				})
			})
			if card.Clicked() {
				a.openArticleForRead(siteID, artID)
			}
		}
		if shown == 0 {
			ui.Column(c).AlignSelf(ui.Stretch).Padding(20).Radius(6).Background(paperBg).Border(0.5, paperEdge).Children(func() {
				ui.Text(c, "暂无匹配的已发布文章").Font(macPlanetFont).FontSize(12.5).TextColor(pal.muted)
			})
		}
	})
}

func (a *App) siteBookmarkChip(c *ui.Context, key, label string, site *store.Site, siteIdx int, dotColor ui.Color, active bool, onClick func()) {
	pal := a.pal
	_, paperBg, _, paperEdge, _ := paperDeskColors(pal)
	bg := ui.Transparent
	fg := pal.secondary
	if active {
		bg = paperBg
		fg = pal.ink
	}
	btn := ui.ButtonBase(c.Key(key+"-btn")).Label(label).
		Radius(6).Padding(5, 10, 5, 8).Background(bg).
		Cursor(ui.CursorPointer).Transition(rowHoverTransition)
	if active {
		btn.Border(0.5, paperEdge).Shadow(0, 1, 2, 0, pal.prefsNavActiveShadow)
	} else if btn.Hovered() {
		btn.Background(pal.hover)
	}
	btn.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			if site != nil {
				a.siteAvatar(c, *site, siteIdx, 16)
			}
			txt := ui.Text(c, label).Font(macPlanetFont).FontSize(12).TextColor(fg).NoWrap()
			if active {
				txt.FontWeight(600)
			}
			if site != nil {
				ui.Box(c).Size(5.5, 5.5).Radius(3).Background(dotColor).Shrink(0)
			}
		})
	})
	if btn.Clicked() {
		onClick()
	}
}

func (a *App) viewHeatmapGrid(c *ui.Context, counts map[string]int) {
	pal := a.pal
	const weeks = 36
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	endOffset := int(time.Saturday - today.Weekday())
	endDay := today.AddDate(0, 0, endOffset)
	startDay := endDay.AddDate(0, 0, -(weeks*7 - 1))

	switch a.activityMode {
	case activityModeWeekly:
		ui.Row(c).Gap(3).AlignItems(ui.End).Children(func() {
			for w := range weeks {
				sum := 0
				for d := range 7 {
					day := startDay.AddDate(0, 0, w*7+d)
					sum += counts[day.Format("2006-01-02")]
				}
				h := float32(6)
				if sum > 0 {
					h = min(float32(56), float32(12+sum*8))
				}
				tip := fmt.Sprintf("第 %d 周：%d 次", w+1, sum)
				ui.Box(c.Key(fmt.Sprintf("hm-week-%d", w))).Size(9, h).Radius(2).Background(heatmapColor(sum, pal)).Tooltip(tip)
			}
		})
	case activityModeCumulative:
		running := 0
		ui.Row(c).Gap(2.5).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-cw-%d", w))).Gap(2.5).Children(func() {
					for d := range 7 {
						day := startDay.AddDate(0, 0, w*7+d)
						key := day.Format("2006-01-02")
						running += counts[key]
						level := 0
						switch {
						case running >= 12:
							level = 6
						case running >= 6:
							level = 4
						case running >= 3:
							level = 2
						case running >= 1:
							level = 1
						}
						ui.Box(c.Key("hm-c-"+key)).Size(9, 9).Radius(2).Background(heatmapColor(level, pal)).
							Tooltip(fmt.Sprintf("%s：累计 %d 次", key, running))
					}
				})
			}
		})
	default:
		ui.Row(c).Gap(2.5).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-w-%d", w))).Gap(2.5).Children(func() {
					for d := range 7 {
						day := startDay.AddDate(0, 0, w*7+d)
						key := day.Format("2006-01-02")
						n := counts[key]
						ui.Box(c.Key("hm-"+key)).Size(9, 9).Radius(2).Background(heatmapColor(n, pal)).
							Tooltip(fmt.Sprintf("%s：%d 次", key, n))
					}
				})
			}
		})
	}
}

func (a *App) viewComposePickerModal(c *ui.Context) {
	pal := a.pal
	macSheet(c, &a.composePickerOpen, macSheetProps{
		Title:      "选择要写作的站点",
		Width:      380,
		HideFooter: true,
	}, pal, func() {
		ui.Text(c, "请选择新文章所属的站点：").Font(macPlanetFont).FontSize(12).TextColor(pal.secondary)
		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			for i, s := range a.sites {
				if s.Archived {
					continue
				}
				idx, site := i, s
				siteID, siteName := site.ID, fallbackTitle(site.Name)
				if interactiveRow(c, "compose-pick-"+siteID, siteName, pal.subtle, pal.hover, pal.active).
					Radius(6).PaddingX(12).PaddingY(8).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(8).AlignSelf(ui.Stretch).Children(func() {
							a.siteAvatar(c, site, idx, 22)
							ui.Text(c, siteName).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink)
						})
					}).Clicked() {
					a.composePickerOpen = false
					a.startNewDraft(siteID)
				}
			}
		})
	})
}

func heatmapColor(count int, pal macPalette) ui.Color {
	if count <= 0 {
		if pal.dark {
			return ui.RGBA(255, 255, 255, 0.06)
		}
		return ui.RGBA(15, 23, 42, 0.06)
	}
	switch {
	case count == 1:
		return pal.ink.Alpha(0.28)
	case count <= 3:
		return pal.ink.Alpha(0.50)
	case count <= 5:
		return pal.ink.Alpha(0.72)
	default:
		return pal.ink.Alpha(0.92)
	}
}

var _ = store.DashboardSite{}
