package nativeui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"windows/internal/store"
)

type viewMode int

const (
	viewOverview viewMode = iota
	viewSite
	viewUnread
	viewStarred
	viewSubscription
)

const (
	editorPaneWrite = 0
	editorPaneSplit = 1
	editorPaneView  = 2
)

// App holds the UI state for the MyGo Native UI shell and editor.
type App struct {
	backend Backend

	mode           viewMode
	selectedSiteID string

	dashboard     store.DashboardSnapshot
	activity      []store.DailyActivity
	sites         []store.Site
	drafts        []store.Draft
	articles      []store.Article
	subscriptions []store.Subscription

	// Reading / editing selection within the selected site.
	editing           bool
	activeDraftID     string
	activeArticleID   string
	selectedReadID    string
	editorTitle       string
	savedTitle        string
	editorPaneMode    int
	editorMetaOpen    bool
	editorTags        map[string]string
	editorTagsDirty   bool
	newTagInput       string
	editorAttachments []string
	editorHeroImage   string
	editorState       *editor.State
	editorBuffer      *editor.Buffer
	markdownOpts      markdown.Options
	previewScroll     ui.ScrollState
	readerScroll      ui.ScrollState
	dashboardScroll   ui.ScrollState
	postsListScroll   ui.ScrollState
	sidebarScroll     ui.ScrollState
	postsPanelWidth   float32
	editorSplitRatio  float32
	postSearchQuery   string

	// Unsaved draft navigation guard.
	unsavedDialogOpen bool
	pendingNavAction  func()

	// Create-site modal state.
	createSiteOpen  bool
	newSiteName     string
	newSiteAbout    string
	createSiteError string

	// Article settings modal state.
	articleSettingsOpen    bool
	articleSettingsID      string
	articleSettingsSlug    string
	articleSettingsType    int
	articleSettingsExtLink string
	articleSettingsInNav   bool
	articleSettingsWeight  string
	articleSettingsErr     string

	// Delete article confirmation dialog state.
	deleteArticleOpen  bool
	deleteArticleID    string
	deleteArticleTitle string

	// Quick post modal state.
	quickPostOpen    bool
	quickPostSiteID  string
	quickPostTitle   string
	quickPostContent string
	quickPostErr     string

	// Dashboard interaction state.
	activityMode        int
	trendRange          int
	trendHoverPoint     int
	articleFilterSiteID string
	composePickerOpen   bool

	// Subscriptions & global search state.
	selectedSubURL        string
	selectedSubItemSubURL string
	selectedSubItemKey    string
	subSearchQuery        string
	subReaderScroll       ui.ScrollState
	followModalOpen       bool
	followAddress         string
	followErr             string
	editSubOpen           bool
	editSubOldURL         string
	editSubNewURL         string
	editSubTitle          string
	editSubErr            string
	unfollowOpen          bool
	unfollowURL           string
	unfollowTitle         string
	searchModalOpen       bool
	searchQuery           string
	searchResults         []store.SearchResult
	searchErr             string
	relatedModalOpen      bool
	relatedTitle          string
	relatedResults        []store.SearchResult
	relatedErr            string

	// Batch 3A modals state.
	siteSettings siteSettingsState
	publishModal publishModalState
	ipfsModal    ipfsModalState
	offlineModal offlineModalState
	foldersModal foldersModalState
	removedSites removedSitesState

	// Batch 3B & 3C modals state.
	integrations integrationsModalState
	toolsModal   toolsModalState

	// Native platform hooks (TTS & external browser preview).
	speaker          Speaker
	openExternal     func(string) error
	pal              macPalette
	settingsMenuOpen bool
	postsCollapsed   bool

	// Feedback status.
	statusText  string
	statusIsErr bool
	loaded      bool
}

// New creates a native UI application bound to backend and loads initial data.
func New(backend Backend) *App {
	buf := editor.NewBuffer("")
	buf.SetLanguage(editor.LanguageFor("post.md"))
	a := &App{
		backend:        backend,
		mode:           viewOverview,
		editorPaneMode: editorPaneWrite,
		editorTags:     map[string]string{},
		editorBuffer:   buf,
		editorState: &editor.State{
			Buffer:         buf,
			Font:           ui.Font{Size: 15},
			LineHeight:     1.75,
			SoftWrap:       true,
			LineNumbers:    editor.LineNumbersOff,
			NoIndentGuides: true,
			NoFolding:      true,
			NoActiveLine:   true,
			Placeholder:    "开始写点什么…",
			Label:          "Markdown 正文编辑器",
		},
		postsPanelWidth:  276,
		editorSplitRatio: 50,
		trendHoverPoint:  -1,
		speaker:          newDefaultSpeaker(),
		openExternal:     openBrowserFile,
	}
	a.Reload()
	return a
}

