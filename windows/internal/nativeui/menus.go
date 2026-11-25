package nativeui

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func (a *App) sidebarSiteContextMenu(site store.Site) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item("写新文章").Shortcut(ui.Ctrl, ui.KeyN).Chosen() {
			a.guardUnsaved(func() { a.startNewDraft(site.ID) })
		}
		if m.Item("导入文章…").Chosen() {
			if a.backend != nil {
				art, err := a.backend.ImportArticle(context.Background(), site.ID)
				if err != nil {
					a.setError("导入文章失败：" + err.Error())
				} else if art.ID != "" {
					a.Reload()
					a.openArticleForRead(site.ID, art.ID)
					a.setStatus("已导入文章：" + fallbackTitle(art.Title))
				}
			}
		}
		if m.Item("站点设置…").Chosen() {
			a.openSiteSettings(site)
		}
		m.Separator()
		if m.Item("发布站点").Chosen() {
			a.triggerPublishSite(site)
		}
		archiveLabel := "归档站点"
		if site.Archived {
			archiveLabel = "取消归档"
		}
		if m.Item(archiveLabel).Chosen() {
			if a.backend != nil {
				if _, err := a.backend.SetSiteArchived(site.ID, !site.Archived); err != nil {
					a.setError("更新归档状态失败：" + err.Error())
				} else {
					if !site.Archived {
						a.archivedExpanded = true
					}
					a.Reload()
					if site.Archived {
						a.setStatus("已取消归档：" + fallbackTitle(site.Name))
					} else {
						a.setStatus("已归档站点：" + fallbackTitle(site.Name))
					}
				}
			}
		}
		m.Separator()
		if m.Item("移入回收站").Danger().Chosen() {
			a.openSiteSettings(site)
			a.siteSettings.removeConfirm = true
		}
	}
}

func (a *App) sidebarSubContextMenu(sub store.Subscription) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item("刷新订阅").Chosen() {
			if _, err := a.backend.RefreshSubscription(sub.URL); err != nil {
				a.setError("刷新订阅失败：" + err.Error())
			} else {
				a.Reload()
				a.setStatus("已刷新订阅")
			}
		}
		if m.Item("全部标为已读").Chosen() {
			if err := a.backend.MarkSubscriptionRead(sub.URL); err != nil {
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
			a.unfollowTitle = subscriptionTitle(sub)
			a.unfollowOpen = true
		}
	}
}

func (a *App) articleContextMenu(art store.Article) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item("编辑文章").Shortcut(ui.Ctrl, ui.KeyE).Chosen() {
			a.guardUnsaved(func() {
				a.openArticleForRead(art.SiteID, art.ID)
				a.openArticleForEdit(art)
			})
		}
		if m.Item("文章设置…").Chosen() {
			a.openArticleSettings(art)
		}
		if m.Item("查找相关文章").Chosen() {
			a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
		}
		m.Separator()
		starLabel := "添加星标"
		if art.Starred {
			starLabel = "移除星标"
		}
		if m.Item(starLabel).Chosen() {
			if a.backend != nil {
				if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, !art.Starred, store.StarTypeDefault); err != nil {
					a.setError("更新星标失败：" + err.Error())
				} else {
					a.Reload()
				}
			}
		}
		m.Submenu("标记星标", func(sm *ui.Menu) {
			for _, opt := range allStarTypes {
				if sm.Item(opt.label).Radio(art.Starred && art.StarType == opt.id).Chosen() {
					if a.backend != nil {
						if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, true, opt.id); err != nil {
							a.setError("更新星标失败：" + err.Error())
						} else {
							a.Reload()
							a.setStatus("已标记为「" + opt.label + "」")
						}
					}
				}
			}
			if art.Starred {
				sm.Separator()
				if sm.Item("移除星标").Chosen() {
					if a.backend != nil {
						if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, false, store.StarTypeDefault); err != nil {
							a.setError("移除星标失败：" + err.Error())
						} else {
							a.Reload()
							a.setStatus("已移除星标")
						}
					}
				}
			}
		})
		pinLabel := "置顶文章"
		if art.Pinned != nil {
			pinLabel = "取消置顶"
		}
		if m.Item(pinLabel).Chosen() {
			a.toggleArticlePinned(art)
		}
		if a.hasOtherActiveSite(art.SiteID) {
			if m.Item("移动到其他站点…").Chosen() {
				a.openMoveArticleSheet(art)
			}
		}
		if m.Item("导出文章…").Chosen() {
			if a.backend != nil {
				if outPath, err := a.backend.ExportArticle(context.Background(), art.SiteID, art.ID); err != nil {
					a.setError("导出文章失败：" + err.Error())
				} else if outPath != "" {
					a.setStatus("文章已导出：" + outPath)
				}
			}
		}
		m.Separator()
		if m.Item("删除文章").Danger().Chosen() {
			a.deleteArticleID = art.ID
			a.deleteArticleTitle = art.Title
			a.deleteArticleOpen = true
		}
	}
}

