package store

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"windows/internal/services"
)

type PlausibleTracking struct {
	Enabled  bool   `json:"enabled"`
	Domain   string `json:"domain,omitempty"`
	ScriptID string `json:"scriptId,omitempty"`
	Server   string `json:"server,omitempty"`
}

type Site struct {
	ID                         string             `json:"id"`
	Name                       string             `json:"name"`
	About                      string             `json:"about"`
	Slug                       string             `json:"slug,omitempty"`
	AuthorName                 string             `json:"authorName,omitempty"`
	Domain                     string             `json:"domain,omitempty"`
	TwitterUsername            string             `json:"twitterUsername,omitempty"`
	GithubUsername             string             `json:"githubUsername,omitempty"`
	TelegramUsername           string             `json:"telegramUsername,omitempty"`
	MastodonUsername           string             `json:"mastodonUsername,omitempty"`
	DiscordLink                string             `json:"discordLink,omitempty"`
	PodcastLanguage            string             `json:"podcastLanguage,omitempty"`
	PodcastExplicit            bool               `json:"podcastExplicit,omitempty"`
	PodcastCategories          []string           `json:"podcastCategories,omitempty"`
	PodcastCoverPNG            string             `json:"podcastCoverPng,omitempty"`
	DoNotIndex                 bool               `json:"doNotIndex,omitempty"`
	CustomCodeHeadEnabled      bool               `json:"customCodeHeadEnabled,omitempty"`
	CustomCodeHead             string             `json:"customCodeHead,omitempty"`
	CustomCodeBodyStartEnabled bool               `json:"customCodeBodyStartEnabled,omitempty"`
	CustomCodeBodyStart        string             `json:"customCodeBodyStart,omitempty"`
	CustomCodeBodyEndEnabled   bool               `json:"customCodeBodyEndEnabled,omitempty"`
	CustomCodeBodyEnd          string             `json:"customCodeBodyEnd,omitempty"`
	PrewarmNewPost             *bool              `json:"prewarmNewPost,omitempty"`
	PublishAsIPNS              *bool              `json:"publishAsIPNS,omitempty"`
	SaveRoundAvatar            bool               `json:"saveRoundAvatar,omitempty"`
	Aggregation                []string           `json:"aggregation,omitempty"`
	AcceptsDonation            bool               `json:"acceptsDonation,omitempty"`
	AcceptsDonationMessage     string             `json:"acceptsDonationMessage,omitempty"`
	AcceptsDonationETHAddress  string             `json:"acceptsDonationETHAddress,omitempty"`
	JuiceboxEnabled            bool               `json:"juiceboxEnabled,omitempty"`
	JuiceboxProjectID          int                `json:"juiceboxProjectID,omitempty"`
	FarcasterManifestJSON      string             `json:"farcasterManifestJson,omitempty"`
	AvatarPNG                  string             `json:"avatarPng,omitempty"`
	Created                    time.Time          `json:"created"`
	IPNS                       string             `json:"ipns,omitempty"`
	LastPublishedCID           string             `json:"lastPublishedCID,omitempty"`
	LastPublished              *time.Time         `json:"lastPublished,omitempty"`
	TemplateID                 string             `json:"templateId,omitempty"`
	TemplateName               string             `json:"templateName,omitempty"`
	TemplateSettings           map[string]string  `json:"templateSettings,omitempty"`
	NextArticleNumber          int                `json:"nextArticleNumber,omitempty"`
	Imported                   bool               `json:"imported,omitempty"`
	TagsUnsupported            bool               `json:"tagsUnsupported,omitempty"`
	Archived                   bool               `json:"archived,omitempty"`
	Removed                    bool               `json:"removed,omitempty"`
	DeletionPending            bool               `json:"deletionPending,omitempty"`
	ContentRevision            uint64             `json:"contentRevision,omitempty"`
	PublishedRevision          uint64             `json:"publishedRevision,omitempty"`
	PublishPending             bool               `json:"publishPending,omitempty"`
	PlausibleOverride          *PlausibleTracking `json:"plausibleOverride,omitempty"`
	// Public analytics values are derived from verified original metadata at
	// render time; the private API key is never stored in this structure.
	PlausibleEnabled   bool              `json:"-"`
	PlausibleDomain    string            `json:"-"`
	PlausibleAPIServer string            `json:"-"`
	PlausibleScriptID  string            `json:"-"`
	Tags               map[string]string `json:"-"`
}

