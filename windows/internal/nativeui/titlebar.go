package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
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
	return 10
}

// viewSidebarHeader renders the 44px top bar integrated seamlessly into Column 1 (Sidebar):
// Brand icon + Sidebar toggle + Sparkles AI on left.
func (a *App) viewSidebarHeader(c *ui.Context) {
	pal := a.pal
	h := a.topBarHeight(c)
	leftPad := a.topBarLeftPad(c)

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).
		Background(pal.sidebar).Padding(0, 8, 0, leftPad).DragWindow().Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Box(c).Size(24, 24).Center().Shrink(0).Children(func() {
				ui.Icon(c, iconPlanet).Size(18, 18).TextColor(pal.ink)
			})
			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				if toolbarIconBtn(c, "tb-toggle-sidebar", lucide.PanelLeftClose(), a.t("nav.collapseSidebar")).Clicked() {
					a.sidebarCollapsed = true
				}
				if toolbarIconBtn(c, "tb-ai-sparkles", lucide.Sparkles(), a.t("nav.aiAssistant")).Clicked() {
					a.openToolsModal(toolsTabAI)
				}
			})
		})
	})
}

// viewListPaneHeader renders the 44px top bar integrated into Column 2 (ArticleListView),
// matching macOS navigationTitle + navigationSubtitle + filter menu icon.
func (a *App) viewListPaneHeader(c *ui.Context, title, subtitle string, isSubMode bool) {
	pal := a.pal
	h := a.topBarHeight(c)
	leftPad := float32(14)
	if a.sidebarCollapsed {
		leftPad = a.topBarLeftPad(c)
	}

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).
		Background(pal.canvas).Padding(0, 10, 0, leftPad).DragWindow().Children(func() {
		if a.sidebarCollapsed {
			if toolbarIconBtn(c, "tb-expand-sidebar", lucide.PanelLeftOpen(), a.t("nav.expandSidebar")).Clicked() {
				a.sidebarCollapsed = false
			}
		}
		ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
			ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink).MaxLines(1).Ellipsis("…")
			if subtitle != "" {
				ui.Text(c, subtitle).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).MaxLines(1).Ellipsis("…")
			}
		})

		filterColor := pal.secondary
		if a.listFilter != listFilterAll {
			filterColor = pal.accent
		}
		toolbarIconBtnColored(c, "list-filter-menu", lucide.ListFilter(), filterColor, a.t("filter.tooltip"), pal).Menu(func(m *ui.Menu) {
			if m.Item(a.t("common.all")).Radio(a.listFilter == listFilterAll && a.starTypeFilter < 0).Chosen() {
				a.listFilter = listFilterAll
				a.starTypeFilter = -1
			}
			if isSubMode {
				if m.Item(a.t("filter.unread")).Radio(a.listFilter == listFilterUnread).Chosen() {
					a.listFilter = listFilterUnread
					a.starTypeFilter = -1
				}
				if m.Item(a.t("filter.starred")).Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
			} else {
				if m.Item(a.t("filter.pages")).Radio(a.listFilter == listFilterPages).Chosen() {
					a.listFilter = listFilterPages
					a.starTypeFilter = -1
				}
				if m.Item(a.t("filter.nav")).Radio(a.listFilter == listFilterNav).Chosen() {
					a.listFilter = listFilterNav
					a.starTypeFilter = -1
				}
				m.Separator()
				if m.Item(a.t("filter.pinned")).Radio(a.listFilter == listFilterPinned).Chosen() {
					a.listFilter = listFilterPinned
					a.starTypeFilter = -1
				}
				if m.Item(a.t("filter.starred")).Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
			}
			m.Submenu(a.t("star.byType"), func(sm *ui.Menu) {
				if sm.Item(a.t("star.all")).Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
				sm.Separator()
				for _, opt := range allStarTypes {
					if sm.Item(a.t(opt.key)).Radio(a.listFilter == listFilterStarred && a.starTypeFilter == opt.id).Chosen() {
						a.listFilter = listFilterStarred
						a.starTypeFilter = opt.id
					}
				}
			})
			m.Separator()
			searchLabel := a.t("filter.showSearch")
			if a.listSearchOpen {
				searchLabel = a.t("filter.hideSearch")
			}
			if m.Item(searchLabel).Shortcut(ui.Ctrl, ui.KeyF).Chosen() {
				a.listSearchOpen = !a.listSearchOpen
				if !a.listSearchOpen {
					a.postSearchQuery = ""
					a.subSearchQuery = ""
				}
			}
		})
	})
	hairlineH(c, pal)
}

