// Package attachments manages files belonging to Windows-authored articles.
package attachments

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"windows/internal/backup"
	"windows/internal/mediainfo"
	"windows/internal/store"
)

func validName(name string) bool {
	if !backup.SafeName(name) || strings.HasSuffix(strings.ToLower(name), ".key") {
		return false
	}
	switch strings.ToLower(name) {
	case "index.html", "article.json", "article.md", "simple.html", "nft.json", "nft.json.cid.txt", "_cover.png", "_grid.jpg", "_videothumbnail.png":
		return false
	}
	return true
}

// CopyImported copies only the attachment names recorded in the original backup.
func CopyImported(sourceDir, destinationDir string, names []string) error {
	seen := map[string]bool{}
	for _, name := range names {
		folded := strings.ToLower(name)
		if !validName(name) || seen[folded] {
			return errors.New("原版附件名称无效或冲突")
		}
		seen[folded] = true
		if err := copyRegular(filepath.Join(sourceDir, name), filepath.Join(destinationDir, name)); err != nil {
			return fmt.Errorf("复制原版附件 %s 失败：%w", name, err)
		}
	}
	return nil
}

func ownedDir(root, siteID, articleID string, create bool) (string, error) {
	return managedDir(root, "attachments", siteID, articleID, create)
}

func managedDir(root, kind, siteID, articleID string, create bool) (string, error) {
	if !backup.SafeName(siteID) || !backup.SafeName(articleID) {
		return "", errors.New("附件路径无效")
	}
	dir := root
	for _, part := range []string{kind, siteID, articleID} {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) && create {
			if err = os.Mkdir(dir, 0700); err != nil {
				return "", err
			}
			info, err = os.Lstat(dir)
		}
		if err != nil {
			return "", err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("附件目录无效")
		}
	}
	return dir, nil
}

// CleanupArticleFiles removes only locally owned files queued by a committed
// draft publication; files still referenced by any article remain untouched.
func CleanupArticleFiles(root string, library *store.Store) {
	protected := map[string]bool{}
	for _, entry := range library.ArticleFileReferences() {
		protected[strings.ToLower(filepath.Join(entry.SiteID, entry.ArticleID, entry.Name))] = true
	}
	for _, entry := range library.PendingArticleCleanup() {
		if !validName(entry.Name) {
			continue
		}
		if protected[strings.ToLower(filepath.Join(entry.SiteID, entry.ArticleID, entry.Name))] {
			_ = library.CompleteArticleCleanup(entry)
			continue
		}
		dir, err := ownedDir(root, entry.SiteID, entry.ArticleID, false)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteArticleCleanup(entry)
			continue
		}
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entry.Name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteArticleCleanup(entry)
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err = os.Remove(path); err == nil || errors.Is(err, os.ErrNotExist) {
			_ = library.CompleteArticleCleanup(entry)
		}
	}
}

func copyRegular(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("附件必须是普通文件")
	}
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	actual, err := input.Stat()
	if err != nil || !actual.Mode().IsRegular() || !os.SameFile(info, actual) {
		return errors.New("附件在复制期间发生变化")
	}
	output, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	if closeErr := output.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

// attachmentKind classifies a file name the way templates treat attachments.
func attachmentKind(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".opus", ".flac":
		return "audio"
	case ".mp4", ".mov", ".m4v", ".webm", ".avi", ".mpeg", ".mpg":
		return "video"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "image"
	}
	return "file"
}

