package sitegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"windows/internal/store"
)

// RenderIndexOnly regenerates only index.html for fast homepage/SPA template
// iteration without rewriting every individual article directory.
func RenderIndexOnly(dir string, site store.Site, articles []store.Article, templateDir string) error {
	if !site.Imported {
		var err error
		site, err = normalizedSiteProfile(site)
		if err != nil {
			return err
		}
	}
	var err error
	site, err = prepareTags(site, articles)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	items := append([]store.Article(nil), articles...)
	sortSiteArticles(items)

	if strings.TrimSpace(templateDir) != "" {
		// Delegate to full template index render if a template directory is configured.
		// RenderWithTemplateSettings already writes index.html and assets; for quick
		// rebuilds when no article directories need rewriting we fall back cleanly
		// if index.html exists or render via template.
		return RenderWithTemplate(dir, site, items, templateDir)
	}

	var blogItems []store.Article
	for _, a := range items {
		if a.ArticleType == 0 {
			blogItems = append(blogItems, a)
		}
	}
	hasPodcast := false
	for _, a := range items {
		if a.AudioFilename != "" {
			hasPodcast = true
			break
		}
	}
	var buf bytes.Buffer
	err = indexTemplate.Execute(&buf, struct {
		Title      string
		Site       store.Site
		IconPrefix string
		Navigation []navigationItem
		Social     []navigationItem
		HasPodcast bool
		Articles   []store.Article
	}{
		Title:      site.Name,
		Site:       site,
		IconPrefix: "",
		Navigation: navigationItems(items, ""),
		Social:     socialLinks(site),
		HasPodcast: hasPodcast,
		Articles:   blogItems,
	})
	if err != nil {
		return fmt.Errorf("渲染首页失败：%w", err)
	}
	return os.WriteFile(filepath.Join(dir, "index.html"), buf.Bytes(), 0600)
}
