package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/attachments"
	"windows/internal/migration"
	"windows/internal/sitegen"
	"windows/internal/store"
)

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
