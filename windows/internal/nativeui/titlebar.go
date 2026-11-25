package nativeui

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
)

const macTopBarHeight float32 = 44

func (a *App) topBarHeight(c *ui.Context) float32 {
	bar := c.TitleBar()
	if bar.Height > macTopBarHeight {
		return bar.Height
	}
	return macTopBarHeight
}

func (a *App) topBarRightPad(c *ui.Context) float32 {
	bar := c.TitleBar()
	if bar.Right > 0 {
		return bar.Right + 10
	}
	return 14
}

func (a *App) topBarLeftPad(c *ui.Context) float32 {
	bar := c.TitleBar()
	if bar.Left > 0 {
		return bar.Left + 10
	}
	return 12
}

// viewSidebarHeader renders the 44px top bar integrated seamlessly into Column 1 (Sidebar),
// matching macOS PlanetSidebarView.toolbar (Brand + Sidebar toggle + Sparkles AI).
func (a *App) viewSidebarHeader(c *ui.Context) {
	pal := a.pal
	h := a.topBarHeight(c)
	leftPad := a.topBarLeftPad(c)

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).
		Background(pal.sidebar).Padding(0, 8, 0, leftPad).DragWindow().Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			ui.Icon(c, iconPlanet).Size(20, 20).TextColor(pal.ink)
			ui.Text(c, "Planet").Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).NoWrap()
		})
		ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
			if action.ActionIcon(c.Key("tb-toggle-sidebar"), lucide.PanelLeftClose(), action.ActionIconProps{
				Variant: action.ActionIconBorderless, Size: theme.Small, Title: "收起侧边栏",
			}).Clicked() {
				a.sidebarCollapsed = true
			}
			if action.ActionIcon(c.Key("tb-ai-sparkles"), lucide.Sparkles(), action.ActionIconProps{
				Variant: action.ActionIconBorderless, Size: theme.Small, Title: "AI 助手",
			}).Clicked() {
				a.openToolsModal(toolsTabAI)
			}
		})
	})
}

// viewListPaneHeader renders the 44px top bar integrated into Column 2 (ArticleListView),
// matching macOS navigationTitle + navigationSubtitle with zero duplicate headers.
func (a *App) viewListPaneHeader(c *ui.Context, title, subtitle string) {
	pal := a.pal
	h := a.topBarHeight(c)
	leftPad := float32(14)
	if a.sidebarCollapsed {
		leftPad = a.topBarLeftPad(c)
	}

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).
		Background(pal.canvas).Padding(0, 12, 0, leftPad).DragWindow().Children(func() {
		if a.sidebarCollapsed {
			if action.ActionIcon(c.Key("tb-expand-sidebar"), lucide.PanelLeftOpen(), action.ActionIconProps{
				Variant: action.ActionIconBorderless, Size: theme.Small, Title: "展开侧边栏",
			}).Clicked() {
				a.sidebarCollapsed = false
			}
		}
		ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink).NoWrap()
			if subtitle != "" {
				ui.Text(c, subtitle).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).NoWrap()
			}
		})
	})
	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
}

