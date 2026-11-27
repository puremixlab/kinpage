package nativeui

import (
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
				// 0. Pinned top-level primary entry: Writing Desk (写作台).
				navIconColor := func(selected bool) ui.Color {
					if selected {
						return pal.ink
					}
					return pal.secondary
				}
				if sidebarNavRow(c, "nav-overview", a.t("nav.overview"), a.mode == viewOverview, 0, func() {
					smartFeedIcon(c, iconGrid, navIconColor(a.mode == viewOverview))
				}, pal) {
					a.guardUnsaved(func() {
						a.mode = viewOverview
						a.editing = false
						a.clearStatus()
						a.Reload()
					})
				}
				ui.Box(c).Height(6)
				ui.Box(c).PaddingX(6).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })
				ui.Box(c).Height(6)

				// 1. Smart Feeds (阅读) — calm monochrome icons; color reserved for site avatars.
				ui.Text(c, a.t("nav.reading")).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
				if sidebarNavRow(c, "nav-today", a.t("nav.today"), a.mode == viewToday, a.totalTodayCount(), func() {
					smartFeedIcon(c, iconSunSolid, navIconColor(a.mode == viewToday))
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewToday, "") })
				}
				if sidebarNavRow(c, "nav-unread", a.t("nav.unread"), a.mode == viewUnread, a.totalUnreadCount(), func() {
					smartFeedIcon(c, iconCircleInsetSolid, navIconColor(a.mode == viewUnread))
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewUnread, "") })
				}
				if sidebarNavRow(c, "nav-starred", a.t("nav.starred"), a.mode == viewStarred, a.totalStarredCount(), func() {
					smartFeedIcon(c, iconStarSolid, navIconColor(a.mode == viewStarred))
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewStarred, "") })
				}

				// 2. My Planets (我的站点) — clean section header without inline '+' button.
				ui.Box(c).Height(12)
				ui.Text(c, a.t("nav.mySites")).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
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
					ui.Text(c, a.t("nav.noSites")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, entry := range activeSites {
					idx, s := entry.idx, entry.site
					siteID, siteName := s.ID, a.fallbackTitle(s.Name)
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
					toggle := ui.ButtonBase(c.Key("toggle-archived-sites")).Label(a.t("nav.archivedSites")).
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
							ui.Text(c, a.tf("nav.archivedCount", len(archivedSites))).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted)
						})
					})
					if toggle.Clicked() {
						a.archivedExpanded = !a.archivedExpanded
					}
					if a.archivedExpanded {
						for _, entry := range archivedSites {
							idx, s := entry.idx, entry.site
							siteID, siteName := s.ID, a.fallbackTitle(s.Name)
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
				ui.Text(c, a.t("nav.following")).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).PaddingX(8).PaddingY(3)
				if len(a.subscriptions) == 0 {
					ui.Text(c, a.t("nav.noFollowing")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
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

		// Sidebar bottom bar (44px):
		// Left: Settings popover anchored cleanly above the button; Right: IPFS status popover.
		hairlineH(c, pal)
		ui.Row(c).Height(44).Padding(0, 10, 0, 10).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.bottomBg).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			settingsLabel := a.t("nav.settings")
			settingsBtn := ui.ButtonBase(c.Key("sidebar-settings-menu")).Label(settingsLabel).MinHeight(30).
				Radius(6).PaddingX(7).Cursor(ui.CursorPointer).Tooltip(settingsLabel).Transition(rowHoverTransition)
			if a.settingsMenuOpen {
				settingsBtn.Background(pal.active)
			} else if settingsBtn.Hovered() {
				settingsBtn.Background(pal.hover)
			}
			settingsBtn.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Icon(c, lucide.Settings()).Size(14, 14).TextColor(pal.secondary).Shrink(0)
					ui.Text(c, settingsLabel).Font(macPlanetFont).FontSize(12.5).TextColor(pal.ink).NoWrap()
				})
			})
			if settingsBtn.Clicked() {
				a.settingsMenuOpen = !a.settingsMenuOpen
			}
			ui.PopoverBase(c, settingsBtn, &a.settingsMenuOpen, func(panel ui.Element) {
				panel.AttachTo(settingsBtn, ui.AnchorTopLeft, ui.AnchorBottomLeft).Margin(0, 0, 8, 0).
					Padding(6).Radius(10).Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).
					Shadow(0, 10, 28, 0, pal.shadowColor).
					Children(func() {
						a.viewSettingsPopoverContent(c)
					})
			})

			dotColor := pal.green
			ipfsLabel := a.t("ipfs.online")
			st := a.ipfsModal.status
			switch {
			case st.Disabled:
				dotColor = pal.muted
				ipfsLabel = a.t("ipfs.disabled")
			case st.Running && st.Online:
				dotColor = pal.green
				if st.Peers > 0 {
					ipfsLabel = a.tf("ipfs.onlinePeers", st.Peers)
				}
			case st.Running:
				dotColor = pal.amber
				ipfsLabel = a.t("ipfs.local")
			default:
				dotColor = pal.danger
				ipfsLabel = a.t("ipfs.offline")
			}

			ipfsBtn := ui.ButtonBase(c.Key("sys-ipfs")).Label(a.t("ipfs.node")).MinHeight(30).
				Radius(6).PaddingX(6).Cursor(ui.CursorPointer).Tooltip(a.t("ipfs.viewStatus")).Transition(rowHoverTransition)
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
		})
	})
}

