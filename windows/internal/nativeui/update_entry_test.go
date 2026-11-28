package nativeui

import (
	"context"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/updates"
)

func TestGeneralSettingsOffersCheckForUpdates(t *testing.T) {
	app := New(&fakeBackend{})
	app.SetLocale(LocaleZhCN)
	tester := ui.NewTester(app.View, 1360, 860)
	tester.Frame()

	for _, label := range []string{"设置", "更多设置…"} {
		if err := tester.Click(label); err != nil {
			t.Fatalf("open General settings using %q: %v", label, err)
		}
	}
	if !app.toolsModal.open || app.toolsModal.tab != toolsTabGeneral {
		t.Fatal("expected General settings to be open")
	}
	if err := tester.Click("检查更新"); err != nil {
		t.Fatalf("General settings must offer a Check for Updates action: %v", err)
	}
	if !app.updateSettings.configOpen {
		t.Fatal("unconfigured update check should open the trust settings")
	}
}

type startupUpdateBackend struct {
	*fakeBackend
	auto    bool
	checked chan struct{}
	release chan struct{}
}

func (b startupUpdateBackend) UpdateSettings() (updates.SettingsInfo, error) {
	return updates.SettingsInfo{
		Configured: true, CurrentVersion: "0.1.0",
		Settings: updates.Config{FeedURL: "https://updates.example.test/manifest.json", AutoCheck: b.auto},
	}, nil
}

func (b startupUpdateBackend) CheckUpdatesContext(ctx context.Context) (updates.CheckInfo, error) {
	close(b.checked)
	select {
	case <-b.release:
		return updates.CheckInfo{Version: "0.1.1", Available: true}, nil
	case <-ctx.Done():
		return updates.CheckInfo{}, ctx.Err()
	}
}

func TestStartupUpdateCheckHonorsOptInWithoutBlockingFrame(t *testing.T) {
	for _, auto := range []bool{false, true} {
		backend := startupUpdateBackend{fakeBackend: &fakeBackend{}, auto: auto, checked: make(chan struct{}), release: make(chan struct{})}
		app := New(backend)
		tester := ui.NewTester(app.View, 1280, 820)
		complete := monitorSubscriptionTasks(t, app, tester)
		app.CheckUpdatesOnStartup()
		tester.Frame()
		if !auto {
			if app.updateSettings.operation != nil {
				t.Fatal("disabled startup check created a task")
			}
			continue
		}
		select {
		case <-backend.checked:
		case <-time.After(time.Second):
			t.Fatal("enabled startup check did not reach the backend")
		}
		if app.updateSettings.operation == nil {
			t.Fatal("startup check lost its pending state")
		}
		close(backend.release)
		complete()
		if check := app.updateSettings.check; check == nil || !check.Available || check.Version != "0.1.1" || app.updateSettings.downloaded {
			t.Fatalf("unexpected update state after startup check: %+v", app.updateSettings)
		}
	}
}