func cloneSite(site Site) Site {
	site.PodcastCategories = append([]string(nil), site.PodcastCategories...)
	site.Aggregation = append([]string(nil), site.Aggregation...)
	if site.PlausibleOverride != nil {
		copy := *site.PlausibleOverride
		site.PlausibleOverride = &copy
	}
	if site.TemplateSettings != nil {
		cloned := make(map[string]string, len(site.TemplateSettings))
		for k, v := range site.TemplateSettings {
			cloned[k] = v
		}
		site.TemplateSettings = cloned
	}
	if site.PrewarmNewPost != nil {
		v := *site.PrewarmNewPost
		site.PrewarmNewPost = &v
	}
	if site.PublishAsIPNS != nil {
		v := *site.PublishAsIPNS
		site.PublishAsIPNS = &v
	}
	return site
}

func (s Site) ShouldPublishIPNS() bool {
	return s.PublishAsIPNS == nil || *s.PublishAsIPNS
}

func (s Site) ShouldPrewarmNewPost() bool {
	return s.PrewarmNewPost == nil || *s.PrewarmNewPost
}

var plausibleDomainPattern = regexp.MustCompile(`(?i)^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)*$`)

var plausibleScriptPattern = regexp.MustCompile(`^pa-[A-Za-z0-9_-]{1,128}$`)

func ValidPlausibleScriptID(id string) bool {
	return id == "" || plausibleScriptPattern.MatchString(id)
}

func ValidPlausibleDomain(domain string) bool {
	return len(domain) <= 253 && plausibleDomainPattern.MatchString(domain)
}

func ApplyPlausibleOverride(site Site) Site {
	if site.PlausibleOverride != nil {
		site.PlausibleEnabled = site.PlausibleOverride.Enabled
		site.PlausibleDomain = site.PlausibleOverride.Domain
		site.PlausibleScriptID = site.PlausibleOverride.ScriptID
		if site.PlausibleEnabled {
			origin, err := services.PlausibleServerOrigin(site.PlausibleOverride.Server)
			if err == nil {
				site.PlausibleAPIServer = strings.TrimPrefix(origin, "https://")
			} else {
				site.PlausibleAPIServer = site.PlausibleOverride.Server
			}
		} else {
			site.PlausibleDomain = ""
			site.PlausibleAPIServer = ""
			site.PlausibleScriptID = ""
		}
	}
	return site
}

func (s *Store) ListSites() ([]Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sites := []Site{}
	for _, site := range s.data.Sites {
		if !site.Removed {
			sites = append(sites, cloneSite(site))
		}
	}
	return sites, nil
}

func (s *Store) RemovedSites() ([]Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sites := []Site{}
	for _, site := range s.data.Sites {
		if site.Removed {
			sites = append(sites, cloneSite(site))
		}
	}
	return sites, nil
}

// RemoveSite moves the site to a recoverable local state. It does not remove
// the IPNS key or make previously published content disappear from IPFS.
func (s *Store) RemoveSite(siteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || site.Removed {
			continue
		}
		if site.PublishPending && site.ContentRevision <= site.PublishedRevision {
			if site.ContentRevision == ^uint64(0) {
				return errors.New("站点修订版本已达到上限")
			}
			site.ContentRevision++
		}
		site.Removed = true
		site.PublishPending = false
		return s.write(next)
	}
	return errors.New("站点不存在")
}

func (s *Store) RestoreSite(siteID string) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || !site.Removed {
			continue
		}
		if site.DeletionPending {
			return Site{}, errors.New("站点已开始永久删除，无法恢复")
		}
		site.Removed = false
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(*site), nil
	}
	return Site{}, errors.New("已移除的站点不存在")
}

// BeginSiteDeletion commits an irreversible deletion intent before files or
// keys are removed, so interrupted cleanup remains visible and retryable.
func (s *Store) BeginSiteDeletion(siteID string) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || !site.Removed {
			continue
		}
		if site.DeletionPending {
			return cloneSite(*site), nil
		}
		site.DeletionPending = true
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(*site), nil
	}
	return Site{}, errors.New("请先移除站点")
}

