package sitegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"windows/internal/store"
)

func TestRenderWritesFarcasterManifest(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	site := store.Site{
		ID:                    "s1",
		Name:                  "Mini App Site",
		Created:               now,
		FarcasterManifestJSON: `{"accountAssociation":{"header":"abc"}}`,
	}
	if err := Render(dir, site, nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ".well-known", "farcaster.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"accountAssociation"`) {
		t.Fatalf("unexpected farcaster.json: %s", string(raw))
	}
}
