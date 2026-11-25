package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/feedback"
)

// View renders the entire native Planet window.
func (a *App) View(c *ui.Context) {
	a.pal = applyMacPlanetTheme(c)
	a.editorState.Theme = paperEditorTheme(c, a.pal)
	if !a.loaded {
		a.Reload()
	}

	if !a.anyModalOpen() {
		if a.editing && c.Shortcut(ui.Ctrl, ui.KeyS) {
			a.saveCurrentDraft()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyK) {
			a.openGlobalSearch()
		}
		if c.Shortcut(ui.Ctrl, ui.KeyN) {
			a.guardUnsaved(func() { a.startNewDraft(a.selectedSiteID) })
		}
		if c.Shortcut(ui.Ctrl, ui.KeyD) {
			a.openQuickPost()
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
	a.viewArticleSettingsModal(c)
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
	a.viewFoldersModal(c)
	a.viewRemovedSitesModal(c)
	a.viewIntegrationsModal(c)
	a.viewToolsModal(c)
	a.viewUnsavedDialog(c)
	a.maybeSaveLayoutPrefs()
}

func (a *App) viewStatusBar(c *ui.Context) {
	alertType := feedback.Info
	if a.statusIsErr {
		alertType = feedback.Error
	}
	closed := false
	ui.Box(c).AlignSelf(ui.Stretch).Padding(8, 16, 0, 16).Shrink(0).Children(func() {
		feedback.Alert(c.Key("status-alert"), a.statusText, feedback.AlertProps{
			Type:    alertType,
			Variant: feedback.Soft,
			Closed:  &closed,
		})
	})
	if closed {
		a.clearStatus()
	}
}

func (a *App) viewDashboardPane(c *ui.Context) {
	a.viewDetailToolbar(c, true)
	if a.statusText != "" {
		a.viewStatusBar(c)
	}
	a.viewDashboard(c)
}

func (a *App) viewCreateSiteModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.createSiteOpen, macSheetProps{
		Title:       "新建站点",
		Width:       440,
		PrimaryText: "创建",
		CancelText:  "取消",
	}, pal, func() {
		ui.TextInput(c.Key("modal-site-name"), &a.newSiteName).Placeholder("站点名称").Label("站点名称").AutoFocus().AlignSelf(ui.Stretch)
		ui.TextArea(c.Key("modal-site-about"), &a.newSiteAbout).Placeholder("站点简介（可选）").Label("站点简介").Lines(2, 4).AlignSelf(ui.Stretch)
		if a.createSiteError != "" {
			ui.Text(c, a.createSiteError).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		name := strings.TrimSpace(a.newSiteName)
		if name == "" {
			a.createSiteError = "请输入站点名称"
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
		a.setStatus("已创建站点：" + site.Name)
	}
}
