package nativeui

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
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

func (a *App) viewDashboard(c *ui.Context) {
	pal := a.pal
	ui.Scroll(c).TrackScroll(&a.dashboardScroll).Grow(1).MinHeight(0).Background(pal.canvas).Children(func() {
		ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(32).PaddingY(24).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(1160).Gap(20).Children(func() {
				// Top 52px header matching workspace-dashboard.png & mac-planet.css:896-905.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).MinHeight(52).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(3).Children(func() {
						ui.Text(c, "写作台").Font(macPlanetFont).FontSize(22).FontWeight(700).LetterSpacing(-0.015).TextColor(pal.ink)
						ui.Text(c, "继续上次的思路，也开始新的记录。").Font(macPlanetFont).FontSize(12).TextColor(pal.secondary)
					})
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						if ui.Button(c.Key("dash-search"), "").Label("搜索文章").Width(212).MinHeight(34).
							Background(pal.canvas).Border(0.5, pal.controlLine).Radius(6).PaddingX(14).PaddingY(6).
							Children(func() {
								ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, "搜索文章").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
									ui.Icon(c, iconSearch).Size(15, 15).TextColor(pal.muted)
								})
							}).Clicked() {
							a.openGlobalSearch()
						}
						if macPrimaryButton(c, "dash-compose", "写文章", pal) {
							a.handleDashboardCompose()
						}
					})
				})

				// Lightweight horizontal metrics bar (0.5px hairline).
				ui.Column(c).AlignSelf(ui.Stretch).Gap(10).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Row(c).Gap(24).AlignItems(ui.Center).Children(func() {
							metricInline(c, fmt.Sprintf("%d", a.dashboard.SiteCount), "个站点", pal)
							metricInline(c, fmt.Sprintf("%d", a.dashboard.ArticleCount), "篇文章", pal)
							metricInline(c, fmt.Sprintf("%d", a.dashboard.DraftCount), "篇草稿", pal)
							if a.dashboard.UnreadCount > 0 {
								metricInline(c, fmt.Sprintf("%d", a.dashboard.UnreadCount), "条未读", pal)
							}
						})
						ui.Text(c, "所有站点").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
					})
					hairlineH(c, pal)
				})

				if a.dashboard.SiteCount == 0 {
					ui.Column(c).Padding(36).Gap(12).Background(pal.subtle).Border(0.5, pal.line).Radius(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "还没有创建站点").Font(macPlanetFont).FontSize(16).FontWeight(600).TextColor(pal.ink)
						ui.Text(c, "创建一个站点后即可开始本地写作、管理草稿与发布到 IPFS。").Font(macPlanetFont).FontSize(13).TextColor(pal.secondary)
						if macPrimaryButton(c, "dash-create-first", "创建第一个站点", pal) {
							a.createSiteOpen = true
						}
					})
					return
				}

				// Two-column Mac Planet dashboard: left main (1fr) + right sites column (244px).
				ui.Row(c).Gap(28).AlignSelf(ui.Stretch).AlignItems(ui.Start).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Gap(24).Children(func() {
						a.viewDashboardDrafts(c)
						a.viewDashboardArticles(c)
						a.viewActivitySection(c)
					})
					ui.Column(c).Width(244).Shrink(0).Gap(18).Children(func() {
						a.viewDashboardSites(c)
						a.viewDashboardNoteCard(c)
					})
				})
			})
		})
	})
}

func metricInline(c *ui.Context, num, unit string, pal macPalette) {
	ui.Row(c).Gap(4).AlignItems(ui.Baseline).Children(func() {
		ui.Text(c, num).Bold().FontSize(14).TextColor(pal.ink)
		ui.Text(c, unit).FontSize(12).TextColor(pal.secondary)
	})
}

