package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"windows/internal/applog"
	"windows/internal/backup"
	"windows/internal/localapi"
	"windows/internal/nativeui"
	"windows/internal/offline"
	"windows/internal/pinnable"
	"windows/internal/services"
	"windows/internal/store"
)

//go:embed resources/icon.png
var appIconPNG []byte

func (l *Library) ensureAPIServer() *localapi.Server {
	l.apiMu.Lock()
	defer l.apiMu.Unlock()
	if l.apiServer == nil {
		l.apiServer = localapi.NewServer(localapi.Backend{
			OpenStore: l.open,
			PublishSite: func(siteID string) error {
				_, err := l.PublishSite(siteID)
				return err
			},
		})
	}
	return l.apiServer
}

func (l *Library) closeAPIServer() {
	l.apiMu.Lock()
	srv := l.apiServer
	l.apiMu.Unlock()
	if srv != nil {
		_ = srv.Close()
	}
}

func (l *Library) DataDir() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return dir
}

func (l *Library) MoveDataDir(ctx context.Context) (string, error) {
	dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择新的空数据仓库目录", Directory: true})
	if err != nil || len(dirs) == 0 {
		return "", err
	}
	currentDir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	defaultDir := l.defaultDataDir
	if defaultDir == "" {
		defaultDir = currentDir
	}
	if err := store.MigrateDataDir(defaultDir, currentDir, dirs[0]); err != nil {
		return "", err
	}
	return dirs[0], nil
}

func (l *Library) PreventSleep() bool {
	s, err := l.open()
	if err != nil {
		return false
	}
	return s.PreventSleep()
}

func (l *Library) SetPreventSleep(enabled bool) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	if err := services.SetPreventSleep(enabled); err != nil {
		return err
	}
	return s.SetPreventSleep(enabled)
}

func (l *Library) AIConfig() (services.AIConfig, error) {
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.AIConfig{}, err
	}
	return services.LoadAIConfig(root)
}

func (l *Library) ConfigureAI(baseURL, model, token string) (services.AIConfig, error) {
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.AIConfig{}, err
	}
	return services.SaveAIConfig(root, baseURL, model, token)
}

func (l *Library) ProbeAI(baseURL string) (services.AIProbeResult, error) {
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.AIProbeResult{}, err
	}
	return services.ProbeAIEndpoints(root, baseURL)
}

func (l *Library) AskAI(prompt, contextTitle, contextBody, historyJSON string) (string, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "", errors.New("请输入提问内容")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	var history []services.AIChatMessage
	if strings.TrimSpace(historyJSON) != "" {
		_ = json.Unmarshal([]byte(historyJSON), &history)
	}
	sys := "你是邻页（Kinpage）本地知识与文章助手，请使用清晰简洁的中文回答用户问题。"
	if strings.TrimSpace(contextTitle) != "" || strings.TrimSpace(contextBody) != "" {
		bodyRunes := []rune(contextBody)
		if len(bodyRunes) > 8000 {
			bodyRunes = bodyRunes[:8000]
		}
		sys += fmt.Sprintf("\n\n当前阅读文章：《%s》\n正文内容：\n%s", contextTitle, string(bodyRunes))
	} else if s, e := l.open(); e == nil {
		if hits, e := s.SearchArticles(prompt); e == nil && len(hits) > 0 {
			var sb strings.Builder
			sb.WriteString("\n\n本地库相关文章摘录：")
			for i, h := range hits {
				if i >= 4 {
					break
				}
				sb.WriteString(fmt.Sprintf("\n- 《%s》（%s）：%s", h.Title, h.SourceName, h.Snippet))
			}
			sys += sb.String()
		}
	}
	msgs := make([]services.AIChatMessage, 0, len(history)+2)
	msgs = append(msgs, services.AIChatMessage{Role: "system", Content: sys})
	for _, m := range history {
		if (m.Role == "user" || m.Role == "assistant") && strings.TrimSpace(m.Content) != "" {
			msgs = append(msgs, m)
		}
	}
	msgs = append(msgs, services.AIChatMessage{Role: "user", Content: prompt})
	return services.ChatCompletion(root, msgs)
}

func (l *Library) LocalAPIStatus() (localapi.Status, error) {
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return localapi.Status{}, err
	}
	return l.ensureAPIServer().Status(root)
}

