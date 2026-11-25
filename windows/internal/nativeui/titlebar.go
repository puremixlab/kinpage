package nativeui

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const defaultTitleBarHeight float32 = 42

func (a *App) viewTitleBar(c *ui.Context) {
	pal := a.pal
	bar := c.TitleBar()
	height := defaultTitleBarHeight
	if bar.Height > height {
		height = bar.Height
	}
	leftPad := float32(12)
	if bar.Left > 0 {
		leftPad = bar.Left + 10
	}
	rightPad := float32(12)
	if bar.Right > 0 {
		rightPad = bar.Right + 10
	}

	ui.Row(c).Height(height).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Stretch).
		Background(pal.canvas).DragWindow().Children(func() {
		// Left zone: matches 228px sidebar surface (#ececec) seamlessly down into the sidebar.
		leftBg := pal.canvas
		if !a.sidebarCollapsed {
			leftBg = pal.sidebar
		}
		leftZone := ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(6).
			Background(leftBg).Padding(0, 10, 0, leftPad).Shrink(0)
		if !a.sidebarCollapsed && bar.Left == 0 {
			leftZone.Width(228)
		}
		leftZone.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, iconPlanet).Size(20, 20).TextColor(pal.ink)
				ui.Text(c, "Planet").Font(macPlanetFont).FontSize(15).FontWeight(600).TextColor(pal.ink).NoWrap()
			})
			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				sbIcon := lucide.PanelLeftClose()
				sbTitle := "收起侧边栏"
				if a.sidebarCollapsed {
					sbIcon = lucide.PanelLeftOpen()
					sbTitle = "展开侧边栏"
				}
				if action.ActionIcon(c.Key("tb-toggle-sidebar"), sbIcon, action.ActionIconProps{
					Variant: action.ActionIconBorderless, Size: theme.Small, Title: sbTitle,
				}).Clicked() {
					a.sidebarCollapsed = !a.sidebarCollapsed
				}
				if a.mode != viewOverview {
					if macIconButton(c, "tb-toggle-posts", iconSidebar, 24, 14, pal.secondary, "切换文章列表") {
						a.postsCollapsed = !a.postsCollapsed
					}
				}
			})
		})

		if !a.sidebarCollapsed {
			ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(pal.line)
		}

		// Right main top bar: pure white canvas (#ffffff) matching main WebView workspace.
		ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).
			Background(pal.canvas).Padding(0, rightPad, 0, 16).DragWindow().Children(func() {
			// Center-left compact search pill (Ctrl K).
			ui.Row(c).Grow(1).MinWidth(0).Justify(ui.Center).AlignItems(ui.Center).PaddingX(8).Children(func() {
				if interactiveRow(c, "tb-search", "搜索文章", pal.subtle, pal.hover, pal.active).
					Width(220).MinHeight(28).Border(0.5, pal.line).Radius(6).PaddingX(10).PaddingY(3).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Icon(c, iconSearch).Size(13, 13).TextColor(pal.muted)
								ui.Text(c, "搜索文章").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).NoWrap()
							})
							ui.Box(c).PaddingX(4).PaddingY(1).Border(0.5, pal.line).Radius(4).Children(func() {
								ui.Text(c, "Ctrl K").Font(macPlanetFont).FontSize(10).FontWeight(500).TextColor(pal.muted).NoWrap()
							})
						})
					}).Clicked() {
					a.openGlobalSearch()
				}
			})

			// Right zone: restrained context-aware primary actions (padded safely left of bar.Right).
			ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
				a.viewTitleBarActions(c)
			})
		})
	})
	// Only draw divider under the right canvas side when sidebar is open, so sidebar flows seamlessly.
	ui.Row(c).Height(1).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
		if !a.sidebarCollapsed {
			ui.Box(c).Width(228).Height(1).Background(pal.sidebar)
			ui.Box(c).Width(1).Height(1).Background(pal.line)
		}
		if a.mode != viewOverview {
			ui.Box(c).Grow(1).Height(1).Background(pal.line)
		} else {
			ui.Box(c).Grow(1).Height(1).Background(pal.canvas)
		}
	})
}