func (a *App) viewDashboardDrafts(c *ui.Context) {
	pal := a.pal
	ui.Column(c).AlignSelf(ui.Stretch).Gap(6).Children(func() {
		ui.Text(c, "继续写作").Bold().FontSize(15).TextColor(pal.ink)
		if len(a.dashboard.RecentDrafts) == 0 {
			ui.Text(c, "暂无本地草稿，点击右上角“写文章”开始新记录。").FontSize(12).TextColor(pal.muted).PaddingY(12)
			return
		}
		for _, d := range a.dashboard.RecentDrafts {
			siteID, draftID := d.SiteID, d.ID
			title := fallbackTitle(d.Title)
			siteName := fallbackTitle(d.SiteName)
			label := fmt.Sprintf("%s  ·  %s", title, siteName)
			if ui.Button(c.Key("dash-draft-"+draftID), "").Label(label).AlignSelf(ui.Stretch).MinHeight(58).
				Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(12).PaddingY(10).
				Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						ui.Icon(c, iconEdit).Size(17, 17).TextColor(pal.secondary).Shrink(0)
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, title).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).NoWrap()
							ui.Text(c, siteName+"   |   已保存在本地").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).NoWrap()
						})
						ui.Text(c, formatShortDate(d.Modified)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted).Shrink(0).NoWrap()
					})
				}).Clicked() {
				a.openDraftByID(siteID, draftID)
			}
			hairlineH(c, pal)
		}
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

	ui.Column(c).AlignSelf(ui.Stretch).Gap(6).Children(func() {
		ui.Text(c, "最近文章").Bold().FontSize(15).TextColor(pal.ink)

		// Underline tab filter bar matching workspace-dashboard.png.
		if len(a.dashboard.Sites) > 1 {
			ui.Column(c).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).Gap(4).Wrap().AlignSelf(ui.Stretch).Children(func() {
					a.dashboardFilterTab(c, "art-filter-all", "所有站点", a.articleFilterSiteID == "", func() {
						a.articleFilterSiteID = ""
					})
					for _, s := range a.dashboard.Sites {
						sid := s.ID
						a.dashboardFilterTab(c, "art-filter-"+sid, fallbackTitle(s.Name), a.articleFilterSiteID == sid, func() {
							a.articleFilterSiteID = sid
						})
					}
				})
				ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
			})
		}

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
			if ui.Button(c.Key("dash-art-"+artID), "").Label(label).AlignSelf(ui.Stretch).MinHeight(54).
				Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(12).PaddingY(10).
				Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						ui.Icon(c, iconFile).Size(17, 17).TextColor(pal.muted).Shrink(0)
						ui.Text(c, title).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).Grow(1).NoWrap()
						ui.Text(c, siteName).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).Shrink(0).NoWrap()
						ui.Text(c, formatShortDate(art.Modified)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted).Width(64).Shrink(0).NoWrap()
					})
				}).Clicked() {
				a.openArticleForRead(siteID, artID)
			}
			hairlineH(c, pal)
		}
		if shown == 0 {
			ui.Text(c, "暂无匹配的已发布文章").FontSize(12).TextColor(pal.muted).PaddingY(12)
		}
	})
}

func (a *App) dashboardFilterTab(c *ui.Context, key, label string, active bool, onClick func()) {
	pal := a.pal
	color := pal.secondary
	if active {
		color = pal.accent
	}
	ui.Column(c.Key(key)).Children(func() {
		if ui.Button(c.Key(key+"-btn"), "").Label(label).
			Background(ui.Color{}).Border(0, ui.Color{}).Radius(4).PaddingX(10).PaddingY(6).
			Children(func() {
				txt := ui.Text(c, label).FontSize(12).TextColor(color).NoWrap()
				if active {
					txt.Bold()
				}
			}).Clicked() {
			onClick()
		}
		barColor := ui.Color{}
		if active {
			barColor = pal.accent
		}
		ui.Box(c).Height(2).AlignSelf(ui.Stretch).Background(barColor)
	})
}

func (a *App) viewDashboardSites(c *ui.Context) {
	pal := a.pal
	ui.Column(c).AlignSelf(ui.Stretch).Gap(4).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "站点状态").Font(macPlanetFont).FontSize(15).FontWeight(600).TextColor(pal.ink)
			if macIconButton(c, "dash-sites-add", iconPlus, 28, 15, pal.muted, "新建站点") {
				a.newSiteName = ""
				a.newSiteAbout = ""
				a.createSiteError = ""
				a.createSiteOpen = true
			}
		})
		for i, s := range a.dashboard.Sites {
			idx, site := i, s
			statusLabel := "尚未发布"
			statusColor := pal.muted
			if site.PublishPending {
				statusLabel = "有未发布改动"
				statusColor = pal.amber
			} else if site.HasPublished {
				statusLabel = "已发布"
				statusColor = pal.green
			}
			var fullSite store.Site
			fullSite.Name = site.Name
			for _, candidate := range a.sites {
				if candidate.ID == site.ID {
					fullSite = candidate
					break
				}
			}
			if ui.Button(c.Key("dash-site-row-"+site.ID), "").Label(fallbackTitle(site.Name)).AlignSelf(ui.Stretch).
				Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(10).
				Children(func() {
					ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(8).AlignSelf(ui.Stretch).Children(func() {
							a.siteAvatar(c, fullSite, idx, 28)
							ui.Text(c, fallbackTitle(site.Name)).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).Grow(1).NoWrap()
							ui.Icon(c, iconArrow).Size(14, 14).TextColor(pal.muted)
						})
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Box(c).Size(5, 5).Radius(2.5).Background(statusColor)
							ui.Text(c, statusLabel).Font(macPlanetFont).FontSize(12).TextColor(statusColor)
						})
						ui.Text(c, fmt.Sprintf("%d 篇文章，%d 篇草稿", site.ArticleCount, site.DraftCount)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
					})
				}).Clicked() {
				a.selectSite(site.ID)
			}
			hairlineH(c, pal)
		}
	})
}

