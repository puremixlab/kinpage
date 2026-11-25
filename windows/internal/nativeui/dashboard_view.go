package nativeui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/store"
)

const (
	activityModeDaily      = 0
	activityModeWeekly     = 1
	activityModeCumulative = 2
)

const (
	trendRange30d = 0
	trendRange90d = 1
	trendRange1y  = 2
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

// dashboardCard renders a macOS Sequoia elevated card over the soft prefsCanvas.
func dashboardCard(c *ui.Context, pal macPalette, padding float32, body func()) {
	ui.Column(c).AlignSelf(ui.Stretch).Padding(padding).Gap(12).
		Background(pal.prefsCard).Radius(12).
		Border(0.5, pal.prefsCardBorder).
		Shadow(0, 1, 3, 0, pal.prefsCardShadow).
		Children(body)
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

func (a *App) viewDashboard(c *ui.Context) {
	pal := a.pal
	last30, streak := a.writingStatsSummary()

	ui.Scroll(c).TrackScroll(&a.dashboardScroll).Grow(1).MinHeight(0).Background(pal.prefsCanvas).Children(func() {
		ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(32).PaddingY(24).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(1140).Gap(18).Children(func() {
				// Hero top bar: clean title + date/streak pill on left, primary actions on right.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
						ui.Text(c, "写作台").Font(macPlanetFont).FontSize(22).FontWeight(700).LetterSpacing(-0.02).TextColor(pal.prefsInk)
						pillText := time.Now().Format("1月2日")
						if streak > 0 {
							pillText = fmt.Sprintf("%s · 连续 %d 天", pillText, streak)
						}
						ui.Box(c).Padding(3, 9, 3, 9).Radius(999).Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).Children(func() {
							ui.Text(c, pillText).Font(macPlanetFont).FontSize(11.5).FontWeight(500).TextColor(pal.prefsMuted).NoWrap()
						})
					})
					ui.Row(c).AlignItems(ui.Center).Gap(8).Shrink(0).Children(func() {
						if len(a.sites) > 0 {
							if macSecondaryButton(c, "dash-hero-quick", "快捷发文", pal) {
								a.openQuickPost()
							}
						}
						if macPrimaryButtonSized(c, "dash-hero-compose", "写文章", 30, 14, 13, pal) {
							a.handleDashboardCompose()
						}
					})
				})

				// 4-column Sequoia KPI strip.
				ui.Row(c).Gap(14).AlignSelf(ui.Stretch).Children(func() {
					dashboardKPICard(c, "kpi-sites", lucide.Globe(), pal.accent, fmt.Sprintf("%d", a.dashboard.SiteCount), "站点", pal)
					dashboardKPICard(c, "kpi-articles", lucide.FileText(), pal.green, fmt.Sprintf("%d", a.dashboard.ArticleCount), "已发布文章", pal)
					dashboardKPICard(c, "kpi-drafts", lucide.PenLine(), pal.orange, fmt.Sprintf("%d", a.dashboard.DraftCount), "本地草稿", pal)
					dashboardKPICard(c, "kpi-30d", lucide.Flame(), ui.Hex("#af52de"), fmt.Sprintf("%d", last30), "近 30 天写作", pal)
				})

				if a.dashboard.SiteCount == 0 {
					dashboardCard(c, pal, 40, func() {
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

				// Two-column Sequoia Bento grid: left main (1fr) + right sites panel (280px).
				ui.Row(c).Gap(18).AlignSelf(ui.Stretch).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(16).Children(func() {
						a.viewDashboardDrafts(c)
						a.viewDashboardArticles(c)
						a.viewActivitySection(c)
					})
					ui.Column(c).Width(280).Shrink(0).Gap(16).Children(func() {
						a.viewDashboardSites(c)
					})
				})

				// Minimal subtle footer.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).PaddingY(4).Children(func() {
					ui.Text(c, "所有草稿与站点均保存在本地").Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
					if ghostIconButton(c, "dash-refresh", "刷新数据", "刷新数据", pal) {
						a.Reload()
						a.setStatus("已刷新数据")
					}
				})
			})
		})
	})
}

