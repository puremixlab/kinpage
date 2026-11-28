package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/egoist/mygo"
	"os"
	"time"
	"windows/internal/updates"
)

//go:embed release.json
var releaseMetadata []byte
var appVersion = func() string {
	var info struct {
		Version string `json:"version"`
	}
	if e := json.Unmarshal(releaseMetadata, &info); e != nil {
		panic("invalid embedded release version")
	}
	if _, e := updates.Compare(info.Version, info.Version); e != nil {
		panic("invalid embedded release version")
	}
	return info.Version
}()

type UpdateSettingsInfo = updates.SettingsInfo
type UpdateCheckInfo = updates.CheckInfo

func (l *Library) UpdateSettings() (UpdateSettingsInfo, error) {
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	root, e := mygo.App.Path(mygo.PathUserData)
	if e != nil {
		return UpdateSettingsInfo{}, e
	}
	cfg, e := updates.LoadConfig(root)
	return UpdateSettingsInfo{Settings: cfg, CurrentVersion: appVersion, Configured: cfg.FeedURL != ""}, e
}
func (l *Library) SaveUpdateSettings(feedURL, publicKey string, autoCheck bool) (UpdateSettingsInfo, error) {
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	if l.updateClosing {
		return UpdateSettingsInfo{}, errors.New("程序正在关闭")
	}
	root, e := mygo.App.Path(mygo.PathUserData)
	if e != nil {
		return UpdateSettingsInfo{}, e
	}
	cfg, e := updates.SaveSettings(root, updates.Config{FeedURL: feedURL, PublicKey: publicKey, AutoCheck: autoCheck})
	if e != nil {
		return UpdateSettingsInfo{}, e
	}
	l.invalidateUpdatesLocked()
	return UpdateSettingsInfo{Settings: cfg, CurrentVersion: appVersion, Configured: cfg.FeedURL != ""}, nil
}
func (l *Library) CheckUpdates() (UpdateCheckInfo, error) {
	return l.CheckUpdatesContext(context.Background())
}

func (l *Library) CheckUpdatesContext(parent context.Context) (UpdateCheckInfo, error) {
	l.updateMu.Lock()
	if err := parent.Err(); err != nil {
		l.updateMu.Unlock()
		return UpdateCheckInfo{}, err
	}
	if l.updateClosing {
		l.updateMu.Unlock()
		return UpdateCheckInfo{}, errors.New("程序正在关闭")
	}
	l.invalidateUpdatesLocked()
	root, e := mygo.App.Path(mygo.PathUserData)
	if e != nil {
		l.updateMu.Unlock()
		return UpdateCheckInfo{}, e
	}
	cfg, e := updates.LoadConfig(root)
	if e != nil {
		l.updateMu.Unlock()
		return UpdateCheckInfo{}, e
	}
	ctx, cancel := context.WithCancel(parent)
	l.updateCancel = cancel
	generation := l.updateGeneration
	l.updateWorkers.Add(1)
	l.updateMu.Unlock()
	defer l.updateWorkers.Done()
	defer cancel()
	r, e := updates.NewClient().Check(ctx, cfg, appVersion, time.Now())
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	if generation != l.updateGeneration {
		return UpdateCheckInfo{}, errors.New("更新检查已取消或设置已改变")
	}
	l.updateCancel = nil
	if err := ctx.Err(); err != nil {
		return UpdateCheckInfo{}, err
	}
	if e != nil {
		return UpdateCheckInfo{}, e
	}
	if e = updates.RecordSeen(root, cfg, r); e != nil {
		return UpdateCheckInfo{}, e
	}
	l.checkedUpdate = &r
	l.updateIdentity = updates.TrustIdentity(cfg)
	cmp, _ := updates.Compare(r.Version, appVersion)
	return UpdateCheckInfo{Version: r.Version, SHA256: r.SHA256, Size: r.Size, Available: cmp > 0}, nil
}
func (l *Library) DownloadUpdate(version, sha256 string) (resultPath string, resultError error) {
	return l.DownloadUpdateContext(context.Background(), version, sha256)
}

