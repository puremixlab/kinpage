package nativeui

import (
	"image"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPetPreferencesRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nativeui-layout.json")
	if err := os.WriteFile(path, []byte(`{"locale":"en","sidebarWidth":260}`), 0o600); err != nil {
		t.Fatal(err)
	}
	app := New(&fakeBackend{})
	app.EnablePrefsPersistence(path)
	if app.desktopPetPrefs != (desktopPetPrefs{}) || app.DesktopPetEnabled() {
		t.Fatal("legacy preferences must start with the desktop pet disabled")
	}

	app.desktopPetPrefs.Enabled = true
	app.rememberDesktopPetPosition(image.Pt(-152, 0))
	restored := New(&fakeBackend{})
	restored.EnablePrefsPersistence(path)
	want := desktopPetPrefs{Enabled: true, X: -152, Y: 0, Positioned: true}
	if got := restored.desktopPetPrefs; got != want {
		t.Fatalf("restored desktop pet preferences = %+v, want %+v", got, want)
	}
	if restored.Locale() != LocaleEn || restored.sidebarWidth != 260 {
		t.Fatal("saving the desktop pet must preserve other layout preferences")
	}
}