func (a *App) viewTitleBarActions(c *ui.Context) {
	pal := a.pal

	// Editor mode actions.
	if a.mode == viewSite && a.editing {
		saveStateText := "草稿已保存（未发布）"
		if a.isEditorDirty() {
			saveStateText = "有未保存更改"
		}
		ui.Text(c, saveStateText).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).NoWrap()
		if action.Button(c.Key("editor-save"), "保存草稿", action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
			a.saveCurrentDraft()
		}
		if action.Button(c.Key("editor-publish"), "发布", action.ButtonProps{Variant: action.Primary, Size: theme.Small}).Clicked() {
			a.publishCurrentDraft()
		}
		if a.activeDraftID != "" {
			if ghostIconButton(c, "editor-delete", "删除", "删除草稿", pal) {
				a.deleteCurrentDraft()
			}
		}
		return
	}

	// Site reader mode actions.
	if a.mode == viewSite {
		if site := a.currentSite(); site != nil {
			siteID := site.ID
			if ghostIconButton(c, "site-new-post", "+ 写文章", "+ 写文章", pal) {
				a.guardUnsaved(func() { a.startNewDraft(siteID) })
			}
			if ghostIconButton(c, "site-publish", "发布站点", "发布站点", pal) {
				a.triggerPublishSite(*site)
			}
			if macIconButton(c, "site-settings", iconSettings, 26, 15, pal.secondary, "站点设置") {
				a.openSiteSettings(*site)
			}
			if macIconButton(c, "site-integrations", iconMore, 26, 15, pal.secondary, "扩展") {
				a.openIntegrationsModal(*site)
			}
			if art := a.currentArticle(); art != nil {
				if action.Button(c.Key("reader-edit"), "编辑文章", action.ButtonProps{Variant: action.Primary, Size: theme.Small}).Clicked() {
					a.openArticleForEdit(*art)
				}
			}
		}
		return
	}

	// Subscription reading mode actions.
	if a.mode == viewToday || a.mode == viewUnread || a.mode == viewStarred || a.mode == viewSubscription {
		if a.mode == viewSubscription {
			if sub := a.currentSubscription(); sub != nil {
				subURL := sub.URL
				if ghostIconButton(c, "sub-refresh", "刷新", "刷新", pal) {
					if _, err := a.backend.RefreshSubscription(subURL); err != nil {
						a.setError("刷新订阅失败：" + err.Error())
					} else {
						a.Reload()
						a.setStatus("已刷新订阅")
					}
				}
				if ghostIconButton(c, "sub-mark-all", "全部已读", "全部已读", pal) {
					if err := a.backend.MarkSubscriptionRead(subURL); err != nil {
						a.setError("标记已读失败：" + err.Error())
					} else {
						a.Reload()
						a.setStatus("已全部标记为已读")
					}
				}
				if ghostIconButton(c, "sub-edit", "编辑", "编辑", pal) {
					a.editSubOldURL = sub.URL
					a.editSubNewURL = sub.URL
					a.editSubTitle = sub.CustomTitle
					a.editSubErr = ""
					a.editSubOpen = true
				}
				if ghostIconButton(c, "sub-unfollow", "取消关注", "取消关注", pal) {
					a.unfollowURL = sub.URL
					a.unfollowTitle = subscriptionTitle(*sub)
					a.unfollowOpen = true
				}
			}
		}
		if cur, ok := a.currentSubItem(); ok {
			subURL, key := cur.subURL, cur.effective
			starLabel := "☆ 星标"
			if cur.item.Starred {
				starLabel = "★ 取消星标"
			}
			if ghostIconButton(c, "sub-star", starLabel, starLabel, pal) {
				if err := a.backend.StarSubscription(subURL, key, !cur.item.Starred); err != nil {
					a.setError("更新星标失败：" + err.Error())
				} else {
					a.Reload()
				}
			}
		}
		return
	}

	// Overview (Dashboard) mode actions (cleanly restrained like main WebView).
	if action.Button(c.Key("dash-compose"), "写文章", action.ButtonProps{Variant: action.Primary, Size: theme.Small}).Clicked() {
		a.handleDashboardCompose()
	}
}