func (a *App) viewDashboardNoteCard(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Padding(16).Gap(6).Background(pal.subtle).Border(0.5, pal.line).Radius(10).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "写好之后，再分享出去").Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink)
		ui.Text(c, "私密草稿只保存在本地。保存并发布后，站点内容才会更新。").Font(macPlanetFont).FontSize(12).TextColor(pal.secondary)
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

	ui.Column(c).Padding(16).Gap(14).Background(pal.canvas).Border(0.5, pal.line).Radius(8).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "写作活跃度").Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink)
				ui.Text(c, fmt.Sprintf("累计 %d 次写作记录", totalActions)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
			})
			macSegmented(c, "activity-mode", &a.activityMode, []string{"按日", "按周", "累计"}, pal)
		})

		a.viewHeatmapGrid(c, counts)

		hairlineH(c, pal)

		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "写作趋势").Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink)
			macSegmented(c, "trend-range", &a.trendRange, []string{"30 天", "90 天", "1 年"}, pal)
		})

		a.viewActivityTrendChart(c, counts)
	})
}

func (a *App) viewHeatmapGrid(c *ui.Context, counts map[string]int) {
	pal := a.pal
	const weeks = 16
	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	endOffset := int(time.Saturday - today.Weekday())
	endDay := today.AddDate(0, 0, endOffset)
	startDay := endDay.AddDate(0, 0, -(weeks*7 - 1))

	switch a.activityMode {
	case activityModeWeekly:
		ui.Row(c).Gap(6).AlignItems(ui.End).Children(func() {
			for w := range weeks {
				sum := 0
				for d := range 7 {
					day := startDay.AddDate(0, 0, w*7+d)
					sum += counts[day.Format("2006-01-02")]
				}
				h := float32(10)
				if sum > 0 {
					h = min(float32(52), float32(14+sum*7))
				}
				tip := fmt.Sprintf("第 %d 周：%d 次", w+1, sum)
				ui.Box(c.Key(fmt.Sprintf("hm-week-%d", w))).Size(18, h).Radius(4).Background(heatmapColor(sum, pal)).Tooltip(tip)
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
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(pal.muted)
	}

	ui.Column(c).AlignSelf(ui.Stretch).Height(148).Children(func() {
		chart.Plot(c.Key("trend-plot"), chart.Axes{Left: yTicks, Bottom: x.Ticks(5), Label: labelFn}, func(area ui.Element) {
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range yTicks {
					py := chart.Y(r, tick.At)
					p.Line(r.X, py, r.X+r.W, py, 1, pal.line)
				}
				p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), pal.accent.Alpha(0.14))
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
				chart.RuleX(c, xs[i], 1).Background(pal.muted)
				chart.Point(c, xs[i], ys[i], 8, 8).Radius(4).Background(pal.accent).Border(2, pal.canvas)
			}
		}).Label("写作趋势图").Fill().GapX(8).GapY(6)
	})
}

func (a *App) viewComposePickerModal(c *ui.Context) {
	pal := a.pal
	ui.Modal(c, &a.composePickerOpen, func() {
		ui.Column(c).Width(360).Gap(12).Children(func() {
			ui.Text(c, "选择要写作的站点").Bold().FontSize(16).TextColor(pal.ink)
			ui.Text(c, "当前有多个站点，请选择新文章所属的站点：").FontSize(12).TextColor(pal.secondary)
			ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
				for _, s := range a.sites {
					siteID := s.ID
					if ui.Button(c.Key("compose-pick-"+siteID), fallbackTitle(s.Name)).AlignSelf(ui.Stretch).Clicked() {
						a.composePickerOpen = false
						a.startNewDraft(siteID)
					}
				}
			})
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("compose-pick-cancel"), "取消").Clicked() {
					a.composePickerOpen = false
				}
			})
		})
	})
}

func heatmapColor(count int, pal macPalette) ui.Color {
	if count <= 0 {
		return pal.selection
	}
	switch {
	case count == 1:
		return pal.accent.Alpha(0.28)
	case count <= 3:
		return pal.accent.Alpha(0.52)
	case count <= 5:
		return pal.accent.Alpha(0.76)
	default:
		return pal.accent
	}
}

var _ = store.DashboardSite{}
