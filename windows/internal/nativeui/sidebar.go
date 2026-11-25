package nativeui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
	"windows/internal/store"
)

func (a *App) viewSidebar(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Fill().MinWidth(220).Background(pal.sidebar).Children(func() {
		// Full-height Column 1 integrated 44px top bar (macOS PlanetSidebarView.toolbar).
		a.viewSidebarHeader(c)

		ui.Scroll(c).TrackScroll(&a.sidebarScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(10).PaddingY(4).Gap(2).Children(func() {
				// 1. Smart Feeds (阅读) — matching macOS PlanetSidebarView.swift:34-76.
				ui.Text(c, "阅读").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
				if sidebarNavRow(c, "nav-today", "今天", a.mode == viewToday, a.totalTodayCount(), func() {
					smartFeedIcon(c, iconSunSolid, pal.orange)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewToday, "") })
				}
				if sidebarNavRow(c, "nav-unread", "未读", a.mode == viewUnread, a.totalUnreadCount(), func() {
					smartFeedIcon(c, iconCircleInsetSolid, pal.accent)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewUnread, "") })
				}
				if sidebarNavRow(c, "nav-starred", "收藏", a.mode == viewStarred, a.totalStarredCount(), func() {
					smartFeedIcon(c, iconStarSolid, pal.yellow)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewStarred, "") })
				}
				if sidebarNavRow(c, "nav-overview", "写作台", a.mode == viewOverview, 0, func() {
					smartFeedIcon(c, iconGrid, pal.secondary)
				}, pal) {
					a.guardUnsaved(func() {
						a.mode = viewOverview
						a.editing = false
						a.clearStatus()
						a.Reload()
					})
				}

				// 2. My Planets (我的站点) — clean section header without inline '+' button.
				ui.Box(c).Height(12)
				ui.Text(c, "我的站点").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
				var activeSites, archivedSites []struct {
					idx  int
					site store.Site
				}
				for i, site := range a.orderedSites() {
					if site.Archived {
						archivedSites = append(archivedSites, struct {
							idx  int
							site store.Site
						}{i, site})
					} else {
						activeSites = append(activeSites, struct {
							idx  int
							site store.Site
						}{i, site})
					}
				}
				if len(activeSites) == 0 {
					ui.Text(c, "暂无站点").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, entry := range activeSites {
					idx, s := entry.idx, entry.site
					siteID, siteName := s.ID, fallbackTitle(s.Name)
					selected := a.mode == viewSite && a.selectedSiteID == siteID
					row := sidebarNavRowElem(c, "site-"+siteID, siteName, selected, 0, func() {
						a.siteAvatar(c, s, idx, 24)
					}, pal).ContextMenu(a.sidebarSiteContextMenu(s))
					if row.Clicked() {
						a.guardUnsaved(func() { a.selectSite(siteID) })
					}
				}
				if len(archivedSites) > 0 {
					ui.Box(c).Height(6)
					toggle := ui.ButtonBase(c.Key("toggle-archived-sites")).Label("已归档站点").
						AlignSelf(ui.Stretch).PaddingX(8).PaddingY(4).Radius(6).Cursor(ui.CursorPointer)
					if toggle.Hovered() {
						toggle.Background(pal.hover)
					}
					toggle.Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
							chev := lucide.ChevronRight()
							if a.archivedExpanded {
								chev = lucide.ChevronDown()
							}
							ui.Icon(c, chev).Size(12, 12).TextColor(pal.muted)
							ui.Text(c, fmt.Sprintf("已归档 (%d)", len(archivedSites))).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted)
						})
					})
					if toggle.Clicked() {
						a.archivedExpanded = !a.archivedExpanded
					}
					if a.archivedExpanded {
						for _, entry := range archivedSites {
							idx, s := entry.idx, entry.site
							siteID, siteName := s.ID, fallbackTitle(s.Name)
							selected := a.mode == viewSite && a.selectedSiteID == siteID
							row := sidebarNavRowElem(c, "site-"+siteID, siteName, selected, 0, func() {
								a.siteAvatar(c, s, idx, 24)
							}, pal).ContextMenu(a.sidebarSiteContextMenu(s))
							if row.Clicked() {
								a.guardUnsaved(func() { a.selectSite(siteID) })
							}
						}
					}
				}

				// 3. Following Planets (关注) — clean section header without inline '+' button.
				ui.Box(c).Height(12)
				ui.Text(c, "关注").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
				if len(a.subscriptions) == 0 {
					ui.Text(c, "暂无关注").Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, sub := range a.orderedSubscriptions() {
					s := sub
					subURL, title := s.URL, subscriptionTitle(s)
					unread := 0
					for _, item := range s.Feed.Items {
						if !item.Read {
							unread++
						}
					}
					selected := a.mode == viewSubscription && a.selectedSubURL == subURL
					row := sidebarNavRowElem(c, "sub-"+subURL, title, selected, unread, func() {
						a.subAvatar(c, s, 24, pal)
					}, pal).ContextMenu(a.sidebarSubContextMenu(s))
					if row.Clicked() {
						a.guardUnsaved(func() { a.openSubscriptionView(viewSubscription, subURL) })
					}
				}
			})
		})

		// macOS PlanetSidebarView bottom bar (44px):
		// Left: IPFS online status + peer count; Right: '+' menu (New Planet / Follow Planet).
		hairlineH(c, pal)
		ui.Row(c).Height(44).Padding(0, 10, 0, 14).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.bottomBg).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			dotColor := pal.green
			ipfsLabel := "IPFS 在线"
			st := a.ipfsModal.status
			switch {
			case st.Disabled:
				dotColor = pal.muted
				ipfsLabel = "IPFS 已停用"
			case st.Running && st.Online:
				dotColor = pal.green
				if st.Peers > 0 {
					ipfsLabel = fmt.Sprintf("IPFS 在线 (%d)", st.Peers)
				}
			case st.Running:
				dotColor = pal.amber
				ipfsLabel = "IPFS 本地"
			default:
				dotColor = pal.danger
				ipfsLabel = "IPFS 离线"
			}

			ipfsBtn := ui.ButtonBase(c.Key("sys-ipfs")).Label("IPFS 节点").MinHeight(30).
				Radius(6).PaddingX(6).Cursor(ui.CursorPointer).Tooltip("查看 IPFS 节点状态").Transition(rowHoverTransition)
			if a.ipfsModal.open {
				ipfsBtn.Background(pal.active)
			} else if ipfsBtn.Hovered() {
				ipfsBtn.Background(pal.hover)
			}
			ipfsBtn.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
					ui.Box(c).Size(10, 10).Radius(5).Background(dotColor).Shrink(0)
					ui.Text(c, ipfsLabel).Font(macPlanetFont).FontSize(12.5).TextColor(pal.ink).NoWrap()
				})
			})
			wasOpen := a.ipfsModal.open
			overlay.Popover(c, ipfsBtn, &a.ipfsModal.open, overlay.PopoverProps{Placement: overlay.Top}, func() {
				a.viewIPFSPopoverContent(c)
			})
			if !wasOpen && a.ipfsModal.open {
				a.refreshIPFSModal()
			}

			toolbarIconBtn(c, "sidebar-add-menu", lucide.Plus(), "新建或关注").Menu(func(m *ui.Menu) {
				if m.Item("新建站点…").Chosen() {
					a.newSiteName = ""
					a.newSiteAbout = ""
					a.createSiteError = ""
					a.createSiteOpen = true
				}
				m.Separator()
				if m.Item("关注站点…").Chosen() {
					a.followAddress = ""
					a.followErr = ""
					a.followModalOpen = true
				}
			})
		})
	})
}