func (l *Library) DownloadUpdateContext(parent context.Context, version, sha256 string) (resultPath string, resultError error) {
	l.updateMu.Lock()
	if err := parent.Err(); err != nil {
		l.updateMu.Unlock()
		return "", err
	}
	if l.updateClosing {
		l.updateMu.Unlock()
		return "", errors.New("程序正在关闭")
	}
	if l.checkedUpdate == nil || l.checkedUpdate.Version != version || l.checkedUpdate.SHA256 != sha256 {
		l.updateMu.Unlock()
		return "", errors.New("请先重新检查更新")
	}
	root, e := mygo.App.Path(mygo.PathUserData)
	if e != nil {
		l.updateMu.Unlock()
		return "", e
	}
	cfg, e := updates.LoadConfig(root)
	if e != nil {
		l.updateMu.Unlock()
		return "", e
	}
	if updates.TrustIdentity(cfg) != l.updateIdentity {
		l.updateMu.Unlock()
		return "", errors.New("更新设置已变化，请重新检查")
	}
	r := *l.checkedUpdate
	if l.updateCancel != nil {
		l.updateCancel()
	}
	l.updateGeneration++
	generation := l.updateGeneration
	ctx, cancel := context.WithCancel(parent)
	l.updateCancel = cancel
	l.updateWorkers.Add(1)
	l.updateMu.Unlock()
	defer l.updateWorkers.Done()
	defer cancel()
	dir, e := updates.NewDownloadDirectory(root)
	if e != nil {
		return "", e
	}
	path := ""
	defer func() {
		if resultError == nil {
			return
		}
		if path != "" {
			if err := updates.RemoveVerifiedDownload(path, r); err != nil {
				resultError = fmt.Errorf("%v；未清理的更新缓存保留在 %s", resultError, dir)
			}
		}
		_ = os.Remove(dir)
	}()
	path, e = updates.NewClient().Download(ctx, cfg, appVersion, r, dir, time.Now())
	if e != nil {
		_ = os.Remove(dir)
		return "", e
	}
	if e = updates.VerifyDownloaded(path, r); e != nil {
		return "", e
	}
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	if generation != l.updateGeneration {
		return "", errors.New("下载已取消或更新设置已改变")
	}
	l.updateCancel = nil
	if err := ctx.Err(); err != nil {
		return "", err
	}
	l.downloadedUpdate = path
	return path, nil
}

func (l *Library) invalidateUpdatesLocked() {
	if l.updateCancel != nil {
		l.updateCancel()
		l.updateCancel = nil
	}
	l.updateGeneration++
	l.checkedUpdate = nil
	l.downloadedUpdate = ""
	l.updateIdentity = ""
}
func (l *Library) CancelUpdateOperation() {
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	l.invalidateUpdatesLocked()
}
func (l *Library) closeUpdates() {
	l.updateMu.Lock()
	l.updateClosing = true
	l.invalidateUpdatesLocked()
	l.updateMu.Unlock()
	l.updateWorkers.Wait()
}
func (l *Library) RevealDownloadedUpdate() error {
	l.updateMu.Lock()
	defer l.updateMu.Unlock()
	if l.checkedUpdate == nil || l.downloadedUpdate == "" {
		return errors.New("尚未保存已验证的更新包")
	}
	root, e := mygo.App.Path(mygo.PathUserData)
	if e != nil {
		return e
	}
	cfg, e := updates.LoadConfig(root)
	if e != nil {
		return e
	}
	if updates.TrustIdentity(cfg) != l.updateIdentity {
		return errors.New("更新设置已变化，请重新检查")
	}
	if e = updates.VerifyDownloaded(l.downloadedUpdate, *l.checkedUpdate); e != nil {
		return e
	}
	mygo.Shell.ShowItemInFolder(l.downloadedUpdate)
	return nil
}
