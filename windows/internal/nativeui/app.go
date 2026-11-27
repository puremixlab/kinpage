package nativeui

import (
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/egoist/mygo/ui/markdown"
	"windows/internal/store"
)

type viewMode int

const (
	viewOverview viewMode = iota
	viewSite
	viewToday
	viewUnread
	viewStarred
	viewSubscription
)

const (
	editorPaneWrite = 0
	editorPaneSplit = 1
	editorPaneView  = 2
)

type listFilterKind int

const (
	listFilterAll listFilterKind = iota
	listFilterPages
	listFilterNav
	listFilterPinned
	listFilterUnread
	listFilterStarred
)

// App holds the UI state for the MyGo Native UI shell and editor.
type App struct {
	backend Backend

	mode           viewMode
	selectedSiteID string

	dashboard         store.DashboardSnapshot
	activity          []store.DailyActivity
	sites             []store.Site
	drafts            []store.Draft
	articles          []store.Article
	subscriptions     []store.Subscription
	articleExcerpts   map[string]string
	articleWordCounts map[string]int

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
	sidebarWidth      float32
	editorSplitRatio  float32
	postSearchQuery   string
	listSearchOpen    bool
	listFilter        listFilterKind
	starTypeFilter    int // -1 = all star types, 0..5 = specific store.StarType
	archivedExpanded  bool

	// Unsaved draft navigation guard.
	unsavedDialogOpen bool
	pendingNavAction  func()

	// Create-site modal state.
	createSiteOpen  bool
	newSiteName     string
	newSiteAbout    string
	createSiteError string

	// Article settings modal state.
	articleSettingsOpen     bool
	articleSettingsID       string
	articleSettingsSlug     string
	articleSettingsType     int
	articleSettingsExtLink  string
	articleSettingsInNav    bool
	articleSettingsWeight   string
	articleSettingsDate     string
	articleSettingsOrigDate string
	articleSettingsErr      string

	// Move article sheet state.
	moveArticleOpen         bool
	moveArticleID           string
	moveArticleTitle        string
	moveArticleFromSiteID   string
	moveArticleTargetSiteID string
	moveArticleErr          string

	// Delete article confirmation dialog state.
	deleteArticleOpen  bool
	deleteArticleID    string
	deleteArticleTitle string

	// Quick post modal state.
	quickPostOpen           bool
	quickPostSiteID         string
	quickPostTitle          string
	quickPostContent        string
	quickPostAttachmentName string
	quickPostAttachmentB64  string
	quickPostDiscardConfirm bool
	quickPostErr            string

	// Dashboard interaction state.
	activityMode        int
	trendRange          int
	trendHoverPoint     int
	articleFilterSiteID string
	composePickerOpen   bool
	badgeWallOpen       bool
	badgesHidden        bool
	badgePositions      map[string]ui.Point
	badgeDragOffsets    map[string]*ui.Point
	badgeZOrder         []string
	badgeDraggingID     string

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
	siteSettings     siteSettingsState
	publishModal     publishModalState
	ipfsModal        ipfsModalState
	offlineModal     offlineModalState
	foldersModal     foldersModalState
	removedSites     removedSitesState
	openIPFSResOpen  bool
	openIPFSResInput string
	openIPFSResErr   string
	tipModalOpen     bool
	tipRecipient     string
	tipAmount        string
	tipErr           string
	externalDirty    atomic.Bool

	// Batch 3B & 3C modals state.
	integrations    integrationsModalState
	toolsModal      toolsModalState
	templatePreview templatePreviewState

	// Native platform hooks (TTS, external browser preview, embedded WebView, image loader).
	speaker          Speaker
	openExternal     func(string) error
	newWebView       WebViewFactory
	imgLoader        *imageLoader
	subShowWebView   bool
	subWebView       WebViewHost
	subWebViewURL    string
	pal              macPalette
	locale           Locale
	onLocaleChange   func(Locale)
	bitmapCache      map[bitmapDataKey]*ui.Bitmap
	settingsMenuOpen bool
	sidebarCollapsed bool
	postsCollapsed   bool
	prefsPath        string
	lastSavedPrefs   layoutPrefs
	siteOrder        []string
	subOrder         []string

	// Feedback status (dispatched as non-blocking macOS bottom-right toasts).
	statusText     string
	statusIsErr    bool
	statusSeq      int
	shownStatusSeq int
	loaded         bool
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
			Font:           ui.Font{Family: macPlanetFont, Size: 15},
			LineHeight:     1.8,
			SoftWrap:       true,
			LineNumbers:    editor.LineNumbersOff,
			NoIndentGuides: true,
			NoFolding:      true,
			NoActiveLine:   true,
			Placeholder:    "开始写点什么…",
			Label:          "Markdown 正文编辑器",
		},
		postsPanelWidth:  276,
		sidebarWidth:     228,
		editorSplitRatio: 50,
		trendHoverPoint:  -1,
		starTypeFilter:   -1,
		speaker:          newDefaultSpeaker(),
		openExternal:     openBrowserFile,
		imgLoader:        newImageLoader(),
		locale:           LocaleZhCN,
	}
	a.syncEditorLocale()
	a.Reload()
	return a
}