// Close stops any active native background playback such as TTS.
func (a *App) Close() {
	if a.speaker != nil {
		a.speaker.Stop()
	}
}

func (a *App) toggleSpeak(title, body string) {
	if a.speaker == nil {
		return
	}
	if a.speaker.Speaking() {
		a.speaker.Stop()
		a.setStatus("已停止朗读")
		return
	}
	text := strings.TrimSpace(title + "\n\n" + body)
	if text == "" {
		return
	}
	if err := a.speaker.Speak(text); err != nil {
		a.setError("语音朗读失败：" + err.Error())
		return
	}
	a.setStatus("正在使用系统语音朗读…")
}

// Reload refreshes the dashboard snapshot, writing activity, site list, and
// currently selected site's drafts and articles.
func (a *App) Reload() {
	if a.backend == nil {
		return
	}
	a.loaded = true
	if dash, err := a.backend.Dashboard(); err != nil {
		a.setError("读取总览失败：" + err.Error())
	} else {
		a.dashboard = dash
	}
	if act, err := a.backend.WritingActivity(); err == nil {
		a.activity = act
	}
	if sites, err := a.backend.Sites(); err != nil {
		a.setError("读取站点列表失败：" + err.Error())
	} else {
		a.sites = sites
	}
	if subs, err := a.backend.Subscriptions(); err == nil {
		a.subscriptions = subs
	}
	if a.selectedSiteID != "" {
		a.loadSiteItems(a.selectedSiteID)
	}
}

func (a *App) loadSiteItems(siteID string) {
	if a.backend == nil || siteID == "" {
		a.drafts = nil
		a.articles = nil
		return
	}
	drafts, err := a.backend.Drafts(siteID)
	if err != nil {
		a.setError("读取草稿失败：" + err.Error())
	} else {
		a.drafts = drafts
	}
	articles, err := a.backend.Articles(siteID)
	if err != nil {
		a.setError("读取文章失败：" + err.Error())
	} else {
		sort.SliceStable(articles, func(i, j int) bool {
			pi, pj := articles[i].Pinned != nil, articles[j].Pinned != nil
			if pi != pj {
				return pi
			}
			return false
		})
		a.articles = articles
	}
}

func (a *App) setStatus(msg string) {
	a.statusText = msg
	a.statusIsErr = false
}

func (a *App) setError(msg string) {
	a.statusText = msg
	a.statusIsErr = true
}

func (a *App) clearStatus() {
	a.statusText = ""
	a.statusIsErr = false
}

func (a *App) isEditorDirty() bool {
	if !a.editing {
		return false
	}
	return a.editorTitle != a.savedTitle || a.editorTagsDirty || a.editorBuffer.Modified()
}

func (a *App) selectSite(siteID string) {
	a.mode = viewSite
	a.selectedSiteID = siteID
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.selectedReadID = ""
	a.postSearchQuery = ""
	a.clearStatus()
	a.loadSiteItems(siteID)
	if len(a.articles) > 0 {
		a.selectedReadID = a.articles[0].ID
	} else if len(a.drafts) > 0 {
		a.openDraft(a.drafts[0])
	}
}

func (a *App) startNewDraft(siteID string) {
	if siteID == "" {
		if len(a.sites) == 0 {
			a.createSiteOpen = true
			return
		}
		siteID = a.sites[0].ID
	}
	a.mode = viewSite
	if a.selectedSiteID != siteID {
		a.selectedSiteID = siteID
		a.loadSiteItems(siteID)
	}
	a.editing = true
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.selectedReadID = ""
	a.editorTitle = ""
	a.savedTitle = ""
	a.editorTags = map[string]string{}
	a.editorTagsDirty = false
	a.newTagInput = ""
	a.editorAttachments = nil
	a.editorHeroImage = ""
	a.editorBuffer.Reset("")
	a.setStatus("新建草稿")
}

func (a *App) openDraft(d store.Draft) {
	a.mode = viewSite
	if a.selectedSiteID != d.SiteID {
		a.selectedSiteID = d.SiteID
		a.loadSiteItems(d.SiteID)
	}
	a.editing = true
	a.activeDraftID = d.ID
	a.activeArticleID = d.ArticleID
	a.selectedReadID = ""
	a.editorTitle = d.Title
	a.savedTitle = d.Title
	a.editorTags = cloneTagsMap(d.Tags)
	a.editorTagsDirty = false
	a.newTagInput = ""
	a.editorAttachments = append([]string(nil), d.Attachments...)
	a.editorHeroImage = d.HeroImage
	a.editorBuffer.Reset(d.Content)
	a.clearStatus()
}

