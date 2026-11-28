package store_test

import (
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"

	"windows/internal/attachments"
	"windows/internal/localapi"
	"windows/internal/store"
)

func TestMigrateDataDirPreservesAttachmentsAndLocalAPIConfig(t *testing.T) {
	source, target := t.TempDir(), t.TempDir()
	library, err := store.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	site, err := library.CreateSite("Portable Blog", "")
	if err != nil {
		t.Fatal(err)
	}
	article, err := library.SaveArticle(site.ID, "", "Published entry", "")
	if err != nil {
		t.Fatal(err)
	}
	draft, err := library.SaveDraft(site.ID, "", "", "Private draft", "")
	if err != nil {
		t.Fatal(err)
	}

	const articleName, articleText = "article.txt", "published attachment"
	const draftName, draftText = "draft.txt", "private draft attachment"
	if _, err := attachments.AddBytes(source, library, site.ID, article.ID, articleName, base64.StdEncoding.EncodeToString([]byte(articleText))); err != nil {
		t.Fatal(err)
	}
	if _, err := attachments.AddDraftBytes(source, library, site.ID, draft.ID, draftName, base64.StdEncoding.EncodeToString([]byte(draftText))); err != nil {
		t.Fatal(err)
	}
	wantConfig, err := localapi.SaveConfig(source, true, 18086, false, "Writer", "")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.ScheduleDataDirMigration(source, source, target); err != nil {
		t.Fatal(err)
	}
	if active := store.ResolveDataDir(source); active != source {
		t.Fatalf("scheduling switched the active repository to %q", active)
	}
	if _, err := library.SaveDraft(site.ID, draft.ID, "", "Saved after scheduling", ""); err != nil {
		t.Fatal(err)
	}
	// Simulate a target that became unavailable after being picked in settings.
	blocker := filepath.Join(target, "unrelated.txt")
	if err := os.WriteFile(blocker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if active, err := store.ApplyScheduledDataDirMigration(source); err == nil || active != source {
		t.Fatalf("failed migration did not keep the active repository: %q, %v", active, err)
	}
	if data, err := os.ReadFile(blocker); err != nil || string(data) != "keep" {
		t.Fatalf("migration changed the unrelated file: %q, %v", data, err)
	}
	if err := os.Remove(blocker); err != nil {
		t.Fatal(err)
	}
	if active, err := store.ApplyScheduledDataDirMigration(source); err != nil || active != target {
		t.Fatalf("migration retry = %q, %v; want %q", active, err, target)
	}
	migratedRoot := store.ResolveDataDir(source)
	if migratedRoot != target {
		t.Fatalf("repository path after migration = %q, want %q", migratedRoot, target)
	}
	migrated, err := store.Open(migratedRoot)
	if err != nil {
		t.Fatal(err)
	}

	assertAttachment := func(t *testing.T, file *os.File, err error, want string) {
		t.Helper()
		if err != nil {
			t.Fatalf("open migrated attachment: %v", err)
		}
		defer file.Close()
		got, err := io.ReadAll(file)
		if err != nil || string(got) != want {
			t.Fatalf("read migrated attachment = %q, %v; want %q", got, err, want)
		}
	}
	t.Run("article attachment", func(t *testing.T) {
		articles, err := migrated.ListArticles(site.ID)
		if err != nil || len(articles) != 1 {
			t.Fatalf("list migrated articles = %v, %v", articles, err)
		}
		file, err := attachments.OpenRegistered(migratedRoot, site, articles[0], articleName)
		assertAttachment(t, file, err, articleText)
	})
	t.Run("draft attachment", func(t *testing.T) {
		drafts, err := migrated.ListDrafts(site.ID)
		if err != nil || len(drafts) != 1 {
			t.Fatalf("list migrated drafts = %v, %v", drafts, err)
		}
		if drafts[0].Title != "Saved after scheduling" {
			t.Fatalf("migration missed the last saved draft: %+v", drafts[0])
		}
		file, err := attachments.OpenDraftAttachment(migratedRoot, drafts[0], draftName)
		assertAttachment(t, file, err, draftText)
	})
	t.Run("local API configuration", func(t *testing.T) {
		got, err := localapi.LoadConfig(migratedRoot)
		if err != nil || got != wantConfig {
			t.Fatalf("load migrated local API configuration = %+v, %v; want %+v", got, err, wantConfig)
		}
	})
	if active, err := store.ApplyScheduledDataDirMigration(source); err != nil || active != target {
		t.Fatalf("subsequent startup tried to migrate again: %q, %v", active, err)
	}
}

func TestScheduledMigrationWaitsForUnavailableSource(t *testing.T) {
	defaultDir, root, target := t.TempDir(), t.TempDir(), t.TempDir()
	source, moved := filepath.Join(root, "source"), filepath.Join(root, "offline")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.ScheduleDataDirMigration(defaultDir, source, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(source, moved); err != nil {
		t.Fatal(err)
	}
	if active, err := store.ApplyScheduledDataDirMigration(defaultDir); err == nil || active != "" {
		t.Fatalf("missing source should prevent fallback to stale data: %q, %v", active, err)
	}
	if err := os.Rename(moved, source); err != nil {
		t.Fatal(err)
	}
	if active, err := store.ApplyScheduledDataDirMigration(defaultDir); err != nil || active != target {
		t.Fatalf("reconnected source: active=%q, err=%v", active, err)
	}
}
