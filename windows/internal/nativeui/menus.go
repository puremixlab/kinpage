package nativeui

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func (a *App) sidebarSiteContextMenu(site store.Site) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item(a.t("menu.siteNewPost")).Shortcut(ui.Ctrl, ui.KeyN).Chosen() {
			a.guardUnsaved(func() { a.startNewDraft(site.ID) })
		}
		if m.Item(a.t("menu.siteImportArt")).Chosen() {
			if a.backend != nil {
				art, err := a.backend.ImportArticle(context.Background(), site.ID)
				if err != nil {
					a.setError(a.tf("status.importArtErr", err.Error()))
				} else if art.ID != "" {
					a.Reload()
					a.openArticleForRead(site.ID, art.ID)
					a.setStatus(a.tf("status.importArtDone", a.fallbackTitle(art.Title)))
				}
			}
		}
		if m.Item(a.t("menu.siteSettings")).Chosen() {
			a.openSiteSettings(site)
		}
		m.Separator()
		if m.Item(a.t("menu.sitePublish")).Chosen() {
			a.triggerPublishSite(site)
		}
		if m.Item(a.t("menu.siteQuickRebuild")).Chosen() {
			if a.backend != nil {
				if n, err := a.backend.RebuildSite(site.ID, true); err != nil {
					a.setError(a.tf("status.quickRebuildErr", err.Error()))
				} else {
					a.setStatus(a.tf("status.quickRebuildDone", n))
				}
			}
		}
		if m.Item(a.t("menu.siteFullRebuild")).Chosen() {
			if a.backend != nil {
				if n, err := a.backend.RebuildSite(site.ID, false); err != nil {
					a.setError(a.tf("status.fullRebuildErr", err.Error()))
				} else {
					a.setStatus(a.tf("status.fullRebuildDone", n))
				}
			}
		}
		archiveLabel := a.t("menu.siteArchive")
		if site.Archived {
			archiveLabel = a.t("menu.siteUnarchive")
		}
		if m.Item(archiveLabel).Chosen() {
			if a.backend != nil {
				if _, err := a.backend.SetSiteArchived(site.ID, !site.Archived); err != nil {
					a.setError(a.tf("status.archiveErr", err.Error()))
				} else {
					if !site.Archived {
						a.archivedExpanded = true
					}
					a.Reload()
					if site.Archived {
						a.setStatus(a.tf("status.unarchived", a.fallbackTitle(site.Name)))
					} else {
						a.setStatus(a.tf("status.archived", a.fallbackTitle(site.Name)))
					}
				}
			}
		}
		if len(a.sites) > 1 {
			m.Submenu(a.t("menu.reorder"), func(sm *ui.Menu) {
				if sm.Item(a.t("menu.moveUp")).Chosen() {
					a.moveSiteOrder(site.ID, -1)
				}
				if sm.Item(a.t("menu.moveDown")).Chosen() {
					a.moveSiteOrder(site.ID, 1)
				}
			})
		}
		m.Separator()
		if m.Item(a.t("menu.siteMoveToTrash")).Danger().Chosen() {
			a.openSiteSettings(site)
			a.siteSettings.removeConfirm = true
		}
	}
}

func (a *App) sidebarSubContextMenu(sub store.Subscription) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item(a.t("menu.subRefresh")).Chosen() {
			a.refreshSubscription(sub.URL)
		}
		if m.Item(a.t("menu.subMarkAllRead")).Chosen() {
			if err := a.backend.MarkSubscriptionRead(sub.URL); err != nil {
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
		if len(a.subscriptions) > 1 {
			m.Submenu(a.t("menu.reorder"), func(sm *ui.Menu) {
				if sm.Item(a.t("menu.moveUp")).Chosen() {
					a.moveSubOrder(sub.URL, -1)
				}
				if sm.Item(a.t("menu.moveDown")).Chosen() {
					a.moveSubOrder(sub.URL, 1)
				}
			})
		}
		m.Separator()
		if m.Item(a.t("menu.subUnfollow")).Danger().Chosen() {
			a.unfollowURL = sub.URL
			a.unfollowTitle = subscriptionTitle(sub)
			a.unfollowOpen = true
		}
	}
}