// Add copies a file and records it on a saved, Windows-authored article.
// Audio and video follow the macOS writer's single-attachment rule: adding one
// replaces the previous file of the same kind, and a re-selected name replaces
// the previous copy.
func Add(root string, library *store.Store, siteID, articleID, source string) (store.Article, error) {
	name := filepath.Base(source)
	if !validName(name) {
		return store.Article{}, errors.New("附件文件名无效或与生成文件冲突")
	}
	CleanupArticleFiles(root, library)
	posts, err := library.ListArticles(siteID)
	if err != nil {
		return store.Article{}, err
	}
	found := false
	kind := attachmentKind(name)
	for _, post := range posts {
		if post.ID != articleID {
			continue
		}
		found = true
		replaced := ""
		for _, old := range post.Attachments {
			if strings.EqualFold(old, name) {
				replaced = old
				break
			}
		}
		if replaced == "" && (kind == "audio" || kind == "video") {
			if kind == "audio" && post.AudioFilename != "" {
				replaced = post.AudioFilename
			}
			if kind == "video" && post.VideoFilename != "" {
				replaced = post.VideoFilename
			}
		}
		if replaced != "" {
			if _, err := Remove(root, library, siteID, articleID, replaced); err != nil {
				return store.Article{}, err
			}
			CleanupArticleFiles(root, library)
		}
		break
	}
	if !found {
		return store.Article{}, errors.New("文章不存在")
	}
	for _, entry := range library.PendingArticleCleanup() {
		if entry.SiteID == siteID && entry.ArticleID == articleID && strings.EqualFold(entry.Name, name) {
			return store.Article{}, errors.New("原附件等待清理，请稍后重试")
		}
	}
	dir, err := ownedDir(root, siteID, articleID, true)
	if err != nil {
		return store.Article{}, err
	}
	target := filepath.Join(dir, name)
	if err = copyRegular(source, target); err != nil {
		return store.Article{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		_ = os.Remove(target)
		return store.Article{}, err
	}
	size := info.Size()
	var duration *int
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".opus", ".flac":
		duration = mediainfo.Duration(target)
	}
	article, err := library.AddAttachmentWithMetadata(siteID, articleID, name, duration, &size)
	if err != nil {
		_ = os.Remove(target)
		return store.Article{}, err
	}
	return article, nil
}

// Remove removes only an attachment registered to this article.
func Remove(root string, library *store.Store, siteID, articleID, name string) (store.Article, error) {
	if !validName(name) {
		return store.Article{}, errors.New("附件名称无效")
	}
	posts, err := library.ListArticles(siteID)
	if err != nil {
		return store.Article{}, err
	}
	found := false
	for _, post := range posts {
		if post.ID == articleID {
			for _, old := range post.Attachments {
				if old == name {
					found = true
				}
			}
			break
		}
	}
	if !found {
		return store.Article{}, errors.New("附件不存在")
	}
	dir, err := ownedDir(root, siteID, articleID, false)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return store.Article{}, err
		}
		// Original imported files remain in the immutable archive.
		sites, e := library.ListSites()
		if e != nil {
			return store.Article{}, e
		}
		for _, site := range sites {
			if site.ID == siteID && site.Imported {
				info, e := os.Lstat(filepath.Join(root, "imports", siteID, articleID, name))
				if e != nil || !info.Mode().IsRegular() {
					return store.Article{}, errors.New("附件文件无效")
				}
				return library.RemoveAttachment(siteID, articleID, name)
			}
		}
		return store.Article{}, err
	}
	path := filepath.Join(dir, name)
	if _, e := os.Lstat(path); errors.Is(e, os.ErrNotExist) {
		info, e := os.Lstat(filepath.Join(root, "imports", siteID, articleID, name))
		if e != nil || !info.Mode().IsRegular() {
			return store.Article{}, errors.New("附件文件无效")
		}
		return library.RemoveAttachment(siteID, articleID, name)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return store.Article{}, errors.New("附件文件无效")
	}
	temp, err := os.CreateTemp(dir, ".remove-*")
	if err != nil {
		return store.Article{}, err
	}
	tempPath := temp.Name()
	if err = temp.Close(); err != nil {
		return store.Article{}, err
	}
	if err = os.Remove(tempPath); err != nil {
		return store.Article{}, err
	}
	if err = os.Rename(path, tempPath); err != nil {
		return store.Article{}, err
	}
	article, err := library.RemoveAttachment(siteID, articleID, name)
	if err != nil {
		if rollbackErr := os.Rename(tempPath, path); rollbackErr != nil {
			return store.Article{}, fmt.Errorf("%v；附件恢复失败：%w", err, rollbackErr)
		}
		return store.Article{}, err
	}
	_ = os.Remove(tempPath)
	return article, nil
}

// CopyToSite adds registered files to an already rendered public site.
func CopyToSite(root, destination string, articles []store.Article) error {
	for _, article := range articles {
		if len(article.Attachments) == 0 {
			continue
		}
		dir, err := ownedDir(root, article.SiteID, article.ID, false)
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for _, name := range article.Attachments {
			lower := strings.ToLower(name)
			if !validName(name) || seen[lower] {
				return errors.New("附件名称无效或冲突")
			}
			seen[lower] = true
			target := filepath.Join(destination, article.ID, name)
			if err = copyRegular(filepath.Join(dir, name), target); err != nil {
				return fmt.Errorf("复制附件 %s 失败：%w", name, err)
			}
		}
	}
	return nil
}

