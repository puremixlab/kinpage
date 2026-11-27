package main

import (
	"errors"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"windows/internal/backup"
	"windows/internal/pinnable"
	"windows/internal/services"
	"windows/internal/store"
)

// ConfigureService stores credentials encrypted for the current Windows user.
func (l *Library) ConfigureService(siteID, kind, name, token string) error {
	return l.configureService(siteID, kind, name, token, "", "")
}

// ConfigurePlausible sets the domain, optional site-specific script ID and key.
func (l *Library) ConfigurePlausible(siteID, domain, token, scriptID string) error {
	return l.ConfigurePlausibleAt(siteID, domain, token, scriptID, "https://plausible.io")
}

// ConfigurePlausibleAt accepts an HTTPS service selected explicitly by the user.
func (l *Library) ConfigurePlausibleAt(siteID, domain, token, scriptID, server string) error {
	domain, token, scriptID = strings.TrimSpace(domain), strings.TrimSpace(token), strings.TrimSpace(scriptID)
	if strings.TrimSpace(server) == "" {
		return errors.New("请明确填写 Plausible HTTPS 服务地址")
	}
	origin, err := services.PlausibleServerOrigin(server)
	if err != nil {
		return err
	}
	if origin == "https://plausible.io" {
		origin = ""
	}
	if token != "" {
		return l.configureService(siteID, "plausible", domain, token, scriptID, origin)
	}
	if !store.ValidPlausibleDomain(domain) || !store.ValidPlausibleScriptID(scriptID) {
		return errors.New("Plausible 域名或脚本 ID 无效")
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	old, err := services.LoadConfig(root, siteID, "plausible")
	if err != nil {
		return err
	}
	if old.Token != "" && old.Server != origin {
		return errors.New("更换 Plausible 服务器时请重新输入该服务器的 API 令牌，或先停用原配置")
	}
	changed := old.Token != "" && old.Name != domain
	if changed {
		next := old
		next.Name = domain
		if err = services.SaveConfig(root, siteID, "plausible", next); err != nil {
			return err
		}
	}
	if _, err = s.SetPlausibleTrackingServer(siteID, true, domain, scriptID, origin); err != nil {
		if changed {
			if rollback := services.SaveConfig(root, siteID, "plausible", old); rollback != nil {
				return errors.New("站点设置保存失败，服务配置恢复也失败，请重新检查")
			}
		}
		return err
	}
	return nil
}

func (l *Library) configureService(siteID, kind, name, token, scriptID, server string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	name, token = strings.TrimSpace(name), strings.TrimSpace(token)
	if kind == "plausible" && token != "" && !store.ValidPlausibleDomain(name) {
		return errors.New("Plausible 站点域名无效")
	}
	if kind == "plausible" && !store.ValidPlausibleScriptID(scriptID) {
		return errors.New("Plausible 脚本 ID 无效，请输入 pa- 开头的站点专属 ID")
	}
	// Disable tracking even when the old encrypted credential is unreadable.
	if kind == "plausible" && token == "" {
		var savedDomain, savedScript, savedServer string
		if site.PlausibleOverride != nil {
			savedDomain = site.PlausibleOverride.Domain
			savedScript = site.PlausibleOverride.ScriptID
			savedServer = site.PlausibleOverride.Server
			if site.PlausibleOverride.Enabled && savedServer == "" {
				savedServer = "https://plausible.io"
			}
		} else if len(raw) > 0 {
			_, savedDomain, savedServer, _ = backup.PlausibleStoredSettings(raw)
		}
		if _, err = s.SetPlausibleTrackingServer(siteID, false, savedDomain, savedScript, savedServer); err != nil {
			return err
		}
		if err = services.SaveConfig(root, siteID, kind, services.Config{}); err != nil {
			return errors.New("站点跟踪已停用，但本机服务凭据未能清理，请重试")
		}
		return nil
	}
	old, err := services.LoadConfig(root, siteID, kind)
	if err != nil {
		return err
	}
	cfg := services.Config{Name: name, Token: token, Server: server}
	if old.Name == name && old.Token == token && old.Server == server {
		cfg = old
	}
	if err = services.SaveConfig(root, siteID, kind, cfg); err != nil {
		return err
	}
	if kind == "plausible" {
		if _, err = s.SetPlausibleTrackingServer(siteID, token != "", name, scriptID, server); err != nil {
			if rollbackErr := services.SaveConfig(root, siteID, kind, old); rollbackErr != nil {
				return errors.New("站点统计设置保存失败，服务配置恢复也失败，请重新检查")
			}
			return err
		}
	}
	return nil
}

type serviceInfo struct {
	Name            string `json:"name"`
	Configured      bool   `json:"configured"`
	LastCID         string `json:"lastCid,omitempty"`
	RequestID       string `json:"requestId,omitempty"`
	PendingCID      string `json:"pendingCid,omitempty"`
	ScriptID        string `json:"scriptId,omitempty"`
	TrackingEnabled bool   `json:"trackingEnabled,omitempty"`
	TrackingWarning string `json:"trackingWarning,omitempty"`
	Server          string `json:"server,omitempty"`
}

func (l *Library) ServiceInfo(siteID, kind string) (serviceInfo, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return serviceInfo{}, err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return serviceInfo{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return serviceInfo{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, kind)
	info := serviceInfo{Name: cfg.Name, Configured: cfg.Token != "", LastCID: cfg.LastCID, RequestID: cfg.RequestID, PendingCID: cfg.PendingCID}
	if kind == "plausible" && site.PlausibleOverride != nil {
		info.ScriptID = site.PlausibleOverride.ScriptID
		info.TrackingEnabled = site.PlausibleOverride.Enabled
		info.Server = site.PlausibleOverride.Server
		if info.Name == "" {
			info.Name = site.PlausibleOverride.Domain
		}
		if !site.PlausibleOverride.Enabled && info.Server == "" && len(raw) > 0 {
			_, domain, server, e := backup.PlausibleStoredSettings(raw)
			if e != nil {
				info.TrackingWarning = "原版统计配置不受支持；请手动填写服务器。"
			} else {
				info.Server = server
				if info.Name == "" {
					info.Name = domain
				}
			}
		}
	} else if kind == "plausible" && len(raw) > 0 {
		enabled, domain, server, e := backup.PlausibleStoredSettings(raw)
		if e != nil {
			info.TrackingWarning = "原版统计配置不受支持；可停用或重新配置。"
		} else {
			info.TrackingEnabled = enabled
			info.Server = server
			if info.Name == "" {
				info.Name = domain
			}
		}
	}
	if kind == "plausible" && info.Server == "" && info.TrackingWarning == "" && (site.PlausibleOverride != nil || len(raw) == 0) {
		info.Server = "https://plausible.io"
	}
	return info, err
}

func (l *Library) PlausibleStats(siteID string) (services.Stats, error) {
	return l.PlausibleStatsRange(siteID, "7d")
}

func (l *Library) PlausibleStatsRange(siteID, period string) (services.Stats, error) {
	if period != "day" && period != "7d" && period != "30d" {
		return services.Stats{}, errors.New("统计时间范围无效")
	}
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return services.Stats{}, err
	}
	site, _, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return services.Stats{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.Stats{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, "plausible")
	if err != nil {
		return services.Stats{}, err
	}
	if cfg.Token == "" {
		return services.Stats{}, errors.New("请先配置 Plausible")
	}
	trackingEnabled := false
	trackingDomain, trackingServer := "", ""
	if site.PlausibleOverride != nil {
		trackingEnabled = site.PlausibleOverride.Enabled
		trackingDomain = site.PlausibleOverride.Domain
		trackingServer = site.PlausibleOverride.Server
	} else if len(raw) > 0 {
		enabled, domain, server, e := backup.PlausiblePublicSettings(raw)
		if e != nil {
			return services.Stats{}, e
		}
		trackingEnabled = enabled
		trackingDomain = domain
		trackingServer = server
	}
	trackingOrigin, e := services.PlausibleServerOrigin(trackingServer)
	if e != nil {
		return services.Stats{}, e
	}
	credentialOrigin, e := services.PlausibleServerOrigin(cfg.Server)
	if e != nil {
		return services.Stats{}, e
	}
	if !trackingEnabled || trackingDomain != cfg.Name || trackingOrigin != credentialOrigin {
		return services.Stats{}, errors.New("统计凭据与站点跟踪设置不一致，请重新保存 Plausible 配置")
	}
	return services.NewClient().PlausibleStatsAtRange(cfg.Server, cfg.Token, cfg.Name, period)
}

func (l *Library) RequestFilebasePin(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	if site.LastPublishedCID == "" {
		return errors.New("请先发布站点")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	cfg, err := services.LoadConfig(root, siteID, "filebase")
	if err != nil {
		return err
	}
	if cfg.Token == "" {
		return errors.New("请先配置 Filebase")
	}
	return services.RetryPublished(root, siteID, site.LastPublishedCID, services.NewClient())
}

func (l *Library) FilebaseStatus(siteID string) (services.PinStatus, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return services.PinStatus{}, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return services.PinStatus{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return services.PinStatus{}, err
	}
	cfg, err := services.LoadConfig(root, siteID, "filebase")
	if err != nil {
		return services.PinStatus{}, err
	}
	if cfg.Token == "" || cfg.RequestID == "" {
		return services.PinStatus{}, errors.New("尚无已提交的固定请求")
	}
	status, err := services.NewClient().FilebaseStatus(cfg.Token, cfg.RequestID)
	if err != nil {
		return status, err
	}
	if status.Pin.CID != cfg.LastCID {
		return services.PinStatus{}, errors.New("服务报告的 CID 与本机记录不一致")
	}
	return status, nil
}

// ConfigurePinnable saves an explicitly supplied endpoint using Windows DPAPI.
// An empty endpoint disables the integration for this site.
func (l *Library) ConfigurePinnable(siteID, endpoint string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	return pinnable.Save(root, siteID, strings.TrimSpace(endpoint))
}

func (l *Library) PinnableConfigured(siteID string) (bool, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return false, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return false, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return false, err
	}
	endpoint, err := pinnable.Load(root, siteID)
	return endpoint != "", err
}

func (l *Library) RequestPinnablePin(siteID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	site, _, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return err
	}
	if site.IPNS == "" || site.LastPublishedCID == "" {
		return errors.New("请先发布站点")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	endpoint, err := pinnable.Load(root, siteID)
	if err != nil {
		return err
	}
	if endpoint == "" {
		return errors.New("请先配置 Pinnable 地址")
	}
	if err := pinnable.NewClient().PinForSite(endpoint, site.IPNS, false); err != nil {
		return err
	}
	l.pushPinnableRecord(endpoint, site.IPNS)
	return nil
}

// pushPinnableRecord best-effort posts the locally signed IPNS record for ipns
// to POST <endpoint>/record after the pin request has verified the endpoint.
func (l *Library) pushPinnableRecord(endpoint, ipns string) {
	node := l.getNode()
	if node == nil || ipns == "" {
		return
	}
	record, err := node.LocalIPNSRecord(ipns)
	if err != nil || len(record) == 0 {
		return
	}
	if err := pinnable.NewClient().PushRecord(endpoint, record); err != nil {
		log.Printf("push IPNS record to Pinnable: %v", err)
	}
}

func (l *Library) PinnableStatus(siteID string) (pinnable.Status, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return pinnable.Status{}, err
	}
	if _, _, _, err = s.PublicationSnapshot(siteID); err != nil {
		return pinnable.Status{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return pinnable.Status{}, err
	}
	endpoint, err := pinnable.Load(root, siteID)
	if err != nil {
		return pinnable.Status{}, err
	}
	if endpoint == "" {
		return pinnable.Status{}, errors.New("请先配置 Pinnable 地址")
	}
	return pinnable.NewClient().Status(endpoint)
}
