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
		backend:          backend,
		mode:             viewOverview,
		editorPaneMode:   editorPaneSplit,
		editorTags:       map[string]string{},
		editorBuffer:     buf,
		editorState:      &editor.State{Buffer: buf, SoftWrap: true, Placeholder: "使用 Markdown 开始写作…", Label: "Markdown 正文编辑器"},
		postsPanelWidth:  260,
		editorSplitRatio: 50,
		trendHoverPoint:  -1,
	}
	a.Reload()
	return a
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
	if !a.loaded {
		a.Reload()
	}

	if a.editing && !a.anyModalOpen() && c.Shortcut(ui.Ctrl, ui.KeyS) {
		a.saveCurrentDraft()
	}
	if !a.anyModalOpen() && c.Shortcut(ui.Ctrl, ui.KeyK) {
		a.openGlobalSearch()
	}

	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		a.viewSidebar(c)
		ui.Box(c).Width(1).AlignSelf(ui.Stretch).Background(t.Border)
		ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Children(func() {
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
	a.viewUnsavedDialog(c)
}

func (a *App) anyModalOpen() bool {
	return a.createSiteOpen || a.composePickerOpen || a.articleSettingsOpen ||
		a.quickPostOpen || a.deleteArticleOpen || a.unsavedDialogOpen ||
		a.followModalOpen || a.editSubOpen || a.unfollowOpen ||
		a.searchModalOpen || a.relatedModalOpen
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
	t := c.Theme()
	ui.Column(c).Width(220).Shrink(0).AlignSelf(ui.Stretch).Background(t.Surface).Children(func() {
		ui.Column(c).Padding(12).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, "Planet").Bold().FontSize(16)
				ui.Badge(c, "Native")
			})
			ui.Row(c).Gap(6).Children(func() {
				overviewBtn := ui.Button(c.Key("nav-overview"), "总览").Grow(1)
				if a.mode == viewOverview {
					overviewBtn.Background(t.Accent).TextColor(t.AccentText)
				}
				if overviewBtn.Clicked() {
					a.guardUnsaved(func() {
						a.mode = viewOverview
						a.editing = false
						a.clearStatus()
						a.Reload()
					})
				}
				if ui.Button(c.Key("nav-new-site"), "+ 站点").Clicked() {
					a.newSiteName = ""
					a.newSiteAbout = ""
					a.createSiteError = ""
					a.createSiteOpen = true
				}
			})
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c.Key("nav-quick-post"), "⚡ 快捷发文").Grow(1).Clicked() {
					a.openQuickPost()
				}
				if ui.Button(c.Key("nav-search"), "🔍 搜索").Clicked() {
					a.openGlobalSearch()
				}
			})
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		ui.Scroll(c).TrackScroll(&a.sidebarScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(10).Gap(4).Children(func() {
				ui.Text(c, "我的站点").FontSize(11).Bold().TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				if len(a.sites) == 0 {
					ui.Text(c, "暂无站点").FontSize(12).TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				}
				for _, site := range a.sites {
					siteID := site.ID
					selected := a.mode == viewSite && a.selectedSiteID == siteID
					btn := ui.Button(c.Key("site-"+siteID), fallbackTitle(site.Name)).AlignSelf(ui.Stretch)
					if selected {
						btn.Background(t.Accent).TextColor(t.AccentText)
					}
					if btn.Clicked() {
						a.guardUnsaved(func() { a.selectSite(siteID) })
					}
				}

				ui.Box(c).Height(8)
				ui.Text(c, "阅读").FontSize(11).Bold().TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				unreadLabel := fmt.Sprintf("未读 (%d)", a.totalUnreadCount())
				unreadBtn := ui.Button(c.Key("nav-unread"), unreadLabel).AlignSelf(ui.Stretch)
				if a.mode == viewUnread {
					unreadBtn.Background(t.Accent).TextColor(t.AccentText)
				}
				if unreadBtn.Clicked() {
					a.guardUnsaved(func() { a.openSubscriptionView(viewUnread, "") })
				}
				starredLabel := fmt.Sprintf("星标 (%d)", a.totalStarredCount())
				starredBtn := ui.Button(c.Key("nav-starred"), starredLabel).AlignSelf(ui.Stretch)
				if a.mode == viewStarred {
					starredBtn.Background(t.Accent).TextColor(t.AccentText)
				}
				if starredBtn.Clicked() {
					a.guardUnsaved(func() { a.openSubscriptionView(viewStarred, "") })
				}

				ui.Box(c).Height(8)
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).PaddingX(6).PaddingY(4).Children(func() {
					ui.Text(c, "订阅").FontSize(11).Bold().TextColor(t.TextMuted)
					if ui.Button(c.Key("nav-follow"), "+ 关注").Clicked() {
						a.followAddress = ""
						a.followErr = ""
						a.followModalOpen = true
					}
				})
				if len(a.subscriptions) == 0 {
					ui.Text(c, "暂无订阅").FontSize(12).TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				}
				for _, sub := range a.subscriptions {
					subURL := sub.URL
					selected := a.mode == viewSubscription && a.selectedSubURL == subURL
					btn := ui.Button(c.Key("sub-"+subURL), subscriptionTitle(sub)).AlignSelf(ui.Stretch)
					if selected {
						btn.Background(t.Accent).TextColor(t.AccentText)
					}
					if btn.Clicked() {
						a.guardUnsaved(func() { a.openSubscriptionView(viewSubscription, subURL) })
					}
				}
			})
		})

		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
		ui.Row(c).Padding(10).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Shrink(0).Children(func() {
			ui.Text(c, "MyGo GPU 模式").FontSize(11).TextColor(t.TextMuted)
			if ui.Button(c.Key("sidebar-refresh"), "刷新").Clicked() {
				a.Reload()
				a.setStatus("已刷新数据")
			}
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
	ui.Split(c.Key("site-split"), &a.postsPanelWidth,
		func() { a.viewPostsPanel(c) },
		func() { a.viewDetailPane(c) },
	).SplitReset(260).Grow(1).MinHeight(0).AlignSelf(ui.Stretch)
}

func (a *App) viewPostsPanel(c *ui.Context) {
	t := c.Theme()
	site := a.currentSite()
	if site == nil {
		return
	}
	siteID := site.ID
	query := strings.ToLower(strings.TrimSpace(a.postSearchQuery))

	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Column(c).Padding(12).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, fallbackTitle(site.Name)).Bold().FontSize(14)
				if ui.PrimaryButton(c.Key("site-new-post"), "+ 写文章").Clicked() {
					a.guardUnsaved(func() { a.startNewDraft(siteID) })
				}
			})
			ui.SearchField(c.Key("post-search"), &a.postSearchQuery).Placeholder("筛选文章或草稿…").AlignSelf(ui.Stretch)
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		ui.Scroll(c).TrackScroll(&a.postsListScroll).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Padding(10).Gap(4).Children(func() {
				if len(a.drafts) > 0 {
					ui.Text(c, fmt.Sprintf("草稿 (%d)", len(a.drafts))).FontSize(11).Bold().TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
					for _, d := range a.drafts {
						draft := d
						title := fallbackTitle(draft.Title)
						if query != "" && !strings.Contains(strings.ToLower(title), query) {
							continue
						}
						btn := ui.Button(c.Key("draft-"+draft.ID), "✎ "+title).AlignSelf(ui.Stretch)
						if a.editing && a.activeDraftID == draft.ID {
							btn.Background(t.Accent).TextColor(t.AccentText)
						}
						if btn.Clicked() {
							a.guardUnsaved(func() { a.openDraft(draft) })
						}
					}
				}

				ui.Text(c, fmt.Sprintf("已发布 (%d)", len(a.articles))).FontSize(11).Bold().TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				if len(a.articles) == 0 {
					ui.Text(c, "暂无文章").FontSize(12).TextColor(t.TextMuted).PaddingX(6).PaddingY(4)
				}
				for _, art := range a.articles {
					artID := art.ID
					title := fallbackTitle(art.Title)
					if query != "" && !strings.Contains(strings.ToLower(title), query) {
						continue
					}
					label := title
					if art.Pinned != nil {
						label = "📌 " + label
					}
					btn := ui.Button(c.Key("article-"+artID), label).AlignSelf(ui.Stretch)
					if !a.editing && a.selectedReadID == artID {
						btn.Background(t.Accent).TextColor(t.AccentText)
					}
					if btn.Clicked() {
						a.guardUnsaved(func() { a.openArticleForRead(siteID, artID) })
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
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).PaddingX(14).PaddingY(10).Gap(8).AlignItems(ui.Center).Background(t.Surface).Shrink(0).Children(func() {
			ui.TextInput(c.Key("editor-title"), &a.editorTitle).Placeholder("输入文章标题…").Label("文章标题").Grow(1)
			ui.Segmented(c.Key("editor-mode"), &a.editorPaneMode, "编辑", "分屏", "预览").Label("视图模式")
			metaBtn := ui.Button(c.Key("editor-meta-toggle"), "元数据")
			if a.editorMetaOpen {
				metaBtn.Background(t.Accent).TextColor(t.AccentText)
			}
			if metaBtn.Clicked() {
				a.editorMetaOpen = !a.editorMetaOpen
			}
			if ui.Button(c.Key("editor-save"), "保存草稿").Clicked() {
				a.saveCurrentDraft()
			}
			if ui.PrimaryButton(c.Key("editor-publish"), "发布").Clicked() {
				a.publishCurrentDraft()
			}
			if a.activeDraftID != "" {
				if ui.Button(c.Key("editor-delete"), "删除").Clicked() {
					a.deleteCurrentDraft()
				}
			}
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

		if a.editorMetaOpen {
			a.viewEditorMetadata(c)
		}

		ui.Box(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
			switch a.editorPaneMode {
			case editorPaneWrite:
				editor.View(c, a.editorState).Fill()
			case editorPaneView:
				a.viewMarkdownPreview(c, a.editorTitle, a.editorBuffer.String(), &a.previewScroll)
			default:
				ui.SplitPercent(c.Key("editor-split"), &a.editorSplitRatio,
					func() { editor.View(c, a.editorState).Fill() },
					func() { a.viewMarkdownPreview(c, a.editorTitle, a.editorBuffer.String(), &a.previewScroll) },
				).SplitReset(50).Fill()
			}
		})

		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
		pos := a.editorState.CursorPos()
		chars := utf8.RuneCountInString(a.editorBuffer.String())
		dirtyText := "已保存"
		if a.isEditorDirty() {
			dirtyText = "● 未保存（Ctrl+S 保存）"
		}
		ui.Row(c).PaddingX(14).PaddingY(6).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(t.Surface).Shrink(0).Children(func() {
			ui.Text(c, dirtyText).FontSize(11).TextColor(t.TextMuted)
			ui.Text(c, fmt.Sprintf("行 %d, 列 %d  ·  %d 行  ·  %d 字  ·  %d 标签  ·  %d 附件",
				pos.Line+1, pos.Col+1, a.editorBuffer.LineCount(), chars, len(a.editorTags), len(a.editorAttachments))).FontSize(11).TextColor(t.TextMuted)
		})
	})
}

func (a *App) viewMarkdownPreview(c *ui.Context, title, content string, scroll *ui.ScrollState) {
	t := c.Theme()
	ui.Scroll(c).TrackScroll(scroll).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(24).Gap(14).AlignSelf(ui.Stretch).Children(func() {
			if strings.TrimSpace(title) != "" {
				ui.Text(c, title).Bold().FontSize(24)
			}
			if strings.TrimSpace(content) == "" {
				ui.Text(c, "预览区域为空，输入 Markdown 内容后将实时渲染。").FontSize(13).TextColor(t.TextMuted)
			} else {
				markdown.View(c, content, &a.markdownOpts)
			}
		})
	})
}