// CopyImportedCurrent combines owned edits with unchanged archived attachments.
func CopyImportedCurrent(root, archive, destination string, article store.Article) error {
	if !backup.SafeName(article.SiteID) || !backup.SafeName(article.ID) {
		return errors.New("附件路径无效")
	}
	seen := map[string]bool{}
	for _, name := range article.Attachments {
		key := strings.ToLower(name)
		if !validName(name) || seen[key] {
			return errors.New("附件名称无效或冲突")
		}
		seen[key] = true
		source := ""
		dir, err := ownedDir(root, article.SiteID, article.ID, false)
		if err == nil {
			candidate := filepath.Join(dir, name)
			if _, e := os.Lstat(candidate); e == nil {
				source = candidate
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if source == "" {
			dir = filepath.Join(archive, article.ID)
			info, e := os.Lstat(dir)
			if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return errors.New("原版文章目录无效")
			}
			source = filepath.Join(dir, name)
		}
		if err := copyRegular(source, filepath.Join(destination, article.ID, name)); err != nil {
			return fmt.Errorf("复制附件 %s 失败：%w", name, err)
		}
	}
	return nil
}

// OpenRegistered opens a registered attachment without following links.
func OpenRegistered(root string, site store.Site, article store.Article, name string) (*os.File, error) {
	if site.ID != article.SiteID || !backup.SafeName(site.ID) || !backup.SafeName(article.ID) || !validName(name) {
		return nil, errors.New("附件路径无效")
	}
	registered := false
	for _, item := range article.Attachments {
		if item == name {
			registered = true
			break
		}
	}
	if !registered {
		return nil, errors.New("附件未登记")
	}
	var path string
	dir, err := ownedDir(root, site.ID, article.ID, false)
	if err == nil {
		candidate := filepath.Join(dir, name)
		if _, err = os.Lstat(candidate); err == nil {
			path = candidate
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if path == "" {
		if !site.Imported {
			return nil, os.ErrNotExist
		}
		dir = root
		for _, part := range []string{"imports", site.ID, article.ID} {
			dir = filepath.Join(dir, part)
			info, err := os.Lstat(dir)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return nil, errors.New("原附件目录无效")
			}
		}
		path = filepath.Join(dir, name)
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("附件文件无效")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		f.Close()
		return nil, errors.New("附件在读取期间发生变化")
	}
	return f, nil
}

func stageAttachmentBytes(name, dataBase64 string, existing []string) (string, string, func(), error) {
	const maxPasteBytes = 32 << 20
	if len(dataBase64) == 0 || len(dataBase64) > base64.StdEncoding.EncodedLen(maxPasteBytes) {
		return "", "", nil, errors.New("附件数据为空或超过 32 MiB 上限")
	}
	raw, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil || len(raw) == 0 || len(raw) > maxPasteBytes {
		return "", "", nil, errors.New("附件编码无效")
	}
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == "/" || name == `\` {
		name = fmt.Sprintf("paste-%d.png", time.Now().UTC().UnixNano())
	}
	if filepath.Ext(name) == "" {
		switch http.DetectContentType(raw) {
		case "image/png":
			name += ".png"
		case "image/jpeg":
			name += ".jpg"
		case "image/gif":
			name += ".gif"
		case "image/webp":
			name += ".webp"
		default:
			return "", "", nil, errors.New("附件缺少文件扩展名")
		}
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	candidate := name
	for n := 2; n < 1000; n++ {
		conflict := false
		for _, old := range existing {
			if strings.EqualFold(old, candidate) {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		candidate = fmt.Sprintf("%s-%d%s", stem, n, ext)
	}
	if !validName(candidate) {
		return "", "", nil, errors.New("附件文件名无效或与生成文件冲突")
	}
	dir, err := os.MkdirTemp("", "planet-paste-*")
	if err != nil {
		return "", "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	target := filepath.Join(dir, candidate)
	if err := os.WriteFile(target, raw, 0600); err != nil {
		cleanup()
		return "", "", nil, err
	}
	return target, candidate, cleanup, nil
}

// AddBytes stages in-memory attachment bytes (such as a pasted or dropped image) into a saved article.
func AddBytes(root string, library *store.Store, siteID, articleID, name, dataBase64 string) (store.Article, error) {
	posts, err := library.ListArticles(siteID)
	if err != nil {
		return store.Article{}, err
	}
	var existing []string
	for _, post := range posts {
		if post.ID == articleID {
			existing = post.Attachments
			break
		}
	}
	stagedPath, _, cleanup, err := stageAttachmentBytes(name, dataBase64, existing)
	if err != nil {
		return store.Article{}, err
	}
	defer cleanup()
	return Add(root, library, siteID, articleID, stagedPath)
}
