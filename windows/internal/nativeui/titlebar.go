package nativeui

import (
	"context"
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

// viewSidebarHeader renders the 44px top bar integrated seamlessly into Column 1 (Sidebar),
// matching macOS PlanetSidebarView.toolbar: Sidebar toggle + Sparkles AI on left, '...' tools menu on right.
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
				if toolbarIconBtn(c, "tb-toggle-sidebar", lucide.PanelLeftClose(), "收起侧边栏").Clicked() {
					a.sidebarCollapsed = true
				}
				if toolbarIconBtn(c, "tb-ai-sparkles", lucide.Sparkles(), "AI 助手").Clicked() {
					a.openToolsModal(toolsTabAI)
				}
			})
		})

		toolbarIconBtn(c, "tb-sidebar-more", lucide.Ellipsis(), "更多工具").Menu(func(m *ui.Menu) {
			if m.Item("快捷发文").Shortcut(ui.Ctrl, ui.KeyD).Chosen() {
				a.openQuickPost()
			}
			if m.Item("打开 IPFS 资源…").Shortcut(ui.Ctrl, ui.KeyO).Chosen() {
				a.openIPFSResourceModal()
			}
			if m.Item("打赏创作者 (ETH)…").Chosen() {
				a.openTipModal("")
			}
			m.Separator()
			if m.Item("已发布文件夹").Chosen() {
				a.openFoldersModal()
			}
			if m.Item("离线存储").Chosen() {
				a.openOfflineModal()
			}
			if m.Item("回收站").Chosen() {
				a.openRemovedSitesModal()
			}
			m.Separator()
			if m.Item("本地 API").Chosen() {
				a.openToolsModal(toolsTabAPI)
			}
			if m.Item("密钥管理").Chosen() {
				a.openToolsModal(toolsTabKeys)
			}
			if m.Item("应用日志").Chosen() {
				a.openToolsModal(toolsTabLogs)
			}
			m.Separator()
			preventSleep := a.backend != nil && a.backend.PreventSleep()
			if m.Item("运行期间防止系统休眠").Radio(preventSleep).Chosen() {
				if a.backend != nil {
					if err := a.backend.SetPreventSleep(!preventSleep); err != nil {
						a.setError("设置防休眠失败：" + err.Error())
					} else if !preventSleep {
						a.setStatus("已开启防休眠")
					} else {
						a.setStatus("已关闭防休眠")
					}
				}
			}
			if m.Item("迁移数据仓库目录…").Chosen() {
				if a.backend != nil {
					if dst, err := a.backend.MoveDataDir(context.Background()); err != nil {
						a.setError("迁移数据目录失败：" + err.Error())
					} else if dst != "" {
						a.setStatus("已迁移至 " + dst + "，重启 Planet 后生效")
					}
				}
			}
			if m.Item("刷新数据").Chosen() {
				a.Reload()
				a.setStatus("已刷新数据")
			}
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
			if toolbarIconBtn(c, "tb-expand-sidebar", lucide.PanelLeftOpen(), "展开侧边栏").Clicked() {
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
		toolbarIconBtnColored(c, "list-filter-menu", lucide.ListFilter(), filterColor, "筛选文章", pal).Menu(func(m *ui.Menu) {
			if m.Item("全部").Radio(a.listFilter == listFilterAll && a.starTypeFilter < 0).Chosen() {
				a.listFilter = listFilterAll
				a.starTypeFilter = -1
			}
			if isSubMode {
				if m.Item("未读").Radio(a.listFilter == listFilterUnread).Chosen() {
					a.listFilter = listFilterUnread
					a.starTypeFilter = -1
				}
				if m.Item("已收藏").Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
			} else {
				if m.Item("独立页面").Radio(a.listFilter == listFilterPages).Chosen() {
					a.listFilter = listFilterPages
					a.starTypeFilter = -1
				}
				if m.Item("导航项").Radio(a.listFilter == listFilterNav).Chosen() {
					a.listFilter = listFilterNav
					a.starTypeFilter = -1
				}
				m.Separator()
				if m.Item("已置顶").Radio(a.listFilter == listFilterPinned).Chosen() {
					a.listFilter = listFilterPinned
					a.starTypeFilter = -1
				}
				if m.Item("已收藏").Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
			}
			m.Submenu("按星标类型", func(sm *ui.Menu) {
				if sm.Item("全部星标").Radio(a.listFilter == listFilterStarred && a.starTypeFilter < 0).Chosen() {
					a.listFilter = listFilterStarred
					a.starTypeFilter = -1
				}
				sm.Separator()
				for _, opt := range allStarTypes {
					if sm.Item(opt.label).Radio(a.listFilter == listFilterStarred && a.starTypeFilter == opt.id).Chosen() {
						a.listFilter = listFilterStarred
						a.starTypeFilter = opt.id
					}
				}
			})
			m.Separator()
			searchLabel := "在列表中筛选…"
			if a.listSearchOpen {
				searchLabel = "隐藏列表筛选栏"
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
				if toolbarIconBtn(c, "tb-expand-sidebar", lucide.PanelLeftOpen(), "展开侧边栏").Clicked() {
					a.sidebarCollapsed = false
				}
			}
			if a.mode != viewOverview {
				if toolbarIconBtn(c, "tb-toggle-posts", lucide.PanelLeft(), "切换文章列表").Clicked() {
					a.postsCollapsed = !a.postsCollapsed
				}
			}
			if a.mode == viewSite {
				if site := a.currentSite(); site != nil {
					siteID := site.ID
					ui.Box(c).Width(1).Height(16).MarginX(4).Background(pal.line)
					if toolbarIconBtn(c, "site-new-post", lucide.Plus(), "写文章").Clicked() {
						a.guardUnsaved(func() { a.startNewDraft(siteID) })
					}
					if toolbarIconBtn(c, "tb-quick-post", lucide.MessageSquarePlus(), "快捷发文").Clicked() {
						a.openQuickPost()
					}
					if toolbarIconBtn(c, "site-settings", lucide.Info(), "站点设置").Clicked() {
						a.openSiteSettings(*site)
					}
					if !a.editing {
						if toolbarIconBtn(c, "site-publish", lucide.Upload(), "发布站点").Clicked() {
							a.triggerPublishSite(*site)
						}
					}
				}
			}
			if a.mode == viewSubscription {
				if sub := a.currentSubscription(); sub != nil {
					subURL := sub.URL
					ui.Box(c).Width(1).Height(16).MarginX(4).Background(pal.line)
					if toolbarIconBtn(c, "sub-refresh", lucide.RefreshCw(), "刷新订阅").Clicked() {
						if _, err := a.backend.RefreshSubscription(subURL); err != nil {
							a.setError("刷新订阅失败：" + err.Error())
						} else {
							a.Reload()
							a.setStatus("已刷新订阅")
						}
					}
					toolbarIconBtn(c, "sub-more", lucide.Info(), "订阅操作").Menu(func(m *ui.Menu) {
						if m.Item("全部标为已读").Chosen() {
							if err := a.backend.MarkSubscriptionRead(subURL); err != nil {
								a.setError("标记已读失败：" + err.Error())
							} else {
								a.Reload()
								a.setStatus("已全部标记为已读")
							}
						}
						if m.Item("编辑订阅…").Chosen() {
							a.editSubOldURL = sub.URL
							a.editSubNewURL = sub.URL
							a.editSubTitle = sub.CustomTitle
							a.editSubErr = ""
							a.editSubOpen = true
						}
						m.Separator()
						if m.Item("取消关注").Danger().Chosen() {
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
					ctxTitle = fallbackTitle(site.Name)
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
				if toolbarIconBtn(c, "tb-search", lucide.Search(), "搜索文章").Clicked() {
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
				ui.Text(c, "未保存").Font(macPlanetFont).FontSize(11).TextColor(pal.amber).NoWrap().PaddingX(4)
			}
			if toolbarIconBtn(c, "editor-save", lucide.Check(), "保存草稿").Clicked() {
				a.saveCurrentDraft()
			}
			if a.activeDraftID != "" {
				if toolbarIconBtn(c, "editor-delete", lucide.Trash2(), "删除草稿").Clicked() {
					a.deleteCurrentDraft()
				}
			}
			if macPrimaryButtonSmall(c, "editor-publish", "发布", pal) {
				a.publishCurrentDraft()
			}
			return
		}

		if art := a.currentArticle(); art != nil {
			if toolbarIconBtn(c, "reader-edit", lucide.PencilLine(), "编辑文章").Clicked() {
				a.openArticleForEdit(*art)
			}
			speakLabel := "朗读"
			if a.speaker != nil && a.speaker.Speaking() {
				speakLabel = "停止朗读"
			}
			pinLabel := "置顶"
			if art.Pinned != nil {
				pinLabel = "取消置顶"
			}
			toolbarIconBtn(c, "reader-more", lucide.Ellipsis(), "更多文章操作").Menu(func(m *ui.Menu) {
				if m.Item(pinLabel).Chosen() {
					a.toggleArticlePinned(*art)
				}
				if m.Item("文章设置").Chosen() {
					a.openArticleSettings(*art)
				}
				if m.Item("相关文章").Chosen() {
					a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
				}
				if m.Item(speakLabel).Chosen() {
					a.toggleSpeak(art.Title, art.Content)
				}
				m.Separator()
				if m.Item("删除文章").Chosen() {
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
			starLabel := "星标"
			if cur.item.Starred {
				starIcon = iconStarSolid
				starColor = pal.yellow
				starLabel = "取消星标"
			}
			if toolbarIconBtnColored(c, "sub-star", starIcon, starColor, starLabel, pal).Clicked() {
				if err := a.backend.StarSubscription(subURL, key, !cur.item.Starred); err != nil {
					a.setError("更新星标失败：" + err.Error())
				} else {
					a.Reload()
				}
			}

			readLabel := "标为未读"
			readColor := pal.secondary
			if !cur.item.Read {
				readLabel = "标为已读"
				readColor = pal.accent
			}
			if toolbarIconBtnColored(c, "sub-toggle-read", lucide.CircleDot(), readColor, readLabel, pal).Clicked() {
				if err := a.backend.SetSubscriptionRead(subURL, key, !cur.item.Read); err != nil {
					a.setError("更新阅读状态失败：" + err.Error())
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
			speakLabel := "朗读"
			if a.speaker != nil && a.speaker.Speaking() {
				speakIcon = lucide.VolumeX()
				speakLabel = "停止朗读"
			}
			if toolbarIconBtn(c, "sub-speak", speakIcon, speakLabel).Clicked() {
				a.toggleSpeak(cur.item.Title, mdBody)
			}

			if toolbarIconBtn(c, "sub-related", lucide.FileText(), "相关文章").Clicked() {
				a.openRelatedArticles("", "", subURL, key, cur.item.Title)
			}

			if link := strings.TrimSpace(cur.item.Link); link != "" {
				webIcon := lucide.Globe()
				webLabel := "查看原网页"
				webColor := pal.secondary
				if a.subShowWebView {
					webIcon = lucide.FileText()
					webLabel = "切换阅读模式"
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
							a.setError("打开原网页失败：" + err.Error())
						}
					}
				}
			}
		}
		return
	}

	// Overview (Dashboard) mode actions: clean macOS icon buttons matching detail toolbar.
	if toolbarIconBtn(c, "tb-search", lucide.Search(), "搜索文章").Clicked() {
		a.openGlobalSearch()
	}
	if toolbarIconBtn(c, "dash-compose", lucide.SquarePen(), "写文章").Clicked() {
		a.handleDashboardCompose()
	}
}
