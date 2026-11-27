package nativeui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
)

// View renders the entire native Planet window.
func (a *App) View(c *ui.Context) {
	a.pal = applyMacPlanetTheme(c)
	a.editorState.Theme = paperEditorTheme(c, a.pal)
	if !a.loaded || a.externalDirty.Swap(false) {
		a.Reload()
	}
	if a.editorInspiration.phase != inspirationClosed &&
		(!a.editing || a.mode != viewSite || a.editorBuffer.Version() != a.editorInspiration.bufferVersion) {
		a.editorInspiration.phase = inspirationClosed
	}

	if !a.anyModalOpen() {
		if a.editing && c.Shortcut(ui.Ctrl, ui.KeyS) {
			a.saveCurrentDraft()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyK) {
			a.openGlobalSearch()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyO) {
			a.openIPFSResourceModal()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyN) {
			a.guardUnsaved(func() { a.startNewDraft(a.selectedSiteID) })
		}
		if c.Shortcut(ui.Ctrl, ui.KeyD) {
			a.openQuickPost()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyComma) {
			a.openToolsModal(toolsTabGeneral)
		}
		if a.mode == viewSite && !a.editing && c.Shortcut(ui.Ctrl, ui.KeyE) {
			if art := a.currentArticle(); art != nil {
				a.openArticleForEdit(*art)
			}
		}
		if a.mode != viewOverview && c.Shortcut(ui.Ctrl, ui.KeyF) {
			a.listSearchOpen = !a.listSearchOpen
			if !a.listSearchOpen {
				a.postSearchQuery = ""
				a.subSearchQuery = ""
			}
		}
		if a.listSearchOpen && c.Shortcut(0, ui.KeyEscape) {
			a.listSearchOpen = false
			a.postSearchQuery = ""
			a.subSearchQuery = ""
		}
	}

	pal := a.pal
	// Full-height 3-column macOS NavigationView split geometry:
	// Column 1 (Sidebar, resizable) | Column 2 (ArticleList) | Column 3 (Article/Editor/Dashboard).
	renderMain := func() {
		ui.Column(c).Fill().MinWidth(0).Background(pal.canvas).Children(func() {
			switch a.mode {
			case viewToday, viewUnread, viewStarred, viewSubscription:
				a.viewSubscriptionWorkspace(c)
			case viewSite:
				if a.currentSite() != nil {
					a.viewSiteWorkspace(c)
				} else {
					a.viewDashboardPane(c)
				}
			default:
				a.viewDashboardPane(c)
			}
		})
	}
	if a.sidebarCollapsed {
		renderMain()
	} else {
		if a.sidebarWidth < 220 {
			a.sidebarWidth = 228
		}
		ui.Split(c.Key("sidebar-split"), &a.sidebarWidth,
			func() { a.viewSidebar(c) },
			renderMain,
		).SplitReset(228).Fill().Background(pal.canvas)
	}

	a.viewCreateSiteModal(c)
	a.viewComposePickerModal(c)
	a.viewBadgeWallModal(c)
	a.viewArticleSettingsModal(c)
	a.viewMoveArticleSheet(c)
	a.viewQuickPostModal(c)
	a.viewDeleteArticleDialog(c)
	a.viewFollowModal(c)
	a.viewEditSubscriptionModal(c)
	a.viewUnfollowDialog(c)
	a.viewGlobalSearchModal(c)
	a.viewRelatedArticlesModal(c)
	a.viewSiteSettingsModal(c)
	a.viewPublishModal(c)
	a.viewOfflineModal(c)
	a.viewOpenIPFSResourceSheet(c)
	a.viewTipCreatorSheet(c)
	a.viewFoldersModal(c)
	a.viewRemovedSitesModal(c)
	a.viewToolsModal(c)
	a.viewTemplatePreviewSheet(c)
	a.viewUnsavedDialog(c)

	// Release embedded subscription WebView when leaving subscription mode or returning to Reader View.
	inSubMode := a.mode == viewToday || a.mode == viewUnread || a.mode == viewStarred || a.mode == viewSubscription
	if (!inSubMode || !a.subShowWebView) && a.subWebView != nil {
		a.destroySubWebView()
	}

	// macOS-style non-blocking floating toast notifications at bottom-right
	// (never pushes article/dashboard content down).
	if a.statusSeq != a.shownStatusSeq && a.statusText != "" {
		a.shownStatusSeq = a.statusSeq
		typ := feedback.ToastSuccess
		timeout := 3 * time.Second
		if a.statusIsErr {
			typ = feedback.ToastError
			timeout = 6 * time.Second
		}
		feedback.ShowToast(c, feedback.ToastProps{
			Title:   a.statusText,
			Type:    typ,
			Timeout: timeout,
		})
	}
	feedback.ToastHost(c, feedback.ToastHostProps{Placement: feedback.ToastBottomRight})

	a.maybeSaveLayoutPrefs()
}

func (a *App) viewDashboardPane(c *ui.Context) {
	a.viewDashboardToolbar(c)
	a.viewDashboard(c)
}

func (a *App) viewCreateSiteModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.createSiteOpen, macSheetProps{
		Title:        a.t("dialog.createSite.title"),
		Width:        440,
		PrimaryText:  a.t("common.create"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.TextInput(c.Key("modal-site-name"), &a.newSiteName).Placeholder(a.t("dialog.createSite.namePlaceholder")).Label(a.t("dialog.createSite.nameLabel")).AutoFocus().AlignSelf(ui.Stretch)
		ui.TextArea(c.Key("modal-site-about"), &a.newSiteAbout).Placeholder(a.t("dialog.createSite.aboutPlaceholder")).Label(a.t("dialog.createSite.aboutLabel")).Lines(2, 4).AlignSelf(ui.Stretch)
		if a.createSiteError != "" {
			ui.Text(c, a.createSiteError).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		name := strings.TrimSpace(a.newSiteName)
		if name == "" {
			a.createSiteError = a.t("dialog.createSite.nameRequired")
			return
		}
		site, err := a.backend.CreateSite(name, strings.TrimSpace(a.newSiteAbout))
		if err != nil {
			a.createSiteError = err.Error()
			return
		}
		a.createSiteOpen = false
		a.createSiteError = ""
		a.Reload()
		a.selectSite(site.ID)
		a.setStatus(a.tf("status.siteCreated", site.Name))
	}
}