func (a *App) openArticleForRead(siteID, articleID string) {
	a.mode = viewSite
	if a.selectedSiteID != siteID {
		a.selectedSiteID = siteID
		a.loadSiteItems(siteID)
	}
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.selectedReadID = articleID
	a.readerScroll.Y = 0
	a.clearStatus()
}

func (a *App) openArticleForEdit(art store.Article) {
	for _, d := range a.drafts {
		if d.ArticleID == art.ID {
			a.openDraft(d)
			return
		}
	}
	a.mode = viewSite
	a.editing = true
	a.activeDraftID = ""
	a.activeArticleID = art.ID
	a.selectedReadID = ""
	a.editorTitle = art.Title
	a.savedTitle = art.Title
	a.editorTags = cloneTagsMap(art.Tags)
	a.editorTagsDirty = false
	a.newTagInput = ""
	a.editorAttachments = append([]string(nil), art.Attachments...)
	a.editorHeroImage = art.HeroImage
	a.editorBuffer.Reset(art.Content)
	a.setStatus("正在编辑已发布文章（保存后将创建草稿）")
}

func (a *App) saveCurrentDraft() bool {
	if !a.editing || a.selectedSiteID == "" || a.backend == nil {
		return false
	}
	tagsJSON, err := a.marshalEditorTags()
	if err != nil {
		a.setError("序列化标签失败：" + err.Error())
		return false
	}
	content := a.editorBuffer.String()
	draft, err := a.backend.SaveDraftWithTags(a.selectedSiteID, a.activeDraftID, a.activeArticleID, a.editorTitle, content, tagsJSON)
	if err != nil {
		a.setError("保存草稿失败：" + err.Error())
		return false
	}
	a.activeDraftID = draft.ID
	a.activeArticleID = draft.ArticleID
	a.savedTitle = draft.Title
	a.editorTags = cloneTagsMap(draft.Tags)
	a.editorTagsDirty = false
	a.editorAttachments = append([]string(nil), draft.Attachments...)
	a.editorHeroImage = draft.HeroImage
	a.editorBuffer.MarkSaved()
	a.Reload()
	a.setStatus("草稿已保存 " + time.Now().Format("15:04:05"))
	return true
}

func (a *App) publishCurrentDraft() {
	if !a.editing || a.selectedSiteID == "" || a.backend == nil {
		return
	}
	tagsJSON, err := a.marshalEditorTags()
	if err != nil {
		a.setError("序列化标签失败：" + err.Error())
		return
	}
	content := a.editorBuffer.String()
	art, err := a.backend.PublishDraftWithTags(a.selectedSiteID, a.activeDraftID, a.editorTitle, content, tagsJSON)
	if err != nil {
		a.setError("发布失败：" + err.Error())
		return
	}
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.selectedReadID = art.ID
	a.editorTagsDirty = false
	a.editorBuffer.MarkSaved()
	a.Reload()
	a.setStatus("已发布：" + fallbackTitle(art.Title))
}

func (a *App) deleteCurrentDraft() {
	if !a.editing || a.selectedSiteID == "" || a.activeDraftID == "" || a.backend == nil {
		return
	}
	if err := a.backend.DeleteDraft(a.selectedSiteID, a.activeDraftID); err != nil {
		a.setError("删除草稿失败：" + err.Error())
		return
	}
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.editorTagsDirty = false
	a.Reload()
	a.setStatus("草稿已删除")
}

func (a *App) currentSite() *store.Site {
	for i := range a.sites {
		if a.sites[i].ID == a.selectedSiteID {
			return &a.sites[i]
		}
	}
	return nil
}

func (a *App) currentArticle() *store.Article {
	for i := range a.articles {
		if a.articles[i].ID == a.selectedReadID {
			return &a.articles[i]
		}
	}
	return nil
}

// View renders the entire native Planet window.
func (a *App) View(c *ui.Context) {
	a.pal = applyMacPlanetTheme(c)
	a.editorState.Theme = paperEditorTheme(c, a.pal)
	if !a.loaded {
		a.Reload()
	}

	if a.editing && !a.anyModalOpen() && c.Shortcut(ui.Ctrl, ui.KeyS) {
		a.saveCurrentDraft()
	}
	if !a.anyModalOpen() && c.Shortcut(ui.Ctrl, ui.KeyK) {
		a.openGlobalSearch()
	}

	pal := a.pal
	ui.Row(c).Fill().Background(pal.canvas).Children(func() {
		a.viewSidebar(c)
		ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(pal.line)
		ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
			if a.statusText != "" {
				a.viewStatusBar(c)
			}
			switch a.mode {
			case viewUnread, viewStarred, viewSubscription:
				a.viewSubscriptionWorkspace(c)
			case viewSite:
				if a.currentSite() != nil {
					a.viewSiteWorkspace(c)
				} else {
					a.viewDashboard(c)
				}
			default:
				a.viewDashboard(c)
			}
		})
	})

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
	a.viewIPFSModal(c)
	a.viewOfflineModal(c)
	a.viewFoldersModal(c)
	a.viewRemovedSitesModal(c)
	a.viewIntegrationsModal(c)
	a.viewToolsModal(c)
	a.viewUnsavedDialog(c)
}

