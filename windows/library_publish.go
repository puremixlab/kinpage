package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/ipfs"
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
	return l.PublishSiteContext(context.Background(), siteID)
}

type preparedPublication struct {
	site     store.Site
	articles []store.Article
	raw      json.RawMessage
	root     string
	dir      string
	node     *ipfs.Client
}

type publicationNode struct {
	*ipfs.Client
	ctx context.Context
}

func (n publicationNode) Add(dir string) (string, error) {
	return n.AddContext(n.ctx, dir)
}

func (n publicationNode) LookupKey(name string) (string, error) {
	return n.LookupKeyContext(n.ctx, name)
}

func (n publicationNode) PublishOnline(key, cid string) (string, error) {
	return n.PublishOnlineContext(n.ctx, key, cid)
}

func (l *Library) prepareSitePublication(ctx context.Context, s *store.Store, siteID string) (preparedPublication, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if err := ctx.Err(); err != nil {
		return preparedPublication{}, err
	}
	site, articles, raw, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return preparedPublication{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return preparedPublication{}, err
	}
	p := preparedPublication{site: site, articles: articles, raw: raw, root: root}
	if site.ShouldPublishIPNS() {
		if _, _, err := l.ensureNode(true); err != nil {
			return preparedPublication{}, err
		}
		p.node = l.node
		if p.site.IPNS == "" {
			key, err := p.node.EnsureKeyContext(ctx, siteID)
			if err != nil {
				return preparedPublication{}, err
			}
			if err := s.SetIdentity(siteID, key); err != nil {
				return preparedPublication{}, err
			}
			p.site.IPNS = key
		}
	}

	if len(raw) == 0 {
		p.dir, err = l.stageRemote(root, p.site, articles)
		return p, err
	}
	templateDir, err := l.templatePath(p.site)
	if err != nil {
		return preparedPublication{}, err
	}
	if templateDir == "" {
		return preparedPublication{}, errors.New("发布导入站点前，请先选择已验证的模板")
	}
	p.dir, err = os.MkdirTemp("", "planet-import-publish-")
	if err != nil {
		return preparedPublication{}, err
	}
	err = migration.StageImportedPublic(filepath.Join(root, "imports", siteID), p.dir, raw, p.site, articles, templateDir)
	if err != nil {
		_ = os.RemoveAll(p.dir)
		return preparedPublication{}, fmt.Errorf("无法发布原版备份：%w", err)
	}
	return p, nil
}

func (l *Library) PublishSiteContext(ctx context.Context, siteID string) (publisher.Result, error) {
	l.activePublishes.Add(1)
	defer l.activePublishes.Add(-1)
	l.publishMu.Lock()
	defer l.publishMu.Unlock()
	s, err := l.open()
	if err != nil {
		return publisher.Result{}, err
	}
	p, err := l.prepareSitePublication(ctx, s, siteID)
	if err != nil {
		return publisher.Result{}, err
	}
	defer os.RemoveAll(p.dir)
	if err := ctx.Err(); err != nil {
		return publisher.Result{}, err
	}

	site, root := p.site, p.root
	result := publisher.Result{Revision: site.ContentRevision}
	if p.node != nil {
		// The directory owns all assets needed by Kubo and external targets.
		// Let authoring operations proceed while the network request is in flight.
		result, err = publisher.PublishDirectory(publicationNode{Client: p.node, ctx: ctx}, p.dir, site)
		if err != nil {
			return result, err
		}
		result.Revision = site.ContentRevision
		result.Pending, err = s.SetPublishedRevision(siteID, result.IPNS, result.CID, result.Revision)
		if err != nil {
			return result, fmt.Errorf("IPNS 已发布到 %s，但保存本地状态失败：%w", result.IPNS, err)
		}
		l.scheduleProvide(result.CID)
		services.SendDesktopNotification("邻页发布完成", site.Name+" 已发布到 IPNS")
		if site.ShouldPrewarmNewPost() {
			l.schedulePrewarm(site, result.CID)
		}
	}

	// Keep the integration records serialized with their configuration and
	// manual retry operations, including uncertain remote results.
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if p.node != nil {
		endpoint, err := pinnable.Load(root, siteID)
		if err != nil {
			result.PinnableWarning = "站点已发布，但无法读取 Pinnable 配置。"
		} else if endpoint != "" {
			if err = pinnable.NewClient().PinForSite(endpoint, result.IPNS, true); err != nil {
				result.PinnableWarning = "站点已发布，但 Pinnable 固定请求未完成：" + err.Error() + "。请在固定服务中检查。"
			} else {
				l.pushPinnableRecord(endpoint, result.IPNS)
			}
		}
		if err = services.PinPublished(root, siteID, result.CID, services.NewClient()); err != nil {
			result.PinnableWarning += " 站点已发布，但 Filebase 固定请求未完成：" + err.Error()
		}
		if err = services.SyncDWebServices(root, siteID, result.CID, nil); err != nil {
			result.PinnableWarning += " 站点已发布，但 dWebServices.xyz 同步未完成：" + err.Error()
		}
	}
	result.Deployments = l.autoDeploy(root, siteID, p.dir)
	if p.node == nil && len(result.Deployments) == 0 {
		result.PinnableWarning = "尚未配置站点发布目标。"
	}
	if warning := l.syncPublishedX(root, site, p.articles); warning != "" {
		if result.PinnableWarning != "" {
			result.PinnableWarning += " "
		}
		result.PinnableWarning += "X 同步未完成：" + warning
	}
	result.TelegramWarning = l.syncPublishedTelegram(root, site, p.articles, p.raw)
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
		if !l.publishMu.TryLock() {
			return
		}
		defer l.publishMu.Unlock()
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
