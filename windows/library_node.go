package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/applog"
	"windows/internal/ipfs"
)

func (l *Library) getNode() *ipfs.Client {
	l.nodeMu.RLock()
	defer l.nodeMu.RUnlock()
	return l.node
}

func (l *Library) setNode(node *ipfs.Client) {
	l.nodeMu.Lock()
	l.node = node
	l.nodeMu.Unlock()
}

// IPFSStatus observes the existing daemon concurrently without blocking behind
// active publish or subscription refresh operations.
func (l *Library) IPFSStatus() (ipfs.KuboStatus, error) {
	s, err := l.open()
	if err != nil {
		return ipfs.KuboStatus{}, err
	}
	disabled := s.IPFSDisabled()
	node := l.getNode()
	if node == nil {
		return ipfs.KuboStatus{Disabled: disabled}, nil
	}
	status, err := node.Status()
	status.Disabled = disabled
	return status, err
}

// SetIPFSEnabled persists the user's node preference and controls the owned daemon.
func (l *Library) SetIPFSEnabled(enabled bool) error {
	if !l.publishMu.TryLock() {
		return errors.New("正在发布站点，请稍后重试")
	}
	defer l.publishMu.Unlock()
	if !l.publicMu.TryLock() {
		return errors.New("节点正忙，请稍后重试")
	}
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	wasDisabled := s.IPFSDisabled()
	if err = s.SetIPFSDisabled(!enabled); err != nil {
		return err
	}
	if !enabled {
		l.stopProvideWorkers()
		if l.node != nil {
			return l.node.Close()
		}
		return nil
	}
	if _, _, err = l.ensureNode(true); err != nil {
		if wasDisabled {
			if restoreErr := s.SetIPFSDisabled(true); restoreErr != nil {
				return fmt.Errorf("启动节点失败：%v；恢复设置失败：%w", err, restoreErr)
			}
		}
		return err
	}
	return nil
}

func (l *Library) IPFSRepoSize() (int64, error) {
	node := l.getNode()
	if node == nil {
		return 0, errors.New("IPFS 节点未运行")
	}
	return node.RepoSize()
}

func (l *Library) IPFSGarbageCollect() error {
	if !l.publishMu.TryLock() {
		return errors.New("正在发布站点，请稍后重试")
	}
	defer l.publishMu.Unlock()
	if !l.publicMu.TryLock() {
		return errors.New("节点正忙，请稍后重试")
	}
	defer l.publicMu.Unlock()
	if l.node == nil {
		return errors.New("IPFS 节点未运行")
	}
	return l.node.GarbageCollect()
}

func (l *Library) ResolveIPFSResourceURL(input string) (string, error) {
	st, err := l.IPFSStatus()
	if err != nil {
		return "", err
	}
	return ipfs.ResolveResourceURL(st.Gateway, input)
}

// Phase 5: Key Manager, Categorized Live Log Viewer, Downloads Manager, Template Live Reload
func (l *Library) ListIPNSKeys() ([]ipfs.KeyInfo, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if _, _, err := l.ensureNode(false); err != nil {
		return nil, err
	}
	return l.node.ListKeys()
}

func (l *Library) ExportIPNSKey(ctx context.Context, name string) (string, error) {
	dir, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择导出 .key 的目录", Directory: true})
	if err != nil || len(dir) == 0 {
		return "", err
	}
	target := filepath.Join(dir[0], name+".key")
	if !l.publishMu.TryLock() {
		return "", errors.New("正在发布站点，请稍后重试")
	}
	defer l.publishMu.Unlock()
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if _, _, err := l.ensureNode(false); err != nil {
		return "", err
	}
	if err := l.node.ExportKey(name, target); err != nil {
		return "", err
	}
	applog.Global.Append("IPFS", "已导出 IPNS 密钥："+name)
	return target, nil
}

