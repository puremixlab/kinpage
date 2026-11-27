package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/migration"
	"windows/internal/pinnable"
	"windows/internal/publisher"
	"windows/internal/services"
	"windows/internal/store"
)

func (l *Library) ActivePublishCount() int {
	return int(l.activePublishes.Load())
}

// PublishSite publishes the saved snapshot; Kubo is started only on request.
func (l *Library) PublishSite(siteID string) (publisher.Result, error) {
	return l.publishSite(siteID, nil)
}

func (l *Library) publishSite(siteID string, mobileArticle *store.Article) (publisher.Result, error) {
	l.activePublishes.Add(1)
	defer l.activePublishes.Add(-1)
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return publisher.Result{}, err
	}
	snapshot, articles, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return publisher.Result{}, err
	}
	site := &snapshot
	if mobileArticle != nil {
		matches := slices.ContainsFunc(articles, func(article store.Article) bool {
			return article.ID == mobileArticle.ID && article.Title == mobileArticle.Title && article.Content == mobileArticle.Content
		})
		if !matches || site.Archived || !site.ShouldPublishIPNS() {
			return publisher.Result{}, errors.New("手机文章或发布站点已更改")
		}
		if site.LastPublishedCID != "" && !site.PublishPending && site.PublishedRevision == site.ContentRevision {
			return publisher.Result{CID: site.LastPublishedCID, IPNS: site.IPNS, Revision: site.PublishedRevision}, nil
		}
	}
	var importedPublic string
	if len(raw) > 0 {
		root, e := mygo.App.Path(mygo.PathUserData)
		if e != nil {
			return publisher.Result{}, e
		}
		importedPublic, e = os.MkdirTemp("", "planet-import-publish-")
		if e != nil {
			return publisher.Result{}, e
		}
		defer os.RemoveAll(importedPublic)
		templateDir, e := l.templatePath(*site)
		if e != nil {
			return publisher.Result{}, e
		}
		if templateDir == "" {
			return publisher.Result{}, errors.New("发布导入站点前，请先选择已验证的模板")
		}
		if e = migration.StageImportedPublic(filepath.Join(root, "imports", siteID), importedPublic, raw, *site, articles, templateDir); e != nil {
			return publisher.Result{}, fmt.Errorf("无法发布原版备份：%w", e)
		}
	}
	if !site.ShouldPublishIPNS() {
		root, e := mygo.App.Path(mygo.PathUserData)
		if e != nil {
			return publisher.Result{}, e
		}
		result := publisher.Result{CID: site.LastPublishedCID, IPNS: site.IPNS, Revision: site.ContentRevision}
		if warning := l.autoDeploy(root, *site, articles); warning != "" {
			result.PinnableWarning = "外部发布未完成：" + warning
		}
		if warning := l.syncPublishedX(root, *site, articles); warning != "" {
			if result.PinnableWarning != "" {
				result.PinnableWarning += " "
			}
			result.PinnableWarning += "X 同步未完成：" + warning
		}
		result.TelegramWarning = l.syncPublishedTelegram(root, *site, articles, raw)
		return result, nil
	}
	if _, _, err := l.ensureNode(true); err != nil {
		return publisher.Result{}, err
	}
	if site.IPNS == "" {
		key, e := l.node.EnsureKey(site.ID)
		if e != nil {
			return publisher.Result{}, e
		}
		if e = s.SetIdentity(site.ID, key); e != nil {
			return publisher.Result{}, e
		}
		site.IPNS = key
	}
	var result publisher.Result
	if importedPublic != "" {
		result, err = publisher.PublishDirectory(l.node, importedPublic, *site)
	} else {
		templateDir, e := l.templatePath(*site)
		if e != nil {
			return publisher.Result{}, e
		}
		root, e := mygo.App.Path(mygo.PathUserData)
		if e != nil {
			return publisher.Result{}, e
		}
		result, err = publisher.PublishWithAssets(l.node, os.TempDir(), *site, articles, templateDir, root)
	}
	if err != nil {
		return publisher.Result{}, err
	}
	result.Revision = site.ContentRevision
	result.Pending, err = s.SetPublishedRevision(siteID, result.IPNS, result.CID, result.Revision)
	if err != nil {
		return result, fmt.Errorf("IPNS 已发布到 %s，但保存本地状态失败：%w", result.IPNS, err)
	}
	l.scheduleProvide(result.CID)
	services.SendDesktopNotification("邻页发布完成", site.Name+" 已发布到 IPNS")
	if site.ShouldPrewarmNewPost() && result.CID != "" {
		l.schedulePrewarm(*site, result.CID)
	}
	// This integration is opt-in. IPNS publication has already succeeded, so a
	// separate warning must not turn it into a failed publication.
	root, e := mygo.App.Path(mygo.PathUserData)
	if e == nil {
		endpoint, e := pinnable.Load(root, siteID)
		if e != nil {
			result.PinnableWarning = "站点已发布，但无法读取 Pinnable 配置。"
		} else if endpoint != "" {
			if e = pinnable.NewClient().PinForSite(endpoint, result.IPNS, true); e != nil {
				result.PinnableWarning = "站点已发布，但 Pinnable 固定请求未完成：" + e.Error() + "。请在固定服务中检查。"
			} else {
				l.pushPinnableRecord(endpoint, result.IPNS)
			}
		}
		if e = services.PinPublished(root, siteID, result.CID, services.NewClient()); e != nil {
			result.PinnableWarning += " 站点已发布，但 Filebase 固定请求未完成：" + e.Error()
		}
		if e = services.SyncDWebServices(root, siteID, result.CID, nil); e != nil {
			result.PinnableWarning += " 站点已发布，但 dWebServices.xyz 同步未完成：" + e.Error()
		}
		if warning := l.autoDeploy(root, *site, articles); warning != "" {
			result.PinnableWarning += " IPNS 已发布，但外部发布未完成：" + warning
		}
		if warning := l.syncPublishedX(root, *site, articles); warning != "" {
			result.PinnableWarning += " 站点已发布，但 X 同步未完成：" + warning
		}
		result.TelegramWarning = l.syncPublishedTelegram(root, *site, articles, raw)
	} else {
		result.TelegramWarning = "站点已发布，但无法读取 Telegram 本机配置。"
	}
	return result, nil
}