func (a *App) anyModalOpen() bool {
	return a.createSiteOpen || a.composePickerOpen || a.articleSettingsOpen ||
		a.quickPostOpen || a.deleteArticleOpen || a.unsavedDialogOpen ||
		a.followModalOpen || a.editSubOpen || a.unfollowOpen ||
		a.searchModalOpen || a.relatedModalOpen ||
		a.siteSettings.open || a.siteSettings.removeConfirm ||
		a.publishModal.open || a.ipfsModal.open || a.offlineModal.open ||
		a.foldersModal.open || a.removedSites.open || a.integrations.open ||
		a.toolsModal.open
}

func (a *App) viewStatusBar(c *ui.Context) {
	t := c.Theme()
	bg := t.Surface
	fg := t.TextMuted
	if a.statusIsErr {
		bg = ui.RGBA(220, 38, 38, 0.12)
		fg = t.Danger
	}
	ui.Row(c).AlignSelf(ui.Stretch).PaddingX(14).PaddingY(6).Gap(8).AlignItems(ui.Center).Background(bg).Children(func() {
		ui.Text(c, a.statusText).FontSize(12).TextColor(fg).Grow(1)
		if ui.Button(c.Key("dismiss-status"), "关闭").Clicked() {
			a.clearStatus()
		}
	})
	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
}

