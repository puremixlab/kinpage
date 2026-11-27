package main

import (
	"context"
	_ "embed"
	"encoding/base64"
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
	"windows/internal/attachments"
	"windows/internal/audiostream"
	"windows/internal/backup"
	"windows/internal/feeds"
	"windows/internal/localapi"
	"windows/internal/migration"
	"windows/internal/nativeui"
	"windows/internal/offline"
	"windows/internal/pinnable"
	"windows/internal/services"
	"windows/internal/sitegen"
	"windows/internal/store"
	"windows/internal/templateinstall"
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

func (l *Library) AddDraftAttachment(ctx context.Context, siteID, draftID string) (store.Draft, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择草稿附件"})
	if err != nil || len(paths) == 0 {
		return store.Draft{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Draft{}, err
	}
	return attachments.AddDraft(root, s, siteID, draftID, paths[0])
}

func (l *Library) AddDraftAttachmentBytes(siteID, draftID, name, dataBase64 string) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Draft{}, err
	}
	return attachments.AddDraftBytes(root, s, siteID, draftID, name, dataBase64)
}

func (l *Library) RemoveDraftAttachment(siteID, draftID, name string) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Draft{}, err
	}
	return attachments.RemoveDraft(root, s, siteID, draftID, name)
}

// AttachArticleFiles opens a native picker for one kind of editor attachment
// ("image", "video", or "audio") and records every selected file on the saved
// article, mirroring the macOS writer toolbar entries.
func (l *Library) AttachArticleFiles(ctx context.Context, siteID, articleID, kind string) (store.Article, error) {
	paths, err := chooseEditorAttachments(ctx, kind, false)
	if err != nil || len(paths) == 0 {
		return store.Article{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	article := store.Article{}
	for i, path := range paths {
		article, err = attachments.Add(root, s, siteID, articleID, path)
		if err != nil {
			if i == 0 {
				return store.Article{}, err
			}
			break
		}
	}
	return article, err
}

// AttachDraftFiles is the draft counterpart of AttachArticleFiles.
func (l *Library) AttachDraftFiles(ctx context.Context, siteID, draftID, kind string) (store.Draft, error) {
	paths, err := chooseEditorAttachments(ctx, kind, false)
	if err != nil || len(paths) == 0 {
		return store.Draft{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Draft{}, err
	}
	draft := store.Draft{}
	for i, path := range paths {
		draft, err = attachments.AddDraft(root, s, siteID, draftID, path)
		if err != nil {
			if i == 0 {
				return store.Draft{}, err
			}
			break
		}
	}
	return draft, err
}

func chooseEditorAttachments(ctx context.Context, kind string, multiple bool) ([]string, error) {
	var options mygo.OpenDialogOptions
	switch kind {
	case "image":
		options = mygo.OpenDialogOptions{
			Title: "选择附件照片（PNG、JPEG、GIF 或 WebP）",
			Filters: []mygo.FileFilter{
				{Name: "图片", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp"}},
			},
			Multiple: multiple,
		}
	case "video":
		options = mygo.OpenDialogOptions{
			Title: "选择附件视频（MP4、MOV、M4V 或 WebM）",
			Filters: []mygo.FileFilter{
				{Name: "视频", Extensions: []string{"mp4", "mov", "m4v", "webm"}},
				{Name: "全部文件", Extensions: []string{"*"}},
			},
		}
	case "audio":
		options = mygo.OpenDialogOptions{
			Title: "选择附件音频（MP3、M4A、AAC 或 WAV）",
			Filters: []mygo.FileFilter{
				{Name: "音频", Extensions: []string{"mp3", "m4a", "aac", "wav"}},
				{Name: "全部文件", Extensions: []string{"*"}},
			},
		}
	default:
		return nil, errors.New("附件类型无效")
	}
	options.Parent = mygo.CallerWindow(ctx)
	return mygo.Dialog.Open(options)
}

// SetArticleHeroImage records an explicit cover choice; an empty name clears it.
func (l *Library) SetArticleHeroImage(siteID, articleID, name string) (store.Article, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.SetHeroImage(siteID, articleID, name)
}

// SetDraftHeroImage records an explicit cover choice on a draft; empty clears it.
func (l *Library) SetDraftHeroImage(siteID, draftID, name string) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	return s.SetDraftHeroImage(siteID, draftID, name)
}

// EditorAttachmentMediaURL streams an article or draft attachment (audio or
// video) for the editor players over the private loopback server. One server
// serves both players; StopEditorMedia revokes it when the editor moves on.
func (l *Library) EditorAttachmentMediaURL(siteID, articleID, draftID, name string) (string, error) {
	s, err := l.open()
	if err != nil {
		return "", err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	lookup := func(requested string) (*os.File, error) {
		if draftID != "" {
			drafts, err := s.ListDrafts(siteID)
			if err != nil {
				return nil, err
			}
			for _, draft := range drafts {
				if draft.ID == draftID {
					return attachments.OpenDraftAttachment(root, draft, requested)
				}
			}
			return nil, os.ErrNotExist
		}
		site, articles, _, err := s.PublicationSnapshot(siteID)
		if err != nil {
			return nil, err
		}
		for _, article := range articles {
			if article.ID == articleID {
				return attachments.OpenRegistered(root, site, article, requested)
			}
		}
		return nil, os.ErrNotExist
	}
	l.editorMediaMu.Lock()
	defer l.editorMediaMu.Unlock()
	if l.editorMedia == nil {
		server, err := audiostream.New(func(_, _, requested string) (*os.File, error) {
			return lookup(requested)
		})
		if err != nil {
			return "", err
		}
		l.editorMedia = server
	}
	address, err := l.editorMedia.URLWithType(siteID, articleID, name, "")
	if err != nil {
		return "", err
	}
	return address, nil
}

// StopEditorMedia revokes the editor media stream when playback ends or the
// attachment is removed.
func (l *Library) StopEditorMedia() {
	l.editorMediaMu.Lock()
	defer l.editorMediaMu.Unlock()
	if l.editorMedia != nil {
		_ = l.editorMedia.Close()
		l.editorMedia = nil
	}
}

// EditorAttachmentThumbnail returns a small inline preview for the media tray.
func (l *Library) EditorAttachmentThumbnail(siteID, articleID, draftID, name string) (string, error) {
	s, err := l.open()
	if err != nil {
		return "", err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	if draftID != "" {
		drafts, err := s.ListDrafts(siteID)
		if err != nil {
			return "", err
		}
		for _, draft := range drafts {
			if draft.ID == draftID {
				data, mime, err := attachments.ReadDraftPreviewImage(root, draft, name)
				if err != nil {
					return "", err
				}
				return attachments.ThumbnailDataURL(data, mime)
			}
		}
		return "", os.ErrNotExist
	}
	site, articles, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return "", err
	}
	for _, article := range articles {
		if article.ID == articleID {
			data, mime, err := attachments.ReadPreviewImage(root, site, article, name)
			if err != nil {
				return "", err
			}
			return attachments.ThumbnailDataURL(data, mime)
		}
	}
	return "", os.ErrNotExist
}

func (l *Library) SetDraftExistingAttachmentRemoved(siteID, draftID, name string, removed bool) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	return s.SetDraftExistingAttachmentRemoved(siteID, draftID, name, removed)
}

func (l *Library) CompressDraftVideo(siteID, draftID, name, preset string) (store.Draft, error) {
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Draft{}, err
	}
	return attachments.CompressDraftVideo(root, s, siteID, draftID, name, preset)
}

func (l *Library) PickQuickPostAttachment(ctx context.Context) (string, string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择快捷发文附件（图片、视频或音频）"})
	if err != nil || len(paths) == 0 {
		return "", "", err
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		return "", "", err
	}
	if len(raw) > 100<<20 {
		return "", "", errors.New("附件大小不能超过 100 MB")
	}
	return filepath.Base(paths[0]), base64.StdEncoding.EncodeToString(raw), nil
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

func (l *Library) RebuildSite(siteID string, quick bool) (int, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return 0, err
	}
	sites, err := s.ListSites()
	if err != nil {
		return 0, err
	}
	var selected *store.Site
	for i := range sites {
		if sites[i].ID == siteID {
			selected = &sites[i]
			break
		}
	}
	if selected == nil {
		return 0, errors.New("站点不存在")
	}
	articles, err := s.ListArticles(siteID)
	if err != nil {
		return 0, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return 0, err
	}
	outDir := filepath.Join(root, "sites", siteID)
	templateDir, err := l.templatePath(*selected)
	if err != nil {
		return 0, err
	}
	if quick {
		if err := sitegen.RenderIndexOnly(outDir, *selected, articles, templateDir); err != nil {
			return 0, err
		}
		return len(articles), nil
	}
	enriched, err := attachments.EnrichMedia(root, *selected, articles)
	if err != nil {
		return 0, err
	}
	if templateDir != "" {
		if err := sitegen.RenderWithTemplate(outDir, *selected, enriched, templateDir); err != nil {
			return 0, err
		}
	} else {
		if err := sitegen.Render(outDir, *selected, enriched); err != nil {
			return 0, err
		}
	}
	if err := attachments.CopyToSite(root, outDir, enriched); err != nil {
		return 0, err
	}
	if err := sitegen.WriteAliases(outDir, enriched); err != nil {
		return 0, err
	}
	return len(enriched), nil
}

func (l *Library) ExportArticle(ctx context.Context, siteID, articleID string) (string, error) {
	dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择导出文章数据包的目录", Directory: true})
	if err != nil || len(dirs) == 0 {
		return "", err
	}
	s, err := l.open()
	if err != nil {
		return "", err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	name := articleID
	if arts, err := s.ListArticles(siteID); err == nil {
		for _, a := range arts {
			if a.ID == articleID && strings.TrimSpace(a.Slug) != "" {
				name = a.Slug
				break
			}
		}
	}
	dest := filepath.Join(dirs[0], name+".article.zip")
	if err := attachments.ExportArticleZip(root, s, siteID, articleID, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func (l *Library) ImportArticle(ctx context.Context, siteID string) (store.Article, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择要导入的文章数据包 (.article.zip)"})
	if err != nil || len(paths) == 0 {
		return store.Article{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.ImportArticleZip(root, s, siteID, paths[0])
}

// AddAttachment copies a selected local file into a saved Windows article.
func (l *Library) AddAttachment(ctx context.Context, siteID, articleID string) (store.Article, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择文章附件"})
	if err != nil || len(paths) == 0 {
		return store.Article{}, err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.Add(root, s, siteID, articleID, paths[0])
}

func (l *Library) AddAttachmentBytes(siteID, articleID, name, dataBase64 string) (store.Article, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.AddBytes(root, s, siteID, articleID, name, dataBase64)
}

// RemoveAttachment removes a file previously added to a Windows article.
func (l *Library) RemoveAttachment(siteID, articleID, name string) (store.Article, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.Remove(root, s, siteID, articleID, name)
}

// AudioURL provides a revocable loopback stream for the selected article audio.
func (l *Library) AudioURL(siteID, articleID string) (string, error) {
	s, err := l.open()
	if err != nil {
		return "", err
	}
	_, articles, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return "", err
	}
	name := ""
	for _, a := range articles {
		if a.ID == articleID {
			name = a.AudioFilename
			break
		}
	}
	if name == "" {
		return "", errors.New("文章没有音频附件")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	open := func(siteID, articleID, name string) (*os.File, error) {
		site, articles, _, err := s.PublicationSnapshot(siteID)
		if err != nil {
			return nil, err
		}
		for _, a := range articles {
			if a.ID == articleID && a.AudioFilename == name {
				return attachments.OpenRegistered(root, site, a, name)
			}
		}
		return nil, os.ErrNotExist
	}
	return l.startAudio(open, siteID, articleID, name, "")
}

// SubscriptionAudioURL streams a registered Planet audio file from a private temporary copy.
func (l *Library) SubscriptionAudioURL(address, key string) (string, error) {
	s, err := l.open()
	if err != nil {
		return "", err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return "", err
	}
	var item *feeds.Item
	for _, sub := range subs {
		if sub.URL == address {
			for i := range sub.Feed.Items {
				a := &sub.Feed.Items[i]
				if feeds.ItemKey(*a) == key {
					item = a
					break
				}
			}
			break
		}
	}
	if item == nil {
		return "", errors.New("订阅文章不存在")
	}
	if item.ContentCID == "" && item.AudioURL != "" {
		return l.SubscriptionMediaURL(address, key, item.AudioURL)
	}
	if item.ContentCID == "" || item.ArticleID == "" || item.AudioFilename == "" {
		return "", errors.New("订阅文章没有音频附件")
	}
	found := false
	for _, name := range item.Attachments {
		if name == item.AudioFilename {
			found = true
			break
		}
	}
	if !found {
		return "", errors.New("音频附件未登记")
	}
	temp, err := os.CreateTemp("", "planet-audio-*")
	if err != nil {
		return "", err
	}
	path := temp.Name()
	l.publicMu.Lock()
	_, _, err = l.ensureNode(true)
	if err == nil {
		path := item.ArticleID + "/" + item.AudioFilename
		if item.ContentPath != "" {
			path = item.ContentPath + "/" + path
		}
		err = l.node.CopyContent(item.ContentCID, path, temp)
	}
	l.publicMu.Unlock()
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	open := func(siteID, articleID, name string) (*os.File, error) {
		if siteID != address || articleID != key || name != item.AudioFilename {
			return nil, os.ErrNotExist
		}
		return os.Open(path)
	}
	return l.startAudio(open, address, key, item.AudioFilename, path)
}

// SubscriptionMediaURL serves an already-fixed feed media file without network access.
func (l *Library) SubscriptionMediaURL(address, key, mediaURL string) (string, error) {
	l.feedMu.Lock()
	defer l.feedMu.Unlock()
	s, err := l.open()
	if err != nil {
		return "", err
	}
	subs, err := s.ListSubscriptions()
	if err != nil {
		return "", err
	}
	var asset feeds.OfflineAsset
	found := false
	for _, sub := range subs {
		if sub.URL == address {
			for _, item := range sub.Feed.Items {
				if feeds.ItemKey(item) == key {
					asset, found = item.OfflineMedia[mediaURL]
					break
				}
			}
			break
		}
	}
	if !found || (!strings.HasPrefix(asset.ContentType, "audio/") && !strings.HasPrefix(asset.ContentType, "video/")) {
		return "", errors.New("媒体尚未固定到本机")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	open := func(site, article, name string) (*os.File, error) {
		if site != address || article != key || name != mediaURL {
			return nil, os.ErrNotExist
		}
		return offline.Open(root, asset)
	}
	return l.startMedia(open, address, key, mediaURL, "", asset.ContentType, asset.Snapshot, asset.File)
}

func (l *Library) startAudio(open func(string, string, string) (*os.File, error), site, article, name, temp string) (string, error) {
	return l.startMedia(open, site, article, name, temp, "", "", "")
}
func (l *Library) startMedia(open func(string, string, string) (*os.File, error), site, article, name, temp, mediaType, snapshot, file string) (string, error) {
	l.audioMu.Lock()
	defer l.audioMu.Unlock()
	l.closeAudioLocked()
	server, err := audiostream.New(open)
	if err != nil {
		if temp != "" {
			_ = os.Remove(temp)
		}
		return "", err
	}
	address, err := server.URLWithType(site, article, name, mediaType)
	if err != nil {
		_ = server.Close()
		if temp != "" {
			_ = os.Remove(temp)
		}
		return "", err
	}
	l.audio = server
	l.audioTemp = temp
	l.audioSnapshot, l.audioFile = snapshot, file
	return address, nil
}

// StopAudio revokes the requested stream without stopping newer playback.
func (l *Library) StopAudio(address string) {
	l.audioMu.Lock()
	defer l.audioMu.Unlock()
	if l.audio != nil && l.audio.Revoke(address) {
		l.closeAudioLocked()
	}
}
func (l *Library) closeAudioLocked() {
	l.audioSnapshot, l.audioFile = "", ""
	if l.audio != nil {
		_ = l.audio.Close()
		l.audio = nil
	}
	if l.audioTemp != "" {
		_ = os.Remove(l.audioTemp)
		l.audioTemp = ""
	}
}
func (l *Library) closeAudio() { l.audioMu.Lock(); defer l.audioMu.Unlock(); l.closeAudioLocked() }

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

func (l *Library) CheckTemplateModified(siteID string) (int64, error) {
	s, err := l.open()
	if err != nil {
		return 0, err
	}
	site, _, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return 0, err
	}
	dir, err := l.templatePath(site)
	if err != nil || dir == "" {
		return 0, err
	}
	var latest int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if mt := info.ModTime().UnixNano(); mt > latest {
				latest = mt
			}
		}
		return nil
	})
	return latest, nil
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

func (l *Library) templatePath(site store.Site) (string, error) {
	if site.TemplateID == "" {
		return "", nil
	}
	if !strings.HasPrefix(site.TemplateID, "template-") || !backup.SafeName(site.TemplateID) {
		return "", errors.New("模板目录无效")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, "templates", site.TemplateID)
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("模板目录不存在或无效")
	}
	if err := templateinstall.Verify(path); err != nil {
		return "", fmt.Errorf("模板未通过验证：%w", err)
	}
	return path, nil
}

func (l *Library) SelectTemplate(ctx context.Context, siteID string) (store.Site, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择模板目录", Directory: true})
	if err != nil || len(paths) == 0 {
		return store.Site{}, err
	}
	return l.installTemplate(siteID, paths[0])
}

func (l *Library) BuiltinTemplates() []templateinstall.Builtin { return templateinstall.Builtins() }

func (l *Library) BuiltinTemplatePreview(id string) (string, error) {
	return templateinstall.BuiltinPreview(id)
}

func (l *Library) ApplyBuiltinTemplate(siteID, id string) (store.Site, error) {
	source, cleanup, err := templateinstall.ExtractBuiltin(id)
	if err != nil {
		return store.Site{}, err
	}
	defer cleanup()
	return l.installTemplate(siteID, source)
}

func (l *Library) installTemplate(siteID, source string) (store.Site, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	raw, err := s.BackupMetadata(siteID)
	if err != nil {
		return store.Site{}, err
	}
	sites, err := s.ListSites()
	if err != nil {
		return store.Site{}, err
	}
	var site *store.Site
	for i := range sites {
		if sites[i].ID == siteID {
			site = &sites[i]
			break
		}
	}
	if site == nil {
		return store.Site{}, errors.New("站点不存在")
	}
	articles, err := s.ListArticles(siteID)
	if err != nil {
		return store.Site{}, err
	}
	if len(raw) > 0 {
		if err := backup.CheckSimplePlain(raw, *site, articles); err != nil {
			return store.Site{}, err
		}
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Site{}, err
	}
	if _, err := templateinstall.Install(root, source, *site, articles, s); err != nil {
		return store.Site{}, err
	}
	updated, err := s.ListSites()
	if err != nil {
		return store.Site{}, err
	}
	for _, entry := range updated {
		if entry.ID == siteID {
			return entry, nil
		}
	}
	return store.Site{}, errors.New("站点不存在")
}

// TemplateSettings returns supported settings, including imported values.
func (l *Library) TemplateSettings(siteID string) (map[string]string, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return nil, err
	}
	if site.TemplateName != "Sepia" || site.TemplateID == "" {
		return nil, errors.New("当前模板没有可编辑的设置")
	}
	settings := site.TemplateSettings
	if settings == nil && len(raw) > 0 {
		root, err := mygo.App.Path(mygo.PathUserData)
		if err != nil {
			return nil, err
		}
		settings, err = migration.LoadTemplateSettings(filepath.Join(root, "imports", siteID), raw, site.TemplateName)
		if err != nil {
			return nil, err
		}
	}
	return sitegen.SepiaSettings(settings)
}

// SaveTemplateSettings validates values before persisting them locally.
func (l *Library) SaveTemplateSettings(siteID, values string) (store.Site, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if len(values) > 1<<20 {
		return store.Site{}, errors.New("模板设置过大")
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal([]byte(values), &raw) != nil || raw == nil {
		return store.Site{}, errors.New("模板设置无效")
	}
	normalized, err := sitegen.NormalizeSettings("Sepia", raw)
	if err != nil {
		return store.Site{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetTemplateSettings(siteID, normalized)
}

// ImportPlanet imports a selected original .planet folder.
func (l *Library) ImportPlanet(ctx context.Context) (store.Site, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择 .planet 备份目录", Directory: true})
	if err != nil {
		return store.Site{}, err
	}
	if len(paths) == 0 {
		return store.Site{}, nil
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	root, exe, err := l.ensureNode(false)
	if err != nil {
		return store.Site{}, err
	}
	return migration.New(root, exe, s, l.node).Import(paths[0])
}

// ExportPlanet writes a .planet backup containing the site's private key.
func (l *Library) ExportPlanet(ctx context.Context, siteID string) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择备份保存位置", Directory: true})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return "", err
	}
	root, exe, err := l.ensureNode(false)
	if err != nil {
		return "", err
	}
	return migration.New(root, exe, s, l.node).Export(siteID, paths[0])
}

// ExportWebsite creates a new folder containing a basic static website.
func (l *Library) ExportWebsite(ctx context.Context, siteID string) (string, error) {
	s, err := l.open()
	if err != nil {
		return "", err
	}
	sites, err := s.ListSites()
	if err != nil {
		return "", err
	}
	var selected *store.Site
	for i := range sites {
		if sites[i].ID == siteID {
			selected = &sites[i]
			break
		}
	}
	if selected == nil {
		return "", errors.New("站点不存在")
	}
	articles, err := s.ListArticles(siteID)
	if err != nil {
		return "", err
	}
	raw, err := s.BackupMetadata(siteID)
	if err != nil {
		return "", err
	}
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择网站导出位置", Directory: true})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	dir, err := os.MkdirTemp(paths[0], "Planet-website-")
	if err != nil {
		return "", err
	}
	templateDir, err := l.templatePath(*selected)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if len(raw) > 0 {
		l.publicMu.Lock()
		defer l.publicMu.Unlock()
		*selected, articles, raw, err = s.PublicationSnapshot(siteID)
		if err == nil {
			templateDir, err = l.templatePath(*selected)
		}
		if err == nil && templateDir == "" {
			err = errors.New("请先选择已验证的模板")
		}
		if err == nil {
			err = migration.StageImportedPublic(filepath.Join(root, "imports", siteID), dir, raw, *selected, articles, templateDir)
		}
	} else {
		articles, err = attachments.EnrichMedia(root, *selected, articles)
		if err == nil && templateDir != "" {
			err = sitegen.RenderWithTemplate(dir, *selected, articles, templateDir)
		} else if err == nil {
			err = sitegen.Render(dir, *selected, articles)
		}
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if len(raw) == 0 {
		err = attachments.CopyToSite(root, dir, articles)
		if err == nil {
			for _, article := range articles {
				if err = attachments.GenerateVideoThumbnail(root, *selected, article, dir); err != nil {
					break
				}
				if err = attachments.GenerateGridThumbnails(root, *selected, article, dir); err != nil {
					break
				}
			}
		}
		if err == nil {
			err = sitegen.WriteAliases(dir, articles)
		}
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
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
