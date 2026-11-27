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

	dashboardFilterDrafts = "__drafts__"
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
		a.setStatus(a.tf("status.followedFeatured", followed))
	} else if lastErr != nil {
		a.setError(a.tf("status.followFeaturedErr", lastErr.Error()))
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

func (a *App) dashboardInnerWidth(c *ui.Context) float32 {
	winW, _ := c.Size()
	if winW <= 0 {
		return 1000
	}
	sideW := float32(0)
	if !a.sidebarCollapsed {
		sideW = a.sidebarWidth
		if sideW < 220 {
			sideW = 228
		}
	}
	inner := winW - sideW - 72 // 36px left + 36px right horizontal padding
	if inner < 420 {
		inner = 420
	}
	return inner
}

func dashboardCardColumns(containerW float32) int {
	if containerW < 680 {
		return 2
	}
	return 3
}

func formatMastheadDate(t time.Time) string {
	weekdays := [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	return fmt.Sprintf("%d月%d日 · %s", int(t.Month()), t.Day(), weekdays[t.Weekday()])
}

func formatWordCount(n int) string {
	if n <= 0 {
		return "暂无正文"
	}
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d 字", n/1000, n%1000)
	}
	return fmt.Sprintf("%d 字", n)
}

func letterpressSurfaceColors(pal macPalette) (hoverBg, hoverBorder, topHighlight ui.Color) {
	if pal.dark {
		return ui.Hex("#282a30"), ui.RGBA(255, 255, 255, 0.13), ui.RGBA(255, 255, 255, 0.065)
	}
	return ui.Hex("#fafbfd"), ui.RGBA(15, 23, 42, 0.13), ui.RGBA(255, 255, 255, 0.94)
}

// viewDashboardToolbar renders a single unified top toolbar seamless with
// pal.prefsCanvas, preserving window drag + caption button insets while hosting
// Search, Quick Post, and the sole primary "写文章" CTA.
func (a *App) viewDashboardToolbar(c *ui.Context) {
	pal := a.pal
	h := a.topBarHeight(c)
	rightPad := a.topBarRightPad(c)
	leftPad := float32(16)
	if a.sidebarCollapsed {
		leftPad = a.topBarLeftPad(c)
	}
	ui.Column(c).AlignSelf(ui.Stretch).Background(pal.prefsCanvas).Children(func() {
		ui.Row(c).Height(h).Padding(0, rightPad, 0, leftPad).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			AlignSelf(ui.Stretch).DragWindow().Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
				if a.sidebarCollapsed {
					if toolbarIconBtn(c, "tb-expand-sidebar", lucide.PanelLeftOpen(), a.t("nav.expandSidebar")).Clicked() {
						a.sidebarCollapsed = false
					}
				}
			})
			ui.Row(c).AlignItems(ui.Center).Gap(8).Shrink(0).Children(func() {
				if toolbarIconBtn(c, "dash-tb-search", lucide.Search(), a.t("toolbar.searchPosts")).Clicked() {
					a.openGlobalSearch()
				}
				if len(a.sites) > 0 {
					if macSecondaryButton(c, "dash-hero-quick", a.t("toolbar.quickPost"), pal) {
						a.openQuickPost()
					}
				}
				composeLabel := a.t("toolbar.writePost")
				btn := ui.ButtonBase(c.Key("dash-hero-compose")).Label(composeLabel).
					Height(28).PaddingX(14).Radius(6).Background(pal.prefsInk).
					Center().Cursor(ui.CursorPointer)
				if btn.Hovered() {
					btn.Background(pal.prefsInk.Alpha(0.86))
				}
				btn.Children(func() {
					ui.Text(c, composeLabel).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.prefsCanvas).NoWrap()
				})
				if btn.Clicked() {
					a.handleDashboardCompose()
				}
			})
		})
		ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)
	})
}

