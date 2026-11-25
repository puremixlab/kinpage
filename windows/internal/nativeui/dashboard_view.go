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
	t := c.Theme()
	ui.Scroll(c).TrackScroll(&a.dashboardScroll).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).Padding(24).Gap(20).AlignSelf(ui.Stretch).Children(func() {
			// Header.
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, "写作总览").Bold().FontSize(22)
					ui.Text(c, "MyGo 原生 GPU 界面 · 直连本地 Store").FontSize(12).TextColor(t.TextMuted)
				})
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c.Key("dash-quick-post"), "快捷发文").Clicked() {
						a.openQuickPost()
					}
					if ui.PrimaryButton(c.Key("dash-compose"), "写文章").Clicked() {
						a.handleDashboardCompose()
					}
				})
			})

			// KPI cards.
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				statCard(c, "站点", fmt.Sprintf("%d", a.dashboard.SiteCount))
				statCard(c, "已发布文章", fmt.Sprintf("%d", a.dashboard.ArticleCount))
				statCard(c, "本地草稿", fmt.Sprintf("%d", a.dashboard.DraftCount))
				statCard(c, "未读订阅", fmt.Sprintf("%d", a.dashboard.UnreadCount))
			})

			if a.dashboard.SiteCount == 0 {
				ui.Column(c).Padding(24).Gap(10).Background(t.Surface).Border(1, t.Border).Radius(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "还没有创建站点").Bold().FontSize(15)
					ui.Text(c, "创建一个站点后即可体验原生 Markdown 编辑器、写作热力图与实时分屏预览。").FontSize(12).TextColor(t.TextMuted)
					if ui.PrimaryButton(c.Key("dash-create-first"), "创建第一个站点").Clicked() {
						a.createSiteOpen = true
					}
				})
				return
			}

			// Writing activity heatmap & trend chart.
			a.viewActivitySection(c)

			// Recent drafts & filtered recent articles.
			ui.Row(c).Gap(16).AlignSelf(ui.Stretch).AlignItems(ui.Start).Children(func() {
				a.viewDashboardDrafts(c)
				a.viewDashboardArticles(c)
			})

			// Sites overview list.
			a.viewDashboardSites(c)
		})
	})
}

func (a *App) viewDashboardDrafts(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).MinWidth(0).Padding(16).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Text(c, "最近草稿").Bold().FontSize(14)
		if len(a.dashboard.RecentDrafts) == 0 {
			ui.Text(c, "暂无本地草稿").FontSize(12).TextColor(t.TextMuted)
		}
		for _, d := range a.dashboard.RecentDrafts {
			siteID, draftID := d.SiteID, d.ID
			label := fmt.Sprintf("%s  ·  %s", fallbackTitle(d.Title), fallbackTitle(d.SiteName))
			if ui.Button(c.Key("dash-draft-"+draftID), label).AlignSelf(ui.Stretch).Clicked() {
				a.openDraftByID(siteID, draftID)
			}
		}
	})
}

func (a *App) viewDashboardArticles(c *ui.Context) {
	t := c.Theme()
	// Reset filter if the selected site no longer exists.
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

	ui.Column(c).Grow(1).MinWidth(0).Padding(16).Gap(8).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "最近文章").Bold().FontSize(14)
		})

		if len(a.dashboard.Sites) > 1 {
			ui.Row(c).Gap(6).Wrap().AlignSelf(ui.Stretch).Children(func() {
				allBtn := ui.Button(c.Key("art-filter-all"), "全部站点")
				if a.articleFilterSiteID == "" {
					allBtn.Background(t.Accent).TextColor(t.AccentText)
				}
				if allBtn.Clicked() {
					a.articleFilterSiteID = ""
				}
				for _, s := range a.dashboard.Sites {
					sid := s.ID
					btn := ui.Button(c.Key("art-filter-"+sid), fallbackTitle(s.Name))
					if a.articleFilterSiteID == sid {
						btn.Background(t.Accent).TextColor(t.AccentText)
					}
					if btn.Clicked() {
						a.articleFilterSiteID = sid
					}
				}
			})
		}

		shown := 0
		for _, art := range a.dashboard.RecentArticles {
			if a.articleFilterSiteID != "" && art.SiteID != a.articleFilterSiteID {
				continue
			}
			shown++
			siteID, artID := art.SiteID, art.ID
			label := fmt.Sprintf("%s  ·  %s", fallbackTitle(art.Title), fallbackTitle(art.SiteName))
			if ui.Button(c.Key("dash-art-"+artID), label).AlignSelf(ui.Stretch).Clicked() {
				a.openArticleForRead(siteID, artID)
			}
		}
		if shown == 0 {
			ui.Text(c, "暂无匹配的已发布文章").FontSize(12).TextColor(t.TextMuted)
		}
	})
}

func (a *App) viewDashboardSites(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(16).Gap(10).Background(t.Surface).Border(1, t.Border).Radius(8).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "站点状态").Bold().FontSize(14)
		for _, s := range a.dashboard.Sites {
			site := s
			statusLabel := "未发布"
			if site.PublishPending {
				statusLabel = "有未发布改动"
			} else if site.HasPublished {
				statusLabel = "已发布"
			}
			ui.Row(c.Key("dash-site-row-"+site.ID)).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(10).PaddingY(8).Background(t.Background).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, fallbackTitle(site.Name)).Bold().FontSize(13)
					ui.Text(c, fmt.Sprintf("%d 篇文章  ·  %d 篇草稿  ·  %s", site.ArticleCount, site.DraftCount, statusLabel)).FontSize(11).TextColor(t.TextMuted)
				})
				ui.Row(c).Gap(6).Shrink(0).Children(func() {
					if ui.Button(c.Key("dash-site-open-"+site.ID), "进入").Clicked() {
						a.selectSite(site.ID)
					}
					if ui.Button(c.Key("dash-site-write-"+site.ID), "写文章").Clicked() {
						a.startNewDraft(site.ID)
					}
				})
			})
		}
	})
}

