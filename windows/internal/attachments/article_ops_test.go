package attachments

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"windows/internal/store"
)

func TestMoveArticleCopiesAttachmentsAndCleansSource(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(filepath.Join(root, "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	siteA, err := lib.CreateSite("Site A", "")
	if err != nil {
		t.Fatal(err)
	}
	siteB, err := lib.CreateSite("Site B", "")
	if err != nil {
		t.Fatal(err)
	}
	art, err := lib.SaveArticle(siteA.ID, "", "With Photo", "![pic](photo.png)")
	if err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(root, "photo.png")
	if err := os.WriteFile(srcFile, []byte("fake-png-bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	art, err = Add(root, lib, siteA.ID, art.ID, srcFile)
	if err != nil {
		t.Fatal(err)
	}

	moved, err := MoveArticle(root, lib, siteA.ID, art.ID, siteB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.SiteID != siteB.ID {
		t.Fatalf("expected moved siteID %s, got %s", siteB.ID, moved.SiteID)
	}
	siteBObj, _, _ := findSite(lib, siteB.ID)
	f, err := OpenRegistered(root, siteBObj, moved, "photo.png")
	if err != nil {
		t.Fatalf("expected attachment in destination site: %v", err)
	}
	_ = f.Close()

	oldDir, err := ownedDir(root, siteA.ID, art.ID, false)
	if err == nil {
		if _, statErr := os.Stat(oldDir); !os.IsNotExist(statErr) {
			t.Fatalf("expected source attachment dir %s to be removed", oldDir)
		}
	}
}

func TestExportAndImportArticleZipRoundTrip(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(filepath.Join(root, "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	siteA, _ := lib.CreateSite("Site A", "")
	siteB, _ := lib.CreateSite("Site B", "")
	art, _ := lib.SaveArticle(siteA.ID, "", "Exportable", "Hello ![a](note.txt)")
	customDate := time.Date(2025, 1, 15, 8, 0, 0, 0, time.UTC)
	art, _ = lib.UpdateArticleDate(siteA.ID, art.ID, customDate)
	art, _ = lib.SetArticleStar(siteA.ID, art.ID, true, store.StarTypeHeart)
	noteFile := filepath.Join(root, "note.txt")
	if err := os.WriteFile(noteFile, []byte("attachment-data"), 0600); err != nil {
		t.Fatal(err)
	}
	art, err = Add(root, lib, siteA.ID, art.ID, noteFile)
	if err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(root, "exported.article.zip")
	if err := ExportArticleZip(root, lib, siteA.ID, art.ID, zipPath); err != nil {
		t.Fatal(err)
	}

	imported, err := ImportArticleZip(root, lib, siteB.ID, zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if imported.ID == art.ID {
		t.Fatal("expected fresh UUID on import")
	}
	if imported.Title != "Exportable" || !imported.Created.Equal(customDate) || !imported.Starred || imported.StarType != store.StarTypeHeart {
		t.Fatalf("imported metadata mismatch: %+v", imported)
	}
	if len(imported.Attachments) != 1 || imported.Attachments[0] != "note.txt" {
		t.Fatalf("expected note.txt attachment, got %v", imported.Attachments)
	}
	siteBObj, _, _ := findSite(lib, siteB.ID)
	f, err := OpenRegistered(root, siteBObj, imported, "note.txt")
	if err != nil {
		t.Fatalf("expected imported attachment readable: %v", err)
	}
	_ = f.Close()
}

func TestImportArticleZipRejectsZipSlipAndSupportsPlanetArticleDir(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(filepath.Join(root, "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, _ := lib.CreateSite("Site", "")

	// 1. Zip Slip archive must be rejected.
	evilZip := filepath.Join(root, "evil.article.zip")
	ef, err := os.Create(evilZip)
	if err != nil {
		t.Fatal(err)
	}
	ezw := zip.NewWriter(ef)
	w1, _ := ezw.Create("article.json")
	_, _ = w1.Write([]byte(`{"title":"Bad","content":"x"}`))
	w2, _ := ezw.Create("../escape.txt")
	_, _ = w2.Write([]byte("pwned"))
	_ = ezw.Close()
	_ = ef.Close()
	if _, err := ImportArticleZip(root, lib, site.ID, evilZip); err == nil {
		t.Fatal("expected Zip Slip archive to be rejected")
	}

	// 2. macOS <UUID>.planetarticle/article.json layout must succeed.
	macZip := filepath.Join(root, "mac.article.zip")
	mf, err := os.Create(macZip)
	if err != nil {
		t.Fatal(err)
	}
	mzw := zip.NewWriter(mf)
	mw1, _ := mzw.Create("POST-123.planetarticle/article.json")
	_, _ = mw1.Write([]byte(`{"title":"From Mac","content":"Hi","attachments":["pic.png"]}`))
	mw2, _ := mzw.Create("POST-123.planetarticle/pic.png")
	_, _ = mw2.Write([]byte("pngdata"))
	_ = mzw.Close()
	_ = mf.Close()

	imp, err := ImportArticleZip(root, lib, site.ID, macZip)
	if err != nil {
		t.Fatalf("expected macOS planetarticle layout to import cleanly: %v", err)
	}
	if imp.Title != "From Mac" || len(imp.Attachments) != 1 {
		t.Fatalf("unexpected imported article: %+v", imp)
	}
}
