package nativeui

import (
	"errors"
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

	// 2. Deploy tab (empty state -> secondary modal -> configured card -> deploy now).
	if err := tt.Click("部署"); err != nil {
		t.Fatalf("failed to switch to deploy tab: %v", err)
	}
	if err := tt.Click("配置 Cloudflare Pages"); err != nil {
		t.Fatalf("failed to open deploy secondary modal: %v", err)
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

	// 3. X Sync tab (empty state -> secondary modal -> save).
	if err := tt.Click("X 同步"); err != nil {
		t.Fatalf("failed to switch to X tab: %v", err)
	}
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to open X secondary modal: %v", err)
	}
	if err := tt.Click("保存 X 配置"); err != nil {
		t.Fatalf("failed to save X config: %v", err)
	}
	if !backend.xConfigured {
		t.Fatal("expected X sync configured")
	}

	// 4. Telegram tab (empty state -> secondary modal -> save).
	if err := tt.Click("Telegram"); err != nil {
		t.Fatalf("failed to switch to Telegram tab: %v", err)
	}
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to open Telegram secondary modal: %v", err)
	}
	if err := tt.Click("保存 Telegram"); err != nil {
		t.Fatalf("failed to save Telegram config: %v", err)
	}
	if !backend.tgConfigured {
		t.Fatal("expected Telegram configured")
	}

	// 5. Aggregation tab (empty state -> secondary modal -> configured list -> sync).
	if err := tt.Click("聚合"); err != nil {
		t.Fatalf("failed to switch to aggregation tab: %v", err)
	}
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to open aggregation secondary modal: %v", err)
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

	// 6. Web3 tab (empty state -> secondary modal -> save).
	if err := tt.Click("Web3"); err != nil {
		t.Fatalf("failed to switch to Web3 tab: %v", err)
	}
	if err := tt.Click("配置"); err != nil {
		t.Fatalf("failed to open Web3 secondary modal: %v", err)
	}
	app.integrations.web3Donate = true
	app.integrations.web3ETH = "vitalik.eth"
	app.integrations.web3Juicebox = true
	app.integrations.web3JuiceboxID = "invalid"
	tt.Frame()
	if err := tt.Click("保存 Web3 设置"); err != nil {
		t.Fatalf("failed to try saving invalid Web3 settings: %v", err)
	}
	if !app.integrations.web3ModalOpen || app.integrations.err != app.t("int.web3.jbIntErr") {
		t.Fatalf("expected Web3 sheet to stay open with the ID error, got open=%v error=%q", app.integrations.web3ModalOpen, app.integrations.err)
	}
	if backend.sites[0].AcceptsDonation || backend.sites[0].JuiceboxProjectID != 0 {
		t.Fatalf("invalid settings changed the site: %+v", backend.sites[0])
	}
	app.integrations.web3JuiceboxID = "42"
	tt.Frame()
	if err := tt.Click("保存 Web3 设置"); err != nil {
		t.Fatalf("failed to save Web3 settings: %v", err)
	}
	if !backend.sites[0].AcceptsDonation || backend.sites[0].AcceptsDonationETHAddress != "vitalik.eth" || backend.sites[0].JuiceboxProjectID != 42 {
		t.Fatalf("expected Web3 settings saved, got %+v", backend.sites[0])
	}
}

type integrationServiceSave struct {
	siteID string
	kind   string
}

type integrationServiceTestBackend struct {
	*fakeBackend
	saves   []integrationServiceSave
	saveErr error
}

func (b *integrationServiceTestBackend) recordSave(siteID, kind string) error {
	b.saves = append(b.saves, integrationServiceSave{siteID: siteID, kind: kind})
	return b.saveErr
}

func (b *integrationServiceTestBackend) ConfigureService(siteID, kind, _, _ string) error {
	return b.recordSave(siteID, kind)
}

func (b *integrationServiceTestBackend) ConfigurePlausibleAt(siteID, _, _, _, _ string) error {
	return b.recordSave(siteID, "plausible")
}

func (b *integrationServiceTestBackend) ConfigurePinnable(siteID, _ string) error {
	return b.recordSave(siteID, "pinnable")
}

func TestNativeUIIntegrationServiceSheetsSaveAndReportErrors(t *testing.T) {
	saveErr := errors.New("test save failed")
	cases := []struct {
		kind     string
		saveKey  string
		savedKey string
		open     func(*integrationsModalState) *bool
	}{
		{"plausible", "int.srv.savePlausible", "int.srv.savedPlausible", func(s *integrationsModalState) *bool { return &s.plausibleModalOpen }},
		{"filebase", "int.srv.saveFilebase", "int.srv.savedFilebase", func(s *integrationsModalState) *bool { return &s.filebaseModalOpen }},
		{"pinnable", "int.srv.savePinnable", "int.srv.savedPinnable", func(s *integrationsModalState) *bool { return &s.pinnableModalOpen }},
		{"dwebservices", "int.srv.saveDweb", "int.srv.savedDweb", func(s *integrationsModalState) *bool { return &s.dwebModalOpen }},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			backend := &integrationServiceTestBackend{fakeBackend: &fakeBackend{}, saveErr: saveErr}
			app := New(backend)
			app.integrations.siteID = "s1"
			open := tc.open(&app.integrations)
			*open = true

			tt := ui.NewTester(app.viewIntegrationsSecondarySheets, 1280, 900)
			tt.Frame()
			if err := tt.Click(app.t(tc.saveKey)); err != nil {
				t.Fatal(err)
			}
			if !*open || app.integrations.err != saveErr.Error() || app.integrations.msg != "" {
				t.Fatalf("failed save left open=%v, error=%q, message=%q", *open, app.integrations.err, app.integrations.msg)
			}

			backend.saveErr = nil
			tt.Frame()
			if err := tt.Click(app.t(tc.saveKey)); err != nil {
				t.Fatal(err)
			}
			if *open || app.integrations.err != "" || app.integrations.msg != app.t(tc.savedKey) {
				t.Fatalf("successful save left open=%v, error=%q, message=%q", *open, app.integrations.err, app.integrations.msg)
			}
			want := integrationServiceSave{siteID: "s1", kind: tc.kind}
			if len(backend.saves) != 2 || backend.saves[0] != want || backend.saves[1] != want {
				t.Fatalf("expected both saves to target %+v, got %+v", want, backend.saves)
			}
		})
	}
}