func (a *App) viewDashboard(c *ui.Context) {
	pal := a.pal
	innerW := a.dashboardInnerWidth(c)
	containerW := min(innerW, float32(1120))

	ui.Scroll(c).TrackScroll(&a.dashboardScroll).Grow(1).MinHeight(0).Background(pal.prefsCanvas).Children(func() {
		ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
			ui.Column(c).Width(containerW).Gap(24).Children(func() {
				if a.dashboard.SiteCount == 0 {
					ui.Column(c).AlignSelf(ui.Stretch).Padding(40).Radius(12).
						Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
						Shadow(0, 4, 14, 0, pal.prefsCardShadow).Children(func() {
						display.Empty(c, display.EmptyProps{
							Icon:        lucide.Globe(),
							Title:       a.t("dashboard.emptyTitle"),
							Description: a.t("dashboard.emptyDesc"),
							Action: func() {
								ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
									if macPrimaryButton(c, "dash-create-first", a.t("dashboard.createFirstSite"), pal) {
										a.createSiteOpen = true
									}
									if len(a.subscriptions) == 0 {
										if macSecondaryButton(c, "dash-follow-featured", a.t("dashboard.followFeatured"), pal) {
											a.followFeaturedPlanets()
										}
									}
								})
							},
						})
					})
					return
				}

				// 1. Editorial Date Masthead (left) + draggable writing badges (center-left) + compact 14-week mini-heatmap (right).
				a.viewDashboardMasthead(c, containerW)

				// 2. Single Elevated Manuscript Paper Sheet (focal point: most recent draft or blank sheet).
				a.viewDashboardHeroPaper(c)

				// 3. Site Filter Shelf + 3:2 Letterpress Magazine Cards.
				a.viewDashboardArticles(c, containerW)
			})
		})
	})
}

func (a *App) viewDashboardMasthead(c *ui.Context, containerW float32) {
	pal := a.pal
	dateText := a.formatMastheadDate(time.Now())
	counts := make(map[string]int, len(a.activity))
	for _, item := range a.activity {
		counts[item.Date] = item.Articles + item.Drafts
	}

	const (
		dateSize     = float32(26)
		heatmapW     = float32(14*9 + 13*3) // 165px
		mastheadGaps = float32(18 + 16)
	)
	dateW, _ := c.MeasureText(0, ui.Span{Text: dateText, Font: macPlanetFont, Size: dateSize, Weight: 600})
	availBadgeW := containerW - dateW - heatmapW - mastheadGaps

	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(16).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(18).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, dateText).
				Font(macPlanetFont).FontSize(dateSize).FontWeight(600).LetterSpacing(-0.02).
				TextColor(pal.prefsInk).NoWrap().Shrink(0)
			a.viewDashboardBadgeTray(c, availBadgeW)
		})
		a.viewMiniHeatmap(c, counts)
	})
}

func (a *App) viewMiniHeatmap(c *ui.Context, counts map[string]int) {
	pal := a.pal
	const weeks = 14
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	endOffset := int(time.Saturday - today.Weekday())
	endDay := today.AddDate(0, 0, endOffset)
	startDay := endDay.AddDate(0, 0, -(weeks*7 - 1))

	ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
		for w := range weeks {
			ui.Column(c.Key(fmt.Sprintf("hm-mw-%d", w))).Gap(3).Children(func() {
				for d := range 7 {
					day := startDay.AddDate(0, 0, w*7+d)
					key := day.Format("2006-01-02")
					n := counts[key]
					ui.Box(c.Key("hm-m-"+key)).Size(9, 9).Radius(2).
						Background(heatmapColor(n, pal)).
						Tooltip(a.tf("dashboard.heatmapTooltip", key, n))
				}
			})
		}
	})
}