func (l *Library) ConfigureLocalAPI(enabled bool, port int, authEnabled bool, username, passcode string) (localapi.Status, error) {
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return localapi.Status{}, err
	}
	if _, err := localapi.SaveConfig(root, enabled, port, authEnabled, username, passcode); err != nil {
		return localapi.Status{}, err
	}
	srv := l.ensureAPIServer()
	if err := srv.Sync(root); err != nil {
		return localapi.Status{}, err
	}
	return srv.Status(root)
}

func (l *Library) ClearLocalAPILogs() {
	l.ensureAPIServer().ClearLogs()
}

// Published Folders (Phase 4.1)
func (l *Library) PublishedFolders() ([]store.PublishedFolder, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.ListPublishedFolders()
}

func (l *Library) AddPublishedFolder(ctx context.Context, name string, autoPublish bool) (store.PublishedFolder, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择要发布到 IPFS 的本地文件夹", Directory: true})
	if err != nil || len(paths) == 0 {
		return store.PublishedFolder{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.PublishedFolder{}, err
	}
	return s.AddPublishedFolder(name, paths[0], autoPublish)
}

func (l *Library) PublishFolder(folderID string) (store.PublishedFolder, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.PublishedFolder{}, err
	}
	folders, err := s.ListPublishedFolders()
	if err != nil {
		return store.PublishedFolder{}, err
	}
	var target *store.PublishedFolder
	for i := range folders {
		if folders[i].ID == folderID {
			target = &folders[i]
			break
		}
	}
	if target == nil {
		return store.PublishedFolder{}, errors.New("发布文件夹不存在")
	}
	if info, err := os.Stat(target.Path); err != nil || !info.IsDir() {
		return store.PublishedFolder{}, errors.New("本地文件夹已不存在或不可访问")
	}
	if _, _, err := l.ensureNode(true); err != nil {
		return store.PublishedFolder{}, err
	}
	keyName := target.KeyName
	if keyName == "" {
		keyName = "folder-" + target.ID
	}
	ipnsKey, err := l.node.EnsureKey(keyName)
	if err != nil {
		return store.PublishedFolder{}, fmt.Errorf("准备文件夹 IPNS 密钥失败：%w", err)
	}
	cid, err := l.node.Add(target.Path)
	if err != nil {
		return store.PublishedFolder{}, fmt.Errorf("添加文件夹到 IPFS 失败：%w", err)
	}
	publishedIPNS, err := l.node.PublishOnline(keyName, cid)
	if err != nil {
		return store.PublishedFolder{}, fmt.Errorf("发布文件夹 IPNS 失败：%w", err)
	}
	if publishedIPNS == "" {
		publishedIPNS = ipnsKey
	}
	l.scheduleProvide(cid)
	return s.RecordFolderPublication(target.ID, publishedIPNS, cid)
}

func (l *Library) RemovePublishedFolder(folderID string) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.RemovePublishedFolder(folderID)
}

func (l *Library) AppLogs(category string) []applog.Entry {
	return applog.Global.List(category)
}

func (l *Library) ClearAppLogs(category string) {
	applog.Global.Clear(category)
}

type DownloadItem struct {
	Name       string `json:"name"`
	SourceName string `json:"sourceName"`
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	Status     string `json:"status"`
}

func (l *Library) ListDownloads() ([]DownloadItem, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return nil, err
	}
	var out []DownloadItem
	for _, sub := range subs {
		source := sub.CustomTitle
		if source == "" {
			source = sub.Feed.Title
		}
		for _, item := range sub.Feed.Items {
			for mediaURL, asset := range item.OfflineMedia {
				out = append(out, DownloadItem{
					Name:       asset.File,
					SourceName: source,
					URL:        mediaURL,
					Size:       asset.Size,
					Status:     "已下载离线",
				})
			}
			for _, att := range item.Attachments {
				out = append(out, DownloadItem{
					Name:       att,
					SourceName: source,
					URL:        item.Link,
					Status:     "IPFS 附件",
				})
			}
		}
	}
	if out == nil {
		out = []DownloadItem{}
	}
	return out, nil
}