func (a *App) viewArticleReader(c *ui.Context) {
	t := c.Theme()
	art := a.currentArticle()
	if art == nil {
		ui.Column(c).Fill().Center().Gap(10).Children(func() {
			ui.Text(c, "选择一篇文章阅读，或点击左上角“+ 写文章”开始创作").FontSize(13).TextColor(t.TextMuted)
			if site := a.currentSite(); site != nil {
				if ui.PrimaryButton(c.Key("empty-compose"), "写新文章").Clicked() {
					a.startNewDraft(site.ID)
				}
			}
		})
		return
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).PaddingX(16).PaddingY(10).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(t.Surface).Shrink(0).Children(func() {
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, fallbackTitle(art.Title)).Bold().FontSize(15)
				var meta []string
				if !art.Modified.IsZero() {
					meta = append(meta, "更新于 "+art.Modified.Local().Format("2006-01-02 15:04"))
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
					ui.Text(c, strings.Join(meta, "  ·  ")).FontSize(11).TextColor(t.TextMuted)
				}
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				pinLabel := "置顶"
				if art.Pinned != nil {
					pinLabel = "取消置顶"
				}
				if ui.Button(c.Key("reader-related"), "相关文章").Clicked() {
					a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
				}
				if ui.Button(c.Key("reader-pin"), pinLabel).Clicked() {
					a.toggleArticlePinned(*art)
				}
				if ui.Button(c.Key("reader-settings"), "文章设置").Clicked() {
					a.openArticleSettings(*art)
				}
				if ui.PrimaryButton(c.Key("reader-edit"), "编辑文章").Clicked() {
					a.openArticleForEdit(*art)
				}
				if ui.Button(c.Key("reader-delete"), "删除").Clicked() {
					a.deleteArticleID = art.ID
					a.deleteArticleTitle = art.Title
					a.deleteArticleOpen = true
				}
			})
		})
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
		a.viewMarkdownPreview(c, "", art.Content, &a.readerScroll)
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