func (a *App) viewDashboardHeroPaper(c *ui.Context) {
	pal := a.pal
	hoverBg, hoverBorder, topHighlight := letterpressSurfaceColors(pal)

	if len(a.dashboard.RecentDrafts) == 0 {
		card := ui.ButtonBase(c.Key("dash-hero-empty-sheet")).Label(a.t("dashboard.startWritingLabel")).
			AlignSelf(ui.Stretch).AlignItems(ui.Stretch).Radius(12).
			Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
			Shadow(0, 4, 14, 0, pal.prefsCardShadow).
			Cursor(ui.CursorPointer)
		if card.Hovered() {
			card.Background(hoverBg).Border(0.5, hoverBorder)
		}
		card.Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Radius(12).Clip().AlignItems(ui.Stretch).Children(func() {
				ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(topHighlight)
				ui.Column(c).AlignSelf(ui.Stretch).Padding(22, 24, 20, 24).Gap(8).AlignItems(ui.Start).Children(func() {
					ui.Text(c, a.t("dashboard.blankPaper")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
					ui.Text(c, a.t("dashboard.writeToday")).Font(macPlanetFont).FontSize(20).FontWeight(600).LetterSpacing(-0.02).TextColor(pal.prefsInk)
					ui.Text(c, a.t("dashboard.blankPaperDesc")).Font(macPlanetFont).FontSize(13.5).TextColor(pal.prefsSecondary)
					ui.Box(c).Height(4)
					ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)
					ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(8, 0, 0, 0).Children(func() {
						ui.Text(c, a.t("dashboard.localWriting")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
						ui.Text(c, a.t("dashboard.startWritingAction")).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(pal.prefsInk)
					})
				})
			})
		})
		if card.Clicked() {
			a.handleDashboardCompose()
		}
		return
	}

	d := a.dashboard.RecentDrafts[0]
	siteID, draftID := d.SiteID, d.ID
	title := a.fallbackTitle(d.Title)
	siteName := a.fallbackTitle(d.SiteName)
	label := fmt.Sprintf("%s  ·  %s", title, siteName)
	key := "draft:" + siteID + ":" + draftID
	excerpt := ""
	wordCount := 0
	if a.articleExcerpts != nil {
		excerpt = a.articleExcerpts[key]
	}
	if a.articleWordCounts != nil {
		wordCount = a.articleWordCounts[key]
	}
	if excerpt == "" {
		excerpt = a.t("dashboard.localDraftHint")
	}

	card := ui.ButtonBase(c.Key("dash-hero-draft-"+draftID)).Label(label).
		AlignSelf(ui.Stretch).AlignItems(ui.Stretch).Radius(12).
		Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
		Shadow(0, 4, 14, 0, pal.prefsCardShadow).
		Cursor(ui.CursorPointer)
	if card.Hovered() {
		card.Background(hoverBg).Border(0.5, hoverBorder)
	}
	card.Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Radius(12).Clip().AlignItems(ui.Stretch).Children(func() {
			// Top 0.5px specular light edge for tactile letterpress paper feel.
			ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(topHighlight)
			ui.Column(c).AlignSelf(ui.Stretch).Padding(22, 24, 18, 24).Gap(8).AlignItems(ui.Start).Children(func() {
				ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					ui.Text(c, a.tf("dashboard.continueWritingPrefix", siteName)).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted).NoWrap()
					ui.Text(c, a.formatShortDate(d.Modified)).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted).NoWrap()
				})
				ui.Text(c, title).Font(macPlanetFont).FontSize(20).FontWeight(600).LetterSpacing(-0.02).
					TextColor(pal.prefsInk).MaxLines(2).Ellipsis("…")
				ui.Text(c, excerpt).Font(macPlanetFont).FontSize(13.5).TextColor(pal.prefsSecondary).
					MaxLines(3).Ellipsis("…")
				ui.Box(c).Height(6)
				ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)
				ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(8, 0, 0, 0).Children(func() {
					meta := a.t("dashboard.localDraft")
					if wordCount > 0 {
						meta += " · " + a.formatWordCount(wordCount)
					}
					ui.Text(c, meta).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted).NoWrap()
					ui.Text(c, a.t("dashboard.continueWritingAction")).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(pal.prefsInk).NoWrap()
				})
			})
		})
	})
	if card.Clicked() {
		a.openDraftByID(siteID, draftID)
	}
}

type dashboardCardItem struct {
	entry   store.DashboardEntry
	isDraft bool
}

func (a *App) siteStatusDotColor(siteID string) ui.Color {
	pal := a.pal
	for _, s := range a.dashboard.Sites {
		if s.ID == siteID {
			if s.PublishPending {
				return pal.amber
			}
			if s.HasPublished {
				return pal.green
			}
			return pal.prefsMuted
		}
	}
	return pal.prefsMuted
}

