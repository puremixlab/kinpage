package nativeui

import (
	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
)

func (a *App) viewSidebar(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Width(228).Shrink(0).AlignSelf(ui.Stretch).Background(pal.sidebar).Children(func() {
		// Full-height Column 1 integrated 44px top bar (macOS PlanetSidebarView.toolbar).
		a.viewSidebarHeader(c)

		ui.Column(c).PaddingX(10).PaddingY(4).Gap(2).Shrink(0).Children(func() {
			if sidebarNavRow(c, "nav-overview", "写作台", a.mode == viewOverview, 0, func() {
				ui.Icon(c, iconGrid).Size(16, 16).TextColor(pal.ink).Shrink(0)
			}, pal) {
				a.guardUnsaved(func() {
					a.mode = viewOverview
					a.editing = false
					a.clearStatus()
					a.Reload()
				})
			}
		})

		ui.Scroll(c).TrackScroll(&a.sidebarScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(10).PaddingY(4).Gap(2).Children(func() {
				// 1. Smart Feeds (阅读) first — matching macOS PlanetSidebarView.swift:34-76.
				ui.Text(c, "阅读").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(2)
				if sidebarNavRow(c, "nav-today", "今天", a.mode == viewToday, a.totalTodayCount(), func() {
					ui.Icon(c, iconClock).Size(16, 16).TextColor(pal.orange).Shrink(0)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewToday, "") })
				}
				if sidebarNavRow(c, "nav-unread", "未读", a.mode == viewUnread, a.totalUnreadCount(), func() {
					ui.Icon(c, iconInbox).Size(16, 16).TextColor(pal.accent).Shrink(0)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewUnread, "") })
				}
				if sidebarNavRow(c, "nav-starred", "收藏", a.mode == viewStarred, a.totalStarredCount(), func() {
					ui.Icon(c, iconStar).Size(16, 16).TextColor(pal.yellow).Shrink(0)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewStarred, "") })
				}

				// 2. My Planets (我的站点) — matching macOS PlanetSidebarView.swift:78-91.
				ui.Box(c).Height(12)
				ui.Row(c).MinHeight(24).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(8).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "我的站点").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted)
					if macIconButton(c, "nav-new-site", iconPlus, 24, 14, pal.muted, "+ 站点") {
						a.newSiteName = ""
						a.newSiteAbout = ""
						a.createSiteError = ""
						a.createSiteOpen = true
					}
				})
				if len(a.sites) == 0 {
					ui.Text(c, "暂无站点").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for i, site := range a.sites {
					idx, s := i, site
					siteID, siteName := s.ID, fallbackTitle(s.Name)
					selected := a.mode == viewSite && a.selectedSiteID == siteID
					if sidebarNavRow(c, "site-"+siteID, siteName, selected, 0, func() {
						a.siteAvatar(c, s, idx, 22)
					}, pal) {
						a.guardUnsaved(func() { a.selectSite(siteID) })
					}
				}

				// 3. Following Planets (订阅) — matching macOS PlanetSidebarView.swift:93-106.
				ui.Box(c).Height(12)
				ui.Row(c).MinHeight(24).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(8).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "订阅").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted)
					if macIconButton(c, "nav-follow", iconPlus, 24, 14, pal.muted, "+ 关注") {
						a.followAddress = ""
						a.followErr = ""
						a.followModalOpen = true
					}
				})
				if len(a.subscriptions) == 0 {
					ui.Text(c, "暂无订阅").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, sub := range a.subscriptions {
					s := sub
					subURL, title := s.URL, subscriptionTitle(s)
					unread := 0
					for _, item := range s.Feed.Items {
						if !item.Read {
							unread++
						}
					}
					selected := a.mode == viewSubscription && a.selectedSubURL == subURL
					if sidebarNavRow(c, "sub-"+subURL, title, selected, unread, func() {
						a.subAvatar(c, s, 22, pal)
					}, pal) {
						a.guardUnsaved(func() { a.openSubscriptionView(viewSubscription, subURL) })
					}
				}
			})
		})

		// Single-row 44px bottom status bar matching main windows/index.html:
		// Settings menu on the left, IPFS node status on the right.
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
		ui.Row(c).Height(44).PaddingX(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.bottomBg).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			settingsAnchor := ui.ButtonBase(c.Key("sys-settings-menu")).Label("设置").MinHeight(30).
				Radius(6).PaddingX(8).Cursor(ui.CursorPointer).Transition(rowHoverTransition)
			if a.settingsMenuOpen {
				settingsAnchor.Background(pal.active)
			} else if settingsAnchor.Hovered() {
				settingsAnchor.Background(pal.hover)
			}
			settingsAnchor.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
					ui.Text(c, "设置").Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).NoWrap()
					ui.Icon(c, iconSettings).Size(15, 15).TextColor(pal.secondary)
				})
			})
			overlay.Popover(c, settingsAnchor, &a.settingsMenuOpen, overlay.PopoverProps{Placement: overlay.Top}, func() {
				ui.Column(c).Width(200).Gap(2).Children(func() {
					menuItem := func(key, label string, onClick func()) {
						if interactiveRow(c, key, label, ui.Transparent, pal.hover, pal.active).
							MinHeight(30).Radius(6).PaddingX(8).PaddingY(5).
							Children(func() {
								ui.Row(c).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, label).Font(macPlanetFont).FontSize(13).TextColor(pal.ink).NoWrap()
								})
							}).Clicked() {
							a.settingsMenuOpen = false
							onClick()
						}
					}
					menuItem("menu-qp", "⚡ 快捷发文", func() { a.openQuickPost() })
					menuItem("menu-ai", "AI 助手", func() { a.openToolsModal(toolsTabAI) })
					menuItem("menu-api", "本地 API", func() { a.openToolsModal(toolsTabAPI) })
					menuItem("menu-keys", "密钥管理", func() { a.openToolsModal(toolsTabKeys) })
					menuItem("menu-logs", "应用日志", func() { a.openToolsModal(toolsTabLogs) })
					hairlineH(c, pal)
					menuItem("menu-offline", "离线存储", func() { a.openOfflineModal() })
					menuItem("menu-folders", "已发布文件夹", func() { a.openFoldersModal() })
					menuItem("menu-trash", "回收站", func() { a.openRemovedSitesModal() })
					menuItem("menu-refresh", "刷新数据", func() {
						a.Reload()
						a.setStatus("已刷新数据")
					})
				})
			})

			dotColor := pal.green
			if a.ipfsModal.status.Disabled {
				dotColor = pal.muted
			}
			ipfsBtn := ui.ButtonBase(c.Key("sys-ipfs")).Label("IPFS 节点").MinHeight(30).
				Radius(6).PaddingX(6).Cursor(ui.CursorPointer).Transition(rowHoverTransition)
			if ipfsBtn.Hovered() {
				ipfsBtn.Background(pal.hover)
			}
			ipfsBtn.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Size(9, 9).Radius(4.5).Background(dotColor)
					ui.Text(c, "IPFS 节点").Font(macPlanetFont).FontSize(13).TextColor(pal.ink).NoWrap()
				})
			})
			if ipfsBtn.Clicked() {
				a.openIPFSModal()
			}
		})
	})
}
