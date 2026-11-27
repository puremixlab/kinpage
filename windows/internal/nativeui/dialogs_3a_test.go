package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/ipfs"
	"windows/internal/offline"
	"windows/internal/siteimage"
	"windows/internal/store"
)

func TestNativeUISiteSettingsAndPublishModal(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "原站点", About: "旧简介", Created: now}},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	tt.Frame()

	if err := tt.Click("原站点"); err != nil {
		t.Fatalf("failed to select site: %v", err)
	}

	// Publish site modal.
	if err := tt.Click("发布站点"); err != nil {
		t.Fatalf("failed to click publish site: %v", err)
	}
	if !app.publishModal.open || app.publishModal.result.CID != "bafy-test-cid" {
		t.Fatalf("expected publish modal with CID, got %+v", app.publishModal)
	}
	if err := tt.Click("完成"); err != nil {
		t.Fatalf("failed to close publish modal: %v", err)
	}

	// Site settings: basic tab.
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("failed to open site settings: %v", err)
	}
	app.siteSettings.name = "新站点名称"
	app.siteSettings.author = "测试作者"
	tt.Frame()
	if err := tt.Click("保存站点设置"); err != nil {
		t.Fatalf("failed to save basic site settings: %v", err)
	}
	if backend.sites[0].Name != "新站点名称" || backend.sites[0].AuthorName != "测试作者" {
		t.Fatalf("expected updated site details, got %+v", backend.sites[0])
	}

	// Site settings: preset avatar in 基本资料, then switch to 社交 tab for domain.
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("failed to reopen site settings: %v", err)
	}
	if err := tt.Click("地球"); err != nil {
		t.Fatalf("failed to apply preset avatar: %v", err)
	}
	if backend.sites[0].AvatarPNG != "preset-earth" {
		t.Fatalf("expected preset avatar applied, got %q", backend.sites[0].AvatarPNG)
	}
	if err := tt.Click("社交"); err != nil {
		t.Fatalf("failed to switch to social tab: %v", err)
	}
	app.siteSettings.domain = "blog.example.com"
	tt.Frame()
	if err := tt.Click("保存站点设置"); err != nil {
		t.Fatalf("failed to save profile: %v", err)
	}
	if backend.sites[0].Domain != "blog.example.com" {
		t.Fatalf("expected updated domain, got %q", backend.sites[0].Domain)
	}

	// Remove site -> restore from trash (Danger Zone sits at bottom of 基本资料).
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("failed to open site settings for remove: %v", err)
	}
	app.siteSettings.scroll.Y = 400
	tt.Frame()
	if err := tt.Click("移入回收站"); err != nil {
		t.Fatalf("failed to click remove site: %v", err)
	}
	if err := tt.Click("移入回收站"); err != nil {
		t.Fatalf("failed to confirm remove site: %v", err)
	}
	if len(backend.sites) != 0 || len(backend.removedSites) != 1 {
		t.Fatalf("expected site moved to trash, live=%d trash=%d", len(backend.sites), len(backend.removedSites))
	}

	if err := tt.Click("设置"); err != nil {
		t.Fatalf("failed to open settings popover: %v", err)
	}
	if err := tt.Click("更多设置…"); err != nil {
		t.Fatalf("failed to open global settings: %v", err)
	}
	if err := tt.Click("回收站"); err != nil {
		t.Fatalf("failed to switch to trash tab: %v", err)
	}
	if err := tt.Click("恢复"); err != nil {
		t.Fatalf("failed to restore site: %v", err)
	}
	if len(backend.sites) != 1 || len(backend.removedSites) != 0 {
		t.Fatalf("expected site restored, live=%d trash=%d", len(backend.sites), len(backend.removedSites))
	}
}

type presetAvatarTestBackend struct{ *fakeBackend }

func (*presetAvatarTestBackend) PresetAvatars() []siteimage.PresetAvatar {
	return siteimage.PresetAvatars()
}

