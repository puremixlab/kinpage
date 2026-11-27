package attachments

import (
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"windows/internal/backup"
	"windows/internal/mediainfo"
	"windows/internal/store"
)

// AddDraft keeps newly selected files outside all public and backup trees.
// Audio and video follow the macOS writer's single-attachment rule: adding one
// replaces the previous draft file of the same kind and, for a draft editing a
// saved article, marks the article's previous file of that kind as removed.
func AddDraft(root string, library *store.Store, siteID, draftID, source string) (store.Draft, error) {
	name := filepath.Base(source)
	if !validName(name) || strings.HasPrefix(strings.ToLower(name), ".draft-stage-") {
		return store.Draft{}, errors.New("附件文件名无效或与生成文件冲突")
	}
	CleanupDraftFiles(root, library)
	drafts, err := library.ListDrafts(siteID)
	if err != nil {
		return store.Draft{}, err
	}
	var matched *store.Draft
	for i := range drafts {
		if drafts[i].ID == draftID {
			matched = &drafts[i]
			break
		}
	}
	if matched == nil {
		return store.Draft{}, errors.New("草稿不存在")
	}
	kind := attachmentKind(name)
	replaced := ""
	for _, old := range matched.Attachments {
		if strings.EqualFold(old, name) || ((kind == "audio" || kind == "video") && attachmentKind(old) == kind) {
			replaced = old
			break
		}
	}
	if replaced == "" && (kind == "audio" || kind == "video") && matched.ArticleID != "" {
		posts, err := library.ListArticles(siteID)
		if err != nil {
			return store.Draft{}, err
		}
		for _, post := range posts {
			if post.ID != matched.ArticleID {
				continue
			}
			existing := post.AudioFilename
			if kind == "video" {
				existing = post.VideoFilename
			}
			removedAlready := false
			for _, removed := range matched.RemovedAttachments {
				if removed == existing {
					removedAlready = true
					break
				}
			}
			if existing != "" && !removedAlready {
				if _, err := library.SetDraftExistingAttachmentRemoved(siteID, draftID, existing, true); err != nil {
					return store.Draft{}, err
				}
			}
			break
		}
	}
	if replaced != "" {
		if _, err := RemoveDraft(root, library, siteID, draftID, replaced); err != nil {
			return store.Draft{}, err
		}
		CleanupDraftFiles(root, library)
	}
	for _, entry := range library.PendingDraftCleanup() {
		if entry.SiteID == siteID && entry.DraftID == draftID && strings.EqualFold(entry.Name, name) {
			return store.Draft{}, errors.New("原草稿附件等待清理，请稍后重试")
		}
	}
	dir, err := managedDir(root, "draft-attachments", siteID, draftID, true)
	if err != nil {
		return store.Draft{}, err
	}
	target := filepath.Join(dir, name)
	_, err = os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		if err = library.BeginDraftAttachment(siteID, draftID, name); err != nil {
			return store.Draft{}, err
		}
	} else if err != nil {
		return store.Draft{}, err
	}
	_, err = copyOrReuse(source, target, dir)
	if err != nil {
		CleanupDraftFiles(root, library)
		return store.Draft{}, err
	}
	draft, err := library.ChangeDraftAttachment(siteID, draftID, name, true)
	if err != nil {
		CleanupDraftFiles(root, library)
	}
	return draft, err
}

// RemoveDraft removes a newly added draft file without changing its article.
func RemoveDraft(root string, library *store.Store, siteID, draftID, name string) (store.Draft, error) {
	if !validName(name) {
		return store.Draft{}, errors.New("附件名称无效")
	}
	drafts, err := library.ListDrafts(siteID)
	if err != nil {
		return store.Draft{}, err
	}
	found := false
	for _, draft := range drafts {
		if draft.ID == draftID {
			for _, item := range draft.Attachments {
				if item == name {
					found = true
				}
			}
		}
	}
	if !found {
		return store.Draft{}, errors.New("草稿附件不存在")
	}
	draft, err := library.ChangeDraftAttachment(siteID, draftID, name, false)
	if err != nil {
		return store.Draft{}, err
	}
	CleanupDraftFiles(root, library)
	return draft, nil
}

