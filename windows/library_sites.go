package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"windows/internal/ipfs"
	"windows/internal/pinnable"
	"windows/internal/services"
	"windows/internal/sitecleanup"
	"windows/internal/store"
)

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