// FinalizeSiteDeletion forgets metadata only after all managed data is removed.
func (s *Store) FinalizeSiteDeletion(siteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	next := s.data
	next.Sites = make([]Site, 0, len(s.data.Sites))
	for _, site := range s.data.Sites {
		if site.ID == siteID {
			if !site.Removed || !site.DeletionPending {
				return errors.New("站点未开始永久删除")
			}
			found = true
			continue
		}
		next.Sites = append(next.Sites, site)
	}
	if !found {
		return errors.New("站点不存在")
	}
	next.Articles = make([]Article, 0, len(s.data.Articles))
	for _, item := range s.data.Articles {
		if item.SiteID != siteID {
			next.Articles = append(next.Articles, item)
		}
	}
	next.Drafts = make([]Draft, 0, len(s.data.Drafts))
	for _, item := range s.data.Drafts {
		if item.SiteID != siteID {
			next.Drafts = append(next.Drafts, item)
		}
	}
	next.DraftCleanup = make([]DraftCleanup, 0, len(s.data.DraftCleanup))
	for _, item := range s.data.DraftCleanup {
		if item.SiteID != siteID {
			next.DraftCleanup = append(next.DraftCleanup, item)
		}
	}
	next.DraftStaging = make([]DraftCleanup, 0, len(s.data.DraftStaging))
	for _, item := range s.data.DraftStaging {
		if item.SiteID != siteID {
			next.DraftStaging = append(next.DraftStaging, item)
		}
	}
	next.ArticleCleanup = make([]ArticleCleanup, 0, len(s.data.ArticleCleanup))
	for _, item := range s.data.ArticleCleanup {
		if item.SiteID != siteID {
			next.ArticleCleanup = append(next.ArticleCleanup, item)
		}
	}
	next.BackupMetadata = make(map[string]json.RawMessage, len(s.data.BackupMetadata))
	for id, raw := range s.data.BackupMetadata {
		if id != siteID {
			next.BackupMetadata[id] = raw
		}
	}
	return s.write(next)
}

// PublicationSnapshot reads a consistent site, article list and backup metadata.
func (s *Store) PublicationSnapshot(siteID string) (Site, []Article, json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var site Site
	found := false
	for _, item := range s.data.Sites {
		if item.ID == siteID && !item.Removed {
			site = cloneSite(item)
			found = true
			break
		}
	}
	if !found {
		return Site{}, nil, nil, errors.New("站点不存在")
	}
	articles := []Article{}
	for _, item := range s.data.Articles {
		if item.SiteID == siteID {
			articles = append(articles, cloneArticle(item))
		}
	}
	return site, articles, append(json.RawMessage(nil), s.data.BackupMetadata[siteID]...), nil
}

// markContentChange is called under the store lock on a detached next snapshot.
func markContentChange(next *data, siteID string, publish bool) {
	next.Sites = append([]Site{}, next.Sites...)
	for i := range next.Sites {
		if next.Sites[i].ID == siteID {
			next.Sites[i].ContentRevision++
			if publish {
				next.Sites[i].PublishPending = true
			}
			return
		}
	}
}

func (s *Store) CreateSite(name, about string) (Site, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Site{}, errors.New("请输入站点名称")
	}
	id, err := newID()
	if err != nil {
		return Site{}, err
	}
	site := Site{ID: id, Name: name, About: about, Created: time.Now().UTC()}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append(append([]Site{}, s.data.Sites...), site)
	if err := s.write(next); err != nil {
		return Site{}, err
	}
	return site, nil
}

func (s *Store) UpdateSite(siteID, name, about string) (Site, error) {
	return s.updateSite(siteID, name, about, nil)
}

// UpdateSiteSettings updates the public site settings as one revision.
func (s *Store) UpdateSiteSettings(siteID, name, about string, doNotIndex bool) (Site, error) {
	return s.updateSite(siteID, name, about, &doNotIndex)
}

func (s *Store) updateSite(siteID, name, about string, doNotIndex *bool) (Site, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Site{}, errors.New("请输入站点名称")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || site.Removed {
			continue
		}
		if site.Name == name && site.About == about && (doNotIndex == nil || site.DoNotIndex == *doNotIndex) {
			return cloneSite(*site), nil
		}
		if site.ContentRevision == ^uint64(0) {
			return Site{}, errors.New("站点修订版本已达到上限")
		}
		site.Name, site.About = name, about
		if doNotIndex != nil {
			site.DoNotIndex = *doNotIndex
		}
		site.ContentRevision++
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(*site), nil
	}
	return Site{}, errors.New("站点不存在")
}

