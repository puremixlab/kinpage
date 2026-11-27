package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"windows/internal/applog"
	"windows/internal/ipfs"
	"windows/internal/localapi"
)

func TestNativeUIToolsModalBatch3C(t *testing.T) {
	backend := &fakeBackend{
		apiStatus: localapi.Status{Port: 8086},
		ipnsKeys:  []ipfs.KeyInfo{{Name: "self", ID: "k51-self"}},
		appLogs:   []applog.Entry{{Time: "10:00:00", Category: "Kinpage", Message: "启动完成"}},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 900)
	tt.Frame()

	// 1. Open AI Assistant from sidebar Sparkles button -> empty state -> secondary modal -> probe, save, chat.
	if err := tt.Click("AI 助手"); err != nil {
		t.Fatalf("failed to open AI modal: %v", err)
	}
	if !tt.HasText("尚未配置本地或兼容 OpenAI 的 AI 服务") {
		t.Fatalf("expected empty state for unconfigured AI")
	}
	if err := tt.Click("添加"); err != nil {
		t.Fatalf("failed to open AI secondary config modal: %v", err)
	}
	if err := tt.Click("探测本地服务与模型"); err != nil {
		t.Fatalf("failed to probe AI: %v", err)
	}
	if app.toolsModal.aiModel != "qwen2.5" {
		t.Fatalf("expected auto-selected model qwen2.5, got %q", app.toolsModal.aiModel)
	}
	if err := tt.Click("保存 AI 配置"); err != nil {
		t.Fatalf("failed to save AI config: %v", err)
	}
	if app.toolsModal.aiModalOpen {
		t.Fatal("expected AI secondary modal closed after save")
	}
	if !app.toolsModal.open {
		t.Fatal("expected underlying global settings window to stay open")
	}
	app.toolsModal.aiPrompt = "帮我总结这段文字"
	tt.Frame()
	if err := tt.Click("发送"); err != nil {
		t.Fatalf("failed to send AI prompt: %v", err)
	}
	if len(app.toolsModal.aiHistory) != 2 {
		t.Fatalf("expected 2 AI messages, got %d", len(app.toolsModal.aiHistory))
	}

	// 2. Local API tab (empty state -> secondary modal -> save -> configured card).
	if err := tt.Click("本地 API"); err != nil {
		t.Fatalf("failed to switch to Local API tab: %v", err)
	}
	if err := tt.Click("配置"); err != nil {
		t.Fatalf("failed to open Local API secondary modal: %v", err)
	}
	app.toolsModal.apiEnabled = true
	app.toolsModal.apiPort = "9090"
	tt.Frame()
	if err := tt.Click("保存 API 配置"); err != nil {
		t.Fatalf("failed to save Local API config: %v", err)
	}
	if !backend.apiStatus.Running || backend.apiStatus.Port != 9090 {
		t.Fatalf("expected Local API running on 9090, got %+v", backend.apiStatus)
	}

	// 3. IPNS Keys tab (existing key list -> open import secondary modal -> import).
	if err := tt.Click("密钥管理"); err != nil {
		t.Fatalf("failed to switch to Keys tab: %v", err)
	}
	if err := tt.Click("导入密钥"); err != nil {
		t.Fatalf("failed to open import key secondary modal: %v", err)
	}
	app.toolsModal.importKeyName = "backup"
	tt.Frame()
	if err := tt.Click("导入密钥"); err != nil {
		t.Fatalf("failed to submit import key modal: %v", err)
	}
	if len(backend.ipnsKeys) != 2 {
		t.Fatalf("expected 2 IPNS keys, got %d", len(backend.ipnsKeys))
	}

	// 4. App Logs tab.
	if err := tt.Click("应用日志"); err != nil {
		t.Fatalf("failed to switch to Logs tab: %v", err)
	}
	if len(app.toolsModal.logs) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(app.toolsModal.logs))
	}
	if err := tt.Click("清空日志"); err != nil {
		t.Fatalf("failed to clear logs: %v", err)
	}
	if len(backend.appLogs) != 0 {
		t.Fatalf("expected logs cleared, got %d", len(backend.appLogs))
	}

	// 5. Sidebar search filters navigation items.
	app.toolsModal.search = "密钥"
	tt.Frame()
	if !tt.HasText("密钥管理") {
		t.Fatal("expected 密钥管理 visible when searching for 密钥")
	}
	if tt.HasText("离线存储") {
		t.Fatal("expected 离线存储 filtered out when searching for 密钥")
	}
}
