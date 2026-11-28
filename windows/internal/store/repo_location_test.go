package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"windows/internal/store"
)

func assertSameDataDir(t *testing.T, got, want string) {
	t.Helper()
	gotInfo, gotErr := os.Stat(got)
	wantInfo, wantErr := os.Stat(want)
	// Windows can spell the same directory with either long or 8.3 names.
	if gotErr != nil || wantErr != nil || !gotInfo.IsDir() || !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("repository = %q (%v), want %q (%v)", got, gotErr, want, wantErr)
	}
}

func TestMigrateAndResolveDataDir(t *testing.T) {
	defaultDir := t.TempDir()
	targetDir := t.TempDir()

	s, err := store.Open(defaultDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateSite("Portable Blog", "Moves cleanly"); err != nil {
		t.Fatal(err)
	}

	// Non-empty target must be rejected.
	nonEmpty := t.TempDir()
	_ = os.WriteFile(filepath.Join(nonEmpty, "existing.txt"), []byte("x"), 0600)
	if err := store.MigrateDataDir(defaultDir, defaultDir, nonEmpty); err == nil {
		t.Fatal("expected error migrating to non-empty directory")
	}

	// Nested directory must be rejected.
	nested := filepath.Join(defaultDir, "sub")
	_ = os.MkdirAll(nested, 0700)
	if err := store.MigrateDataDir(defaultDir, defaultDir, nested); err == nil {
		t.Fatal("expected error migrating into nested directory")
	}

	// Clean migration succeeds and ResolveDataDir points to targetDir.
	if err := store.MigrateDataDir(defaultDir, defaultDir, targetDir); err != nil {
		t.Fatal(err)
	}
	assertSameDataDir(t, store.ResolveDataDir(defaultDir), targetDir)
	migrated, err := store.Open(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	sites, err := migrated.ListSites()
	if err != nil || len(sites) != 1 || sites[0].Name != "Portable Blog" {
		t.Fatalf("expected migrated site in targetDir, got %+v err=%v", sites, err)
	}
}
