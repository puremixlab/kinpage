package desktopshortcut

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
	"windows/resources"
)

func TestDesktopShortcutSwitchesIconAndKeepsTarget(t *testing.T) {
	initShortcutTests(t)
	root := t.TempDir()
	path := filepath.Join(root, shortcutName)
	icons := filepath.Join(root, "icons")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	target := shortcutTarget{exe: exe}
	if changed, err := applyAt(path, icons, target, "tide", false); err != nil || changed {
		t.Fatalf("changing the icon with no shortcut: changed=%v, err=%v", changed, err)
	}
	for _, icon := range []resources.AppIcon{"tide", "orbit", resources.ClassicIcon} {
		if changed, err := applyAt(path, icons, target, icon, icon == "tide"); err != nil || !changed {
			t.Fatalf("applying %s: changed=%v, err=%v", icon, changed, err)
		}
		assertShortcut(t, path, target, icon)
	}

	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	other := shortcutTarget{exe: filepath.Join(root, "AnotherApp.exe")}
	if _, err := applyAt(path, icons, other, "dawn", true); !errors.Is(err, ErrOccupied) {
		t.Fatalf("expected an occupied shortcut name, got %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("existing shortcut was changed, err=%v", err)
	}
}

func TestDesktopShortcutSwitchesIconForPackagedApp(t *testing.T) {
	appID := os.Getenv("KINPAGE_TEST_APPS_FOLDER_ID")
	if appID == "" {
		t.Skip("set KINPAGE_TEST_APPS_FOLDER_ID to a registered app's ID")
	}
	initShortcutTests(t)
	root := t.TempDir()
	path := filepath.Join(root, shortcutName)
	icons := filepath.Join(root, "icons")
	target := shortcutTarget{appID: appID}
	for _, icon := range []resources.AppIcon{"dawn", "orbit"} {
		if changed, err := applyAt(path, icons, target, icon, icon == "dawn"); err != nil || !changed {
			t.Fatalf("applying %s: changed=%v, err=%v", icon, changed, err)
		}
		assertShortcut(t, path, target, icon)
	}
}

func TestDesktopShortcutMatchesShortExecutablePath(t *testing.T) {
	root := t.TempDir()
	name := filepath.Join(root, "Application with a long name.exe")
	if err := os.WriteFile(name, []byte("test executable"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	var buffer [32768]uint16
	n, err := windows.GetShortPathName(value, &buffer[0], uint32(len(buffer)))
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		t.Fatalf("resolve short alias: %v", err)
	}
	short := windows.UTF16ToString(buffer[:n])
	if filepath.Base(short) == filepath.Base(name) {
		t.Skip("the test volume does not create 8.3 aliases")
	}
	if !sameExecutablePath(short, name) || !sameExecutablePath(name, short) {
		t.Fatalf("%q and %q resolve to the same executable", short, name)
	}
}

func initShortcutTests(t *testing.T) {
	t.Helper()
	cleanup, err := initCOM()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
}

func assertShortcut(t *testing.T, path string, target shortcutTarget, icon resources.AppIcon) {
	t.Helper()
	link, persist, err := newShellLink()
	if err != nil {
		t.Fatal(err)
	}
	defer link.close()
	defer persist.close()
	if err := persist.load(path); err != nil {
		t.Fatal(err)
	}
	if !link.matches(target) {
		t.Fatalf("shortcut target changed: %+v", target)
	}
	var filename [32768]uint16
	var index int32
	hr, _, _ := syscall.SyscallN(link.vt.getIconLocation, uintptr(unsafe.Pointer(link)),
		uintptr(unsafe.Pointer(&filename[0])), uintptr(len(filename)), uintptr(unsafe.Pointer(&index)))
	if err := checkHRESULT("read shortcut icon path", hr); err != nil {
		t.Fatal(err)
	}
	if index != 0 {
		t.Fatalf("unexpected shortcut icon index %d", index)
	}
	iconPath := windows.UTF16ToString(filename[:])
	data, err := os.ReadFile(iconPath)
	if err != nil || !bytes.Equal(data, icon.ICO()) {
		t.Fatalf("shortcut icon at %q does not match %q, err=%v", iconPath, icon, err)
	}
}