func (s *Store) SetPublished(siteID, ipns, cid string) error {
	if ipns == "" || cid == "" {
		return errors.New("发布结果缺少 IPNS 或 CID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site{}, s.data.Sites...)
	for i := range next.Sites {
		if next.Sites[i].ID == siteID && !next.Sites[i].Removed {
			if next.Sites[i].IPNS != "" && next.Sites[i].IPNS != ipns {
				return errors.New("站点 IPNS 身份不匹配")
			}
			now := time.Now().UTC()
			next.Sites[i].IPNS, next.Sites[i].LastPublishedCID, next.Sites[i].LastPublished = ipns, cid, &now
			return s.write(next)
		}
	}
	return errors.New("站点不存在")
}

// SetPublishedRevision records the exact snapshot sent to IPNS without losing
// a newer save or content change that happened while publication was running.
func (s *Store) SetPublishedRevision(siteID, ipns, cid string, revision uint64) (bool, error) {
	if ipns == "" || cid == "" {
		return false, errors.New("发布结果缺少 IPNS 或 CID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site{}, s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || site.Removed {
			continue
		}
		if site.IPNS != "" && site.IPNS != ipns {
			return false, errors.New("站点 IPNS 身份不匹配")
		}
		if revision > site.ContentRevision {
			return false, errors.New("发布版本无效")
		}
		now := time.Now().UTC()
		site.IPNS, site.LastPublishedCID, site.LastPublished = ipns, cid, &now
		site.PublishedRevision = revision
		site.PublishPending = site.ContentRevision != revision
		if err := s.write(next); err != nil {
			return false, err
		}
		return site.PublishPending, nil
	}
	return false, errors.New("站点不存在")
}

func (s *Store) SetIdentity(siteID, ipns string) error {
	if ipns == "" {
		return errors.New("IPNS 身份不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site{}, s.data.Sites...)
	for i := range next.Sites {
		if next.Sites[i].ID == siteID && !next.Sites[i].Removed {
			if next.Sites[i].IPNS != "" && next.Sites[i].IPNS != ipns {
				return errors.New("站点 IPNS 身份不匹配")
			}
			next.Sites[i].IPNS = ipns
			return s.write(next)
		}
	}
	return errors.New("站点不存在")
}

func (s *Store) SetTemplate(siteID, templateID, name string) error {
	if templateID == "" || name == "" {
		return errors.New("模板信息无效")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site{}, s.data.Sites...)
	for i := range next.Sites {
		if next.Sites[i].ID == siteID && !next.Sites[i].Removed {
			if next.Sites[i].TemplateName != name {
				next.Sites[i].TemplateSettings = nil
			}
			next.Sites[i].TemplateID = templateID
			next.Sites[i].TemplateName = name
			next.Sites[i].ContentRevision++
			return s.write(next)
		}
	}
	return errors.New("站点不存在")
}

func (s *Store) SetPlausibleTracking(siteID string, enabled bool, domain string) (Site, error) {
	return s.SetPlausibleTrackingScript(siteID, enabled, domain, "")
}

func (s *Store) SetPlausibleTrackingScript(siteID string, enabled bool, domain, scriptID string) (Site, error) {
	return s.SetPlausibleTrackingServer(siteID, enabled, domain, scriptID, "")
}

func (s *Store) SetPlausibleTrackingServer(siteID string, enabled bool, domain, scriptID, server string) (Site, error) {
	if (enabled || domain != "") && !ValidPlausibleDomain(domain) {
		return Site{}, errors.New("Plausible 站点域名无效")
	}
	if !ValidPlausibleScriptID(scriptID) {
		return Site{}, errors.New("Plausible 脚本 ID 无效")
	}
	if enabled || server != "" {
		origin, err := services.PlausibleServerOrigin(server)
		if err != nil {
			return Site{}, err
		}
		server = origin
		if enabled && server == "https://plausible.io" {
			server = ""
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || site.Removed {
			continue
		}
		if site.PlausibleOverride != nil && site.PlausibleOverride.Enabled == enabled && site.PlausibleOverride.Domain == domain && site.PlausibleOverride.ScriptID == scriptID && site.PlausibleOverride.Server == server {
			return cloneSite(*site), nil
		}
		if site.ContentRevision == ^uint64(0) {
			return Site{}, errors.New("站点修订版本已达到上限")
		}
		site.PlausibleOverride = &PlausibleTracking{Enabled: enabled, Domain: domain, ScriptID: scriptID, Server: server}
		site.ContentRevision++
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(*site), nil
	}
	return Site{}, errors.New("站点不存在")
}

func (s *Store) SetTemplateSettings(siteID string, settings map[string]string) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		if next.Sites[i].ID != siteID || next.Sites[i].Removed {
			continue
		}
		if next.Sites[i].TemplateName != "Sepia" || next.Sites[i].TemplateID == "" {
			return Site{}, errors.New("当前模板不支持此设置")
		}
		if next.Sites[i].ContentRevision == ^uint64(0) {
			return Site{}, errors.New("站点修订版本已达到上限")
		}
		next.Sites[i].TemplateSettings = make(map[string]string, len(settings))
		for k, v := range settings {
			next.Sites[i].TemplateSettings[k] = v
		}
		next.Sites[i].ContentRevision++
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(next.Sites[i]), nil
	}
	return Site{}, errors.New("站点不存在")
}

func (s *Store) hasSite(id string) bool {
	for _, site := range s.data.Sites {
		if site.ID == id && !site.Removed {
			return true
		}
	}
	return false
}