// OfflineMediaUsage reports managed feed-cache usage without modifying files.
func (l *Library) OfflineMediaUsage() (offline.Usage, error) { return l.collectOfflineMedia(false) }

// CleanupOfflineMedia removes only files no saved article or active player references.
func (l *Library) CleanupOfflineMedia() (offline.Usage, error) { return l.collectOfflineMedia(true) }
func (l *Library) collectOfflineMedia(clean bool) (offline.Usage, error) {
	l.feedMu.Lock()
	defer l.feedMu.Unlock()
	s, err := l.open()
	if err != nil {
		return offline.Usage{}, err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return offline.Usage{}, err
	}
	refs := map[string]map[string]bool{}
	add := func(snapshot, file string) {
		if snapshot == "" || file == "" {
			return
		}
		if refs[snapshot] == nil {
			refs[snapshot] = map[string]bool{}
		}
		refs[snapshot][file] = true
	}
	for _, sub := range subs {
		for _, item := range sub.Feed.Items {
			for _, asset := range item.OfflineMedia {
				add(asset.Snapshot, asset.File)
			}
		}
	}
	l.audioMu.Lock()
	defer l.audioMu.Unlock()
	add(l.audioSnapshot, l.audioFile)
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return offline.Usage{}, err
	}
	return offline.Collect(root, refs, clean)
}

// ConfigureService stores credentials encrypted for the current Windows user.
func (l *Library) ConfigureService(siteID, kind, name, token string) error {
	return l.configureService(siteID, kind, name, token, "", "")
}

// ConfigurePlausible sets the domain, optional site-specific script ID and key.
func (l *Library) ConfigurePlausible(siteID, domain, token, scriptID string) error {
	return l.ConfigurePlausibleAt(siteID, domain, token, scriptID, "https://plausible.io")
}