// PublishDraft stages files before committing article and draft state together.
func PublishDraft(root string, library *store.Store, siteID, draftID, title, content string) (store.Article, error) {
	return publishDraft(root, library, siteID, draftID, title, content, nil)
}

func PublishDraftWithTags(root string, library *store.Store, siteID, draftID, title, content, tagsJSON string) (store.Article, error) {
	return publishDraft(root, library, siteID, draftID, title, content, &tagsJSON)
}

func publishDraft(root string, library *store.Store, siteID, draftID, title, content string, tagsJSON *string) (store.Article, error) {
	CleanupArticleFiles(root, library)
	prepare := func(article store.Article, names []string) (store.Article, func(), error) {
		src, err := managedDir(root, "draft-attachments", siteID, draftID, false)
		if err != nil {
			return store.Article{}, nil, err
		}
		dst, err := ownedDir(root, siteID, article.ID, true)
		if err != nil {
			return store.Article{}, nil, err
		}
		var created []string
		rollback := func() {
			for _, path := range created {
				_ = os.Remove(path)
			}
		}
		article.Attachments = append([]string(nil), article.Attachments...)
		for _, name := range names {
			if !validName(name) {
				rollback()
				return store.Article{}, nil, errors.New("草稿附件名称无效")
			}
			for _, old := range article.Attachments {
				if strings.EqualFold(old, name) {
					rollback()
					return store.Article{}, nil, errors.New("文章已有同名附件")
				}
			}
			target := filepath.Join(dst, name)
			wasCreated, err := copyOrReuse(filepath.Join(src, name), target, src)
			if err != nil {
				rollback()
				return store.Article{}, nil, err
			}
			if wasCreated {
				created = append(created, target)
			}
			article.Attachments = append(article.Attachments, name)
			switch strings.ToLower(filepath.Ext(name)) {
			case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".opus", ".flac":
				info, err := os.Stat(target)
				if err != nil {
					rollback()
					return store.Article{}, nil, err
				}
				size := info.Size()
				article.AudioFilename, article.AudioByteLength, article.AudioDuration = name, &size, mediainfo.Duration(target)
			case ".mp4", ".mov", ".m4v", ".webm", ".avi", ".mpeg", ".mpg":
				article.VideoFilename = name
			}
		}
		return article, rollback, nil
	}
	var article store.Article
	var err error
	if tagsJSON != nil {
		article, err = library.PublishDraftPreparedWithTags(siteID, draftID, title, content, *tagsJSON, prepare)
	} else {
		article, err = library.PublishDraftPrepared(siteID, draftID, title, content, prepare)
	}
	if err == nil {
		CleanupDraftFiles(root, library)
		CleanupArticleFiles(root, library)
	}
	return article, err
}

// CleanupDraftFiles retries only removals recorded in the durable store.
func CleanupDraftFiles(root string, library *store.Store) {
	protected := map[string]bool{}
	for _, entry := range library.DraftFileReferences() {
		protected[strings.ToLower(filepath.Join(entry.SiteID, entry.DraftID, entry.Name))] = true
	}
	for _, entry := range library.PendingDraftStaging() {
		if !validName(entry.Name) {
			continue
		}
		if protected[strings.ToLower(filepath.Join(entry.SiteID, entry.DraftID, entry.Name))] {
			_ = library.CompleteDraftStaging(entry)
			continue
		}
		dir, err := managedDir(root, "draft-attachments", entry.SiteID, entry.DraftID, false)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftStaging(entry)
			continue
		}
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entry.Name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftStaging(entry)
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err = os.Remove(path); err == nil || errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftStaging(entry)
		}
	}
	for _, entry := range library.PendingDraftCleanup() {
		if !validName(entry.Name) {
			continue
		}
		dir, err := managedDir(root, "draft-attachments", entry.SiteID, entry.DraftID, false)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftCleanup(entry)
			continue
		}
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entry.Name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftCleanup(entry)
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err = os.Remove(path); err == nil || errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteDraftCleanup(entry)
		}
	}
	// Interrupted copies can leave private staging files before registration.
	// Never remove a name recorded by any draft, including a removed site's.
	base := filepath.Join(root, "draft-attachments")
	info, err := os.Lstat(base)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return
	}
	sites, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, site := range sites {
		if !backup.SafeName(site.Name()) {
			continue
		}
		sitePath := filepath.Join(base, site.Name())
		info, err := os.Lstat(sitePath)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		drafts, err := os.ReadDir(sitePath)
		if err != nil {
			continue
		}
		for _, draft := range drafts {
			if !backup.SafeName(draft.Name()) {
				continue
			}
			dir, err := managedDir(root, "draft-attachments", site.Name(), draft.Name(), false)
			if err != nil {
				continue
			}
			files, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, file := range files {
				name := file.Name()
				if !strings.HasPrefix(strings.ToLower(name), ".draft-stage-") || protected[strings.ToLower(filepath.Join(site.Name(), draft.Name(), name))] {
					continue
				}
				path := filepath.Join(dir, name)
				info, err := os.Lstat(path)
				if err == nil && info.Mode().IsRegular() {
					_ = os.Remove(path)
				}
			}
		}
	}
}