// viewSettingsPopoverContent renders the ChatGPT-style compact Settings popover
// anchored cleanly above the bottom-left Settings button without covering it.
func (a *App) viewSettingsPopoverContent(c *ui.Context) {
	pal := a.pal
	popRow := func(key string, icon *ui.SVG, label, shortcut string, onClick func()) {
		row := interactiveRow(c, key, label, ui.Transparent, pal.hover, pal.active).
			MinHeight(30).Radius(6).Padding(4, 10, 4, 10).
			Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Grow(1).MinWidth(0).Children(func() {
					if icon != nil {
						ui.Icon(c, icon).Size(14, 14).TextColor(pal.secondary).Shrink(0)
					}
					ui.Text(c, label).Font(macPlanetFont).FontSize(12.5).TextColor(pal.ink).Grow(1).NoWrap()
					if shortcut != "" {
						ui.Text(c, shortcut).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).NoWrap().Shrink(0)
					}
				})
			})
		if row.Clicked() {
			a.settingsMenuOpen = false
			onClick()
		}
	}

	ui.Column(c).Width(244).Padding(4).Gap(3).Children(func() {
		popRow("sm-new-site", lucide.Plus(), a.t("nav.newSiteEllipsis"), "", func() {
			a.newSiteName = ""
			a.newSiteAbout = ""
			a.createSiteError = ""
			a.createSiteOpen = true
		})
		popRow("sm-follow-site", lucide.Rss(), a.t("nav.followSiteEllipsis"), "", func() {
			a.followAddress = ""
			a.followErr = ""
			a.followModalOpen = true
		})

		ui.Box(c).PaddingY(2).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })

		popRow("sm-quick-post", lucide.MessageSquarePlus(), a.t("menu.quickPost"), "Ctrl+D", func() {
			a.openQuickPost()
		})
		popRow("sm-open-ipfs", lucide.Globe(), a.t("menu.openIPFS"), "Ctrl+O", func() {
			a.openIPFSResourceModal()
		})

		ui.Box(c).PaddingY(2).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })

		// Inline language quick switcher.
		ui.Column(c).Padding(4, 8, 4, 8).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Icon(c, lucide.Globe()).Size(13, 13).TextColor(pal.muted).Shrink(0)
				ui.Text(c, a.t("menu.language")).Font(macPlanetFont).FontSize(11.5).FontWeight(500).TextColor(pal.muted)
			})
			locales := SupportedLocales()
			labels := make([]string, len(locales))
			curIdx := 0
			for i, loc := range locales {
				labels[i] = LocaleDisplayName(loc)
				if a.Locale() == loc {
					curIdx = i
				}
			}
			prevIdx := curIdx
			macSegmented(c, "sm-locale-seg", &curIdx, labels, pal)
			if curIdx != prevIdx && curIdx >= 0 && curIdx < len(locales) {
				chosen := locales[curIdx]
				a.SetLocale(chosen)
				a.setStatus(a.tf("status.languageSwitched", LocaleDisplayName(chosen)))
			}
		})

		ui.Box(c).PaddingY(2).AlignSelf(ui.Stretch).Children(func() { hairlineH(c, pal) })

		popRow("sm-more-settings", lucide.Settings(), a.t("menu.moreSettings"), "Ctrl+,", func() {
			a.openToolsModal(toolsTabGeneral)
		})
	})
}