func prewarmURLsForSite(site store.Site, cid string) []string {
	var urls []string
	if site.IPNS != "" {
		base := "https://" + site.IPNS + ".eth.sucks"
		for _, suffix := range []string{"/", "/avatar.png", "/favicon.ico", "/planet.json", "/tags.html", "/archive.html"} {
			urls = append(urls, base+suffix)
		}
	}
	if cid != "" {
		if strings.HasPrefix(cid, "Qm") {
			urls = append(urls, "https://dweb.link/ipfs/"+cid+"/")
		} else {
			urls = append(urls, "https://"+cid+".eth.sucks/")
		}
	}
	return urls
}

func (l *Library) schedulePrewarm(site store.Site, cid string) {
	node := l.getNode()
	if node == nil {
		return
	}
	status, err := node.Status()
	if err != nil || !status.Running || !status.Online {
		return
	}
	urls := prewarmURLsForSite(site, cid)
	if len(urls) == 0 {
		return
	}
	go func() {
		client := &http.Client{Timeout: 15 * time.Second}
		for _, target := range urls {
			resp, err := client.Get(target)
			if err != nil {
				continue
			}
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
			resp.Body.Close()
		}
	}()
}

func (l *Library) maintainPublished(ctx context.Context) {
	l.resumeSiteDeletions()
	s, err := l.open()
	if err != nil {
		log.Printf("read library: %v", err)
		return
	}
	sites, err := s.ListSites()
	if err != nil {
		log.Printf("list sites: %v", err)
		return
	}
	var startupCIDs []string
	for _, site := range sites {
		if site.LastPublishedCID != "" {
			startupCIDs = append(startupCIDs, site.LastPublishedCID)
		}
	}
	if len(startupCIDs) > 0 {
		l.publicMu.Lock()
		_, _, err = l.ensureNode(true)
		l.publicMu.Unlock()
		if err != nil {
			log.Printf("start IPFS node: %v", err)
		} else {
			l.scheduleProvide(startupCIDs...)
		}
	}
	refresh := func() {
		l.publicMu.Lock()
		defer l.publicMu.Unlock()
		if ctx.Err() != nil || s.IPFSDisabled() {
			return
		}
		current, err := s.ListSites()
		if err != nil {
			log.Printf("list sites: %v", err)
			return
		}
		needsNode := false
		for _, site := range current {
			if site.LastPublishedCID != "" {
				needsNode = true
				break
			}
		}
		if !needsNode {
			return
		}
		if l.node == nil || !l.node.Alive() {
			if _, _, err = l.ensureNode(true); err != nil {
				log.Printf("start IPFS node: %v", err)
				return
			}
		}
		for _, site := range current {
			if ctx.Err() != nil {
				return
			}
			if !site.ShouldPublishIPNS() || site.PublishPending || site.IPNS == "" || site.LastPublishedCID == "" || !publisher.RenewDue(time.Now().UTC(), site.LastPublished) {
				continue
			}
			key, err := l.node.LookupKeyContext(ctx, site.ID)
			if err != nil || key != site.IPNS {
				log.Printf("skip IPNS renewal for %s: key mismatch or unavailable", site.ID)
				continue
			}
			name, err := l.node.PublishOnlineContext(ctx, site.ID, site.LastPublishedCID)
			if err != nil {
				log.Printf("IPNS renewal failed for %s: %v", site.ID, err)
				continue
			}
			if name != site.IPNS {
				log.Printf("IPNS renewal identity mismatch for %s", site.ID)
				continue
			}
			if err = s.SetPublished(site.ID, name, site.LastPublishedCID); err != nil {
				log.Printf("saving renewal for %s: %v", site.ID, err)
			}
			l.scheduleProvide(site.LastPublishedCID)
		}
	}
	refresh()
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}