func (a *App) viewDashboardArticles(c *ui.Context, containerW float32) {
	pal := a.pal
	hoverBg, hoverBorder, topHighlight := letterpressSurfaceColors(pal)

	if a.articleFilterSiteID == dashboardFilterDrafts && len(a.dashboard.RecentDrafts) == 0 {
		a.articleFilterSiteID = ""
	} else if a.articleFilterSiteID != "" && a.articleFilterSiteID != dashboardFilterDrafts {
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

	ui.Column(c).AlignSelf(ui.Stretch).Gap(16).Children(func() {
		// Unified Site Bookmark Shelf.
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Grow(1).MinWidth(0).Children(func() {
				a.siteBookmarkChip(c, "art-filter-all", a.t("common.all"), nil, -1, ui.Transparent, a.articleFilterSiteID == "", func() {
					a.articleFilterSiteID = ""
				})
				if len(a.dashboard.RecentDrafts) > 1 {
					a.draftBookmarkChip(c, a.tf("dashboard.draftsChip", len(a.dashboard.RecentDrafts)), a.articleFilterSiteID == dashboardFilterDrafts, func() {
						if a.articleFilterSiteID == dashboardFilterDrafts {
							a.articleFilterSiteID = ""
						} else {
							a.articleFilterSiteID = dashboardFilterDrafts
						}
					})
				}
				for i, s := range a.dashboard.Sites {
					idx, sid := i, s.ID
					dotColor := a.siteStatusDotColor(sid)
					var fullSite store.Site
					fullSite.Name = s.Name
					for _, candidate := range a.sites {
						if candidate.ID == sid {
							fullSite = candidate
							break
						}
					}
					fs := fullSite
					a.siteBookmarkChip(c, "art-filter-"+sid, a.fallbackTitle(s.Name), &fs, idx, dotColor, a.articleFilterSiteID == sid, func() {
						if a.articleFilterSiteID == sid {
							a.selectSite(sid)
						} else {
							a.articleFilterSiteID = sid
						}
					})
				}
			})
			if macIconButton(c, "dash-sites-add", iconPlus, 26, 13, pal.prefsMuted, a.t("dashboard.newSiteTooltip")) {
				a.newSiteName = ""
				a.newSiteAbout = ""
				a.createSiteError = ""
				a.createSiteOpen = true
			}
		})

		// Collect cards: when "全部" is active, the first draft is already showcased in the Hero Paper Sheet.
		var items []dashboardCardItem
		if a.articleFilterSiteID == dashboardFilterDrafts {
			for _, d := range a.dashboard.RecentDrafts {
				items = append(items, dashboardCardItem{entry: d, isDraft: true})
			}
		} else {
			for i, d := range a.dashboard.RecentDrafts {
				if a.articleFilterSiteID == "" && i == 0 {
					continue
				}
				if a.articleFilterSiteID != "" && d.SiteID != a.articleFilterSiteID {
					continue
				}
				items = append(items, dashboardCardItem{entry: d, isDraft: true})
			}
			for _, art := range a.dashboard.RecentArticles {
				if a.articleFilterSiteID != "" && art.SiteID != a.articleFilterSiteID {
					continue
				}
				items = append(items, dashboardCardItem{entry: art, isDraft: false})
			}
		}

		if len(items) == 0 {
			ui.Column(c).AlignSelf(ui.Stretch).Padding(24).Radius(10).
				Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).Children(func() {
				ui.Text(c, a.t("dashboard.noMatchingPosts")).Font(macPlanetFont).FontSize(12.5).TextColor(pal.prefsMuted)
			})
			return
		}

		cols := dashboardCardColumns(containerW)
		if len(items) < cols {
			cols = len(items)
		}
		ui.Column(c).AlignSelf(ui.Stretch).Gap(16).Children(func() {
			for rowStart := 0; rowStart < len(items); rowStart += cols {
				ui.Row(c).AlignSelf(ui.Stretch).Gap(16).Children(func() {
					for col := range cols {
						idx := rowStart + col
						if idx >= len(items) {
							ui.Box(c).Grow(1).MinWidth(0)
							continue
						}
						item := items[idx]
						entry := item.entry
						siteID, itemID := entry.SiteID, entry.ID
						title := a.fallbackTitle(entry.Title)
						siteName := a.fallbackTitle(entry.SiteName)
						label := fmt.Sprintf("%s  ·  %s", title, siteName)
						prefix := "art:"
						cardKey := "dash-art-" + itemID
						if item.isDraft {
							prefix = "draft:"
							cardKey = "dash-draft-" + itemID
						}
						cacheKey := prefix + siteID + ":" + itemID
						excerpt := ""
						wordCount := 0
						if a.articleExcerpts != nil {
							excerpt = a.articleExcerpts[cacheKey]
						}
						if a.articleWordCounts != nil {
							wordCount = a.articleWordCounts[cacheKey]
						}
						if item.isDraft && excerpt == "" {
							excerpt = a.t("dashboard.localDraftHint")
						}

						metaLeft := siteName
						if item.isDraft {
							metaLeft = a.tf("dashboard.draftSuffix", siteName)
						}
						dotColor := a.siteStatusDotColor(siteID)

						card := ui.ButtonBase(c.Key(cardKey)).Label(label).
							Grow(1).MinWidth(0).MinHeight(148).Radius(10).
							Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
							Shadow(0, 2, 8, 0, pal.prefsCardShadow).
							Cursor(ui.CursorPointer)
						hovered := card.Hovered()
						if hovered {
							card.Background(hoverBg).Border(0.5, hoverBorder)
						}
						card.Children(func() {
							ui.Column(c).Fill().Radius(10).Clip().AlignItems(ui.Start).Children(func() {
								// Top 0.5px specular light edge (Letterpress Paper).
								ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(topHighlight)
								ui.Column(c).Fill().Padding(18, 20, 15, 20).Gap(8).AlignItems(ui.Start).Children(func() {
									// Top metadata: site dot + site name on left, date on right.
									ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Children(func() {
										ui.Row(c).AlignItems(ui.Center).Gap(6).MinWidth(0).Children(func() {
											ui.Box(c).Size(5.5, 5.5).Radius(3).Background(dotColor).Shrink(0)
											ui.Text(c, metaLeft).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).NoWrap().Ellipsis("…")
										})
										ui.Text(c, a.formatShortDate(entry.Modified)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).Shrink(0).NoWrap()
									})
									// Bold editorial title.
									ui.Text(c, title).Font(macPlanetFont).FontSize(15.5).FontWeight(600).LetterSpacing(-0.015).
										TextColor(pal.prefsInk).MaxLines(2).Ellipsis("…")
									// Prose excerpt.
									if excerpt != "" {
										ui.Text(c, excerpt).Font(macPlanetFont).FontSize(12.5).TextColor(pal.prefsSecondary).
											MaxLines(3).Ellipsis("…")
									}
									ui.Box(c).Grow(1)
									// Subtle hairline footer: word count left, hover-revealed action right.
									ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)
									ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Padding(6, 0, 0, 0).Children(func() {
										ui.Text(c, a.formatWordCount(wordCount)).Font(macPlanetFont).FontSize(11).TextColor(pal.prefsMuted).NoWrap()
										if hovered {
											actionText := a.t("dashboard.readAction")
											if item.isDraft {
												actionText = a.t("dashboard.continueWritingAction")
											}
											ui.Text(c, actionText).Font(macPlanetFont).FontSize(11).FontWeight(500).TextColor(pal.prefsInk).NoWrap()
										}
									})
								})
							})
						})
						if card.Clicked() {
							if item.isDraft {
								a.openDraftByID(siteID, itemID)
							} else {
								a.openArticleForRead(siteID, itemID)
							}
						}
					}
				})
			}
		})
	})
}