// viewDetailToolbar renders the 44px top bar integrated into Column 3 (ArticleView / Editor / Dashboard),
// matching macOS ArticleView.toolbar: leading planet actions, draggable spacer, trailing document actions + search.
func (a *App) viewDetailToolbar(c *ui.Context, showBottomBorder bool) {
	pal := a.pal
	h := a.topBarHeight(c)
	rightPad := a.topBarRightPad(c)
	leftPad := float32(12)
	if a.sidebarCollapsed && (a.mode == viewOverview || a.postsCollapsed) {
		leftPad = a.topBarLeftPad(c)
	}

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).
		Background(pal.canvas).Padding(0, rightPad, 0, leftPad).DragWindow().Children(func() {
		// Leading group: macOS toolbarPlanetView (always accessible in site mode, even when editing).
		ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
			if a.sidebarCollapsed && (a.mode == viewOverview || a.postsCollapsed) {
				if action.ActionIcon(c.Key("tb-expand-sidebar"), lucide.PanelLeftOpen(), action.ActionIconProps{
					Variant: action.ActionIconBorderless, Size: theme.Small, Title: "展开侧边栏",
				}).Clicked() {
					a.sidebarCollapsed = false
				}
			}
			if a.mode != viewOverview {
				if macIconButton(c, "tb-toggle-posts", iconSidebar, 28, 15, pal.secondary, "切换文章列表") {
					a.postsCollapsed = !a.postsCollapsed
				}
			}
			if a.mode == viewSite {
				if site := a.currentSite(); site != nil {
					siteID := site.ID
					ui.Box(c).Width(1).Height(16).Background(pal.line)
					if macIconButton(c, "site-new-post", iconEdit, 28, 15, pal.secondary, "+ 写文章") {
						a.guardUnsaved(func() { a.startNewDraft(siteID) })
					}
					if action.ActionIcon(c.Key("tb-quick-post"), lucide.Zap(), action.ActionIconProps{
						Variant: action.ActionIconBorderless, Size: theme.Small, Title: "⚡ 快捷发文",
					}).Clicked() {
						a.openQuickPost()
					}
					if macIconButton(c, "site-settings", iconSettings, 28, 15, pal.secondary, "站点设置") {
						a.openSiteSettings(*site)
					}
					if macIconButton(c, "site-integrations", iconMore, 28, 15, pal.secondary, "扩展") {
						a.openIntegrationsModal(*site)
					}
				}
			}
		})

		// Flexible draggable middle spacer.
		ui.Box(c).Grow(1).MinWidth(12).Height(h).DragWindow()

		// Trailing group: document/mode actions + global search button.
		ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
			a.viewDetailToolbarActions(c)
			if a.mode != viewOverview {
				ui.Box(c).Width(1).Height(16).Background(pal.line)
				if macIconButton(c, "tb-search", iconSearch, 28, 15, pal.secondary, "搜索文章") {
					a.openGlobalSearch()
				}
			}
		})
	})

	if showBottomBorder {
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
	}
}

func (a *App) viewDetailToolbarActions(c *ui.Context) {
	pal := a.pal

	if a.mode == viewSite {
		if a.editing {
			if a.isEditorDirty() {
				display.Tag(c, "未保存", display.TagProps{Color: display.TagWarning, Size: theme.Small})
			} else {
				display.Tag(c, "已保存本地", display.TagProps{Size: theme.Small})
			}
			if action.Button(c.Key("editor-save"), "保存草稿", action.ButtonProps{Variant: action.Default, Size: theme.Small}).Clicked() {
				a.saveCurrentDraft()
			}
			if macPrimaryButtonSmall(c, "editor-publish", "发布", pal) {
				a.publishCurrentDraft()
			}
			if a.activeDraftID != "" {
				if ghostIconButton(c, "editor-delete", "删除", "删除草稿", pal) {
					a.deleteCurrentDraft()
				}
			}
			return
		}

		if site := a.currentSite(); site != nil {
			if ghostIconButton(c, "site-publish", "发布站点", "发布站点", pal) {
				a.triggerPublishSite(*site)
			}
			if art := a.currentArticle(); art != nil {
				if macPrimaryButtonSmall(c, "reader-edit", "编辑文章", pal) {
					a.openArticleForEdit(*art)
				}
			}
		}
		return
	}

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

	// Overview (Dashboard) mode actions: compact search pill + Mac Planet blue 写文章 button.
	if interactiveRow(c, "tb-search", "搜索文章", pal.subtle, pal.hover, pal.active).
		Width(208).MinHeight(28).Border(0.5, pal.line).Radius(6).PaddingX(10).PaddingY(3).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Grow(1).MinWidth(0).Children(func() {
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
	if macPrimaryButtonSmall(c, "dash-compose", "写文章", pal) {
		a.handleDashboardCompose()
	}
}