func dashboardKPICard(c *ui.Context, key string, icon *ui.SVG, tint ui.Color, value, label string, pal macPalette) {
	badgeAlpha := float32(0.13)
	if pal.dark {
		badgeAlpha = 0.22
	}
	ui.Column(c.Key(key)).Grow(1).MinWidth(0).Padding(14, 16, 14, 16).Gap(10).
		Background(pal.prefsCard).Radius(12).
		Border(0.5, pal.prefsCardBorder).
		Shadow(0, 1, 3, 0, pal.prefsCardShadow).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, label).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(pal.prefsMuted).NoWrap()
				ui.Box(c).Size(24, 24).Radius(6).Background(tint.Alpha(badgeAlpha)).Center().Shrink(0).Children(func() {
					ui.Icon(c, icon).Size(13, 13).TextColor(tint)
				})
			})
			ui.Text(c, value).Font(macPlanetFont).FontSize(24).FontWeight(700).LetterSpacing(-0.02).TextColor(pal.prefsInk).NoWrap()
		})
}

func (a *App) viewDashboardDrafts(c *ui.Context) {
	pal := a.pal
	dashboardCard(c, pal, 18, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, "继续写作").Font(macPlanetFont).FontSize(14.5).FontWeight(600).TextColor(pal.prefsInk)
				if len(a.dashboard.RecentDrafts) > 0 {
					ui.Box(c).Padding(1, 7, 1, 7).Radius(999).Background(pal.pillBg).Children(func() {
						ui.Text(c, fmt.Sprintf("%d", len(a.dashboard.RecentDrafts))).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.prefsSecondary)
					})
				}
			})
		})
		if len(a.dashboard.RecentDrafts) == 0 {
			ui.Text(c, "暂无本地草稿").Font(macPlanetFont).FontSize(12.5).TextColor(pal.prefsMuted).PaddingY(6)
			return
		}
		ui.Column(c).AlignSelf(ui.Stretch).Gap(0).Children(func() {
			for i, d := range a.dashboard.RecentDrafts {
				siteID, draftID := d.SiteID, d.ID
				title := fallbackTitle(d.Title)
				siteName := fallbackTitle(d.SiteName)
				label := fmt.Sprintf("%s  ·  %s", title, siteName)
				if interactiveRow(c, "dash-draft-"+draftID, label, ui.Transparent, pal.hover, pal.active).MinHeight(52).
					Radius(8).PaddingX(10).PaddingY(8).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(10).Grow(1).MinWidth(0).Children(func() {
							ui.Box(c).Size(28, 28).Radius(7).Background(pal.orange.Alpha(0.14)).Center().Shrink(0).Children(func() {
								ui.Icon(c, iconEdit).Size(14, 14).TextColor(pal.orange)
							})
							ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, title).Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.prefsInk).MaxLines(1).Ellipsis("…")
								ui.Text(c, siteName+" · 本地草稿").Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).NoWrap()
							})
							ui.Text(c, formatShortDate(d.Modified)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).Shrink(0).NoWrap()
						})
					}).Clicked() {
					a.openDraftByID(siteID, draftID)
				}
				if i < len(a.dashboard.RecentDrafts)-1 {
					ui.Box(c).PaddingX(10).AlignSelf(ui.Stretch).Children(func() { prefsHairlineH(c, pal) })
				}
			}
		})
	})
}