// viewDetailToolbar renders the 44px icon-only top bar integrated into Column 3
// (ArticleView / Editor / Dashboard), matching macOS ArticleView.toolbar.
func (a *App) viewDetailToolbar(c *ui.Context, showBottomBorder bool) {
	pal := a.pal
	h := a.topBarHeight(c)
	rightPad := a.topBarRightPad(c)
	leftPad := float32(10)
	if a.sidebarCollapsed && (a.mode == viewOverview || a.postsCollapsed) {
		leftPad = a.topBarLeftPad(c)
	}

	ui.Row(c).Height(h).Shrink(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).
		Background(pal.canvas).Padding(0, rightPad, 0, leftPad).DragWindow().Children(func() {
		// Leading group: macOS toolbarPlanetView (always accessible in site/subscription mode).
		ui.Row(c).AlignItems(ui.Center).Gap(2).Shrink(0).Children(func() {
			if a.sidebarCollapsed && (a.mode == viewOverview || a.postsCollapsed) {
				if toolbarIconBtn(c, "tb-expand-sidebar", lucide.PanelLeftOpen(), a.t("nav.expandSidebar")).Clicked() {
					a.sidebarCollapsed = false
				}
			}
			if a.mode != viewOverview {
				if toolbarIconBtn(c, "tb-toggle-posts", lucide.PanelLeft(), a.t("nav.togglePosts")).Clicked() {
					a.postsCollapsed = !a.postsCollapsed
				}
			}
			if a.mode == viewSite {
				if site := a.currentSite(); site != nil {
					siteID := site.ID
					ui.Box(c).Width(1).Height(16).MarginX(4).Background(pal.line)
					if toolbarIconBtn(c, "site-new-post", lucide.Plus(), a.t("toolbar.writePost")).Clicked() {
						a.guardUnsaved(func() { a.startNewDraft(siteID) })
					}
					if toolbarIconBtn(c, "tb-quick-post", lucide.MessageSquarePlus(), a.t("toolbar.quickPost")).Clicked() {
						a.openQuickPost()
					}
					if toolbarIconBtn(c, "site-settings", lucide.Info(), a.t("toolbar.siteSettings")).Clicked() {
						a.openSiteSettings(*site)
					}
					if !a.editing {
						if toolbarIconBtn(c, "site-publish", lucide.Upload(), a.t("toolbar.publishSite")).Clicked() {
							a.triggerPublishSite(*site)
						}
					}
				}
			}
			if a.mode == viewSubscription {
				if sub := a.currentSubscription(); sub != nil {
					subURL := sub.URL
					ui.Box(c).Width(1).Height(16).MarginX(4).Background(pal.line)
					if toolbarIconBtn(c, "sub-refresh", lucide.RefreshCw(), a.t("toolbar.refreshSub")).Clicked() {
						if _, err := a.backend.RefreshSubscription(subURL); err != nil {
							a.setError(a.tf("status.refreshSubErr", err.Error()))
						} else {
							a.Reload()
							a.setStatus(a.t("status.refreshedSub"))
						}
					}
					toolbarIconBtn(c, "sub-more", lucide.Info(), a.t("toolbar.subActions")).Menu(func(m *ui.Menu) {
						if m.Item(a.t("menu.subMarkAllRead")).Chosen() {
							if err := a.backend.MarkSubscriptionRead(subURL); err != nil {
								a.setError(a.tf("status.markReadErr", err.Error()))
							} else {
								a.Reload()
								a.setStatus(a.t("status.markedAllRead"))
							}
						}
						if m.Item(a.t("menu.subEdit")).Chosen() {
							a.editSubOldURL = sub.URL
							a.editSubNewURL = sub.URL
							a.editSubTitle = sub.CustomTitle
							a.editSubErr = ""
							a.editSubOpen = true
						}
						m.Separator()
						if m.Item(a.t("menu.subUnfollow")).Danger().Chosen() {
							a.unfollowURL = sub.URL
							a.unfollowTitle = subscriptionTitle(*sub)
							a.unfollowOpen = true
						}
					})
				}
			}
		})

		// Flexible draggable middle area with subtle context title (site or subscription name).
		ui.Row(c).Grow(1).MinWidth(12).Height(h).AlignItems(ui.Center).PaddingX(8).DragWindow().Children(func() {
			ctxTitle := ""
			if a.mode == viewSite {
				if site := a.currentSite(); site != nil {
					ctxTitle = a.fallbackTitle(site.Name)
				}
			} else if cur, ok := a.currentSubItem(); ok {
				ctxTitle = cur.subTitle
			}
			if ctxTitle != "" {
				ui.Text(c, ctxTitle).Font(macPlanetFont).FontSize(13).FontWeight(500).
					TextColor(pal.secondary).MaxLines(1).Ellipsis("…")
			}
		})

		// Trailing group: document/mode actions + global search & share.
		ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
			a.viewDetailToolbarActions(c)
			if a.mode != viewOverview {
				ui.Box(c).Width(1).Height(16).MarginX(2).Background(pal.line)
				if toolbarIconBtn(c, "tb-search", lucide.Search(), a.t("toolbar.searchPosts")).Clicked() {
					a.openGlobalSearch()
				}
			}
		})
	})

	if showBottomBorder {
		hairlineH(c, pal)
	}
}