func (a *App) viewSidebar(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Width(228).Shrink(0).AlignSelf(ui.Stretch).Background(pal.sidebar).Children(func() {
		// Brand header & primary nav (matching workspace-dashboard.png).
		ui.Column(c).PaddingX(12).PaddingY(12).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).PaddingX(4).Children(func() {
				ui.Box(c).Size(18, 18).Radius(9).Background(pal.ink).Center().Children(func() {
					ui.Text(c, "P").FontSize(10).Bold().TextColor(pal.canvas).NoWrap()
				})
				ui.Text(c, "Planet").Bold().FontSize(14).TextColor(pal.ink)
			})
			if sidebarNavRow(c, "nav-overview", "写作台", a.mode == viewOverview, 0, func() {
				ui.Text(c, "⊞").FontSize(14).TextColor(pal.secondary).Shrink(0)
			}, pal) {
				a.guardUnsaved(func() {
					a.mode = viewOverview
					a.editing = false
					a.clearStatus()
					a.Reload()
				})
			}
			// Subtle search pill matching writing-editor.png.
			if ui.Button(c.Key("nav-search"), "").Label("搜索文章").AlignSelf(ui.Stretch).
				Background(pal.hover).Border(0, ui.Color{}).Radius(6).PaddingX(10).PaddingY(6).
				Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "🔍  搜索文章").FontSize(12).TextColor(pal.muted).NoWrap()
						ui.Text(c, "Ctrl K").FontSize(11).TextColor(pal.muted).NoWrap()
					})
				}).Clicked() {
				a.openGlobalSearch()
			}
		})

		ui.Scroll(c).TrackScroll(&a.sidebarScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).PaddingX(10).PaddingY(4).Gap(2).Children(func() {
				// My Sites section.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(6).PaddingY(4).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "我的站点").FontSize(11).Bold().TextColor(pal.muted)
					if ghostIconButton(c, "nav-new-site", "+", "+ 站点", pal) {
						a.newSiteName = ""
						a.newSiteAbout = ""
						a.createSiteError = ""
						a.createSiteOpen = true
					}
				})
				if len(a.sites) == 0 {
					ui.Text(c, "暂无站点").FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, site := range a.sites {
					siteID, siteName := site.ID, fallbackTitle(site.Name)
					selected := a.mode == viewSite && a.selectedSiteID == siteID
					if sidebarNavRow(c, "site-"+siteID, siteName, selected, 0, func() {
						avatarCircle(c, siteName, 20)
					}, pal) {
						a.guardUnsaved(func() { a.selectSite(siteID) })
					}
				}

				// Reading section.
				ui.Box(c).Height(10)
				ui.Text(c, "阅读").FontSize(11).Bold().TextColor(pal.muted).PaddingX(6).PaddingY(4)
				if sidebarNavRow(c, "nav-unread", "未读", a.mode == viewUnread, a.totalUnreadCount(), func() {
					ui.Text(c, "▤").FontSize(13).TextColor(pal.accent).Shrink(0)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewUnread, "") })
				}
				if sidebarNavRow(c, "nav-starred", "收藏", a.mode == viewStarred, a.totalStarredCount(), func() {
					ui.Text(c, "☆").FontSize(13).TextColor(ui.Hex("#f5b800")).Shrink(0)
				}, pal) {
					a.guardUnsaved(func() { a.openSubscriptionView(viewStarred, "") })
				}

				// Subscriptions section.
				ui.Box(c).Height(10)
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(6).PaddingY(4).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "订阅").FontSize(11).Bold().TextColor(pal.muted)
					if ghostIconButton(c, "nav-follow", "+", "+ 关注", pal) {
						a.followAddress = ""
						a.followErr = ""
						a.followModalOpen = true
					}
				})
				if len(a.subscriptions) == 0 {
					ui.Text(c, "暂无订阅").FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(4)
				}
				for _, sub := range a.subscriptions {
					subURL, title := sub.URL, subscriptionTitle(sub)
					unread := 0
					for _, item := range sub.Feed.Items {
						if !item.Read {
							unread++
						}
					}
					selected := a.mode == viewSubscription && a.selectedSubURL == subURL
					if sidebarNavRow(c, "sub-"+subURL, title, selected, unread, func() {
						avatarCircle(c, title, 20)
					}, pal) {
						a.guardUnsaved(func() { a.openSubscriptionView(viewSubscription, subURL) })
					}
				}
			})
		})

		// Single minimal bottom bar matching workspace-dashboard.png (IPFS node + Settings popover).
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
		ui.Row(c).MinHeight(42).PaddingX(12).AlignItems(ui.Center).Justify(ui.SpaceBetween).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			dotColor := pal.green
			if a.ipfsModal.status.Disabled {
				dotColor = pal.muted
			}
			if ui.Button(c.Key("sys-ipfs"), "").Label("IPFS 节点").
				Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(6).PaddingY(4).
				Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Box(c).Size(7, 7).Radius(4).Background(dotColor)
						ui.Text(c, "IPFS 节点").FontSize(12).TextColor(pal.secondary).NoWrap()
					})
				}).Clicked() {
				a.openIPFSModal()
			}

			settingsAnchor := ui.Button(c.Key("sys-settings-menu"), "").Label("设置").
				Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(4).
				Children(func() {
					ui.Text(c, "设置 ⚙").FontSize(12).TextColor(pal.secondary).NoWrap()
				})
			if settingsAnchor.Clicked() {
				a.settingsMenuOpen = !a.settingsMenuOpen
			}
			ui.Popover(c, settingsAnchor, &a.settingsMenuOpen, func() {
				ui.Column(c).Width(180).Gap(2).Padding(4).Children(func() {
					menuItem := func(key, label string, onClick func()) {
						if ui.Button(c.Key(key), "").Label(label).AlignSelf(ui.Stretch).
							Background(ui.Color{}).Border(0, ui.Color{}).Radius(5).PaddingX(10).PaddingY(6).
							Children(func() {
								ui.Row(c).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, label).FontSize(12).TextColor(pal.ink).NoWrap()
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
					ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
					menuItem("menu-offline", "离线存储", func() { a.openOfflineModal() })
					menuItem("menu-folders", "已发布文件夹", func() { a.openFoldersModal() })
					menuItem("menu-trash", "回收站", func() { a.openRemovedSitesModal() })
					menuItem("menu-refresh", "刷新数据", func() {
						a.Reload()
						a.setStatus("已刷新数据")
					})
				})
			})
		})
	})
}

func (a *App) openDraftByID(siteID, draftID string) {
	a.selectSite(siteID)
	for _, d := range a.drafts {
		if d.ID == draftID {
			a.openDraft(d)
			return
		}
	}
}

func (a *App) viewSiteWorkspace(c *ui.Context) {
	if a.postsCollapsed {
		a.viewDetailPane(c)
		return
	}
	ui.Split(c.Key("site-split"), &a.postsPanelWidth,
		func() { a.viewPostsPanel(c) },
		func() { a.viewDetailPane(c) },
	).SplitReset(276).Grow(1).MinHeight(0).AlignSelf(ui.Stretch)
}