func (l *Library) ImportIPNSKey(ctx context.Context, name string) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "选择要导入的 .key 文件"})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	if !l.publishMu.TryLock() {
		return "", errors.New("正在发布站点，请稍后重试")
	}
	defer l.publishMu.Unlock()
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if _, _, err := l.ensureNode(false); err != nil {
		return "", err
	}
	id, err := l.node.ImportKey(name, paths[0])
	if err != nil {
		return "", err
	}
	applog.Global.Append("IPFS", fmt.Sprintf("已导入 IPNS 密钥：%s (%s)", name, id))
	return id, nil
}

// ensureNode is called with publicMu held. Import/export only need an offline repository.
func (l *Library) ensureNode(online bool) (string, string, error) {
	if online {
		s, err := l.open()
		if err != nil {
			return "", "", err
		}
		if s.IPFSDisabled() {
			return "", "", errors.New("IPFS 节点已关闭，请在节点状态面板中开启")
		}
	}
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return "", "", err
	}
	dataDir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return "", "", err
	}
	executable := filepath.Join(resources, "ipfs.exe")
	if _, err = os.Stat(executable); err != nil {
		return "", "", errors.New("缺少 Kubo 程序，请使用包含 Kubo 的安装包")
	}
	if l.node == nil {
		l.setNode(ipfs.New(executable, filepath.Join(dataDir, "ipfs")))
	}
	if !l.node.Alive() {
		if err = l.node.Init(); err != nil {
			return "", "", err
		}
		if online {
			if err = l.node.Start(false); err != nil {
				return "", "", err
			}
		}
	}
	return dataDir, executable, nil
}

func (l *Library) stopProvideWorkers() {
	l.provideMu.Lock()
	cancel := l.provideCancel
	l.provideCancel = nil
	l.provideCtx = nil
	l.provideMu.Unlock()
	if cancel != nil {
		cancel()
	}
	l.provideWorkers.Wait()
}

// scheduleProvide pushes recursive provider records for published CIDs and,
// when Kubo has just started behind NAT without /p2p-circuit relays yet,
// re-announces once public or relay reachability becomes available.
func (l *Library) scheduleProvide(cids ...string) {
	node := l.getNode()
	if node == nil {
		return
	}
	unique := make([]string, 0, len(cids))
	seen := make(map[string]struct{}, len(cids))
	for _, cid := range cids {
		cid = strings.TrimSpace(cid)
		if cid == "" {
			continue
		}
		if _, exists := seen[cid]; exists {
			continue
		}
		seen[cid] = struct{}{}
		unique = append(unique, cid)
	}
	if len(unique) == 0 {
		return
	}
	l.provideMu.Lock()
	if l.provideCtx == nil {
		l.provideCtx, l.provideCancel = context.WithCancel(context.Background())
	}
	ctx := l.provideCtx
	pollInterval := l.providePollInterval
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	waitLimit := l.provideWaitLimit
	if waitLimit <= 0 {
		waitLimit = 90 * time.Second
	}
	l.provideWorkers.Add(1)
	l.provideMu.Unlock()

	go func() {
		defer l.provideWorkers.Done()
		provideAll := func() {
			for _, cid := range unique {
				if ctx.Err() != nil {
					return
				}
				provideCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				err := node.ProvideContext(provideCtx, cid)
				cancel()
				if err != nil && ctx.Err() == nil {
					log.Printf("routing provide %s: %v", cid, err)
				}
			}
		}
		status, err := node.Status()
		if err == nil && status.Running && status.Online {
			provideAll()
			if status.Reachability == ipfs.ReachabilityDirect || status.Reachability == ipfs.ReachabilityRelay {
				return
			}
		}
		waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
		_, ready := node.WaitForPublicReachability(waitCtx, pollInterval)
		cancel()
		if ready && ctx.Err() == nil {
			provideAll()
		}
	}()
}

func (l *Library) closeNode() {
	l.stopProvideWorkers()
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	if l.node != nil {
		if err := l.node.Close(); err != nil {
			log.Printf("close Kubo: %v", err)
		}
	}
}
