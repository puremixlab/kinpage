package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPreventSleepPreferenceAndExternalReload(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.PreventSleep() {
		t.Fatal("expected PreventSleep default false")
	}
	if err := s.SetPreventSleep(true); err != nil {
		t.Fatal(err)
	}
	if !s.PreventSleep() {
		t.Fatal("expected PreventSleep true")
	}

	// Internal write should NOT trigger external reload.
	if changed, err := s.ReloadIfExternallyModified(); err != nil || changed {
		t.Fatalf("expected no external change after own write, got changed=%v err=%v", changed, err)
	}

	// Simulate an external tool modifying library.json.
	ext, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ext.CreateSite("External Site", "Added outside Planet"); err != nil {
		t.Fatal(err)
	}
	// Ensure mtime advances even on coarse filesystems.
	libPath := filepath.Join(dir, "library.json")
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(libPath, future, future)

	changed, err := s.ReloadIfExternallyModified()
	if err != nil || !changed {
		t.Fatalf("expected external change to be detected, got changed=%v err=%v", changed, err)
	}
	sites, err := s.ListSites()
	if err != nil || len(sites) != 1 || sites[0].Name != "External Site" {
		t.Fatalf("expected reloaded external site, got %+v err=%v", sites, err)
	}
}