func (a *App) viewDashboardArticles(c *ui.Context) {
	pal := a.pal
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

	dashboardCard(c, pal, 18, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "最近文章").Font(macPlanetFont).FontSize(14.5).FontWeight(600).TextColor(pal.prefsInk)
			if len(a.dashboard.Sites) > 1 {
				ui.Row(c).Gap(4).Wrap().AlignItems(ui.Center).Children(func() {
					a.dashboardFilterPill(c, "art-filter-all", "所有站点", a.articleFilterSiteID == "", func() {
						a.articleFilterSiteID = ""
					})
					for _, s := range a.dashboard.Sites {
						sid := s.ID
						a.dashboardFilterPill(c, "art-filter-"+sid, fallbackTitle(s.Name), a.articleFilterSiteID == sid, func() {
							a.articleFilterSiteID = sid
						})
					}
				})
			}
		})

		var visibleIdx []int
		for i, art := range a.dashboard.RecentArticles {
			if a.articleFilterSiteID != "" && art.SiteID != a.articleFilterSiteID {
				continue
			}
			visibleIdx = append(visibleIdx, i)
		}
		if len(visibleIdx) == 0 {
			ui.Text(c, "暂无匹配的已发布文章").Font(macPlanetFont).FontSize(12.5).TextColor(pal.prefsMuted).PaddingY(6)
			return
		}
		ui.Column(c).AlignSelf(ui.Stretch).Gap(0).Children(func() {
			for pos, idx := range visibleIdx {
				art := a.dashboard.RecentArticles[idx]
				siteID, artID := art.SiteID, art.ID
				title := fallbackTitle(art.Title)
				siteName := fallbackTitle(art.SiteName)
				label := fmt.Sprintf("%s  ·  %s", title, siteName)
				if interactiveRow(c, "dash-art-"+artID, label, ui.Transparent, pal.hover, pal.active).MinHeight(46).
					Radius(8).PaddingX(10).PaddingY(8).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(10).Grow(1).MinWidth(0).Children(func() {
							ui.Icon(c, iconFile).Size(15, 15).TextColor(pal.prefsMuted).Shrink(0)
							ui.Text(c, title).Font(macPlanetFont).FontSize(13.5).FontWeight(500).TextColor(pal.prefsInk).MaxLines(1).Ellipsis("…").Grow(1)
							ui.Box(c).Padding(2, 7, 2, 7).Radius(5).Background(pal.prefsCanvas).Border(0.5, pal.prefsCardBorder).Shrink(0).Children(func() {
								ui.Text(c, siteName).Font(macPlanetFont).FontSize(11).TextColor(pal.prefsSecondary).NoWrap()
							})
							ui.Text(c, formatShortDate(art.Modified)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).Width(56).Shrink(0).NoWrap()
						})
					}).Clicked() {
					a.openArticleForRead(siteID, artID)
				}
				if pos < len(visibleIdx)-1 {
					ui.Box(c).PaddingX(10).AlignSelf(ui.Stretch).Children(func() { prefsHairlineH(c, pal) })
				}
			}
		})
	})
}

func (a *App) dashboardFilterPill(c *ui.Context, key, label string, active bool, onClick func()) {
	pal := a.pal
	bg := ui.Transparent
	fg := pal.prefsSecondary
	if active {
		bg = pal.accent.Alpha(0.14)
		if pal.dark {
			bg = pal.accent.Alpha(0.24)
		}
		fg = pal.accent
	}
	btn := ui.ButtonBase(c.Key(key+"-btn")).Label(label).
		Radius(999).Padding(4, 10, 4, 10).Background(bg).
		Cursor(ui.CursorPointer).Transition(rowHoverTransition)
	if !active && btn.Hovered() {
		btn.Background(pal.hover)
	}
	btn.Children(func() {
		txt := ui.Text(c, label).Font(macPlanetFont).FontSize(11.5).TextColor(fg).NoWrap()
		if active {
			txt.FontWeight(600)
		}
	})
	if btn.Clicked() {
		onClick()
	}
}

