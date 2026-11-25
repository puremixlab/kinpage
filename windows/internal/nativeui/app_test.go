package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/applog"
	"windows/internal/feeds"
	"windows/internal/ipfs"
	"windows/internal/localapi"
	"windows/internal/offline"
	"windows/internal/publisher"
	"windows/internal/services"
	"windows/internal/siteimage"
	"windows/internal/store"
	"windows/internal/templateinstall"
)

type fakeBackend struct {
	sites         []store.Site
	removedSites  []store.Site
	articles      []store.Article
	drafts        []store.Draft
	activity      []store.DailyActivity
	subscriptions []store.Subscription
	folders       []store.PublishedFolder
	ipfsStatus    ipfs.KuboStatus
	ipfsRepoSize  int64
	offlineUsage  offline.Usage
	gcCount       int
	deployCalls   []string
	xConfigured   bool
	tgConfigured  bool
	aggSynced     int
	aiConfig      services.AIConfig
	apiStatus     localapi.Status
	ipnsKeys      []ipfs.KeyInfo
	appLogs       []applog.Entry
	nextID        int
	nextFile      string
}

func (f *fakeBackend) Dashboard() (store.DashboardSnapshot, error) {
	snap := store.DashboardSnapshot{
		SiteCount:    len(f.sites),
		ArticleCount: len(f.articles),
		DraftCount:   len(f.drafts),
	}
	for _, s := range f.sites {
		snap.Sites = append(snap.Sites, store.DashboardSite{ID: s.ID, Name: s.Name})
	}
	for _, d := range f.drafts {
		snap.RecentDrafts = append(snap.RecentDrafts, store.DashboardEntry{
			ID: d.ID, SiteID: d.SiteID, SiteName: f.siteName(d.SiteID), Title: d.Title, Modified: d.Modified,
		})
	}
	for _, a := range f.articles {
		snap.RecentArticles = append(snap.RecentArticles, store.DashboardEntry{
			ID: a.ID, SiteID: a.SiteID, SiteName: f.siteName(a.SiteID), Title: a.Title, Modified: a.Modified,
		})
	}
	return snap, nil
}

func (f *fakeBackend) siteName(id string) string {
	for _, s := range f.sites {
		if s.ID == id {
			return s.Name
		}
	}
	return ""
}

func (f *fakeBackend) WritingActivity() ([]store.DailyActivity, error) {
	return append([]store.DailyActivity(nil), f.activity...), nil
}

func (f *fakeBackend) Sites() ([]store.Site, error) {
	return append([]store.Site(nil), f.sites...), nil
}

func (f *fakeBackend) CreateSite(name, about string) (store.Site, error) {
	f.nextID++
	s := store.Site{ID: fmt.Sprintf("site-%d", f.nextID), Name: name, About: about, Created: time.Now()}
	f.sites = append(f.sites, s)
	return s, nil
}

func (f *fakeBackend) UpdateSiteDetails(siteID, name, about string, doNotIndex bool, slug, author string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].Name = name
			f.sites[i].About = about
			f.sites[i].DoNotIndex = doNotIndex
			f.sites[i].Slug = slug
			f.sites[i].AuthorName = author
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) UpdateSiteProfile(siteID, profileJSON string) (store.Site, error) {
	var p store.SiteProfile
	if err := json.Unmarshal([]byte(profileJSON), &p); err != nil {
		return store.Site{}, err
	}
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].Domain = p.Domain
			f.sites[i].TwitterUsername = p.TwitterUsername
			f.sites[i].GithubUsername = p.GithubUsername
			f.sites[i].TelegramUsername = p.TelegramUsername
			f.sites[i].MastodonUsername = p.MastodonUsername
			f.sites[i].DiscordLink = p.DiscordLink
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) UpdateSiteCustomCode(siteID, customCodeJSON string) (store.Site, error) {
	var cfg store.SiteCustomCode
	if err := json.Unmarshal([]byte(customCodeJSON), &cfg); err != nil {
		return store.Site{}, err
	}
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].CustomCodeHeadEnabled = cfg.HeadEnabled
			f.sites[i].CustomCodeHead = cfg.Head
			f.sites[i].CustomCodeBodyStartEnabled = cfg.BodyStartEnabled
			f.sites[i].CustomCodeBodyStart = cfg.BodyStart
			f.sites[i].CustomCodeBodyEndEnabled = cfg.BodyEndEnabled
			f.sites[i].CustomCodeBodyEnd = cfg.BodyEnd
			f.sites[i].PublishAsIPNS = cfg.PublishAsIPNS
			f.sites[i].PrewarmNewPost = cfg.PrewarmNewPost
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) UpdatePodcastSettings(siteID, language string, explicit bool, categoriesJSON string) (store.Site, error) {
	var cats []string
	if err := json.Unmarshal([]byte(categoriesJSON), &cats); err != nil {
		return store.Site{}, err
	}
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].PodcastLanguage = language
			f.sites[i].PodcastExplicit = explicit
			f.sites[i].PodcastCategories = cats
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) SelectSiteAvatar(_ context.Context, siteID string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].AvatarPNG = "fake-png"
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) RemoveSiteAvatar(siteID string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].AvatarPNG = ""
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) PresetAvatars() []siteimage.PresetAvatar {
	return []siteimage.PresetAvatar{{Key: "earth", Title: "地球"}}
}

