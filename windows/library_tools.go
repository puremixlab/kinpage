package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/applog"
	"windows/internal/localapi"
	"windows/internal/offline"
	"windows/internal/services"
	"windows/internal/store"
)

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
