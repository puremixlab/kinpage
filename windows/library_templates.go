package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/backup"
	"windows/internal/migration"
	"windows/internal/sitegen"
	"windows/internal/store"
	"windows/internal/templateinstall"
)

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
