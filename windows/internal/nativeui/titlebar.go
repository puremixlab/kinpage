package nativeui

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const defaultTitleBarHeight float32 = 44

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
		// Left zone: macOS PlanetSidebarView top bar (228px #ececec seamlessly flowing into sidebar).
		leftBg := pal.canvas
		if !a.sidebarCollapsed {
			leftBg = pal.sidebar
		}
		leftZone := ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(4).
			Background(leftBg).Padding(0, 8, 0, leftPad).Shrink(0)
		if !a.sidebarCollapsed && bar.Left == 0 {
			leftZone.Width(228)
		}
		leftZone.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
				ui.Icon(c, iconPlanet).Size(20, 20).TextColor(pal.ink)
				ui.Text(c, "Planet").Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).NoWrap()
			})
			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				// macOS PlanetSidebarView toolbar: Sidebar toggle + Sparkles AI Chat button.
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
				if action.ActionIcon(c.Key("tb-ai-sparkles"), lucide.Sparkles(), action.ActionIconProps{
					Variant: action.ActionIconBorderless, Size: theme.Small, Title: "AI 助手",
				}).Clicked() {
					a.openToolsModal(toolsTabAI)
				}
			})
		})

		if !a.sidebarCollapsed {
			ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(pal.line)
		}

		// Right main workspace top bar: pure white canvas (#ffffff) matching macOS ArticleView toolbar.
		ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).
			Background(pal.canvas).Padding(0, rightPad, 0, 12).DragWindow().Children(func() {
			// Leading: list-pane toggle + current workspace context label.
			ui.Row(c).AlignItems(ui.Center).Gap(8).Shrink(0).Children(func() {
				if a.mode != viewOverview {
					if macIconButton(c, "tb-toggle-posts", iconSidebar, 26, 15, pal.secondary, "切换文章列表") {
						a.postsCollapsed = !a.postsCollapsed
					}
					if ctxTitle := a.titleBarContextTitle(); ctxTitle != "" {
						ui.Text(c, ctxTitle).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.secondary).NoWrap()
					}
				}
			})

			// Center: compact global search pill using lobe-mygo display.Hotkey.
			ui.Row(c).Grow(1).MinWidth(0).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
				if interactiveRow(c, "tb-search", "搜索文章", pal.subtle, pal.hover, pal.active).
					Width(224).MinHeight(28).Border(0.5, pal.line).Radius(6).PaddingX(10).PaddingY(3).
					Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Icon(c, iconSearch).Size(13, 13).TextColor(pal.muted)
								ui.Text(c, "搜索文章").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).NoWrap()
							})
							display.Hotkey(c, "mod+k", display.HotkeyProps{
								Variant: display.HotkeyOutlined,
								Size:    theme.Small,
							})
						})
					}).Clicked() {
					a.openGlobalSearch()
				}
			})

			// Trailing: context-aware primary actions (padded safely left of bar.Right).
			ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
				a.viewTitleBarActions(c)
			})
		})
	})

	// Seamless divider: sidebar flows uninterrupted; workspace gets a crisp 1px hairline when not on Overview.
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

func (a *App) titleBarContextTitle() string {
	switch a.mode {
	case viewSite:
		if a.editing {
			return "撰写文章"
		}
		if site := a.currentSite(); site != nil {
			return fallbackTitle(site.Name)
		}
	case viewToday:
		return "今天"
	case viewUnread:
		return "未读"
	case viewStarred:
		return "收藏"
	case viewSubscription:
		if sub := a.currentSubscription(); sub != nil {
			return subscriptionTitle(*sub)
		}
	}
	return ""
}

func (a *App) viewTitleBarActions(c *ui.Context) {
	pal := a.pal

	// Site workspace (both editor & reader): keep site-level quick icons accessible
	// even when the middle posts list is collapsed, while keeping document actions on the right.
	if a.mode == viewSite {
		site := a.currentSite()
		if site != nil && a.postsCollapsed {
			siteID := site.ID
			if macIconButton(c, "site-new-post", iconEdit, 26, 15, pal.secondary, "+ 写文章") {
				a.guardUnsaved(func() { a.startNewDraft(siteID) })
			}
			if macIconButton(c, "site-settings", iconSettings, 26, 15, pal.secondary, "站点设置") {
				a.openSiteSettings(*site)
			}
			if macIconButton(c, "site-integrations", iconMore, 26, 15, pal.secondary, "扩展") {
				a.openIntegrationsModal(*site)
			}
			ui.Box(c).Width(1).Height(16).Background(pal.line)
		}

		if a.editing {
			if a.isEditorDirty() {
				display.Tag(c, "未保存", display.TagProps{Color: display.TagWarning, Size: theme.Small})
			} else {
				display.Tag(c, "已保存本地", display.TagProps{Size: theme.Small})
			}
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

		if site != nil {
			if ghostIconButton(c, "site-publish", "发布站点", "发布站点", pal) {
				a.triggerPublishSite(*site)
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