func (a *App) viewDashboardSites(c *ui.Context) {
	pal := a.pal
	dashboardCard(c, pal, 18, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "站点状态").Font(macPlanetFont).FontSize(14.5).FontWeight(600).TextColor(pal.prefsInk)
			if macIconButton(c, "dash-sites-add", iconPlus, 26, 14, pal.prefsMuted, "新建站点") {
				a.newSiteName = ""
				a.newSiteAbout = ""
				a.createSiteError = ""
				a.createSiteOpen = true
			}
		})
		ui.Column(c).AlignSelf(ui.Stretch).Gap(0).Children(func() {
			for i, s := range a.dashboard.Sites {
				idx, site := i, s
				statusLabel := "未发布"
				dotColor := pal.prefsMuted
				if site.PublishPending {
					statusLabel = "有改动"
					dotColor = pal.amber
				} else if site.HasPublished {
					statusLabel = "已发布"
					dotColor = pal.green
				}
				var fullSite store.Site
				fullSite.Name = site.Name
				for _, candidate := range a.sites {
					if candidate.ID == site.ID {
						fullSite = candidate
						break
					}
				}
				if interactiveRow(c, "dash-site-row-"+site.ID, fallbackTitle(site.Name), ui.Transparent, pal.hover, pal.active).
					Radius(8).PaddingX(8).PaddingY(9).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(10).Grow(1).MinWidth(0).Children(func() {
							a.siteAvatar(c, fullSite, idx, 30)
							ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, fallbackTitle(site.Name)).Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.prefsInk).MaxLines(1).Ellipsis("…")
								ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
									ui.Box(c).Size(6, 6).Radius(3).Background(dotColor).Shrink(0)
									ui.Text(c, fmt.Sprintf("%s · %d 篇 · %d 草稿", statusLabel, site.ArticleCount, site.DraftCount)).
										Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).NoWrap()
								})
							})
							ui.Icon(c, iconArrow).Size(13, 13).TextColor(pal.prefsMuted).Shrink(0)
						})
					}).Clicked() {
					a.selectSite(site.ID)
				}
				if i < len(a.dashboard.Sites)-1 {
					ui.Box(c).PaddingX(8).AlignSelf(ui.Stretch).Children(func() { prefsHairlineH(c, pal) })
				}
			}
		})
	})
}

func (a *App) viewActivitySection(c *ui.Context) {
	pal := a.pal
	counts := make(map[string]int, len(a.activity))
	totalActions := 0
	for _, item := range a.activity {
		n := item.Articles + item.Drafts
		counts[item.Date] = n
		totalActions += n
	}

	dashboardCard(c, pal, 18, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "写作活跃度").Font(macPlanetFont).FontSize(14.5).FontWeight(600).TextColor(pal.prefsInk)
				ui.Text(c, fmt.Sprintf("累计 %d 次写作记录", totalActions)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			})
			macSegmented(c, "activity-mode", &a.activityMode, []string{"按日", "按周", "累计"}, pal)
		})

		a.viewHeatmapGrid(c, counts)

		// Compact legend bar.
		ui.Row(c).AlignItems(ui.Center).Justify(ui.End).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "少").Font(macPlanetFont).FontSize(10.5).TextColor(pal.prefsMuted)
			for _, lvl := range []int{0, 1, 2, 4, 6} {
				ui.Box(c).Size(10, 10).Radius(2.5).Background(heatmapColor(lvl, pal))
			}
			ui.Text(c, "多").Font(macPlanetFont).FontSize(10.5).TextColor(pal.prefsMuted)
		})

		prefsHairlineH(c, pal)

		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "写作趋势").Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.prefsInk)
			macSegmented(c, "trend-range", &a.trendRange, []string{"30 天", "90 天", "1 年"}, pal)
		})

		a.viewActivityTrendChart(c, counts)
	})
}

