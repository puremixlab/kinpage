package attachments

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"windows/internal/store"
)

const (
	maxArticleImportFiles = 256
	maxArticleImportBytes = 512 << 20 // 512 MiB
)

type articlePackageMeta struct {
	ID                     string            `json:"id,omitempty"`
	Title                  string            `json:"title"`
	Content                string            `json:"content"`
	Created                time.Time         `json:"created"`
	Modified               time.Time         `json:"modified,omitempty"`
	Attachments            []string          `json:"attachments,omitempty"`
	Slug                   string            `json:"slug,omitempty"`
	ArticleType            int               `json:"articleType,omitempty"`
	ExternalLink           string            `json:"externalLink,omitempty"`
	IsIncludedInNavigation bool              `json:"isIncludedInNavigation,omitempty"`
	NavigationWeight       int               `json:"navigationWeight,omitempty"`
	Tags                   map[string]string `json:"tags,omitempty"`
	AudioFilename          string            `json:"audioFilename,omitempty"`
	VideoFilename          string            `json:"videoFilename,omitempty"`
	HeroImage              string            `json:"heroImage,omitempty"`
	HeroImageManual        bool              `json:"heroImageManual,omitempty"`
	Pinned                 *time.Time        `json:"pinned,omitempty"`
	Starred                bool              `json:"starred,omitempty"`
	StarType               int               `json:"starType,omitempty"`
}

func findSite(library *store.Store, siteID string) (store.Site, bool, error) {
	sites, err := library.ListSites()
	if err != nil {
		return store.Site{}, false, err
	}
	for _, s := range sites {
		if s.ID == siteID {
			return s, true, nil
		}
	}
	return store.Site{}, false, nil
}

// MoveArticle moves an article and its attachment files from fromSiteID to toSiteID.
func MoveArticle(root string, library *store.Store, fromSiteID, articleID, toSiteID string) (store.Article, error) {
	if fromSiteID == toSiteID {
		return store.Article{}, errors.New("目标站点不能与当前站点相同")
	}
	fromSite, ok, err := findSite(library, fromSiteID)
	if err != nil {
		return store.Article{}, err
	}
	if !ok {
		return store.Article{}, errors.New("源站点不存在")
	}
	if _, ok, err := findSite(library, toSiteID); err != nil {
		return store.Article{}, err
	} else if !ok {
		return store.Article{}, errors.New("目标站点不存在")
	}
	articles, err := library.ListArticles(fromSiteID)
	if err != nil {
		return store.Article{}, err
	}
	var srcArt *store.Article
	for i := range articles {
		if articles[i].ID == articleID {
			srcArt = &articles[i]
			break
		}
	}
	if srcArt == nil {
		return store.Article{}, errors.New("文章不存在")
	}

	var dstDir string
	if len(srcArt.Attachments) > 0 {
		dstDir, err = ownedDir(root, toSiteID, articleID, true)
		if err != nil {
			return store.Article{}, err
		}
		for _, name := range srcArt.Attachments {
			if !validName(name) {
				continue
			}
			f, err := OpenRegistered(root, fromSite, *srcArt, name)
			if err != nil {
				_ = os.RemoveAll(dstDir)
				return store.Article{}, fmt.Errorf("读取文章附件 %s 失败：%w", name, err)
			}
			err = copyReaderToFile(f, filepath.Join(dstDir, name))
			_ = f.Close()
			if err != nil {
				_ = os.RemoveAll(dstDir)
				return store.Article{}, fmt.Errorf("复制文章附件 %s 失败：%w", name, err)
			}
		}
	}

	moved, err := library.MoveArticleRecord(fromSiteID, articleID, toSiteID)
	if err != nil {
		if dstDir != "" {
			_ = os.RemoveAll(dstDir)
		}
		return store.Article{}, err
	}
	if oldDir, err := ownedDir(root, fromSiteID, articleID, false); err == nil {
		_ = os.RemoveAll(oldDir)
	}
	return moved, nil
}

// ExportArticleZip exports an article and its registered attachments into a
// portable zip package (article.json + attachment files).
func ExportArticleZip(root string, library *store.Store, siteID, articleID, destPath string) error {
	if strings.TrimSpace(destPath) == "" {
		return errors.New("导出路径无效")
	}
	site, ok, err := findSite(library, siteID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("站点不存在")
	}
	articles, err := library.ListArticles(siteID)
	if err != nil {
		return err
	}
	var art *store.Article
	for i := range articles {
		if articles[i].ID == articleID {
			art = &articles[i]
			break
		}
	}
	if art == nil {
		return errors.New("文章不存在")
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0700); err != nil {
		return err
	}
	tmpPath := destPath + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(out)
	cleanup := func(cause error) error {
		_ = zw.Close()
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return cause
	}

	meta := articlePackageMeta{
		ID:                     art.ID,
		Title:                  art.Title,
		Content:                art.Content,
		Created:                art.Created,
		Modified:               art.Modified,
		Attachments:            art.Attachments,
		Slug:                   art.Slug,
		ArticleType:            art.ArticleType,
		ExternalLink:           art.ExternalLink,
		IsIncludedInNavigation: art.IsIncludedInNavigation,
		NavigationWeight:       art.NavigationWeight,
		Tags:                   art.Tags,
		AudioFilename:          art.AudioFilename,
		VideoFilename:          art.VideoFilename,
		HeroImage:              art.HeroImage,
		HeroImageManual:        art.HeroImageManual,
		Pinned:                 art.Pinned,
		Starred:                art.Starred,
		StarType:               art.StarType,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return cleanup(err)
	}
	w, err := zw.Create("article.json")
	if err != nil {
		return cleanup(err)
	}
	if _, err := w.Write(metaBytes); err != nil {
		return cleanup(err)
	}

	for _, name := range art.Attachments {
		if !validName(name) {
			continue
		}
		f, err := OpenRegistered(root, site, *art, name)
		if err != nil {
			return cleanup(fmt.Errorf("读取附件 %s 失败：%w", name, err))
		}
		aw, err := zw.Create(name)
		if err != nil {
			_ = f.Close()
			return cleanup(err)
		}
		_, copyErr := io.Copy(aw, f)
		_ = f.Close()
		if copyErr != nil {
			return cleanup(copyErr)
		}
	}

	if err := zw.Close(); err != nil {
		_ = out.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, destPath)
}

