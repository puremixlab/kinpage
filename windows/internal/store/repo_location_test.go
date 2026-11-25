package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateAndResolveDataDir(t *testing.T) {
	defaultDir := t.TempDir()
	targetDir := t.TempDir()

	s, err := Open(defaultDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSite("Portable Blog", "Moves cleanly"); err != nil {
		t.Fatal(err)
	}

	// Non-empty target must be rejected.
	nonEmpty := t.TempDir()
	_ = os.WriteFile(filepath.Join(nonEmpty, "existing.txt"), []byte("x"), 0600)
	if err := MigrateDataDir(defaultDir, defaultDir, nonEmpty); err == nil {
		t.Fatal("expected error migrating to non-empty directory")
	}

	// Nested directory must be rejected.
	nested := filepath.Join(defaultDir, "sub")
	_ = os.MkdirAll(nested, 0700)
	if err := MigrateDataDir(defaultDir, defaultDir, nested); err == nil {
		t.Fatal("expected error migrating into nested directory")
	}

	// Clean migration succeeds and ResolveDataDir points to targetDir.
	if err := MigrateDataDir(defaultDir, defaultDir, targetDir); err != nil {
		t.Fatal(err)
	}
	if got := ResolveDataDir(defaultDir); got != targetDir {
		t.Fatalf("expected ResolveDataDir=%q, got %q", targetDir, got)
	}
	migrated, err := Open(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	sites, err := migrated.ListSites()
	if err != nil || len(sites) != 1 || sites[0].Name != "Portable Blog" {
		t.Fatalf("expected migrated site in targetDir, got %+v err=%v", sites, err)
	}
}
