package main

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/nativeui"
	"windows/internal/publisher"
	"windows/internal/services"
	"windows/internal/store"
)

type DeploymentInfo struct {
	Configured   bool   `json:"configured"`
	Name         string `json:"name"`
	Server       string `json:"server"`
	AutoDeploy   bool   `json:"autoDeploy"`
	DeleteRemote bool   `json:"deleteRemote"`
	Pending      bool   `json:"pending"`
	LastURL      string `json:"lastUrl"`
}

func (l *Library) deploymentSite(siteID string) (store.Site, []store.Article, string, error) {
	s, err := l.open()
	if err != nil {
		return store.Site{}, nil, "", err
	}
	site, articles, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return store.Site{}, nil, "", err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	return site, articles, root, err
}

func (l *Library) DeploymentInfo(siteID, kind string) (DeploymentInfo, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if kind != "ssh" && kind != "cloudflare" {
		return DeploymentInfo{}, errors.New("发布目标无效")
	}
	_, _, root, err := l.deploymentSite(siteID)
	if err != nil {
		return DeploymentInfo{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, kind)
	if err != nil {
		return DeploymentInfo{}, err
	}
	return DeploymentInfo{Configured: cfg.Token != "", Name: cfg.Name, Server: cfg.Server, AutoDeploy: cfg.AutoDeploy, DeleteRemote: cfg.DeleteRemote, Pending: cfg.PendingCID != "", LastURL: cfg.LastURL}, nil
}

func (l *Library) DeploymentSummary(siteID, kind string) (nativeui.DeploymentSummary, error) {
	info, err := l.DeploymentInfo(siteID, kind)
	if err != nil {
		return nativeui.DeploymentSummary{}, err
	}
	return nativeui.DeploymentSummary{
		Configured:   info.Configured,
		Name:         info.Name,
		Server:       info.Server,
		AutoDeploy:   info.AutoDeploy,
		DeleteRemote: info.DeleteRemote,
		Pending:      info.Pending,
		LastURL:      info.LastURL,
	}, nil
}

// ConfigureDeployment stores a target locally; credentials never enter backups.
func (l *Library) ConfigureDeployment(siteID, kind, name, server, token string, auto, deleteRemote bool) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if kind != "ssh" && kind != "cloudflare" {
		return errors.New("发布目标无效")
	}
	_, _, root, err := l.deploymentSite(siteID)
	if err != nil {
		return err
	}
	old, err := services.LoadConfig(root, siteID, kind)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	server = strings.TrimSpace(server)
	var cfg services.Config
	if kind == "cloudflare" {
		if token == "" {
			if old.Name != name || old.Server != server || old.Token == "" {
				return errors.New("更换 Cloudflare 账号或项目时请重新输入令牌")
			}
			token = old.Token
		}
		cfg = services.Config{Name: name, Server: server, Token: token, AutoDeploy: auto}
	} else {
		cfg = services.Config{Name: name, Server: server, Token: "ssh-agent", AutoDeploy: auto, DeleteRemote: deleteRemote}
	}
	if old.Name == cfg.Name && old.Server == cfg.Server {
		cfg.PendingCID = old.PendingCID
		cfg.LastURL = old.LastURL
	}
	return services.SaveConfig(root, siteID, kind, cfg)
}

func (l *Library) DisableDeployment(siteID, kind string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if kind != "ssh" && kind != "cloudflare" {
		return errors.New("发布目标无效")
	}
	_, _, root, err := l.deploymentSite(siteID)
	if err != nil {
		return err
	}
	return services.SaveConfig(root, siteID, kind, services.Config{})
}

func (l *Library) SelectSSHKey(ctx context.Context) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择 SSH 私钥"})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

func (l *Library) stageRemote(root string, site store.Site, articles []store.Article) (string, error) {
	templateDir, err := l.templatePath(site)
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "planet-remote-publish-")
	if err != nil {
		return "", err
	}
	if err = publisher.StagePublicSite(dir, site, articles, templateDir, root); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// DeploySite is an explicit manual deployment of the last saved snapshot.
func (l *Library) DeploySite(siteID, kind string) (string, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if kind != "ssh" && kind != "cloudflare" {
		return "", errors.New("发布目标无效")
	}
	site, articles, root, err := l.deploymentSite(siteID)
	if err != nil {
		return "", err
	}
	if site.Domain == "" && site.IPNS == "" {
		for _, article := range articles {
			if article.ArticleType != 1 && article.AudioFilename != "" {
				return "", errors.New("播客音频需要绝对地址；请先配置站点域名或发布 IPNS")
			}
		}
	}
	dir, err := l.stageRemote(root, site, articles)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	return services.DeployConfigured(root, siteID, kind, dir, true, services.NetworkDeploymentRunner{})
}

func (l *Library) autoDeploy(root, siteID, dir string) []publisher.DeploymentResult {
	var results []publisher.DeploymentResult
	for _, kind := range []string{"ssh", "cloudflare"} {
		target := "Cloudflare Pages"
		if kind == "ssh" {
			target = "SSH / rsync"
		}
		cfg, err := services.LoadConfig(root, siteID, kind)
		if err != nil {
			results = append(results, publisher.DeploymentResult{Target: target, Error: "读取发布配置失败：" + err.Error()})
			continue
		}
		if cfg.Token == "" || !cfg.AutoDeploy {
			continue
		}
		address, err := services.DeployConfigured(root, siteID, kind, dir, false, services.NetworkDeploymentRunner{})
		result := publisher.DeploymentResult{Target: target, URL: address}
		if err != nil {
			result.Error = err.Error()
		}
		results = append(results, result)
	}
	return results
}