func (a *App) viewPostsPanel(c *ui.Context) {
	pal := a.pal
	site := a.currentSite()
	if site == nil {
		return
	}
	siteID := site.ID
	query := strings.ToLower(strings.TrimSpace(a.postSearchQuery))

	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		ui.Column(c).PaddingX(14).PaddingY(12).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, fallbackTitle(site.Name)).Bold().FontSize(15).TextColor(pal.ink).NoWrap()
				if ui.PrimaryButton(c.Key("site-new-post"), "+ 写文章").PaddingX(10).PaddingY(4).Clicked() {
					a.guardUnsaved(func() { a.startNewDraft(siteID) })
				}
			})
			ui.Row(c).Gap(4).Wrap().Children(func() {
				if ghostIconButton(c, "site-publish", "发布站点", "发布站点", pal) {
					a.triggerPublishSite(*site)
				}
				if ghostIconButton(c, "site-settings", "站点设置", "站点设置", pal) {
					a.openSiteSettings(*site)
				}
				if ghostIconButton(c, "site-integrations", "扩展", "扩展", pal) {
					a.openIntegrationsModal(*site)
				}
			})
			ui.SearchField(c.Key("post-search"), &a.postSearchQuery).Placeholder("筛选文章或草稿…").AlignSelf(ui.Stretch)
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(8).Gap(2).Children(func() {
				if len(a.drafts) > 0 {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(8).PaddingY(4).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, "草稿").FontSize(11).Bold().TextColor(pal.muted)
						ui.Text(c, fmt.Sprintf("%d 篇", len(a.drafts))).FontSize(11).TextColor(pal.muted)
					})
					for _, d := range a.drafts {
						draft := d
						title := fallbackTitle(draft.Title)
						if query != "" && !strings.Contains(strings.ToLower(title), query) {
							continue
						}
						selected := a.editing && a.activeDraftID == draft.ID
						cardBg := ui.Color{}
						titleColor := pal.ink
						excerptColor := pal.secondary
						metaColor := pal.muted
						if selected {
							cardBg = pal.listSelected
							titleColor = ui.Hex("#ffffff")
							excerptColor = pal.listSelMuted
							metaColor = pal.listSelMuted
						}
						snippet := excerptText(draft.Content, 44)
						if ui.Button(c.Key("draft-"+draft.ID), "").Label("✎ "+title).AlignSelf(ui.Stretch).
							Background(cardBg).Border(0, ui.Color{}).Radius(8).PaddingX(12).PaddingY(10).
							Children(func() {
								ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, "✎ "+title).Bold().FontSize(13).TextColor(titleColor).NoWrap()
									if snippet != "" {
										ui.Text(c, snippet).FontSize(12).TextColor(excerptColor)
									}
									ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
										ui.Text(c, "草稿").FontSize(11).TextColor(metaColor)
										ui.Text(c, formatSlashDate(draft.Modified)).FontSize(11).TextColor(metaColor)
									})
								})
							}).Clicked() {
							a.guardUnsaved(func() { a.openDraft(draft) })
						}
						if !selected {
							ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
						}
					}
				}

				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(8).PaddingY(6).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "文章").FontSize(11).Bold().TextColor(pal.muted)
					ui.Text(c, fmt.Sprintf("%d 篇", len(a.articles))).FontSize(11).TextColor(pal.muted)
				})
				if len(a.articles) == 0 {
					ui.Text(c, "暂无文章").FontSize(12).TextColor(pal.muted).PaddingX(8).PaddingY(8)
				}
				for _, art := range a.articles {
					artID := art.ID
					title := fallbackTitle(art.Title)
					if query != "" && !strings.Contains(strings.ToLower(title), query) {
						continue
					}
					selected := !a.editing && a.selectedReadID == artID
					cardBg := ui.Color{}
					titleColor := pal.ink
					excerptColor := pal.secondary
					metaColor := pal.muted
					if selected {
						cardBg = pal.listSelected
						titleColor = ui.Hex("#ffffff")
						excerptColor = pal.listSelMuted
						metaColor = pal.listSelMuted
					}
					displayTitle := title
					if art.Pinned != nil {
						displayTitle = "📌 " + displayTitle
					}
					snippet := excerptText(art.Content, 44)
					if ui.Button(c.Key("article-"+artID), "").Label(displayTitle).AlignSelf(ui.Stretch).
						Background(cardBg).Border(0, ui.Color{}).Radius(8).PaddingX(12).PaddingY(10).
						Children(func() {
							ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, displayTitle).Bold().FontSize(13).TextColor(titleColor).NoWrap()
								if snippet != "" {
									ui.Text(c, snippet).FontSize(12).TextColor(excerptColor)
								}
								ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
									ui.Text(c, fallbackTitle(site.Name)).FontSize(11).TextColor(metaColor).NoWrap()
									ui.Text(c, formatSlashDate(art.Modified)).FontSize(11).TextColor(metaColor).NoWrap()
								})
							})
						}).Clicked() {
						a.guardUnsaved(func() { a.openArticleForRead(siteID, artID) })
					}
					if !selected {
						ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
					}
				}
			})
		})
	})
}

func (a *App) viewDetailPane(c *ui.Context) {
	if a.editing {
		a.viewEditor(c)
		return
	}
	a.viewArticleReader(c)
}