// ImportArticleZip imports a portable article zip package into siteID.
// It rejects zip-slip paths, supports both root-level article.json and a single
// wrapping directory (such as <UUID>.planetarticle/article.json), and rolls back
// cleanly on extraction failure.
func ImportArticleZip(root string, library *store.Store, siteID, srcPath string) (store.Article, error) {
	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return store.Article{}, fmt.Errorf("无法打开文章数据包：%w", err)
	}
	defer zr.Close()

	if len(zr.File) == 0 || len(zr.File) > maxArticleImportFiles {
		return store.Article{}, errors.New("文章数据包文件数量无效")
	}

	// Validate every archive entry against path traversal first.
	var metaFile *zip.File
	metaPrefix := ""
	filesByRel := map[string]*zip.File{}
	for _, f := range zr.File {
		if strings.Contains(f.Name, `\`) {
			return store.Article{}, errors.New("文章数据包包含不安全的路径")
		}
		clean := path.Clean(f.Name)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") || strings.Contains(clean, ":") {
			return store.Article{}, errors.New("文章数据包包含不安全的路径")
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if clean == "article.json" {
			metaFile = f
			metaPrefix = ""
		} else if strings.HasSuffix(clean, "/article.json") && strings.Count(clean, "/") == 1 && metaFile == nil {
			metaFile = f
			metaPrefix = strings.TrimSuffix(clean, "article.json")
		}
		filesByRel[clean] = f
	}
	if metaFile == nil {
		return store.Article{}, errors.New("文章数据包缺少 article.json")
	}

	rc, err := metaFile.Open()
	if err != nil {
		return store.Article{}, err
	}
	metaBytes, err := io.ReadAll(io.LimitReader(rc, 16<<20))
	_ = rc.Close()
	if err != nil {
		return store.Article{}, err
	}
	var meta articlePackageMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return store.Article{}, fmt.Errorf("解析 article.json 失败：%w", err)
	}

	var validAttachments []string
	var attachmentEntries []*zip.File
	seen := map[string]bool{}
	for _, name := range meta.Attachments {
		if !validName(name) || seen[strings.ToLower(name)] {
			continue
		}
		zf, ok := filesByRel[metaPrefix+name]
		if !ok {
			continue
		}
		seen[strings.ToLower(name)] = true
		validAttachments = append(validAttachments, name)
		attachmentEntries = append(attachmentEntries, zf)
	}

	hero := meta.HeroImage
	if hero != "" && !seen[strings.ToLower(hero)] {
		hero = ""
	}
	audio := meta.AudioFilename
	if audio != "" && !seen[strings.ToLower(audio)] {
		audio = ""
	}
	video := meta.VideoFilename
	if video != "" && !seen[strings.ToLower(video)] {
		video = ""
	}

	created, err := library.ImportArticleRecord(siteID, store.Article{
		Title:                  meta.Title,
		Content:                meta.Content,
		Created:                meta.Created,
		Modified:               meta.Modified,
		Attachments:            validAttachments,
		Slug:                   meta.Slug,
		ArticleType:            meta.ArticleType,
		ExternalLink:           meta.ExternalLink,
		IsIncludedInNavigation: meta.IsIncludedInNavigation,
		NavigationWeight:       meta.NavigationWeight,
		Tags:                   meta.Tags,
		AudioFilename:          audio,
		VideoFilename:          video,
		HeroImage:              hero,
		HeroImageManual:        meta.HeroImageManual && hero != "",
		Pinned:                 meta.Pinned,
		Starred:                meta.Starred,
		StarType:               meta.StarType,
	})
	if err != nil {
		return store.Article{}, err
	}

	if len(attachmentEntries) > 0 {
		dstDir, err := ownedDir(root, siteID, created.ID, true)
		if err != nil {
			_ = library.DeleteArticle(siteID, created.ID)
			return store.Article{}, err
		}
		var totalBytes int64
		for i, zf := range attachmentEntries {
			name := validAttachments[i]
			arc, err := zf.Open()
			if err != nil {
				_ = os.RemoveAll(dstDir)
				_ = library.DeleteArticle(siteID, created.ID)
				return store.Article{}, err
			}
			n, copyErr := copyLimitedToFile(arc, filepath.Join(dstDir, name), maxArticleImportBytes-totalBytes)
			_ = arc.Close()
			if copyErr != nil {
				_ = os.RemoveAll(dstDir)
				_ = library.DeleteArticle(siteID, created.ID)
				return store.Article{}, copyErr
			}
			totalBytes += n
		}
	}

	return created, nil
}

func copyReaderToFile(r io.Reader, dst string) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return err
	}
	return f.Close()
}

func copyLimitedToFile(r io.Reader, dst string, limit int64) (int64, error) {
	if limit <= 0 {
		return 0, errors.New("文章数据包总大小超出限制")
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, io.LimitReader(r, limit+1))
	if err != nil {
		_ = f.Close()
		_ = os.Remove(dst)
		return 0, err
	}
	if n > limit {
		_ = f.Close()
		_ = os.Remove(dst)
		return 0, errors.New("文章数据包总大小超出限制")
	}
	return n, f.Close()
}