func (f *fakeBackend) ApplyPresetAvatar(siteID, key string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].AvatarPNG = "preset-" + key
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) SetSaveRoundAvatar(siteID string, round bool) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].SaveRoundAvatar = round
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) RemoveSite(siteID string) error {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.removedSites = append(f.removedSites, f.sites[i])
			f.sites = append(f.sites[:i], f.sites[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) RemovedSites() ([]store.Site, error) {
	return append([]store.Site(nil), f.removedSites...), nil
}

func (f *fakeBackend) RestoreSite(siteID string) (store.Site, error) {
	for i := range f.removedSites {
		if f.removedSites[i].ID == siteID {
			s := f.removedSites[i]
			f.removedSites = append(f.removedSites[:i], f.removedSites[i+1:]...)
			f.sites = append(f.sites, s)
			return s, nil
		}
	}
	return store.Site{}, fmt.Errorf("removed site not found")
}

func (f *fakeBackend) PermanentlyDeleteSite(siteID string) error {
	for i := range f.removedSites {
		if f.removedSites[i].ID == siteID {
			f.removedSites = append(f.removedSites[:i], f.removedSites[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) PublishSite(siteID string) (publisher.Result, error) {
	return publisher.Result{CID: "bafy-test-cid", IPNS: "k51-test-ipns", Revision: 3}, nil
}

func (f *fakeBackend) IPFSStatus() (ipfs.KuboStatus, error) { return f.ipfsStatus, nil }
func (f *fakeBackend) SetIPFSEnabled(enabled bool) error {
	f.ipfsStatus.Disabled = !enabled
	f.ipfsStatus.Running = enabled
	return nil
}
func (f *fakeBackend) IPFSRepoSize() (int64, error) { return f.ipfsRepoSize, nil }
func (f *fakeBackend) IPFSGarbageCollect() error    { f.gcCount++; return nil }

func (f *fakeBackend) OfflineMediaUsage() (offline.Usage, error) { return f.offlineUsage, nil }
func (f *fakeBackend) CleanupOfflineMedia() (offline.Usage, error) {
	u := f.offlineUsage
	u.ReclaimedBytes = u.ReclaimableBytes
	u.ReclaimableBytes = 0
	u.ReclaimableFiles = 0
	f.offlineUsage = u
	return u, nil
}

func (f *fakeBackend) PublishedFolders() ([]store.PublishedFolder, error) {
	return append([]store.PublishedFolder(nil), f.folders...), nil
}

func (f *fakeBackend) AddPublishedFolder(_ context.Context, name string, autoPublish bool) (store.PublishedFolder, error) {
	f.nextID++
	if name == "" {
		name = "共享目录"
	}
	folder := store.PublishedFolder{ID: fmt.Sprintf("fld-%d", f.nextID), Name: name, Path: `D:\share`, AutoPublish: autoPublish}
	f.folders = append(f.folders, folder)
	return folder, nil
}

func (f *fakeBackend) PublishFolder(folderID string) (store.PublishedFolder, error) {
	for i := range f.folders {
		if f.folders[i].ID == folderID {
			f.folders[i].LastCID = "bafy-folder"
			return f.folders[i], nil
		}
	}
	return store.PublishedFolder{}, fmt.Errorf("folder not found")
}

func (f *fakeBackend) RemovePublishedFolder(folderID string) error {
	for i := range f.folders {
		if f.folders[i].ID == folderID {
			f.folders = append(f.folders[:i], f.folders[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) BuiltinTemplates() []templateinstall.Builtin {
	return []templateinstall.Builtin{{ID: "plain", Name: "Plain", Description: "简洁文字博客"}}
}

func (f *fakeBackend) BuiltinTemplatePreview(id string) (string, error) {
	return "<!doctype html><html><body><h1>" + id + "</h1></body></html>", nil
}

func (f *fakeBackend) ApplyBuiltinTemplate(siteID, id string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].TemplateID = id
			f.sites[i].TemplateName = "Plain"
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) SelectTemplate(_ context.Context, siteID string) (store.Site, error) {
	return f.ApplyBuiltinTemplate(siteID, "custom")
}

func (f *fakeBackend) TemplateSettings(siteID string) (map[string]string, error) {
	return map[string]string{"accentColor": "#2563eb"}, nil
}

func (f *fakeBackend) SaveTemplateSettings(siteID, values string) (store.Site, error) {
	var m map[string]string
	if err := json.Unmarshal([]byte(values), &m); err != nil {
		return store.Site{}, err
	}
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].TemplateSettings = m
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) ConfigureDeployment(_, kind, _, _, _ string, _, _ bool) error {
	f.deployCalls = append(f.deployCalls, "cfg:"+kind)
	return nil
}
func (f *fakeBackend) DisableDeployment(_, kind string) error {
	f.deployCalls = append(f.deployCalls, "off:"+kind)
	return nil
}
func (f *fakeBackend) DeploySite(_, kind string) (string, error) {
	f.deployCalls = append(f.deployCalls, "run:"+kind)
	return "ok", nil
}
func (f *fakeBackend) ConfigureService(_, _, _, _ string) error        { return nil }
func (f *fakeBackend) ConfigurePlausibleAt(_, _, _, _, _ string) error { return nil }
func (f *fakeBackend) RequestFilebasePin(_ string) error               { return nil }
func (f *fakeBackend) ConfigurePinnable(_, _ string) error             { return nil }
func (f *fakeBackend) RequestPinnablePin(_ string) error               { return nil }
func (f *fakeBackend) ConfigureXSync(_, _, _, _, _ string, _ bool) error {
	f.xConfigured = true
	return nil
}
func (f *fakeBackend) DisableXSync(_ string) error            { f.xConfigured = false; return nil }
func (f *fakeBackend) ConfigureTelegram(_, _, _ string) error { f.tgConfigured = true; return nil }
func (f *fakeBackend) DisableTelegram(_ string) error         { f.tgConfigured = false; return nil }
func (f *fakeBackend) RetryTelegram(_ string) (int, error)    { return 2, nil }
func (f *fakeBackend) UpdateSiteAggregation(siteID, sources string) (store.Site, error) {
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].Aggregation = strings.Fields(sources)
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}
func (f *fakeBackend) SyncSiteAggregation(_ string) (int, error) { f.aggSynced++; return 3, nil }
func (f *fakeBackend) UpdateSiteWeb3(siteID, web3JSON string) (store.Site, error) {
	var cfg store.SiteWeb3Settings
	if err := json.Unmarshal([]byte(web3JSON), &cfg); err != nil {
		return store.Site{}, err
	}
	for i := range f.sites {
		if f.sites[i].ID == siteID {
			f.sites[i].AcceptsDonation = cfg.AcceptsDonation
			f.sites[i].AcceptsDonationETHAddress = cfg.AcceptsDonationETHAddress
			f.sites[i].JuiceboxEnabled = cfg.JuiceboxEnabled
			f.sites[i].JuiceboxProjectID = cfg.JuiceboxProjectID
			return f.sites[i], nil
		}
	}
	return store.Site{}, fmt.Errorf("site not found")
}

func (f *fakeBackend) AIConfig() (services.AIConfig, error) { return f.aiConfig, nil }
func (f *fakeBackend) ConfigureAI(baseURL, model, token string) (services.AIConfig, error) {
	f.aiConfig = services.AIConfig{BaseURL: baseURL, Model: model, HasKey: token != ""}
	return f.aiConfig, nil
}
func (f *fakeBackend) ProbeAI(_ string) (services.AIProbeResult, error) {
	return services.AIProbeResult{OllamaDetected: true, Models: []string{"qwen2.5", "llama3"}}, nil
}
func (f *fakeBackend) AskAI(prompt, _, _, _ string) (string, error) {
	return "**回复**：" + prompt, nil
}
func (f *fakeBackend) LocalAPIStatus() (localapi.Status, error) { return f.apiStatus, nil }
func (f *fakeBackend) ConfigureLocalAPI(enabled bool, port int, authEnabled bool, username, passcode string) (localapi.Status, error) {
	f.apiStatus = localapi.Status{Running: enabled, Enabled: enabled, Port: port, AuthEnabled: authEnabled, Username: username, HasPasscode: passcode != ""}
	return f.apiStatus, nil
}
func (f *fakeBackend) ClearLocalAPILogs() { f.apiStatus.Logs = nil }
func (f *fakeBackend) ListIPNSKeys() ([]ipfs.KeyInfo, error) {
	return append([]ipfs.KeyInfo(nil), f.ipnsKeys...), nil
}
func (f *fakeBackend) ExportIPNSKey(_ context.Context, name string) (string, error) {
	return `D:\keys\` + name + ".key", nil
}
func (f *fakeBackend) ImportIPNSKey(_ context.Context, name string) (string, error) {
	k := ipfs.KeyInfo{Name: name, ID: "k51-" + name}
	f.ipnsKeys = append(f.ipnsKeys, k)
	return k.ID, nil
}
func (f *fakeBackend) AppLogs(category string) []applog.Entry {
	var out []applog.Entry
	for _, e := range f.appLogs {
		if category == "" || e.Category == category {
			out = append(out, e)
		}
	}
	return out
}
func (f *fakeBackend) ClearAppLogs(_ string) { f.appLogs = nil }

func (f *fakeBackend) Articles(siteID string) ([]store.Article, error) {
	var out []store.Article
	for _, a := range f.articles {
		if a.SiteID == siteID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeBackend) Drafts(siteID string) ([]store.Draft, error) {
	var out []store.Draft
	for _, d := range f.drafts {
		if d.SiteID == siteID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeBackend) SaveDraft(siteID, draftID, articleID, title, content string) (store.Draft, error) {
	return f.SaveDraftWithTags(siteID, draftID, articleID, title, content, "{}")
}

func (f *fakeBackend) SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (store.Draft, error) {
	var tags map[string]string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	now := time.Now()
	for i := range f.drafts {
		if f.drafts[i].ID == draftID && draftID != "" {
			f.drafts[i].Title = title
			f.drafts[i].Content = content
			f.drafts[i].Tags = tags
			f.drafts[i].Modified = now
			return f.drafts[i], nil
		}
	}
	f.nextID++
	d := store.Draft{
		ID: fmt.Sprintf("draft-%d", f.nextID), SiteID: siteID, ArticleID: articleID,
		Title: title, Content: content, Tags: tags, Created: now, Modified: now,
	}
	f.drafts = append(f.drafts, d)
	return d, nil
}

func (f *fakeBackend) PublishDraft(siteID, draftID, title, content string) (store.Article, error) {
	return f.PublishDraftWithTags(siteID, draftID, title, content, "{}")
}

func (f *fakeBackend) PublishDraftWithTags(siteID, draftID, title, content, tagsJSON string) (store.Article, error) {
	var tags map[string]string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	now := time.Now()
	if draftID != "" {
		_ = f.DeleteDraft(siteID, draftID)
	}
	f.nextID++
	art := store.Article{
		ID: fmt.Sprintf("art-%d", f.nextID), SiteID: siteID,
		Title: title, Content: content, Tags: tags, Created: now, Modified: now,
	}
	f.articles = append(f.articles, art)
	return art, nil
}

func (f *fakeBackend) DeleteDraft(siteID, draftID string) error {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts = append(f.drafts[:i], f.drafts[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) AddDraftAttachment(_ context.Context, siteID, draftID string) (store.Draft, error) {
	name := f.nextFile
	if name == "" {
		name = "cover.png"
	}
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].Attachments = append(f.drafts[i].Attachments, name)
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) RemoveDraftAttachment(siteID, draftID, name string) (store.Draft, error) {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].Attachments = slices.DeleteFunc(f.drafts[i].Attachments, func(s string) bool { return s == name })
			if f.drafts[i].HeroImage == name {
				f.drafts[i].HeroImage = ""
			}
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) SetDraftHeroImage(siteID, draftID, name string) (store.Draft, error) {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].HeroImage = name
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) SetArticlePinned(siteID, articleID string, pinned bool) (store.Article, error) {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			if pinned {
				now := time.Now()
				f.articles[i].Pinned = &now
			} else {
				f.articles[i].Pinned = nil
			}
			return f.articles[i], nil
		}
	}
	return store.Article{}, fmt.Errorf("article not found")
}

func (f *fakeBackend) UpdateArticleSettings(siteID, articleID, slug string, articleType int, externalLink string, inNavigation bool, weight int) (store.Article, error) {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			f.articles[i].Slug = slug
			f.articles[i].ArticleType = articleType
			f.articles[i].ExternalLink = externalLink
			f.articles[i].IsIncludedInNavigation = inNavigation
			f.articles[i].NavigationWeight = weight
			return f.articles[i], nil
		}
	}
	return store.Article{}, fmt.Errorf("article not found")
}

func (f *fakeBackend) DeleteArticle(siteID, articleID string) error {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			f.articles = append(f.articles[:i], f.articles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) QuickPost(siteID, title, content, _, _ string) (store.Article, error) {
	now := time.Now()
	f.nextID++
	art := store.Article{
		ID: fmt.Sprintf("qp-%d", f.nextID), SiteID: siteID,
		Title: title, Content: content, Created: now, Modified: now,
	}
	f.articles = append(f.articles, art)
	return art, nil
}

func (f *fakeBackend) Subscriptions() ([]store.Subscription, error) {
	out := make([]store.Subscription, len(f.subscriptions))
	for i, s := range f.subscriptions {
		s.Feed.Items = append([]feeds.Item(nil), s.Feed.Items...)
		out[i] = s
	}
	return out, nil
}

func (f *fakeBackend) Follow(address string) (store.Subscription, error) {
	sub := store.Subscription{
		URL: address,
		Feed: feeds.Feed{
			Title: "订阅源 " + address,
			Items: []feeds.Item{{
				Key:     "k1",
				Title:   "首篇订阅文章",
				Content: "<p>Hello <strong>Native</strong> Reader</p>",
			}},
		},
	}
	f.subscriptions = append(f.subscriptions, sub)
	return sub, nil
}

func (f *fakeBackend) RefreshSubscription(address string) (store.Subscription, error) {
	for _, s := range f.subscriptions {
		if s.URL == address {
			return s, nil
		}
	}
	return store.Subscription{}, fmt.Errorf("subscription not found")
}

func (f *fakeBackend) UpdateSubscription(oldAddress, newAddress, customTitle string) (store.Subscription, error) {
	for i := range f.subscriptions {
		if f.subscriptions[i].URL == oldAddress {
			f.subscriptions[i].URL = newAddress
			f.subscriptions[i].CustomTitle = customTitle
			return f.subscriptions[i], nil
		}
	}
	return store.Subscription{}, fmt.Errorf("subscription not found")
}

func (f *fakeBackend) Unfollow(address string) error {
	for i := range f.subscriptions {
		if f.subscriptions[i].URL == address {
			f.subscriptions = append(f.subscriptions[:i], f.subscriptions[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) StarSubscription(address, key string, starred bool) error {
	for i := range f.subscriptions {
		if f.subscriptions[i].URL != address {
			continue
		}
		for j := range f.subscriptions[i].Feed.Items {
			if feedItemKey(f.subscriptions[i].Feed.Items[j]) == key {
				f.subscriptions[i].Feed.Items[j].Starred = starred
				return nil
			}
		}
	}
	return nil
}

func (f *fakeBackend) SetSubscriptionRead(address, key string, read bool) error {
	for i := range f.subscriptions {
		if f.subscriptions[i].URL != address {
			continue
		}
		for j := range f.subscriptions[i].Feed.Items {
			if feedItemKey(f.subscriptions[i].Feed.Items[j]) == key {
				f.subscriptions[i].Feed.Items[j].Read = read
				return nil
			}
		}
	}
	return nil
}

func (f *fakeBackend) MarkSubscriptionRead(address string) error {
	for i := range f.subscriptions {
		if f.subscriptions[i].URL != address {
			continue
		}
		for j := range f.subscriptions[i].Feed.Items {
			f.subscriptions[i].Feed.Items[j].Read = true
		}
	}
	return nil
}

func (f *fakeBackend) SearchArticles(query string) ([]store.SearchResult, error) {
	q := strings.ToLower(query)
	var out []store.SearchResult
	for _, a := range f.articles {
		if strings.Contains(strings.ToLower(a.Title+" "+a.Content), q) {
			out = append(out, store.SearchResult{
				Title: a.Title, SourceName: f.siteName(a.SiteID), Snippet: a.Content,
				SiteID: a.SiteID, ArticleID: a.ID,
			})
		}
	}
	for _, s := range f.subscriptions {
		for _, item := range s.Feed.Items {
			if strings.Contains(strings.ToLower(item.Title+" "+item.Content), q) {
				out = append(out, store.SearchResult{
					Title: item.Title, SourceName: subscriptionTitle(s), Snippet: item.Title,
					SubscriptionURL: s.URL, ItemKey: feedItemKey(item),
				})
			}
		}
	}
	return out, nil
}

func (f *fakeBackend) RelatedArticles(_, _, _, _ string) ([]store.SearchResult, error) {
	var out []store.SearchResult
	for _, a := range f.articles {
		out = append(out, store.SearchResult{
			Title: a.Title, SourceName: f.siteName(a.SiteID),
			SiteID: a.SiteID, ArticleID: a.ID,
		})
	}
	return out, nil
}

func TestNativeUIDashboardAndEditorFlow(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "我的星球", Created: now}},
		articles: []store.Article{{
			ID: "a1", SiteID: "s1", Title: "欢迎文章", Content: "# 你好\n\n这是**原生渲染**的 Markdown。",
			Created: now, Modified: now,
		}},
		drafts: []store.Draft{{
			ID: "d1", SiteID: "s1", Title: "未完成草稿", Content: "## 草稿正文\n\n- 第一项\n- 第二项",
			Created: now, Modified: now,
		}},
		activity: []store.DailyActivity{{Date: now.Format("2006-01-02"), Articles: 1, Drafts: 1}},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	if err := tt.Click("我的星球"); err != nil {
		t.Fatalf("failed to click site in sidebar: %v", err)
	}
	if app.mode != viewSite || app.selectedSiteID != "s1" {
		t.Fatalf("expected site s1 selected, got mode=%v site=%q", app.mode, app.selectedSiteID)
	}

	if err := tt.Click("✎ 未完成草稿"); err != nil {
		t.Fatalf("failed to click draft: %v", err)
	}
	if !app.editing || app.activeDraftID != "d1" {
		t.Fatalf("expected editing draft d1, got editing=%v draft=%q", app.editing, app.activeDraftID)
	}

	// Open metadata panel, add tag and attachment, set hero image, insert markdown.
	if err := tt.Click("元数据"); err != nil {
		t.Fatalf("failed to toggle metadata panel: %v", err)
	}
	app.newTagInput = "Go Native"
	tt.Frame()
	if err := tt.Click("+ 标签"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}
	if app.editorTags["go-native"] != "Go Native" {
		t.Fatalf("expected tag go-native, got %+v", app.editorTags)
	}

	backend.nextFile = "hero.png"
	if err := tt.Click("+ 添加附件"); err != nil {
		t.Fatalf("failed to add attachment: %v", err)
	}
	if len(app.editorAttachments) != 1 || app.editorAttachments[0] != "hero.png" {
		t.Fatalf("expected hero.png attachment, got %+v", app.editorAttachments)
	}
	if err := tt.Click("设为封面"); err != nil {
		t.Fatalf("failed to set hero image: %v", err)
	}
	if app.editorHeroImage != "hero.png" {
		t.Fatalf("expected hero image hero.png, got %q", app.editorHeroImage)
	}
	if err := tt.Click("插入"); err != nil {
		t.Fatalf("failed to insert attachment markdown: %v", err)
	}
	if !strings.Contains(app.editorBuffer.String(), "![hero.png](hero.png)") {
		t.Fatalf("expected inserted image markdown, got %q", app.editorBuffer.String())
	}

	// Modify title and save draft with tags.
	app.editorTitle = "更新后的标题"
	tt.Frame()
	if err := tt.Click("保存草稿"); err != nil {
		t.Fatalf("failed to click save draft: %v", err)
	}
	if backend.drafts[0].Title != "更新后的标题" || backend.drafts[0].Tags["go-native"] != "Go Native" {
		t.Fatalf("expected saved draft title and tags, got %+v", backend.drafts[0])
	}

	if err := tt.Click("预览"); err != nil {
		t.Fatalf("failed to switch to preview mode: %v", err)
	}
	if app.editorPaneMode != editorPaneView {
		t.Fatalf("expected preview mode, got %d", app.editorPaneMode)
	}

	if err := tt.Click("发布"); err != nil {
		t.Fatalf("failed to publish draft: %v", err)
	}
	if app.editing {
		t.Fatal("expected editor to exit after publish")
	}
	if len(backend.articles) != 2 {
		t.Fatalf("expected 2 published articles, got %d", len(backend.articles))
	}
}

func TestNativeUIUnsavedGuardAndArticleActions(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "主站", Created: now}},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "第一篇", Content: "内容一", Created: now, Modified: now},
			{ID: "a2", SiteID: "s1", Title: "第二篇", Content: "内容二", Created: now, Modified: now},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// Quick Post from sidebar.
	if err := tt.Click("⚡ 快捷发文"); err != nil {
		t.Fatalf("failed to open quick post: %v", err)
	}
	app.quickPostTitle = "快讯"
	app.quickPostContent = "刚刚发布"
	tt.Frame()
	if err := tt.Click("立即发布"); err != nil {
		t.Fatalf("failed to submit quick post: %v", err)
	}
	if len(backend.articles) != 3 {
		t.Fatalf("expected 3 articles after quick post, got %d", len(backend.articles))
	}

	// Pin current article and verify it sorts first.
	if err := tt.Click("置顶"); err != nil {
		t.Fatalf("failed to pin article: %v", err)
	}
	if app.articles[0].Title != "快讯" || app.articles[0].Pinned == nil {
		t.Fatalf("expected pinned article first, got %+v", app.articles[0])
	}

	// Open article settings and save slug.
	if err := tt.Click("文章设置"); err != nil {
		t.Fatalf("failed to open article settings: %v", err)
	}
	app.articleSettingsSlug = "quick-news"
	app.articleSettingsInNav = true
	app.articleSettingsWeight = "10"
	tt.Frame()
	if err := tt.Click("保存设置"); err != nil {
		t.Fatalf("failed to save article settings: %v", err)
	}
	if app.currentArticle().Slug != "quick-news" || !app.currentArticle().IsIncludedInNavigation || app.currentArticle().NavigationWeight != 10 {
		t.Fatalf("expected updated article settings, got %+v", app.currentArticle())
	}

	// Enter editor, dirty it, click 总览 -> should open unsaved guard.
	if err := tt.Click("编辑文章"); err != nil {
		t.Fatalf("failed to enter editor: %v", err)
	}
	app.editorTitle = "未保存的修改"
	tt.Frame()
	if err := tt.Click("总览"); err != nil {
		t.Fatalf("failed to click overview: %v", err)
	}
	if !app.unsavedDialogOpen || app.mode != viewSite {
		t.Fatalf("expected unsaved dialog to block navigation, open=%v mode=%v", app.unsavedDialogOpen, app.mode)
	}
	if err := tt.Click("放弃更改"); err != nil {
		t.Fatalf("failed to click discard in unsaved dialog: %v", err)
	}
	if app.mode != viewOverview || app.editing {
		t.Fatalf("expected overview after discard, mode=%v editing=%v", app.mode, app.editing)
	}
}

func TestNativeUIEmptyStateAndCreateSite(t *testing.T) {
	backend := &fakeBackend{}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	if err := tt.Click("创建第一个站点"); err != nil {
		t.Fatalf("failed to open create site modal: %v", err)
	}
	if !app.createSiteOpen {
		t.Fatal("expected createSiteOpen to be true")
	}

	app.newSiteName = "测试新站点"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatalf("failed to submit create site modal: %v", err)
	}
	if len(backend.sites) != 1 || backend.sites[0].Name != "测试新站点" {
		t.Fatalf("expected created site, got %+v", backend.sites)
	}
	if app.mode != viewSite || app.selectedSiteID != backend.sites[0].ID {
		t.Fatalf("expected switched to new site, got mode=%v site=%q", app.mode, app.selectedSiteID)
	}
}