func (a *App) viewEditor(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		// 52px Mac Planet editor toolbar matching writing-editor.png.
		ui.Row(c).MinHeight(52).PaddingX(20).Gap(10).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.canvas).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				if ghostIconButton(c, "toggle-posts", "≡", "切换文章列表", pal) {
					a.postsCollapsed = !a.postsCollapsed
				}
				ui.Text(c, "撰写文章").FontSize(13).TextColor(pal.secondary)
			})
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				saveStateText := "草稿已保存（未发布）"
				if a.isEditorDirty() {
					saveStateText = "● 有未保存更改"
				}
				ui.Text(c, saveStateText).FontSize(12).TextColor(pal.muted)
				if ui.Button(c.Key("editor-save"), "保存草稿").Background(pal.canvas).Border(1, pal.controlLine).Radius(6).PaddingX(12).PaddingY(6).Clicked() {
					a.saveCurrentDraft()
				}
				if ui.PrimaryButton(c.Key("editor-publish"), "发布").PaddingX(14).PaddingY(6).Clicked() {
					a.publishCurrentDraft()
				}
				if a.activeDraftID != "" {
					if ghostIconButton(c, "editor-delete", "删除", "删除草稿", pal) {
						a.deleteCurrentDraft()
					}
				}
			})
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)

		// Centered pure-paper writing surface (matching writing-editor.png).
		ui.Column(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Background(pal.canvas).Children(func() {
			maxW := float32(780)
			if a.editorPaneMode == editorPaneSplit {
				maxW = 1400
			}
			ui.Column(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).MaxWidth(maxW).PaddingX(36).PaddingY(24).Gap(14).Children(func() {
				// 26px bold borderless title input directly on the white page.
				ui.TextInput(c.Key("editor-title"), &a.editorTitle).
					Placeholder("输入文章标题…").Label("文章标题").
					FontSize(26).Bold().
					Background(ui.Color{}).Border(0, ui.Color{}).Padding(0).
					AlignSelf(ui.Stretch)

				// Metadata pill & segmented view switch directly beneath title.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Shrink(0).Children(func() {
					metaSummary := fmt.Sprintf("标签与附件   %d 个标签 / %d 个附件  ∨", len(a.editorTags), len(a.editorAttachments))
					metaBg := pal.subtle
					if a.editorMetaOpen {
						metaBg = pal.selection
					}
					if ui.Button(c.Key("editor-meta-toggle"), "").Label("元数据").
						Background(metaBg).Border(1, pal.line).Radius(6).PaddingX(12).PaddingY(5).
						Children(func() {
							ui.Text(c, metaSummary).FontSize(12).TextColor(pal.secondary).NoWrap()
						}).Clicked() {
						a.editorMetaOpen = !a.editorMetaOpen
					}
					ui.Segmented(c.Key("editor-mode"), &a.editorPaneMode, "编辑", "分屏", "预览").Label("视图模式")
				})

				if a.editorMetaOpen {
					a.viewEditorMetadata(c)
				}

				ui.Box(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					switch a.editorPaneMode {
					case editorPaneWrite:
						editor.View(c, a.editorState).Fill()
					case editorPaneView:
						a.viewMarkdownPreview(c, "", a.editorBuffer.String(), &a.previewScroll)
					default:
						ui.SplitPercent(c.Key("editor-split"), &a.editorSplitRatio,
							func() { editor.View(c, a.editorState).Fill() },
							func() { a.viewMarkdownPreview(c, "", a.editorBuffer.String(), &a.previewScroll) },
						).SplitReset(50).Fill()
					}
				})
			})
		})

		// Minimal 36px footer matching writing-editor.png.
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
		pos := a.editorState.CursorPos()
		chars := utf8.RuneCountInString(a.editorBuffer.String())
		ui.Row(c).MinHeight(36).PaddingX(24).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.canvas).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "草稿仅保存在本地；保存并发布后会公开整个站点及其附件。").FontSize(11).TextColor(pal.muted)
			ui.Text(c, fmt.Sprintf("行 %d, 列 %d  ·  %d 字", pos.Line+1, pos.Col+1, chars)).FontSize(11).TextColor(pal.muted)
		})
	})
}

func (a *App) viewMarkdownPreview(c *ui.Context, title, content string, scroll *ui.ScrollState) {
	pal := a.pal
	ui.Scroll(c).TrackScroll(scroll).Fill().Background(pal.canvas).Children(func() {
		ui.Column(c).PaddingX(16).PaddingY(12).Gap(14).AlignSelf(ui.Stretch).Children(func() {
			if strings.TrimSpace(title) != "" {
				ui.Text(c, title).Bold().FontSize(26).TextColor(pal.ink)
			}
			if strings.TrimSpace(content) == "" {
				ui.Text(c, "预览区域为空，输入 Markdown 内容后将实时渲染。").FontSize(13).TextColor(pal.muted)
			} else {
				markdown.View(c, content, &a.markdownOpts)
			}
		})
	})
}