func (a *App) viewHeatmapGrid(c *ui.Context, counts map[string]int) {
	pal := a.pal
	const weeks = 28
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	endOffset := int(time.Saturday - today.Weekday())
	endDay := today.AddDate(0, 0, endOffset)
	startDay := endDay.AddDate(0, 0, -(weeks*7 - 1))

	switch a.activityMode {
	case activityModeWeekly:
		ui.Row(c).Gap(5).AlignItems(ui.End).Children(func() {
			for w := range weeks {
				sum := 0
				for d := range 7 {
					day := startDay.AddDate(0, 0, w*7+d)
					sum += counts[day.Format("2006-01-02")]
				}
				h := float32(8)
				if sum > 0 {
					h = min(float32(52), float32(14+sum*7))
				}
				tip := fmt.Sprintf("第 %d 周：%d 次", w+1, sum)
				ui.Box(c.Key(fmt.Sprintf("hm-week-%d", w))).Size(14, h).Radius(3.5).Background(heatmapColor(sum, pal)).Tooltip(tip)
			}
		})
	case activityModeCumulative:
		running := 0
		ui.Row(c).Gap(3).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-cw-%d", w))).Gap(3).Children(func() {
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
						ui.Box(c.Key("hm-c-"+key)).Size(12, 12).Radius(3).Background(heatmapColor(level, pal)).
							Tooltip(fmt.Sprintf("%s：累计 %d 次", key, running))
					}
				})
			}
		})
	default:
		ui.Row(c).Gap(3).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-w-%d", w))).Gap(3).Children(func() {
					for d := range 7 {
						day := startDay.AddDate(0, 0, w*7+d)
						key := day.Format("2006-01-02")
						n := counts[key]
						ui.Box(c.Key("hm-"+key)).Size(12, 12).Radius(3).Background(heatmapColor(n, pal)).
							Tooltip(fmt.Sprintf("%s：%d 次", key, n))
					}
				})
			}
		})
	}
}

func (a *App) viewActivityTrendChart(c *ui.Context, counts map[string]int) {
	pal := a.pal
	days := 30
	switch a.trendRange {
	case trendRange90d:
		days = 90
	case trendRange1y:
		days = 365
	}

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	start := today.AddDate(0, 0, -(days - 1))

	dates := make([]time.Time, days)
	values := make([]float64, days)
	maxVal := 1.0
	for i := range days {
		d := start.AddDate(0, 0, i)
		dates[i] = d
		v := float64(counts[d.Format("2006-01-02")])
		values[i] = v
		if v > maxVal {
			maxVal = v
		}
	}

	x := chart.NewTime(start, today)
	y := chart.NewLinear(0, maxVal).Nice(4)
	yTicks := y.Ticks(4)
	xs := make([]float32, days)
	ys := make([]float32, days)
	for i := range days {
		xs[i] = x.At(dates[i])
		ys[i] = y.At(values[i])
	}

	labelFn := func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(pal.prefsMuted)
	}

	ui.Column(c).AlignSelf(ui.Stretch).Height(152).Children(func() {
		chart.Plot(c.Key("trend-plot"), chart.Axes{Left: yTicks, Bottom: x.Ticks(5), Label: labelFn}, func(area ui.Element) {
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range yTicks {
					py := chart.Y(r, tick.At)
					p.Line(r.X, py, r.X+r.W, py, 0.5, pal.prefsCardBorder)
				}
				p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), pal.accent.Alpha(0.13))
				p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, pal.accent)
			})
			describe := func(i int) string {
				if i < 0 || i >= len(dates) {
					return ""
				}
				return fmt.Sprintf("%s：%.0f 次", dates[i].Format("2006-01-02"), values[i])
			}
			chart.Select(c, area, &a.trendHoverPoint, xs, describe)
			if i := a.trendHoverPoint; i >= 0 && i < len(xs) {
				chart.RuleX(c, xs[i], 0.5).Background(pal.prefsMuted)
				chart.Point(c, xs[i], ys[i], 8, 8).Radius(4).Background(pal.accent).Border(2, pal.prefsCard)
			}
		}).Label("写作趋势图").Fill().GapX(8).GapY(6)
	})
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
		return ui.RGBA(15, 23, 42, 0.055)
	}
	switch {
	case count == 1:
		return pal.accent.Alpha(0.30)
	case count <= 3:
		return pal.accent.Alpha(0.54)
	case count <= 5:
		return pal.accent.Alpha(0.78)
	default:
		return pal.accent
	}
}

var _ = store.DashboardSite{}
