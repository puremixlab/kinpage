package nativeui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"windows/resources"
)

func TestAppIconSettingsPersistSelectionAndSyncShortcut(t *testing.T) {
	prefs := filepath.Join(t.TempDir(), "nativeui-layout.json")
	if err := os.WriteFile(prefs, []byte(`{"locale":"en","appIcon":"unknown"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	app := New(&fakeBackend{})
	app.EnablePrefsPersistence(prefs)
	if app.AppIcon() != resources.ClassicIcon {
		t.Fatalf("unknown saved icon resolved to %q", app.AppIcon())
	}
	var applied []resources.AppIcon
	type shortcutCall struct {
		icon   resources.AppIcon
		create bool
	}
	var shortcuts []shortcutCall
	app.SetAppIconHandlers(func(icon resources.AppIcon) error {
		applied = append(applied, icon)
		return nil
	}, func(icon resources.AppIcon, create bool) (bool, error) {
		shortcuts = append(shortcuts, shortcutCall{icon, create})
		return true, nil
	})
	tt := ui.NewTester(app.View, 860, 600)
	tt.Frame()
	for _, label := range []string{"Settings", "More Settings…"} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	app.toolsModal.search = "desktop icon"
	tt.Frame()
	if err := tt.Click("App Icon (General · General)"); err != nil {
		t.Fatalf("open app icon setting from search: %v", err)
	}
	if app.toolsModal.tab != toolsTabGeneral || app.toolsModal.highlightID != "global-app-icon" {
		t.Fatalf("app icon search did not locate the setting: tab=%d, highlight=%q", app.toolsModal.tab, app.toolsModal.highlightID)
	}
	for _, label := range []string{"Tide", "Create or Update Desktop Shortcut"} {
		if err := tt.Click(label); err != nil {
			t.Fatalf("click %q: %v", label, err)
		}
	}
	if !reflect.DeepEqual(applied, []resources.AppIcon{"tide"}) {
		t.Fatalf("unexpected application icon changes: %v", applied)
	}
	if !reflect.DeepEqual(shortcuts, []shortcutCall{{"tide", false}, {"tide", true}}) {
		t.Fatalf("unexpected desktop shortcut updates: %v", shortcuts)
	}
	restarted := New(&fakeBackend{})
	restarted.EnablePrefsPersistence(prefs)
	if got := restarted.AppIcon(); got != "tide" {
		t.Fatalf("restarted with icon %q, want tide", got)
	}
}

func TestAppIconApplyFailureKeepsPreviousSelection(t *testing.T) {
	prefs := filepath.Join(t.TempDir(), "nativeui-layout.json")
	app := New(&fakeBackend{})
	app.EnablePrefsPersistence(prefs)
	app.SetAppIconHandlers(func(icon resources.AppIcon) error {
		if icon == "tide" {
			return errors.New("icon unavailable")
		}
		return nil
	}, func(resources.AppIcon, bool) (bool, error) {
		t.Error("should not change desktop shortcut after an icon failure")
		return false, nil
	})
	app.changeAppIcon("tide")
	if got := app.AppIcon(); got != resources.ClassicIcon || !app.statusIsErr {
		t.Fatalf("failed switch selected %q, error visible: %v", got, app.statusIsErr)
	}
}

func TestAppIconSaveFailureRestoresPreviousIcon(t *testing.T) {
	app := New(&fakeBackend{})
	app.EnablePrefsPersistence(t.TempDir()) // A directory cannot be replaced with the preferences file.
	var applied []resources.AppIcon
	app.SetAppIconHandlers(func(icon resources.AppIcon) error {
		applied = append(applied, icon)
		return nil
	}, func(resources.AppIcon, bool) (bool, error) {
		t.Error("should not change desktop shortcut after a preferences failure")
		return false, nil
	})
	app.changeAppIcon("dawn")
	if app.AppIcon() != resources.ClassicIcon || !app.statusIsErr ||
		!reflect.DeepEqual(applied, []resources.AppIcon{"dawn", resources.ClassicIcon}) {
		t.Fatalf("failed save left icon %q, visible error: %v, application calls: %v", app.AppIcon(), app.statusIsErr, applied)
	}
}
