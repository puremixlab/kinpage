package main

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/attachments"
	"windows/internal/audiostream"
	"windows/internal/feeds"
	"windows/internal/offline"
	"windows/internal/store"
)

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
