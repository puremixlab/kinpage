package nativeui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func TestNativeUIIntegrationsModalBatch3B(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "集成测试站", Created: now}},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 900)
	tt.Frame()

	if err := tt.Click("集成测试站"); err != nil {
		t.Fatalf("failed to select site: %v", err)
	}
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("failed to open site settings: %v", err)
	}
	if err := tt.Click("模板"); err != nil {
		t.Fatalf("failed to switch to templates tab: %v", err)
	}

	// 1. Templates tab.
	if err := tt.Click("应用"); err != nil {
		t.Fatalf("failed to apply builtin template: %v", err)
	}
	if backend.sites[0].TemplateName != "Plain" {
		t.Fatalf("expected Plain template applied, got %q", backend.sites[0].TemplateName)
	}

	// 2. Deploy tab.
	if err := tt.Click("部署"); err != nil {
		t.Fatalf("failed to switch to deploy tab: %v", err)
	}
	app.integrations.deployName = "my-blog"
	app.integrations.deployToken = "secret"
	tt.Frame()
	if err := tt.Click("保存部署配置"); err != nil {
		t.Fatalf("failed to save deployment: %v", err)
	}
	if err := tt.Click("立即部署"); err != nil {
		t.Fatalf("failed to trigger deploy: %v", err)
	}
	if len(backend.deployCalls) != 2 {
		t.Fatalf("expected 2 deploy calls, got %+v", backend.deployCalls)
	}

	// 3. X Sync tab.
	if err := tt.Click("X 同步"); err != nil {
		t.Fatalf("failed to switch to X tab: %v", err)
	}
	if err := tt.Click("保存 X 配置"); err != nil {
		t.Fatalf("failed to save X config: %v", err)
	}
	if !backend.xConfigured {
		t.Fatal("expected X sync configured")
	}

	// 4. Telegram tab.
	if err := tt.Click("Telegram"); err != nil {
		t.Fatalf("failed to switch to Telegram tab: %v", err)
	}
	if err := tt.Click("保存 Telegram"); err != nil {
		t.Fatalf("failed to save Telegram config: %v", err)
	}
	if !backend.tgConfigured {
		t.Fatal("expected Telegram configured")
	}

	// 5. Aggregation tab.
	if err := tt.Click("聚合"); err != nil {
		t.Fatalf("failed to switch to aggregation tab: %v", err)
	}
	app.integrations.aggSources = "https://a.example/rss"
	tt.Frame()
	if err := tt.Click("保存聚合源"); err != nil {
		t.Fatalf("failed to save aggregation: %v", err)
	}
	if err := tt.Click("立即抓取同步"); err != nil {
		t.Fatalf("failed to sync aggregation: %v", err)
	}
	if backend.aggSynced != 1 {
		t.Fatalf("expected aggSynced=1, got %d", backend.aggSynced)
	}

	// 6. Web3 tab.
	if err := tt.Click("Web3"); err != nil {
		t.Fatalf("failed to switch to Web3 tab: %v", err)
	}
	app.integrations.web3Donate = true
	app.integrations.web3ETH = "vitalik.eth"
	app.integrations.web3Juicebox = true
	app.integrations.web3JuiceboxID = "42"
	tt.Frame()
	if err := tt.Click("保存 Web3 设置"); err != nil {
		t.Fatalf("failed to save Web3 settings: %v", err)
	}
	if !backend.sites[0].AcceptsDonation || backend.sites[0].AcceptsDonationETHAddress != "vitalik.eth" || backend.sites[0].JuiceboxProjectID != 42 {
		t.Fatalf("expected Web3 settings saved, got %+v", backend.sites[0])
	}
}