func TestNativeUISiteSettingsCentersPresetAvatarRows(t *testing.T) {
	site := store.Site{ID: "s1", Name: "Example site"}
	backend := &presetAvatarTestBackend{&fakeBackend{sites: []store.Site{site}}}
	app := New(backend)
	app.openSiteSettings(site)
	tt := ui.NewTester(app.View, 820, 760)
	tt.Frame()

	header, ok := tt.Find(app.t("siteSettings.avatarPresets"))
	if !ok {
		t.Fatal("preset gallery is missing")
	}
	rows := map[float32][2]float32{}
	for _, preset := range backend.PresetAvatars() {
		button, ok := tt.Find(preset.Title)
		if !ok {
			t.Fatalf("preset button %q is missing", preset.Title)
		}
		limits, seen := rows[button.Y]
		if !seen {
			limits[0] = button.X
		}
		limits[1] = button.X + button.W
		rows[button.Y] = limits
	}
	if len(rows) < 2 {
		t.Fatal("the narrow settings pane should wrap the presets")
	}
	for y, limits := range rows {
		rowCenter := (limits[0] + limits[1]) / 2
		paneCenter := header.X + header.W/2
		if delta := rowCenter - paneCenter; delta < -0.5 || delta > 0.5 {
			t.Errorf("preset row at y=%.1f is centered at %.1f, want %.1f", y, rowCenter, paneCenter)
		}
	}
}

func TestNativeUIIPFSOfflineAndFoldersModals(t *testing.T) {
	backend := &fakeBackend{
		ipfsStatus:   ipfs.KuboStatus{Running: true, Online: true, Version: "0.35.0", Peers: 12},
		ipfsRepoSize: 10 << 20,
		offlineUsage: offline.Usage{ManagedFiles: 5, TotalBytes: 2 << 20, ReclaimableFiles: 2, ReclaimableBytes: 512 << 10},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 860)
	tt.Frame()

	// IPFS modal: GC + toggle.
	if err := tt.Click("IPFS 节点"); err != nil {
		t.Fatalf("failed to open IPFS modal: %v", err)
	}
	if err := tt.Click("清理未固定数据"); err != nil {
		t.Fatalf("failed to click IPFS GC: %v", err)
	}
	if backend.gcCount != 1 {
		t.Fatalf("expected gcCount=1, got %d", backend.gcCount)
	}
	if err := tt.Click("停用节点"); err != nil {
		t.Fatalf("failed to disable IPFS: %v", err)
	}
	if !backend.ipfsStatus.Disabled {
		t.Fatal("expected IPFS disabled")
	}
	if err := tt.Click("关闭"); err != nil {
		t.Fatalf("failed to close IPFS modal: %v", err)
	}

	// Offline storage tab in Global Settings: cleanup.
	if err := tt.Click("设置"); err != nil {
		t.Fatalf("failed to open settings popover: %v", err)
	}
	if err := tt.Click("更多设置…"); err != nil {
		t.Fatalf("failed to open global settings: %v", err)
	}
	if err := tt.Click("离线存储"); err != nil {
		t.Fatalf("failed to switch to offline tab: %v", err)
	}
	if err := tt.Click("清理可回收空间"); err != nil {
		t.Fatalf("failed to cleanup offline storage: %v", err)
	}
	if backend.offlineUsage.ReclaimableBytes != 0 {
		t.Fatalf("expected reclaimable bytes cleared, got %d", backend.offlineUsage.ReclaimableBytes)
	}

	// Published folders tab in Global Settings: empty state -> secondary add modal -> publish + remove.
	if err := tt.Click("已发布文件夹"); err != nil {
		t.Fatalf("failed to switch to folders tab: %v", err)
	}
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to open add folder secondary modal: %v", err)
	}
	app.foldersModal.newName = "图库"
	tt.Frame()
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to submit add folder modal: %v", err)
	}
	if len(backend.folders) != 1 || backend.folders[0].Name != "图库" {
		t.Fatalf("expected added folder, got %+v", backend.folders)
	}
	if err := tt.Click("发布"); err != nil {
		t.Fatalf("failed to publish folder: %v", err)
	}
	if backend.folders[0].LastCID != "bafy-folder" {
		t.Fatalf("expected folder published CID, got %q", backend.folders[0].LastCID)
	}
	if err := tt.Click("移除"); err != nil {
		t.Fatalf("failed to remove folder: %v", err)
	}
	if len(backend.folders) != 0 {
		t.Fatalf("expected folder removed, got %d", len(backend.folders))
	}
}