func (a *App) viewDetailToolbarActions(c *ui.Context) {
	pal := a.pal

	if a.mode == viewSite {
		if a.editing {
			if a.isEditorDirty() {
				ui.Text(c, a.t("toolbar.unsaved")).Font(macPlanetFont).FontSize(11).TextColor(pal.amber).NoWrap().PaddingX(4)
			}
			if toolbarIconBtn(c, "editor-save", lucide.Check(), a.t("toolbar.saveDraft")).Clicked() {
				a.saveCurrentDraft()
			}
			if a.activeDraftID != "" {
				if toolbarIconBtn(c, "editor-delete", lucide.Trash2(), a.t("toolbar.deleteDraft")).Clicked() {
					a.deleteCurrentDraft()
				}
			}
			if macPrimaryButtonSmall(c, "editor-publish", a.t("common.publish"), pal) {
				a.publishCurrentDraft()
			}
			return
		}

		if art := a.currentArticle(); art != nil {
			if toolbarIconBtn(c, "reader-edit", lucide.PencilLine(), a.t("toolbar.editArticle")).Clicked() {
				a.openArticleForEdit(*art)
			}
			speakLabel := a.t("toolbar.speak")
			if a.speaker != nil && a.speaker.Speaking() {
				speakLabel = a.t("toolbar.stopSpeak")
			}
			pinLabel := a.t("toolbar.pin")
			if art.Pinned != nil {
				pinLabel = a.t("toolbar.unpin")
			}
			toolbarIconBtn(c, "reader-more", lucide.Ellipsis(), a.t("toolbar.moreArticleActions")).Menu(func(m *ui.Menu) {
				if m.Item(pinLabel).Chosen() {
					a.toggleArticlePinned(*art)
				}
				if m.Item(a.t("toolbar.articleSettings")).Chosen() {
					a.openArticleSettings(*art)
				}
				if m.Item(a.t("toolbar.relatedArticles")).Chosen() {
					a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
				}
				if m.Item(speakLabel).Chosen() {
					a.toggleSpeak(art.Title, art.Content)
				}
				m.Separator()
				if m.Item(a.t("menu.artDelete")).Chosen() {
					a.deleteArticleID = art.ID
					a.deleteArticleTitle = art.Title
					a.deleteArticleOpen = true
				}
			})
		}
		return
	}

	if a.mode == viewToday || a.mode == viewUnread || a.mode == viewStarred || a.mode == viewSubscription {
		if cur, ok := a.currentSubItem(); ok {
			subURL, key := cur.subURL, cur.effective
			starIcon := lucide.Star()
			starColor := pal.secondary
			starLabel := a.t("toolbar.star")
			if cur.item.Starred {
				starIcon = iconStarSolid
				starColor = pal.yellow
				starLabel = a.t("toolbar.unstar")
			}
			if toolbarIconBtnColored(c, "sub-star", starIcon, starColor, starLabel, pal).Clicked() {
				if err := a.backend.StarSubscription(subURL, key, !cur.item.Starred); err != nil {
					a.setError(a.tf("status.starErr", err.Error()))
				} else {
					a.Reload()
				}
			}

			readLabel := a.t("menu.subMarkUnread")
			readColor := pal.secondary
			if !cur.item.Read {
				readLabel = a.t("menu.subMarkRead")
				readColor = pal.accent
			}
			if toolbarIconBtnColored(c, "sub-toggle-read", lucide.CircleDot(), readColor, readLabel, pal).Clicked() {
				if err := a.backend.SetSubscriptionRead(subURL, key, !cur.item.Read); err != nil {
					a.setError(a.tf("status.readStateErr", err.Error()))
				} else {
					a.Reload()
				}
			}

			body := cur.item.Content
			if body == "" {
				body = cur.item.Description
			}
			mdBody := htmlToReadableMarkdown(body)
			speakIcon := lucide.Volume2()
			speakLabel := a.t("toolbar.speak")
			if a.speaker != nil && a.speaker.Speaking() {
				speakIcon = lucide.VolumeX()
				speakLabel = a.t("toolbar.stopSpeak")
			}
			if toolbarIconBtn(c, "sub-speak", speakIcon, speakLabel).Clicked() {
				a.toggleSpeak(cur.item.Title, mdBody)
			}

			if toolbarIconBtn(c, "sub-related", lucide.FileText(), a.t("toolbar.relatedArticles")).Clicked() {
				a.openRelatedArticles("", "", subURL, key, cur.item.Title)
			}

			if link := strings.TrimSpace(cur.item.Link); link != "" {
				webIcon := lucide.Globe()
				webLabel := a.t("toolbar.viewOriginalWeb")
				webColor := pal.secondary
				if a.subShowWebView {
					webIcon = lucide.FileText()
					webLabel = a.t("toolbar.switchReaderMode")
					webColor = pal.accent
				}
				if toolbarIconBtnColored(c, "sub-toggle-web", webIcon, webColor, webLabel, pal).Clicked() {
					if a.newWebView != nil {
						a.subShowWebView = !a.subShowWebView
						if !a.subShowWebView {
							a.destroySubWebView()
						}
					} else if a.openExternal != nil {
						if err := a.openExternal(link); err != nil {
							a.setError(a.tf("status.openWebErr", err.Error()))
						}
					}
				}
			}
		}
		return
	}

	// Overview (Dashboard) mode actions: clean macOS icon buttons matching detail toolbar.
	if toolbarIconBtn(c, "tb-search", lucide.Search(), a.t("toolbar.searchPosts")).Clicked() {
		a.openGlobalSearch()
	}
	if toolbarIconBtn(c, "dash-compose", lucide.SquarePen(), a.t("toolbar.writePost")).Clicked() {
		a.handleDashboardCompose()
	}
}
