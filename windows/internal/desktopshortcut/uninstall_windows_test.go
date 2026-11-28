package desktopshortcut

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveDesktopShortcutPreservesForeignLinks(t *testing.T) {
	for _, test := range []struct {
		name          string
		foreignTarget bool
		foreignIcon   bool
	}{
		{name: "owned shortcut"},
		{name: "same name and managed icon but different executable", foreignTarget: true},
		{name: "same executable but user-created shortcut", foreignIcon: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			initShortcutTests(t)
			root := t.TempDir()
			path := filepath.Join(root, shortcutName)
			managedIcons := filepath.Join(root, "managed-icons")
			install := filepath.Join(root, "installed")
			executables := []string{filepath.Join(install, "Kinpage.exe"), filepath.Join(root, "AnotherApp.exe")}
			if err := os.Mkdir(install, 0700); err != nil {
				t.Fatal(err)
			}
			for _, exe := range executables {
				if err := os.WriteFile(exe, []byte("synthetic target; never launched"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			installedExe, linkedExe := executables[0], executables[0]
			if test.foreignTarget {
				linkedExe = executables[1]
			}
			iconDir := managedIcons
			if test.foreignIcon {
				iconDir = filepath.Join(root, "personal-icons")
			}
			changed, err := applyAt(path, iconDir, shortcutTarget{exe: linkedExe}, "tide", true)
			if err != nil || !changed {
				t.Fatalf("could not create fixture shortcut: changed=%v, err=%v", changed, err)
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := removeDesktopShortcutAt(path, managedIcons, installedExe); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(path)
			if test.foreignTarget || test.foreignIcon {
				if err != nil || !bytes.Equal(before, after) {
					t.Fatalf("foreign shortcut changed: %v", err)
				}
				return
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("owned shortcut still exists: %v", err)
			}
			if err := removeDesktopShortcutAt(path, managedIcons, installedExe); err != nil {
				t.Fatalf("a repeat cleanup should accept a missing shortcut: %v", err)
			}
		})
	}
}
