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

	"github.com/egoist/mygo"
	"windows/internal/applog"
	"windows/internal/feeds"
	"windows/internal/follow"
	"windows/internal/ipfs"
	"windows/internal/pinnable"
	"windows/internal/services"
	"windows/internal/sitecleanup"
	"windows/internal/siteimage"
	"windows/internal/store"
)

func (l *Library) Sites() ([]store.Site, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.ListSites()
}

func (l *Library) CreateSite(name, about string) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.CreateSite(name, about)
}

func (l *Library) UpdateSite(siteID, name, about string) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSite(siteID, name, about)
}

func (l *Library) UpdateSiteSettings(siteID, name, about string, doNotIndex bool) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSiteSettings(siteID, name, about, doNotIndex)
}

func (l *Library) UpdateSiteDetails(siteID, name, about string, doNotIndex bool, slug, author string) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSiteDetails(siteID, name, about, doNotIndex, slug, author)
}

func (l *Library) UpdateSiteProfile(siteID, profileJSON string) (store.Site, error) {
	var profile store.SiteProfile
	if len(profileJSON) > 16<<10 || json.Unmarshal([]byte(profileJSON), &profile) != nil {
		return store.Site{}, errors.New("站点公开资料无效")
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSiteProfile(siteID, profile)
}

func (l *Library) UpdateSiteCustomCode(siteID, customCodeJSON string) (store.Site, error) {
	var cfg store.SiteCustomCode
	if len(customCodeJSON) > 256<<10 || json.Unmarshal([]byte(customCodeJSON), &cfg) != nil {
		return store.Site{}, errors.New("站点自定义代码设置无效")
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSiteCustomCode(siteID, cfg)
}

func (l *Library) UpdatePodcastSettings(siteID, language string, explicit bool, categoriesJSON string) (store.Site, error) {
	var categories []string
	if len(categoriesJSON) > 4096 || json.Unmarshal([]byte(categoriesJSON), &categories) != nil {
		return store.Site{}, errors.New("播客分类无效")
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdatePodcastSettings(siteID, language, explicit, categories)
}

func (l *Library) SelectPodcastCover(ctx context.Context, siteID string) (store.Site, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择播客封面（PNG、JPEG 或 GIF）"})
	if err != nil || len(paths) == 0 {
		return store.Site{}, err
	}
	cover, err := siteimage.ReadAvatar(paths[0])
	if err != nil {
		return store.Site{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetPodcastCover(siteID, cover)
}

func (l *Library) RemovePodcastCover(siteID string) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetPodcastCover(siteID, "")
}

// SelectSiteAvatar uses the native file picker and saves a normalized PNG.
func (l *Library) SelectSiteAvatar(ctx context.Context, siteID string) (store.Site, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择站点头像（PNG、JPEG 或 GIF）"})
	if err != nil || len(paths) == 0 {
		return store.Site{}, err
	}
	avatar, err := siteimage.ReadAvatar(paths[0])
	if err != nil {
		return store.Site{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetSiteAvatar(siteID, avatar)
}

func (l *Library) RemoveSiteAvatar(siteID string) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetSiteAvatar(siteID, "")
}

func (l *Library) PresetAvatars() []siteimage.PresetAvatar {
	return siteimage.PresetAvatars()
}

func (l *Library) ApplyPresetAvatar(siteID, key string) (store.Site, error) {
	avatar, err := siteimage.PresetAvatarPNG(key)
	if err != nil {
		return store.Site{}, err
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetSiteAvatar(siteID, avatar)
}

func (l *Library) SetSaveRoundAvatar(siteID string, round bool) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetSaveRoundAvatar(siteID, round)
}

func (l *Library) SetSiteArchived(siteID string, archived bool) (store.Site, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.SetSiteArchived(siteID, archived)
}

// Content Aggregation (Phase 4.2)
func (l *Library) UpdateSiteAggregation(siteID, sourcesText string) (store.Site, error) {
	lines := strings.Split(sourcesText, "\n")
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.UpdateSiteAggregation(siteID, lines)
}

func (l *Library) SyncSiteAggregation(siteID string) (int, error) {
	s, err := l.open()
	if err != nil {
		return 0, err
	}
	sites, err := s.ListSites()
	if err != nil {
		return 0, err
	}
	var target *store.Site
	for i := range sites {
		if sites[i].ID == siteID {
			target = &sites[i]
			break
		}
	}
	if target == nil {
		return 0, errors.New("站点不存在")
	}
	if len(target.Aggregation) == 0 {
		return 0, nil
	}
	existingArticles, _ := s.ListArticles(siteID)
	seenTitles := map[string]bool{}
	for _, a := range existingArticles {
		seenTitles[strings.ToLower(strings.TrimSpace(a.Title))] = true
	}
	added := 0
	for _, rawSource := range target.Aggregation {
		address, err := follow.Normalize(rawSource)
		if err != nil {
			continue
		}
		var feed feeds.Feed
		if strings.HasPrefix(strings.ToLower(address), "http://") || strings.HasPrefix(strings.ToLower(address), "https://") {
			feed, err = feeds.Fetch(address)
		} else {
			l.publicMu.Lock()
			if _, _, e := l.ensureNode(true); e == nil {
				res, e2 := follow.Fetch(l.node, address, false)
				feed, err = res.Feed, e2
			} else {
				err = e
			}
			l.publicMu.Unlock()
		}
		if err != nil {
			continue
		}
		for _, item := range feed.Items {
			title := strings.TrimSpace(item.Title)
			if title == "" {
				continue
			}
			key := strings.ToLower(title)
			if seenTitles[key] {
				continue
			}
			body := strings.TrimSpace(item.Content)
			if body == "" {
				body = strings.TrimSpace(item.Description)
			}
			if item.Link != "" {
				body += "\n\n---\n原文链接：" + item.Link
			}
			if _, err := s.SaveArticle(siteID, "", title, body); err == nil {
				seenTitles[key] = true
				added++
			}
		}
	}
	return added, nil
}

// Phase 6: Web3 Donation / Juicebox Settings
func (l *Library) UpdateSiteWeb3(siteID, web3JSON string) (store.Site, error) {
	var cfg store.SiteWeb3Settings
	if len(web3JSON) > 16<<10 || json.Unmarshal([]byte(web3JSON), &cfg) != nil {
		return store.Site{}, errors.New("Web3 捐赠配置无效")
	}
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	updated, err := s.UpdateSiteWeb3(siteID, cfg)
	if err == nil {
		applog.Global.Append("Kinpage", "已更新站点 Web3 捐赠设置："+updated.Name)
	}
	return updated, err
}

func (l *Library) RemovedSites() ([]store.Site, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.RemovedSites()
}

func (l *Library) RemoveSite(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.RemoveSite(siteID)
}

func (l *Library) RestoreSite(siteID string) (store.Site, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Site{}, err
	}
	return s.RestoreSite(siteID)
}

// PermanentlyDeleteSite starts or resumes deletion of a previously removed site.
func (l *Library) PermanentlyDeleteSite(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	return l.permanentlyDeleteSiteLocked(siteID)
}

func (l *Library) permanentlyDeleteSiteLocked(siteID string) error {
	if !sitecleanup.ValidSiteID(siteID) {
		return errors.New("站点标识无效")
	}
	s, err := l.open()
	if err != nil {
		return err
	}
	sites, err := s.RemovedSites()
	if err != nil {
		return err
	}
	var selected *store.Site
	for i := range sites {
		if sites[i].ID == siteID {
			selected = &sites[i]
			break
		}
	}
	if selected == nil {
		return errors.New("请先移除站点")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	// Check any existing key before committing irreversible deletion intent.
	hasRepo, err := sitecleanup.KuboRepositoryPresent(root)
	if err != nil {
		return err
	}
	if hasRepo {
		if l.node == nil {
			resources, e := mygo.App.Path(mygo.PathResources)
			if e != nil {
				return e
			}
			executable := filepath.Join(resources, "ipfs.exe")
			if _, e = os.Stat(executable); e != nil {
				return errors.New("缺少 Kubo 程序，无法核对密钥")
			}
			l.setNode(ipfs.New(executable, filepath.Join(root, "ipfs")))
		}
		if err = l.node.ValidateKeyAPI(); err != nil {
			return err
		}
		if !l.node.Alive() {
			if err = sitecleanup.CheckOfflineKuboConfig(root); err != nil {
				return err
			}
			if err = l.node.Start(true); err != nil {
				return err
			}
			defer func() {
				if e := l.node.Close(); e != nil {
					log.Printf("close temporary Kubo: %v", e)
				}
			}()
		}
		if err = l.node.ValidateKeyAPI(); err != nil {
			return err
		}
		key, err := l.node.LookupKey(siteID)
		if err != nil {
			return err
		}
		if key != "" {
			if selected.IPNS == "" {
				return errors.New("站点密钥身份未知，无法安全删除")
			}
			match, err := l.node.KeyMatches(siteID, selected.IPNS)
			if err != nil {
				return err
			}
			if !match {
				return errors.New("站点密钥与原 IPNS 不匹配，无法安全删除")
			}
		}
	}
	if _, err = s.BeginSiteDeletion(siteID); err != nil {
		return err
	}
	// After the intent is stored, each step is idempotent and a failure remains
	// visible as pending instead of presenting an unsafe restore operation.
	for _, kind := range []string{"attachments", "draft-attachments", "imports"} {
		if err = sitecleanup.RemoveOwnedDirectory(root, kind, siteID); err != nil {
			return fmt.Errorf("永久删除已开始，清理本机文件失败，可稍后重试：%w", err)
		}
	}
	if err = pinnable.Save(root, siteID, ""); err != nil {
		return fmt.Errorf("永久删除已开始，清理固定服务配置失败，可稍后重试：%w", err)
	}
	for _, kind := range []string{"filebase", "plausible", "ssh", "cloudflare", "x"} {
		if err = services.SaveConfig(root, siteID, kind, services.Config{}); err != nil {
			return fmt.Errorf("永久删除已开始，清理服务配置失败，可稍后重试：%w", err)
		}
	}
	if err = services.DeleteXPostsForSite(root, siteID); err != nil {
		return fmt.Errorf("永久删除已开始，清理 X 同步记录失败，可稍后重试：%w", err)
	}
	if err = services.DeleteTelegramConfig(root, siteID); err != nil {
		return fmt.Errorf("永久删除已开始，清理 Telegram 配置失败，可稍后重试：%w", err)
	}
	if hasRepo {
		if err = l.node.RemoveKeyIfMatches(siteID, selected.IPNS); err != nil {
			return fmt.Errorf("永久删除已开始，清理站点密钥失败，可稍后重试：%w", err)
		}
	}
	if err = s.FinalizeSiteDeletion(siteID); err != nil {
		return fmt.Errorf("永久删除已开始，保存清理状态失败，可稍后重试：%w", err)
	}
	return nil
}

func (l *Library) resumeSiteDeletions() {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return
	}
	sites, err := s.RemovedSites()
	if err != nil {
		return
	}
	for _, site := range sites {
		if site.DeletionPending {
			if err := l.permanentlyDeleteSiteLocked(site.ID); err != nil {
				log.Printf("site deletion remains pending for %s: %v", site.ID, err)
			}
		}
	}
}