func (a *App) viewActivitySection(c *ui.Context) {
	t := c.Theme()
	counts := make(map[string]int, len(a.activity))
	totalActions := 0
	for _, item := range a.activity {
		n := item.Articles + item.Drafts
		counts[item.Date] = n
		totalActions += n
	}

	ui.Column(c).Padding(16).Gap(14).Background(t.Surface).Border(1, t.Border).Radius(8).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "写作活跃度").Bold().FontSize(14)
				ui.Text(c, fmt.Sprintf("累计 %d 次写作记录", totalActions)).FontSize(12).TextColor(t.TextMuted)
			})
			ui.Segmented(c.Key("activity-mode"), &a.activityMode, "按日", "按周", "累计").Label("热力图粒度")
		})

		a.viewHeatmapGrid(c, counts)

		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "写作趋势").Bold().FontSize(13)
			ui.Segmented(c.Key("trend-range"), &a.trendRange, "30 天", "90 天", "1 年").Label("趋势图时间范围")
		})

		a.viewActivityTrendChart(c, counts)
	})
}

func (a *App) viewHeatmapGrid(c *ui.Context, counts map[string]int) {
	t := c.Theme()
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
					h = min(float32(56), float32(14+sum*7))
				}
				tip := fmt.Sprintf("第 %d 周：%d 次", w+1, sum)
				ui.Box(c.Key(fmt.Sprintf("hm-week-%d", w))).Size(18, h).Radius(4).Background(heatmapColor(sum, t.Dark)).Tooltip(tip)
			}
		})
	case activityModeCumulative:
		running := 0
		ui.Row(c).Gap(4).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-cw-%d", w))).Gap(4).Children(func() {
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
						ui.Box(c.Key("hm-c-"+key)).Size(14, 14).Radius(3).Background(heatmapColor(level, t.Dark)).
							Tooltip(fmt.Sprintf("%s：累计 %d 次", key, running))
					}
				})
			}
		})
	default:
		ui.Row(c).Gap(4).Children(func() {
			for w := range weeks {
				ui.Column(c.Key(fmt.Sprintf("hm-w-%d", w))).Gap(4).Children(func() {
					for d := range 7 {
						day := startDay.AddDate(0, 0, w*7+d)
						key := day.Format("2006-01-02")
						n := counts[key]
						ui.Box(c.Key("hm-"+key)).Size(14, 14).Radius(3).Background(heatmapColor(n, t.Dark)).
							Tooltip(fmt.Sprintf("%s：%d 次", key, n))
					}
				})
			}
		})
	}
}

func (a *App) viewActivityTrendChart(c *ui.Context, counts map[string]int) {
	t := c.Theme()
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
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
	}

	ui.Column(c).AlignSelf(ui.Stretch).Height(160).Children(func() {
		chart.Plot(c.Key("trend-plot"), chart.Axes{Left: yTicks, Bottom: x.Ticks(5), Label: labelFn}, func(area ui.Element) {
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range yTicks {
					py := chart.Y(r, tick.At)
					p.Line(r.X, py, r.X+r.W, py, 1, t.Border)
				}
				p.FillPath(chart.Area(r, xs, nil, ys, chart.CurveMonotone), t.Accent.Alpha(0.16))
				p.StrokePath(chart.Line(r, xs, ys, chart.CurveMonotone), 2, t.Accent)
			})
			describe := func(i int) string {
				if i < 0 || i >= len(dates) {
					return ""
				}
				return fmt.Sprintf("%s：%.0f 次", dates[i].Format("2006-01-02"), values[i])
			}
			chart.Select(c, area, &a.trendHoverPoint, xs, describe)
			if i := a.trendHoverPoint; i >= 0 && i < len(xs) {
				chart.RuleX(c, xs[i], 1).Background(t.TextMuted)
				chart.Point(c, xs[i], ys[i], 8, 8).Radius(4).Background(t.Accent).Border(2, t.Background)
			}
		}).Label("写作趋势图").Fill().GapX(8).GapY(6)
	})
}

func (a *App) viewComposePickerModal(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.composePickerOpen, func() {
		ui.Column(c).Width(360).Gap(12).Children(func() {
			ui.Text(c, "选择要写作的站点").Bold().FontSize(16)
			ui.Text(c, "当前有多个站点，请选择新文章所属的站点：").FontSize(12).TextColor(t.TextMuted)
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

func statCard(c *ui.Context, title, value string) {
	t := c.Theme()
	ui.Column(c.Key("stat-"+title)).Grow(1).MinWidth(0).Padding(14).Gap(4).Background(t.Surface).Border(1, t.Border).Radius(8).Children(func() {
		ui.Text(c, title).FontSize(12).TextColor(t.TextMuted)
		ui.Text(c, value).Bold().FontSize(22)
	})
}

func heatmapColor(count int, dark bool) ui.Color {
	if count <= 0 {
		if dark {
			return ui.Hex("#27272a")
		}
		return ui.Hex("#ebedf0")
	}
	switch {
	case count == 1:
		return ui.Hex("#9be9a8")
	case count <= 3:
		return ui.Hex("#40c463")
	case count <= 5:
		return ui.Hex("#30a14e")
	default:
		return ui.Hex("#216e39")
	}
}

// Keep store import used even if DashboardSite helper expands later.
var _ = store.DashboardSite{}
