package sitegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"windows/internal/store"
)

func TestRenderIndexOnlyUpdatesHomepageWithoutRewritingPosts(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	site := store.Site{ID: "s1", Name: "Quick Site", About: "Updated About", Created: now}
	articles := []store.Article{
		{ID: "POST-1", SiteID: "s1", Title: "Post One", Content: "Body", Created: now, Modified: now},
	}
	if err := RenderIndexOnly(dir, site, articles, ""); err != nil {
		t.Fatal(err)
	}
	idx, err := os.ReadFile(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(idx), "Quick Site") || !strings.Contains(string(idx), "Post One") {
		t.Fatalf("expected rendered index.html to contain site & post, got %s", string(idx))
	}
	if _, err := os.Stat(filepath.Join(dir, "POST-1", "index.html")); !os.IsNotExist(err) {
		t.Fatal("RenderIndexOnly should not create individual post directories")
	}
}
