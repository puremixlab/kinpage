package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestGlobalSettingsFieldSearchLocateAndReturn(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "山野通信", Created: now}},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1360, 860)

	app.openToolsModal(toolsTabAI)
	tt.Frame()

	// Search for "休眠" -> should return Prevent Sleep under General.
	app.toolsModal.search = "休眠"
	tt.Frame()
	if err := tt.Click("运行期间防止系统休眠 (通用 · 通用)"); err != nil {
		t.Fatalf("expected sleep setting search result to be clickable: %v", err)
	}
	if app.toolsModal.tab != toolsTabGeneral || app.toolsModal.highlightID != "global-sleep" {
		t.Fatalf("expected General tab with global-sleep highlighted, got tab=%d highlight=%q", app.toolsModal.tab, app.toolsModal.highlightID)
	}
	if app.toolsModal.returnSearch != "休眠" {
		t.Fatalf("expected returnSearch preserved, got %q", app.toolsModal.returnSearch)
	}

	// Return to search results restores previous query.
	if err := tt.Click("返回搜索结果"); err != nil {
		t.Fatalf("expected return-to-search button to work: %v", err)
	}
	if app.toolsModal.search != "休眠" {
		t.Fatalf("expected search restored to 休眠, got %q", app.toolsModal.search)
	}

	// Search for "端口" -> opens Local API tab and opens the secondary config sheet.
	app.toolsModal.search = "端口"
	tt.Frame()
	if err := tt.Click("本地 API · 端口与认证 (网络与工具 · 本地 API)"); err != nil {
		t.Fatalf("expected Local API port search result to be clickable: %v", err)
	}
	if app.toolsModal.tab != toolsTabAPI || !app.toolsModal.apiModalOpen {
		t.Fatalf("expected Local API tab and config modal open, got tab=%d modal=%v", app.toolsModal.tab, app.toolsModal.apiModalOpen)
	}
}

func TestSiteSettingsIdentitySearchPublishIsolationAndDeployPersistence(t *testing.T) {
	now := time.Now()
	pubTrue := true
	backend := &fakeBackend{
		sites: []store.Site{
			{
				ID:                    "s1",
				Name:                  "山野通信",
				Created:               now,
				CustomCodeHeadEnabled: true,
				CustomCodeHead:        "<meta name='test' content='1'>",
				PublishAsIPNS:         &pubTrue,
			},
			{
				ID:      "s2",
				Name:    "设计观察",
				Created: now,
			},
		},
		deployInfos: map[string]DeploymentSummary{
			"s1:cloudflare": {
				Configured: true,
				Name:       "my-pages",
				Server:     "acc-123",
				AutoDeploy: true,
				LastURL:    "https://my-pages.pages.dev",
			},
		},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1360, 860)

	// Open Site Settings for s1: verify site identity shown and Deploy loads persisted summary.
	app.openSiteSettings(backend.sites[0])
	tt.Frame()
	if !tt.HasText("山野通信") {
		t.Fatal("expected current site name visible in Site Settings")
	}

	// Search "预热" in Site Settings -> jumps to Publish Behavior tab.
	app.siteSettings.search = "预热"
	tt.Frame()
	if err := tt.Click("发布行为 (发布与固定 · 发布行为)"); err != nil {
		t.Fatalf("expected Publish Behavior search result to be clickable: %v", err)
	}
	if app.siteSettings.tab != siteSettingsTabPublish || app.siteSettings.highlightID != "site-publish-behavior" {
		t.Fatalf("expected Publish Behavior tab highlighted, got tab=%d highlight=%q", app.siteSettings.tab, app.siteSettings.highlightID)
	}

	// Toggle prewarm and save on Publish Behavior -> must preserve CustomCodeHead and keep window open.
	app.siteSettings.prewarmPost = true
	app.saveSiteSettings()
	if !app.siteSettings.open {
		t.Fatal("expected Site Settings window to stay open after per-page save")
	}
	if backend.sites[0].CustomCodeHead != "<meta name='test' content='1'>" || !backend.sites[0].CustomCodeHeadEnabled {
		t.Fatalf("saving Publish Behavior must preserve CustomCodeHead, got %+v", backend.sites[0])
	}

	// Switch to Deploy tab: verify persisted Cloudflare Pages summary is loaded and cancel modal preserves it.
	app.siteSettings.tab = siteSettingsTabDeploy
	tt.Frame()
	if !tt.HasText("Cloudflare Pages · my-pages") {
		t.Fatalf("expected persisted Cloudflare Pages summary, got texts %v", tt.Texts())
	}
	app.openDeployModal("cloudflare")
	app.integrations.deployName = "unsaved-draft-name"
	app.integrations.deployModalOpen = false
	tt.Frame()
	if !tt.HasText("Cloudflare Pages · my-pages") {
		t.Fatal("canceling deploy config modal must keep the saved summary intact")
	}

	// Switch to site s2: verify s2 shows its own unconfigured Deploy state.
	app.openSiteSettings(backend.sites[1])
	app.siteSettings.tab = siteSettingsTabDeploy
	tt.Frame()
	if !tt.HasText("配置 Cloudflare Pages") {
		t.Fatalf("expected unconfigured deploy action for s2, got texts %v", tt.Texts())
	}
}