// ConfigurePlausibleAt accepts an HTTPS service selected explicitly by the user.
func (l *Library) ConfigurePlausibleAt(siteID, domain, token, scriptID, server string) error {
	domain, token, scriptID = strings.TrimSpace(domain), strings.TrimSpace(token), strings.TrimSpace(scriptID)
	if strings.TrimSpace(server) == "" {
		return errors.New("请明确填写 Plausible HTTPS 服务地址")
	}
	origin, err := services.PlausibleServerOrigin(server)
	if err != nil {
		return err
	}
	if origin == "https://plausible.io" {
		origin = ""
	}
	if token != "" {
		return l.configureService(siteID, "plausible", domain, token, scriptID, origin)
	}
	if !store.ValidPlausibleDomain(domain) || !store.ValidPlausibleScriptID(scriptID) {
		return errors.New("Plausible 域名或脚本 ID 无效")
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	old, err := services.LoadConfig(root, siteID, "plausible")
	if err != nil {
		return err
	}
	if old.Token != "" && old.Server != origin {
		return errors.New("更换 Plausible 服务器时请重新输入该服务器的 API 令牌，或先停用原配置")
	}
	changed := old.Token != "" && old.Name != domain
	if changed {
		next := old
		next.Name = domain
		if err = services.SaveConfig(root, siteID, "plausible", next); err != nil {
			return err
		}
	}
	if _, err = s.SetPlausibleTrackingServer(siteID, true, domain, scriptID, origin); err != nil {
		if changed {
			if rollback := services.SaveConfig(root, siteID, "plausible", old); rollback != nil {
				return errors.New("站点设置保存失败，服务配置恢复也失败，请重新检查")
			}
		}
		return err
	}
	return nil
}
func (l *Library) configureService(siteID, kind, name, token, scriptID, server string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	name, token = strings.TrimSpace(name), strings.TrimSpace(token)
	if kind == "plausible" && token != "" && !store.ValidPlausibleDomain(name) {
		return errors.New("Plausible 站点域名无效")
	}
	if kind == "plausible" && !store.ValidPlausibleScriptID(scriptID) {
		return errors.New("Plausible 脚本 ID 无效，请输入 pa- 开头的站点专属 ID")
	}
	// Disable tracking even when the old encrypted credential is unreadable.
	if kind == "plausible" && token == "" {
		var savedDomain, savedScript, savedServer string
		if site.PlausibleOverride != nil {
			savedDomain = site.PlausibleOverride.Domain
			savedScript = site.PlausibleOverride.ScriptID
			savedServer = site.PlausibleOverride.Server
			if site.PlausibleOverride.Enabled && savedServer == "" {
				savedServer = "https://plausible.io"
			}
		} else if len(raw) > 0 {
			_, savedDomain, savedServer, _ = backup.PlausibleStoredSettings(raw)
		}
		if _, err = s.SetPlausibleTrackingServer(siteID, false, savedDomain, savedScript, savedServer); err != nil {
			return err
		}
		if err = services.SaveConfig(root, siteID, kind, services.Config{}); err != nil {
			return errors.New("站点跟踪已停用，但本机服务凭据未能清理，请重试")
		}
		return nil
	}
	old, err := services.LoadConfig(root, siteID, kind)
	if err != nil {
		return err
	}
	cfg := services.Config{Name: name, Token: token, Server: server}
	if old.Name == name && old.Token == token && old.Server == server {
		cfg = old
	}
	if err = services.SaveConfig(root, siteID, kind, cfg); err != nil {
		return err
	}
	if kind == "plausible" {
		if _, err = s.SetPlausibleTrackingServer(siteID, token != "", name, scriptID, server); err != nil {
			if rollbackErr := services.SaveConfig(root, siteID, kind, old); rollbackErr != nil {
				return errors.New("站点统计设置保存失败，服务配置恢复也失败，请重新检查")
			}
			return err
		}
	}
	return nil
}

type serviceInfo struct {
	Name            string `json:"name"`
	Configured      bool   `json:"configured"`
	LastCID         string `json:"lastCid,omitempty"`
	RequestID       string `json:"requestId,omitempty"`
	PendingCID      string `json:"pendingCid,omitempty"`
	ScriptID        string `json:"scriptId,omitempty"`
	TrackingEnabled bool   `json:"trackingEnabled,omitempty"`
	TrackingWarning string `json:"trackingWarning,omitempty"`
	Server          string `json:"server,omitempty"`
}

func (l *Library) ServiceInfo(siteID, kind string) (serviceInfo, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return serviceInfo{}, err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return serviceInfo{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return serviceInfo{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, kind)
	info := serviceInfo{Name: cfg.Name, Configured: cfg.Token != "", LastCID: cfg.LastCID, RequestID: cfg.RequestID, PendingCID: cfg.PendingCID}
	if kind == "plausible" && site.PlausibleOverride != nil {
		info.ScriptID = site.PlausibleOverride.ScriptID
		info.TrackingEnabled = site.PlausibleOverride.Enabled
		info.Server = site.PlausibleOverride.Server
		if info.Name == "" {
			info.Name = site.PlausibleOverride.Domain
		}
		if !site.PlausibleOverride.Enabled && info.Server == "" && len(raw) > 0 {
			_, domain, server, e := backup.PlausibleStoredSettings(raw)
			if e != nil {
				info.TrackingWarning = "原版统计配置不受支持；请手动填写服务器。"
			} else {
				info.Server = server
				if info.Name == "" {
					info.Name = domain
				}
			}
		}
	} else if kind == "plausible" && len(raw) > 0 {
		enabled, domain, server, e := backup.PlausibleStoredSettings(raw)
		if e != nil {
			info.TrackingWarning = "原版统计配置不受支持；可停用或重新配置。"
		} else {
			info.TrackingEnabled = enabled
			info.Server = server
			if info.Name == "" {
				info.Name = domain
			}
		}
	}
	if kind == "plausible" && info.Server == "" && info.TrackingWarning == "" && (site.PlausibleOverride != nil || len(raw) == 0) {
		info.Server = "https://plausible.io"
	}
	return info, err
}
func (l *Library) PlausibleStats(siteID string) (services.Stats, error) {
	return l.PlausibleStatsRange(siteID, "7d")
}
func (l *Library) PlausibleStatsRange(siteID, period string) (services.Stats, error) {
	if period != "day" && period != "7d" && period != "30d" {
		return services.Stats{}, errors.New("统计时间范围无效")
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return services.Stats{}, err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return services.Stats{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.Stats{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, "plausible")
	if err != nil {
		return services.Stats{}, err
	}
	if cfg.Token == "" {
		return services.Stats{}, errors.New("请先配置 Plausible")
	}
	trackingEnabled := false
	trackingDomain, trackingServer := "", ""
	if site.PlausibleOverride != nil {
		trackingEnabled = site.PlausibleOverride.Enabled
		trackingDomain = site.PlausibleOverride.Domain
		trackingServer = site.PlausibleOverride.Server
	} else if len(raw) > 0 {
		enabled, domain, server, e := backup.PlausiblePublicSettings(raw)
		if e != nil {
			return services.Stats{}, e
		}
		trackingEnabled = enabled
		trackingDomain = domain
		trackingServer = server
	}
	trackingOrigin, e := services.PlausibleServerOrigin(trackingServer)
	if e != nil {
		return services.Stats{}, e
	}
	credentialOrigin, e := services.PlausibleServerOrigin(cfg.Server)
	if e != nil {
		return services.Stats{}, e
	}
	if !trackingEnabled || trackingDomain != cfg.Name || trackingOrigin != credentialOrigin {
		return services.Stats{}, errors.New("统计凭据与站点跟踪设置不一致，请重新保存 Plausible 配置")
	}
	return services.NewClient().PlausibleStatsAtRange(cfg.Server, cfg.Token, cfg.Name, period)
}

func (l *Library) RequestFilebasePin(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	if site.LastPublishedCID == "" {
		return errors.New("请先发布站点")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	cfg, err := services.LoadConfig(root, siteID, "filebase")
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return errors.New("请先配置 Filebase")
	}
	return services.RetryPublished(root, siteID, site.LastPublishedCID, services.NewClient())
}
func (l *Library) FilebaseStatus(siteID string) (services.PinStatus, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return services.PinStatus{}, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return services.PinStatus{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.PinStatus{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, "filebase")
	if err != nil {
		return services.PinStatus{}, err
	}
	if cfg.Token == "" || cfg.RequestID == "" {
		return services.PinStatus{}, errors.New("尚无已提交的固定请求")
	}
	status, err := services.NewClient().FilebaseStatus(cfg.Token, cfg.RequestID)
	if err != nil {
		return status, err
	}
	if status.Pin.CID != cfg.LastCID {
		return services.PinStatus{}, errors.New("服务报告的 CID 与本机记录不一致")
	}
	return status, nil
}

// ConfigurePinnable saves an explicitly supplied endpoint using Windows DPAPI.
// An empty endpoint disables the integration for this site.
func (l *Library) ConfigurePinnable(siteID, endpoint string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	return pinnable.Save(root, siteID, strings.TrimSpace(endpoint))
}

func (l *Library) PinnableConfigured(siteID string) (bool, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return false, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return false, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return false, err
	}
	endpoint, err := pinnable.Load(root, siteID)
	return endpoint != "", err
}

func (l *Library) RequestPinnablePin(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	if site.IPNS == "" || site.LastPublishedCID == "" {
		return errors.New("请先发布站点")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	endpoint, err := pinnable.Load(root, siteID)
	if err != nil {
		return err
	}
	if endpoint == "" {
		return errors.New("请先配置 Pinnable 地址")
	}
	if err := pinnable.NewClient().PinForSite(endpoint, site.IPNS, false); err != nil {
		return err
	}
	l.pushPinnableRecord(endpoint, site.IPNS)
	return nil
}

// pushPinnableRecord best-effort posts the locally signed IPNS record for ipns
// to POST <endpoint>/record after the pin request has verified the endpoint.
func (l *Library) pushPinnableRecord(endpoint, ipns string) {
	node := l.getNode()
	if node == nil || ipns == "" {
		return
	}
	record, err := node.LocalIPNSRecord(ipns)
	if err != nil || len(record) == 0 {
		return
	}
	if err := pinnable.NewClient().PushRecord(endpoint, record); err != nil {
		log.Printf("push IPNS record to Pinnable: %v", err)
	}
}

func (l *Library) PinnableStatus(siteID string) (pinnable.Status, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return pinnable.Status{}, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return pinnable.Status{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return pinnable.Status{}, err
	}
	endpoint, err := pinnable.Load(root, siteID)
	if err != nil {
		return pinnable.Status{}, err
	}
	if endpoint == "" {
		return pinnable.Status{}, errors.New("请先配置 Pinnable 地址")
	}
	return pinnable.NewClient().Status(endpoint)
}

func main() {
	// MyGo CLI invokes the executable with MYGO_GENERATE set during builds.
	// Return early before touching user data directories or acquiring locks.
	if os.Getenv("MYGO_GENERATE") != "" {
		if err := mygo.App.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--planet-version" {
		fmt.Println(appVersion)
		return
	}
	if executable, err := os.Executable(); err != nil {
		showStartupError(err)
		log.Fatal(err)
	} else if err = checkUpdateReady(executable); err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	mygo.App.SetName("Kinpage")
	if mygo.IsDev() {
		path, err := mygo.App.Path(mygo.PathUserData)
		if err != nil {
			log.Fatal(err)
		}
		mygo.App.SetPath(mygo.PathUserData, filepath.Join(path, "development"))
	}
	defaultUserDataDir, _ := mygo.App.Path(mygo.PathUserData)
	if defaultUserDataDir != "" {
		if resolved := store.ResolveDataDir(defaultUserDataDir); resolved != "" && resolved != defaultUserDataDir {
			mygo.App.SetPath(mygo.PathUserData, resolved)
		}
	}
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	releaseUpdateLock, err := acquireUpdateLock()
	if err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	defer releaseUpdateLock()
	library := &Library{defaultDataDir: defaultUserDataDir}
	maintenanceCtx, cancelMaintenance := context.WithCancel(context.Background())
	var maintenance sync.WaitGroup
	defer func() {
		_ = services.SetPreventSleep(false)
		library.closeAPIServer()
		library.closeUpdates()
		cancelMaintenance()
		maintenance.Wait()
		library.closeNode()
		library.closeAudio()
		library.StopEditorMedia()
	}()
	mygo.App.WhenReady(func() {
		if root, err := mygo.App.Path(mygo.PathUserData); err == nil {
			_ = library.ensureAPIServer().Sync(root)
		}
		if library.PreventSleep() {
			_ = services.SetPreventSleep(true)
		}
		maintenance.Add(1)
		go func() { defer maintenance.Done(); library.maintainPublished(maintenanceCtx) }()
		nativeApp := nativeui.New(library)
		if s, err := library.open(); err == nil {
			maintenance.Add(1)
			go func() {
				defer maintenance.Done()
				s.WatchExternalChanges(maintenanceCtx, 2*time.Second, nativeApp.MarkDirtyExternal)
			}()
		}
		if root, err := mygo.App.Path(mygo.PathUserData); err == nil {
			nativeApp.EnablePrefsPersistence(filepath.Join(root, "nativeui-layout.json"))
		}
		opts := mygo.WindowOptions{
			Title: "邻页 · Kinpage", Width: 1280, Height: 820, MinWidth: 860, MinHeight: 600,
			BackgroundColor: "#f7f7f5", StateKey: "main",
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 44,
			Content:        ui.View(nativeApp.View),
		}
		win := mygo.NewWindow(opts)
		if len(appIconPNG) > 0 {
			_ = win.SetIcon(appIconPNG)
		}
		nativeApp.SetWebViewFactory(func() (nativeui.WebViewHost, error) {
			wv, err := win.NewWebView(mygo.WebViewOptions{
				Page: mygo.PageOptions{Untrusted: true},
			})
			if err != nil {
				return nil, err
			}
			return &mygoWebViewHost{wv: wv}, nil
		})

		showMainWindow := func() {
			if win.IsDestroyed() {
				return
			}
			if win.IsMinimized() {
				win.Restore()
			}
			win.Show()
			win.Focus()
		}

		var tray *mygo.Tray
		var quitting atomic.Bool
		var closing atomic.Bool

		quitApp := func() {
			if !closing.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer closing.Store(false)
				if library.ActivePublishCount() > 0 {
					showMainWindow()
					title, msg, detail, cancelBtn, quitBtn := trayQuitConfirmStrings(nativeApp.Locale())
					result, err := mygo.Dialog.Message(mygo.MessageOptions{
						Parent:        win,
						Type:          mygo.MessageQuestion,
						Title:         title,
						Message:       msg,
						Detail:        detail,
						Buttons:       []string{cancelBtn, quitBtn},
						DefaultButton: 0,
						CancelButton:  0,
					})
					if err != nil || result.Button != 1 {
						return
					}
				}
				quitting.Store(true)
				if tray != nil {
					tray.Destroy()
					tray = nil
				}
				nativeApp.Close()
				if !win.IsDestroyed() {
					win.Destroy()
				}
				mygo.App.Quit()
			}()
		}

		buildTrayMenu := func(loc nativeui.Locale) *mygo.Menu {
			showLabel, quitLabel := trayMenuStrings(loc)
			return mygo.NewMenu([]*mygo.MenuItem{
				{
					Label: showLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						showMainWindow()
					},
				},
				mygo.Separator(),
				{
					Label: quitLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						quitApp()
					},
				},
			})
		}

		if createdTray, err := mygo.NewTray(mygo.TrayOptions{
			ID:      "kinpage-tray",
			Icon:    appIconPNG,
			ToolTip: "邻页 · Kinpage",
			Menu:    buildTrayMenu(nativeApp.Locale()),
		}); err == nil {
			tray = createdTray
			tray.OnClick(showMainWindow)
			nativeApp.SetOnLocaleChange(func(loc nativeui.Locale) {
				if tray != nil {
					tray.SetMenu(buildTrayMenu(loc))
				}
			})
		} else {
			log.Printf("system tray unavailable: %v", err)
		}

		win.OnClose(func(event *mygo.CloseEvent) {
			if quitting.Load() {
				return
			}
			event.PreventDefault()
			// Hide to the Windows taskbar system tray so Planet continues running
			// in the background (IPFS node, Local API, and site maintenance).
			if tray != nil {
				win.Hide()
				return
			}
			// Fallback when system tray is unavailable: confirm and exit.
			if !closing.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer closing.Store(false)
				detail := "尚未保存的文章更改将会丢失。"
				if library.ActivePublishCount() > 0 {
					detail = "当前正在进行 IPNS 发布，强制关闭可能导致发布中断或站点无法解析。"
				}
				result, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageQuestion, Title: "关闭邻页", Message: "确定关闭窗口？", Detail: detail, Buttons: []string{"取消", "关闭"}, DefaultButton: 0, CancelButton: 0})
				if err == nil && result.Button == 1 {
					quitting.Store(true)
					nativeApp.Close()
					win.Destroy()
				}
			}()
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func trayMenuStrings(loc nativeui.Locale) (showLabel, quitLabel string) {
	switch loc {
	case nativeui.LocaleZhCN:
		return "显示邻页", "退出邻页"
	case nativeui.LocaleZhTW:
		return "顯示鄰頁", "結束鄰頁"
	default:
		return "Show Kinpage", "Quit Kinpage"
	}
}

func trayQuitConfirmStrings(loc nativeui.Locale) (title, msg, detail, cancelBtn, quitBtn string) {
	switch loc {
	case nativeui.LocaleZhCN:
		return "退出邻页", "确定退出邻页？", "当前正在进行 IPNS 发布，强制退出可能导致发布中断或站点无法解析。", "取消", "退出"
	case nativeui.LocaleZhTW:
		return "結束鄰頁", "確定要結束鄰頁？", "目前正在進行 IPNS 發布，強制結束可能導致發布中斷或站點無法解析。", "取消", "結束"
	default:
		return "Quit Kinpage", "Quit Kinpage?", "An IPNS publish is currently in progress. Quitting now may interrupt publishing.", "Cancel", "Quit"
	}
}

type mygoWebViewHost struct {
	wv *mygo.WebView
}

func (h *mygoWebViewHost) Native() ui.NativeWebView {
	if h == nil || h.wv == nil {
		return nil
	}
	return h.wv
}

func (h *mygoWebViewHost) LoadHTML(html, baseURL string) {
	if h != nil && h.wv != nil && !h.wv.IsDestroyed() {
		h.wv.Page().LoadHTML(html, baseURL)
	}
}

func (h *mygoWebViewHost) LoadURL(rawURL string) error {
	if h == nil || h.wv == nil || h.wv.IsDestroyed() {
		return nil
	}
	return h.wv.Page().LoadURL(rawURL)
}

func (h *mygoWebViewHost) Destroy() {
	if h != nil && h.wv != nil && !h.wv.IsDestroyed() {
		h.wv.Destroy()
	}
}