func (a *App) draftBookmarkChip(c *ui.Context, label string, active bool, onClick func()) {
	pal := a.pal
	bg := ui.Transparent
	fg := pal.prefsSecondary
	if active {
		bg = pal.prefsNavActiveBg
		fg = pal.prefsInk
	}
	btn := ui.ButtonBase(c.Key("art-filter-drafts-btn")).Label(label).
		Radius(6).Padding(5, 10, 5, 9).Background(bg).
		Cursor(ui.CursorPointer)
	if active {
		btn.Border(0.5, pal.prefsCardBorder).Shadow(0, 1, 2, 0, pal.prefsNavActiveShadow)
	} else if btn.Hovered() {
		btn.Background(pal.hover)
	}
	btn.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
			ui.Icon(c, lucide.PenLine()).Size(12, 12).TextColor(fg)
			txt := ui.Text(c, label).Font(macPlanetFont).FontSize(12).TextColor(fg).NoWrap()
			if active {
				txt.FontWeight(600)
			}
		})
	})
	if btn.Clicked() {
		onClick()
	}
}

func (a *App) siteBookmarkChip(c *ui.Context, key, label string, site *store.Site, siteIdx int, dotColor ui.Color, active bool, onClick func()) {
	pal := a.pal
	bg := ui.Transparent
	fg := pal.prefsSecondary
	if active {
		bg = pal.prefsNavActiveBg
		fg = pal.prefsInk
	}
	btn := ui.ButtonBase(c.Key(key+"-btn")).Label(label).
		Radius(6).Padding(5, 10, 5, 8).Background(bg).
		Cursor(ui.CursorPointer)
	if active {
		btn.Border(0.5, pal.prefsCardBorder).Shadow(0, 1, 2, 0, pal.prefsNavActiveShadow)
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

func (a *App) viewComposePickerModal(c *ui.Context) {
	pal := a.pal
	macSheet(c, &a.composePickerOpen, macSheetProps{
		Title:        a.t("dashboard.composePickerTitle"),
		Width:        380,
		HideFooter:   true,
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Text(c, a.t("dashboard.composePickerPrompt")).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary)
		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			for i, s := range a.sites {
				if s.Archived {
					continue
				}
				idx, site := i, s
				siteID, siteName := site.ID, a.fallbackTitle(site.Name)
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
			return ui.RGBA(255, 255, 255, 0.09)
		}
		return ui.RGBA(15, 23, 42, 0.06)
	}
	switch {
	case count == 1:
		return pal.prefsInk.Alpha(0.26)
	case count <= 3:
		return pal.prefsInk.Alpha(0.48)
	case count <= 5:
		return pal.prefsInk.Alpha(0.70)
	default:
		return pal.prefsInk.Alpha(0.90)
	}
}

var _ = store.DashboardSite{}
