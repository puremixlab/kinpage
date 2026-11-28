package nativeui

import (
	"context"
	"time"

	"windows/internal/applog"
	"windows/internal/ipfs"
	"windows/internal/localapi"
	"windows/internal/offline"
	"windows/internal/publisher"
	"windows/internal/services"
	"windows/internal/siteimage"
	"windows/internal/store"
	"windows/internal/templateinstall"
)

// DeploymentSummary mirrors the backend's persisted deployment configuration
// and status for a single site and target kind ("cloudflare" or "ssh").
type DeploymentSummary struct {
	Configured   bool
	Name         string
	Server       string
	AutoDeploy   bool
	DeleteRemote bool
	Pending      bool
	LastURL      string
}

// Backend abstracts the Planet Library operations needed by the native UI.
// Having this interface lets the native UI run directly against *Library in
// production and against an in-memory stub in headless ui.Tester tests.
type Backend interface {
	Dashboard() (store.DashboardSnapshot, error)
	WritingActivity() ([]store.DailyActivity, error)
	Sites() ([]store.Site, error)
	CreateSite(name, about string) (store.Site, error)
	UpdateSiteDetails(siteID, name, about string, doNotIndex bool, slug, author string) (store.Site, error)
	UpdateSiteProfile(siteID, profileJSON string) (store.Site, error)
	UpdateSiteCustomCode(siteID, customCodeJSON string) (store.Site, error)
	UpdatePodcastSettings(siteID, language string, explicit bool, categoriesJSON string) (store.Site, error)
	SelectSiteAvatar(ctx context.Context, siteID string) (store.Site, error)
	RemoveSiteAvatar(siteID string) (store.Site, error)
	PresetAvatars() []siteimage.PresetAvatar
	ApplyPresetAvatar(siteID, key string) (store.Site, error)
	SetSaveRoundAvatar(siteID string, round bool) (store.Site, error)
	RemoveSite(siteID string) error
	SetSiteArchived(siteID string, archived bool) (store.Site, error)
	RemovedSites() ([]store.Site, error)
	RestoreSite(siteID string) (store.Site, error)
	PermanentlyDeleteSite(siteID string) error
	PublishSite(siteID string) (publisher.Result, error)
	IPFSStatus() (ipfs.KuboStatus, error)
	SetIPFSEnabled(enabled bool) error
	IPFSRepoSize() (int64, error)
	IPFSGarbageCollect() error
	ResolveIPFSResourceURL(input string) (string, error)
	RebuildSite(siteID string, quick bool) (int, error)
	PreventSleep() bool
	SetPreventSleep(enabled bool) error
	DataDir() string
	MoveDataDir(ctx context.Context) (string, error)
	OfflineMediaUsage() (offline.Usage, error)
	CleanupOfflineMedia() (offline.Usage, error)
	PublishedFolders() ([]store.PublishedFolder, error)
	AddPublishedFolder(ctx context.Context, name string, autoPublish bool) (store.PublishedFolder, error)
	PublishFolder(folderID string) (store.PublishedFolder, error)
	RemovePublishedFolder(folderID string) error
	BuiltinTemplates() []templateinstall.Builtin
	BuiltinTemplatePreview(id string) (string, error)
	ApplyBuiltinTemplate(siteID, id string) (store.Site, error)
	SelectTemplate(ctx context.Context, siteID string) (store.Site, error)
	TemplateSettings(siteID string) (map[string]string, error)
	SaveTemplateSettings(siteID, values string) (store.Site, error)
	DeploymentSummary(siteID, kind string) (DeploymentSummary, error)
	ConfigureDeployment(siteID, kind, name, server, token string, auto, deleteRemote bool) error
	DisableDeployment(siteID, kind string) error
	DeploySite(siteID, kind string) (string, error)
	ConfigureService(siteID, kind, name, token string) error
	ConfigurePlausibleAt(siteID, domain, token, scriptID, server string) error
	RequestFilebasePin(siteID string) error
	ConfigurePinnable(siteID, endpoint string) error
	RequestPinnablePin(siteID string) error
	ConfigureXSync(siteID, apiKey, apiSecret, accessToken, accessTokenSecret string, autoSync bool) error
	DisableXSync(siteID string) error
	ConfigureTelegram(siteID, chatID, token string) error
	DisableTelegram(siteID string) error
	RetryTelegram(siteID string) (int, error)
	UpdateSiteAggregation(siteID, sourcesText string) (store.Site, error)
	SyncSiteAggregation(siteID string) (int, error)
	UpdateSiteWeb3(siteID, web3JSON string) (store.Site, error)
	AIConfig() (services.AIConfig, error)
	ConfigureAI(baseURL, model, token string) (services.AIConfig, error)
	ProbeAI(baseURL string) (services.AIProbeResult, error)
	AskAI(prompt, contextTitle, contextBody, historyJSON string) (string, error)
	LocalAPIStatus() (localapi.Status, error)
	ConfigureLocalAPI(enabled bool, port int, authEnabled bool, username, passcode string) (localapi.Status, error)
	ClearLocalAPILogs()
	ListIPNSKeys() ([]ipfs.KeyInfo, error)
	ExportIPNSKey(ctx context.Context, name string) (string, error)
	ImportIPNSKey(ctx context.Context, name string) (string, error)
	AppLogs(category string) []applog.Entry
	ClearAppLogs(category string)
	Articles(siteID string) ([]store.Article, error)
	Drafts(siteID string) ([]store.Draft, error)
	SaveDraft(siteID, draftID, articleID, title, content string) (store.Draft, error)
	SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (store.Draft, error)
	PublishDraft(siteID, draftID, title, content string) (store.Article, error)
	PublishDraftWithTags(siteID, draftID, title, content, tagsJSON string) (store.Article, error)
	DeleteDraft(siteID, draftID string) error
	AddDraftAttachment(ctx context.Context, siteID, draftID string) (store.Draft, error)
	RemoveDraftAttachment(siteID, draftID, name string) (store.Draft, error)
	SetDraftHeroImage(siteID, draftID, name string) (store.Draft, error)
	CompressDraftVideo(siteID, draftID, name, preset string) (store.Draft, error)
	SetArticlePinned(siteID, articleID string, pinned bool) (store.Article, error)
	ToggleArticleTodo(siteID, articleID string, lineIndex int) (store.Article, error)
	UpdateArticleSettings(siteID, articleID, slug string, articleType int, externalLink string, inNavigation bool, weight int) (store.Article, error)
	UpdateArticleDate(siteID, articleID string, created time.Time) (store.Article, error)
	SetArticleStar(siteID, articleID string, starred bool, starType int) (store.Article, error)
	MoveArticle(fromSiteID, articleID, toSiteID string) (store.Article, error)
	ExportArticle(ctx context.Context, siteID, articleID string) (string, error)
	ImportArticle(ctx context.Context, siteID string) (store.Article, error)
	DeleteArticle(siteID, articleID string) error
	QuickPost(siteID, title, content, attachmentName, attachmentBase64 string) (store.Article, error)
	PickQuickPostAttachment(ctx context.Context) (string, string, error)
	Subscriptions() ([]store.Subscription, error)
	ReadMarkdownAttachment(siteID, articleID, draftID, name string) ([]byte, string, error)
	FollowContext(ctx context.Context, address string) (SubscriptionUpdate, error)
	RefreshSubscriptionContext(ctx context.Context, address string) (SubscriptionUpdate, error)
	UpdateSubscriptionContext(ctx context.Context, oldAddress, newAddress, customTitle string) (SubscriptionUpdate, error)
	Unfollow(address string) error
	StarSubscription(address, key string, starred bool) error
	SetSubscriptionStar(address, key string, starred bool, starType int) error
	SetSubscriptionRead(address, key string, read bool) error
	MarkSubscriptionRead(address string) error
	SearchArticles(query string) ([]store.SearchResult, error)
	RelatedArticles(siteID, articleID, subscriptionURL, itemKey string) ([]store.SearchResult, error)
}