func (a *App) draftContextMenu(draft store.Draft) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item("打开编辑").Chosen() {
			a.guardUnsaved(func() { a.openDraft(draft) })
		}
		m.Separator()
		if m.Item("删除草稿").Danger().Chosen() {
			if a.backend != nil {
				if err := a.backend.DeleteDraft(draft.SiteID, draft.ID); err != nil {
					a.setError("删除草稿失败：" + err.Error())
				} else {
					if a.editing && a.activeDraftID == draft.ID {
						a.editing = false
						a.activeDraftID = ""
						a.activeArticleID = ""
					}
					a.Reload()
					a.setStatus("草稿已删除")
				}
			}
		}
	}
}

func (a *App) subItemContextMenu(entry subItemEntry) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		readLabel := "标为未读"
		if !entry.item.Read {
			readLabel = "标为已读"
		}
		if m.Item(readLabel).Chosen() {
			if err := a.backend.SetSubscriptionRead(entry.subURL, entry.effective, !entry.item.Read); err != nil {
				a.setError("更新阅读状态失败：" + err.Error())
			} else {
				a.Reload()
			}
		}
		starLabel := "添加星标"
		if entry.item.Starred {
			starLabel = "移除星标"
		}
		if m.Item(starLabel).Chosen() {
			if err := a.backend.StarSubscription(entry.subURL, entry.effective, !entry.item.Starred); err != nil {
				a.setError("更新星标失败：" + err.Error())
			} else {
				a.Reload()
			}
		}
		m.Submenu("标记星标", func(sm *ui.Menu) {
			for _, opt := range allStarTypes {
				if sm.Item(opt.label).Radio(entry.item.Starred && entry.item.StarType == opt.id).Chosen() {
					if err := a.backend.SetSubscriptionStar(entry.subURL, entry.effective, true, opt.id); err != nil {
						a.setError("更新星标失败：" + err.Error())
					} else {
						a.Reload()
					}
				}
			}
			if entry.item.Starred {
				sm.Separator()
				if sm.Item("移除星标").Chosen() {
					if err := a.backend.SetSubscriptionStar(entry.subURL, entry.effective, false, store.StarTypeDefault); err != nil {
						a.setError("移除星标失败：" + err.Error())
					} else {
						a.Reload()
					}
				}
			}
		})
		m.Separator()
		if m.Item("查找相关文章").Chosen() {
			a.openRelatedArticles("", "", entry.subURL, entry.effective, entry.item.Title)
		}
		if link := strings.TrimSpace(entry.item.Link); link != "" && a.openExternal != nil {
			if m.Item("在浏览器中打开").Chosen() {
				if err := a.openExternal(link); err != nil {
					a.setError("打开外部链接失败：" + err.Error())
				}
			}
		}
	}
}