// Close stops any active native background playback and releases embedded WebViews.
func (a *App) Close() {
	if a.speaker != nil {
		a.speaker.Stop()
	}
	a.destroyTemplatePreviewWV()
	a.destroySubWebView()
}

// MarkDirtyExternal flags that library.json was modified outside Planet so the
// next UI frame reloads cleanly on the main thread.
func (a *App) MarkDirtyExternal() {
	a.externalDirty.Store(true)
}

func (a *App) toggleSpeak(title, body string) {
	if a.speaker == nil {
		return
	}
	if a.speaker.Speaking() {
		a.speaker.Stop()
		a.setStatus(a.t("status.stoppedSpeak"))
		return
	}
	text := strings.TrimSpace(title + "\n\n" + body)
	if text == "" {
		return
	}
	if err := a.speaker.Speak(text); err != nil {
		a.setError(a.tf("status.speakErr", err.Error()))
		return
	}
	a.setStatus(a.t("status.speaking"))
}

// Reload refreshes the dashboard snapshot, writing activity, site list, and
// currently selected site's drafts and articles.
func (a *App) Reload() {
	if a.backend == nil {
		return
	}
	a.loaded = true
	if dash, err := a.backend.Dashboard(); err != nil {
		a.setError(a.tf("status.loadOverviewErr", err.Error()))
	} else {
		a.dashboard = dash
	}
	if act, err := a.backend.WritingActivity(); err == nil {
		a.activity = act
	}
	if sites, err := a.backend.Sites(); err != nil {
		a.setError(a.tf("status.loadSitesErr", err.Error()))
	} else {
		a.sites = sites
	}
	if subs, err := a.backend.Subscriptions(); err == nil {
		a.subscriptions = subs
	}
	if st, err := a.backend.IPFSStatus(); err == nil {
		a.ipfsModal.status = st
	}
	a.refreshDashboardExcerpts()
	if a.selectedSiteID != "" {
		a.loadSiteItems(a.selectedSiteID)
	}
}

func (a *App) refreshDashboardExcerpts() {
	capHint := len(a.dashboard.RecentArticles) + len(a.dashboard.RecentDrafts)
	excerpts := make(map[string]string, capHint)
	wordCounts := make(map[string]int, capHint)
	seenArtSites := map[string]bool{}
	for _, entry := range a.dashboard.RecentArticles {
		if entry.SiteID == "" || seenArtSites[entry.SiteID] {
			continue
		}
		seenArtSites[entry.SiteID] = true
		if arts, err := a.backend.Articles(entry.SiteID); err == nil {
			for _, art := range arts {
				k := "art:" + art.SiteID + ":" + art.ID
				excerpts[k] = excerptText(art.Content, 120)
				wordCounts[k] = countBodyChars(art.Content)
			}
		}
	}
	seenDraftSites := map[string]bool{}
	for _, entry := range a.dashboard.RecentDrafts {
		if entry.SiteID == "" || seenDraftSites[entry.SiteID] {
			continue
		}
		seenDraftSites[entry.SiteID] = true
		if drafts, err := a.backend.Drafts(entry.SiteID); err == nil {
			for _, d := range drafts {
				k := "draft:" + d.SiteID + ":" + d.ID
				excerpts[k] = excerptText(d.Content, 140)
				wordCounts[k] = countBodyChars(d.Content)
			}
		}
	}
	a.articleExcerpts = excerpts
	a.articleWordCounts = wordCounts
}

func countBodyChars(raw string) int {
	n := 0
	for _, r := range raw {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			n++
		}
	}
	return n
}