func (a *App) articleContextMenu(art store.Article) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item(a.t("menu.artEdit")).Shortcut(ui.Ctrl, ui.KeyE).Chosen() {
			a.guardUnsaved(func() {
				a.openArticleForRead(art.SiteID, art.ID)
				a.openArticleForEdit(art)
			})
		}
		if m.Item(a.t("menu.artSettings")).Chosen() {
			a.openArticleSettings(art)
		}
		if m.Item(a.t("menu.artRelated")).Chosen() {
			a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
		}
		m.Separator()
		starLabel := a.t("menu.artAddStar")
		if art.Starred {
			starLabel = a.t("menu.artRemoveStar")
		}
		if m.Item(starLabel).Chosen() {
			if a.backend != nil {
				if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, !art.Starred, store.StarTypeDefault); err != nil {
					a.setError(a.tf("status.starErr", err.Error()))
				} else {
					a.Reload()
				}
			}
		}
		m.Submenu(a.t("menu.artStarAs"), func(sm *ui.Menu) {
			for _, opt := range allStarTypes {
				optLabel := a.t(opt.key)
				if sm.Item(optLabel).Radio(art.Starred && art.StarType == opt.id).Chosen() {
					if a.backend != nil {
						if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, true, opt.id); err != nil {
							a.setError(a.tf("status.starErr", err.Error()))
						} else {
							a.Reload()
							a.setStatus(a.tf("status.starredAs", optLabel))
						}
					}
				}
			}
			if art.Starred {
				sm.Separator()
				if sm.Item(a.t("menu.artRemoveStar")).Chosen() {
					if a.backend != nil {
						if _, err := a.backend.SetArticleStar(art.SiteID, art.ID, false, store.StarTypeDefault); err != nil {
							a.setError(a.tf("status.unstarErr", err.Error()))
						} else {
							a.Reload()
							a.setStatus(a.t("status.unstarred"))
						}
					}
				}
			}
		})
		pinLabel := a.t("menu.artPin")
		if art.Pinned != nil {
			pinLabel = a.t("menu.artUnpin")
		}
		if m.Item(pinLabel).Chosen() {
			a.toggleArticlePinned(art)
		}
		if a.hasOtherActiveSite(art.SiteID) {
			if m.Item(a.t("menu.artMoveToSite")).Chosen() {
				a.openMoveArticleSheet(art)
			}
		}
		if m.Item(a.t("menu.artExport")).Chosen() {
			if a.backend != nil {
				if outPath, err := a.backend.ExportArticle(context.Background(), art.SiteID, art.ID); err != nil {
					a.setError(a.tf("status.exportArtErr", err.Error()))
				} else if outPath != "" {
					a.setStatus(a.tf("status.exportedArt", outPath))
				}
			}
		}
		m.Separator()
		if m.Item(a.t("menu.artDelete")).Danger().Chosen() {
			a.deleteArticleID = art.ID
			a.deleteArticleTitle = art.Title
			a.deleteArticleOpen = true
		}
	}
}

func (a *App) draftContextMenu(draft store.Draft) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if m.Item(a.t("menu.draftOpen")).Chosen() {
			a.guardUnsaved(func() { a.openDraft(draft) })
		}
		m.Separator()
		if m.Item(a.t("menu.draftDelete")).Danger().Chosen() {
			if a.backend != nil {
				if err := a.backend.DeleteDraft(draft.SiteID, draft.ID); err != nil {
					a.setError(a.tf("status.deleteDraftErr", err.Error()))
				} else {
					if a.editing && a.activeDraftID == draft.ID {
						a.editing = false
						a.activeDraftID = ""
						a.activeArticleID = ""
					}
					a.Reload()
					a.setStatus(a.t("status.draftDeleted"))
				}
			}
		}
	}
}

func (a *App) subItemContextMenu(entry subItemEntry) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		readLabel := a.t("menu.subMarkUnread")
		if !entry.item.Read {
			readLabel = a.t("menu.subMarkRead")
		}
		if m.Item(readLabel).Chosen() {
			if err := a.backend.SetSubscriptionRead(entry.subURL, entry.effective, !entry.item.Read); err != nil {
				a.setError(a.tf("status.readStateErr", err.Error()))
			} else {
				a.Reload()
			}
		}
		starLabel := a.t("menu.artAddStar")
		if entry.item.Starred {
			starLabel = a.t("menu.artRemoveStar")
		}
		if m.Item(starLabel).Chosen() {
			if err := a.backend.StarSubscription(entry.subURL, entry.effective, !entry.item.Starred); err != nil {
				a.setError(a.tf("status.starErr", err.Error()))
			} else {
				a.Reload()
			}
		}
		m.Submenu(a.t("menu.artStarAs"), func(sm *ui.Menu) {
			for _, opt := range allStarTypes {
				if sm.Item(a.t(opt.key)).Radio(entry.item.Starred && entry.item.StarType == opt.id).Chosen() {
					if err := a.backend.SetSubscriptionStar(entry.subURL, entry.effective, true, opt.id); err != nil {
						a.setError(a.tf("status.starErr", err.Error()))
					} else {
						a.Reload()
					}
				}
			}
			if entry.item.Starred {
				sm.Separator()
				if sm.Item(a.t("menu.artRemoveStar")).Chosen() {
					if err := a.backend.SetSubscriptionStar(entry.subURL, entry.effective, false, store.StarTypeDefault); err != nil {
						a.setError(a.tf("status.unstarErr", err.Error()))
					} else {
						a.Reload()
					}
				}
			}
		})
		m.Separator()
		if m.Item(a.t("menu.artRelated")).Chosen() {
			a.openRelatedArticles("", "", entry.subURL, entry.effective, entry.item.Title)
		}
		if link := strings.TrimSpace(entry.item.Link); link != "" && a.openExternal != nil {
			if m.Item(a.t("menu.openInBrowser")).Chosen() {
				if err := a.openExternal(link); err != nil {
					a.setError(a.tf("status.openLinkErr", err.Error()))
				}
			}
		}
	}
}