func (a *App) viewArticleReader(c *ui.Context) {
	pal := a.pal
	art := a.currentArticle()
	if art == nil {
		ui.Column(c).Fill().Background(pal.canvas).Center().Gap(10).Children(func() {
			ui.Text(c, "选择一篇文章阅读，或点击左侧“+ 写文章”开始创作").FontSize(13).TextColor(pal.muted)
			if site := a.currentSite(); site != nil {
				if ui.PrimaryButton(c.Key("empty-compose"), "写新文章").Clicked() {
					a.startNewDraft(site.ID)
				}
			}
		})
		return
	}

	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		// 52px Mac Planet reader toolbar.
		ui.Row(c).MinHeight(52).PaddingX(20).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.canvas).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				if ghostIconButton(c, "reader-toggle-posts", "≡", "切换文章列表", pal) {
					a.postsCollapsed = !a.postsCollapsed
				}
				ui.Text(c, "阅读文章").FontSize(13).TextColor(pal.secondary)
			})
			ui.Row(c).Gap(6).AlignItems(ui.Center).Shrink(0).Children(func() {
				pinLabel := "置顶"
				if art.Pinned != nil {
					pinLabel = "取消置顶"
				}
				speakLabel := "朗读"
				if a.speaker != nil && a.speaker.Speaking() {
					speakLabel = "停止朗读"
				}
				if ghostIconButton(c, "reader-speak", speakLabel, speakLabel, pal) {
					a.toggleSpeak(art.Title, art.Content)
				}
				if ghostIconButton(c, "reader-related", "相关文章", "相关文章", pal) {
					a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
				}
				if ghostIconButton(c, "reader-pin", pinLabel, pinLabel, pal) {
					a.toggleArticlePinned(*art)
				}
				if ghostIconButton(c, "reader-settings", "文章设置", "文章设置", pal) {
					a.openArticleSettings(*art)
				}
				if ui.PrimaryButton(c.Key("reader-edit"), "编辑文章").PaddingX(12).PaddingY(5).Clicked() {
					a.openArticleForEdit(*art)
				}
				if ghostIconButton(c, "reader-delete", "删除", "删除文章", pal) {
					a.deleteArticleID = art.ID
					a.deleteArticleTitle = art.Title
					a.deleteArticleOpen = true
				}
			})
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)

		ui.Scroll(c).TrackScroll(&a.readerScroll).Fill().Background(pal.canvas).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(740).Gap(14).Children(func() {
					ui.Text(c, fallbackTitle(art.Title)).Bold().FontSize(26).TextColor(pal.ink)
					var meta []string
					if site := a.currentSite(); site != nil {
						meta = append(meta, fallbackTitle(site.Name))
					}
					if !art.Modified.IsZero() {
						meta = append(meta, formatSlashDate(art.Modified))
					}
					if art.Pinned != nil {
						meta = append(meta, "已置顶")
					}
					if art.Slug != "" {
						meta = append(meta, "/"+art.Slug)
					}
					if len(art.Tags) > 0 {
						meta = append(meta, fmt.Sprintf("%d 标签", len(art.Tags)))
					}
					if len(meta) > 0 {
						ui.Text(c, strings.Join(meta, "   ·   ")).FontSize(12).TextColor(pal.muted)
					}
					ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(pal.line)
					markdown.View(c, art.Content, &a.markdownOpts)
				})
			})
		})
	})
}

func (a *App) viewCreateSiteModal(c *ui.Context) {
	t := c.Theme()
	ui.Modal(c, &a.createSiteOpen, func() {
		ui.Column(c).Width(360).Gap(12).Children(func() {
			ui.Text(c, "新建站点").Bold().FontSize(16)
			ui.TextInput(c.Key("modal-site-name"), &a.newSiteName).Placeholder("站点名称").Label("站点名称").AutoFocus().AlignSelf(ui.Stretch)
			ui.TextArea(c.Key("modal-site-about"), &a.newSiteAbout).Placeholder("站点简介（可选）").Label("站点简介").Lines(2, 4).AlignSelf(ui.Stretch)
			if a.createSiteError != "" {
				ui.Text(c, a.createSiteError).FontSize(12).TextColor(t.Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("modal-site-cancel"), "取消").Clicked() {
					a.createSiteOpen = false
				}
				if ui.PrimaryButton(c.Key("modal-site-submit"), "创建").Clicked() {
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
			})
		})
	})
}

func fallbackTitle(s string) string {
	if strings.TrimSpace(s) == "" {
		return "无标题"
	}
	return s
}