func (a *App) loadSiteItems(siteID string) {
	if a.backend == nil || siteID == "" {
		a.drafts = nil
		a.articles = nil
		return
	}
	drafts, err := a.backend.Drafts(siteID)
	if err != nil {
		a.setError(a.tf("status.loadDraftsErr", err.Error()))
	} else {
		a.drafts = drafts
	}
	articles, err := a.backend.Articles(siteID)
	if err != nil {
		a.setError(a.tf("status.loadArticlesErr", err.Error()))
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
	a.statusSeq++
}

func (a *App) setError(msg string) {
	a.statusText = msg
	a.statusIsErr = true
	a.statusSeq++
}

func (a *App) clearStatus() {
	a.statusText = ""
	a.statusIsErr = false
	a.shownStatusSeq = a.statusSeq
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
	a.listSearchOpen = false
	a.listFilter = listFilterAll
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
	a.setStatus(a.t("status.newDraft"))
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

func (a *App) openDraftByID(siteID, draftID string) {
	a.selectSite(siteID)
	for _, d := range a.drafts {
		if d.ID == draftID {
			a.openDraft(d)
			return
		}
	}
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
	a.setStatus(a.t("status.editingPublished"))
}

func (a *App) saveCurrentDraft() bool {
	if !a.editing || a.selectedSiteID == "" || a.backend == nil {
		return false
	}
	tagsJSON, err := a.marshalEditorTags()
	if err != nil {
		a.setError(a.tf("status.marshalTagsErr", err.Error()))
		return false
	}
	content := a.editorBuffer.String()
	draft, err := a.backend.SaveDraftWithTags(a.selectedSiteID, a.activeDraftID, a.activeArticleID, a.editorTitle, content, tagsJSON)
	if err != nil {
		a.setError(a.tf("status.saveDraftErr", err.Error()))
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
	a.setStatus(a.tf("status.draftSaved", time.Now().Format("15:04:05")))
	return true
}

func (a *App) publishCurrentDraft() {
	if !a.editing || a.selectedSiteID == "" || a.backend == nil {
		return
	}
	tagsJSON, err := a.marshalEditorTags()
	if err != nil {
		a.setError(a.tf("status.marshalTagsErr", err.Error()))
		return
	}
	content := a.editorBuffer.String()
	art, err := a.backend.PublishDraftWithTags(a.selectedSiteID, a.activeDraftID, a.editorTitle, content, tagsJSON)
	if err != nil {
		a.setError(a.tf("status.publishErr", err.Error()))
		return
	}
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.selectedReadID = art.ID
	a.editorTagsDirty = false
	a.editorBuffer.MarkSaved()
	a.Reload()
	a.setStatus(a.tf("status.published", a.fallbackTitle(art.Title)))
}

func (a *App) deleteCurrentDraft() {
	if !a.editing || a.selectedSiteID == "" || a.activeDraftID == "" || a.backend == nil {
		return
	}
	if err := a.backend.DeleteDraft(a.selectedSiteID, a.activeDraftID); err != nil {
		a.setError(a.tf("status.deleteDraftErr", err.Error()))
		return
	}
	a.editing = false
	a.activeDraftID = ""
	a.activeArticleID = ""
	a.editorTagsDirty = false
	a.Reload()
	a.setStatus(a.t("status.draftDeleted"))
}

func (a *App) currentSite() *store.Site {
	return a.findSiteByID(a.selectedSiteID)
}

func (a *App) findSiteByID(id string) *store.Site {
	for i := range a.sites {
		if a.sites[i].ID == id {
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

func (a *App) anyModalOpen() bool {
	return a.createSiteOpen || a.composePickerOpen || a.badgeWallOpen || a.articleSettingsOpen ||
		a.moveArticleOpen || a.quickPostOpen || a.deleteArticleOpen || a.unsavedDialogOpen ||
		a.followModalOpen || a.editSubOpen || a.unfollowOpen ||
		a.searchModalOpen || a.relatedModalOpen ||
		a.siteSettings.open || a.siteSettings.removeConfirm ||
		a.publishModal.open || a.ipfsModal.open || a.offlineModal.open ||
		a.openIPFSResOpen || a.tipModalOpen || a.foldersModal.open || a.removedSites.open ||
		a.toolsModal.open || a.templatePreview.open
}

func fallbackTitle(s string) string {
	if strings.TrimSpace(s) == "" {
		return "无标题"
	}
	return s
}