// copyOrReuse uses a completed temporary copy and a no-replace hard link. An
// identical leftover from an interrupted commit can safely be reused.
func copyOrReuse(src, dst, temporaryDir string) (bool, error) {
	if _, err := os.Lstat(dst); err == nil {
		equal, err := sameRegular(src, dst)
		if err != nil {
			return false, err
		}
		if !equal {
			return false, errors.New("存在不同内容的同名残留附件")
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	temp, err := os.CreateTemp(temporaryDir, ".draft-stage-*")
	if err != nil {
		return false, err
	}
	name := temp.Name()
	if err = temp.Close(); err != nil {
		_ = os.Remove(name)
		return false, err
	}
	if err = os.Remove(name); err != nil {
		return false, err
	}
	defer os.Remove(name)
	if err = copyRegular(src, name); err != nil {
		return false, err
	}
	if err = os.Link(name, dst); err != nil {
		if errors.Is(err, os.ErrExist) {
			equal, e := sameRegular(src, dst)
			if e == nil && equal {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func sameRegular(a, b string) (bool, error) {
	digest := func(path string) ([32]byte, error) {
		var zero [32]byte
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			return zero, errors.New("附件必须是普通文件")
		}
		f, err := os.Open(path)
		if err != nil {
			return zero, err
		}
		defer f.Close()
		actual, err := f.Stat()
		if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
			return zero, errors.New("附件在读取期间发生变化")
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return zero, err
		}
		var sum [32]byte
		copy(sum[:], h.Sum(nil))
		return sum, nil
	}
	x, err := digest(a)
	if err != nil {
		return false, err
	}
	y, err := digest(b)
	if err != nil {
		return false, err
	}
	return x == y, nil
}

// OpenDraftAttachment opens a staged draft attachment without following links.
func OpenDraftAttachment(root string, draft store.Draft, name string) (*os.File, error) {
	if !validName(name) {
		return nil, errors.New("附件名称无效")
	}
	registered := false
	for _, item := range draft.Attachments {
		if item == name {
			registered = true
			break
		}
	}
	if !registered {
		return nil, errors.New("草稿附件未登记")
	}
	dir, err := managedDir(root, "draft-attachments", draft.SiteID, draft.ID, false)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("草稿附件文件无效")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	actual, err := f.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		f.Close()
		return nil, errors.New("草稿附件在读取期间发生变化")
	}
	return f, nil
}

// AddDraftBytes stages in-memory attachment bytes (such as a pasted or dropped image) into a draft.
func AddDraftBytes(root string, library *store.Store, siteID, draftID, name, dataBase64 string) (store.Draft, error) {
	drafts, err := library.ListDrafts(siteID)
	if err != nil {
		return store.Draft{}, err
	}
	var existing []string
	for _, draft := range drafts {
		if draft.ID == draftID {
			existing = draft.Attachments
			break
		}
	}
	stagedPath, _, cleanup, err := stageAttachmentBytes(name, dataBase64, existing)
	if err != nil {
		return store.Draft{}, err
	}
	defer cleanup()
	return AddDraft(root, library, siteID, draftID, stagedPath)
}
