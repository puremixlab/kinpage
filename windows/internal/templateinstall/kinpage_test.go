package templateinstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"windows/internal/publisher"
	"windows/internal/store"
	"windows/internal/templatecatalog"
)

func TestKinpageTemplatesStagePublicSite(t *testing.T) {
	for _, item := range templatecatalog.Kinpage() {
		t.Run(item.ID, func(t *testing.T) {
			source, cleanup, err := ExtractBuiltin(item.ID)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(cleanup)
			created := time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC)
			site := store.Site{
				ID: "site", Name: "Harbor & Coast", Created: created,
				Tags: map[string]string{"unused": "Unused tag"},
			}
			duration := 5
			articles := []store.Article{
				{
					ID: "first", SiteID: site.ID, Title: "Tide & Maps", Slug: "tide-and-maps",
					Content: "A **careful** look at the harbor & the tide.\n\n![Port at dawn](harbor.webp)",
					Created: created.AddDate(0, -1, 0), Pinned: &created,
					Tags:          map[string]string{"research": "Research & Notes"},
					Attachments:   []string{"harbor.webp", "harbor.ogg", "walk notes.txt"},
					AudioFilename: "harbor.ogg", AudioDuration: &duration,
				},
				{
					ID: "resource", SiteID: site.ID, Title: "Harbor reference",
					Content: "A useful reference for the shoreline.", Created: created,
					ExternalLink: "https://example.org/maps?q=harbor&view=1",
					Attachments:  []string{"library.webp"},
				},
				{
					ID: "about-page", SiteID: site.ID, Title: "About & Contact",
					Content: "About this site.", Created: created, Slug: "about", ArticleType: 1,
					IsIncludedInNavigation: true, Tags: map[string]string{"private-topic": "Page-only topic"},
				},
			}
			for i := 1; i <= 11; i++ {
				articles = append(articles, store.Article{
					ID: fmt.Sprintf("entry-%d", i), SiteID: site.ID, Title: fmt.Sprintf("Entry %02d", i),
					Created: created.AddDate(0, 0, -i), Content: "More field notes.",
				})
			}

			root := t.TempDir()
			for _, article := range articles {
				if len(article.Attachments) == 0 {
					continue
				}
				dir := filepath.Join(root, "attachments", site.ID, article.ID)
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				for _, name := range article.Attachments {
					data := []byte("A map of the walk.")
					if name != "walk notes.txt" {
						data, err = previewMedia.ReadFile("previewmedia/" + name)
						if err != nil {
							t.Fatal(err)
						}
					}
					if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}

			public := t.TempDir()
			if err := publisher.StagePublicSite(public, site, articles, source, root); err != nil {
				t.Fatal(err)
			}
			read := func(name string) string {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(public, filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			assertContains := func(page string, markers ...string) {
				t.Helper()
				body := read(page)
				for _, marker := range markers {
					if !strings.Contains(body, marker) {
						t.Errorf("%s is missing %q", page, marker)
					}
				}
			}

			assertContains("index.html", "Harbor &amp; Coast", "Tide &amp; Maps",
				`href="./tide-and-maps/"`, `href="./about/"`, "About &amp; Contact",
				`href="./podcast.xml"`, `href="./page2.html"`, "Pinned")
			if item.ID != "folio" {
				assertContains("index.html", "A careful look at the harbor &amp; the tide.")
			}
			index := read("index.html")
			lead, latest := strings.Index(index, "Tide &amp; Maps"), strings.Index(index, "Harbor reference")
			if lead < 0 || latest < 0 || lead > latest || strings.Contains(index, "Entry 11") {
				t.Error("homepage lost its pinned order or includes an article from page two")
			}
			assertContains("page2.html", "Entry 11", `rel="prev" href="./"`)
			assertContains("archive.html", "January 2026", "December 2025", "Entry 11")
			assertContains("tags.html", "Research &amp; Notes", `href="./research.html"`)
			if strings.Contains(read("tags.html"), "unused.html") || strings.Contains(read("tags.html"), "private-topic.html") {
				t.Error("tag index contains a link without a corresponding tag page")
			}
			assertContains("research.html", "Research &amp; Notes", "Tide &amp; Maps")
			assertContains("about/index.html", "About this site.", `href="../assets/base.css"`)
			assertContains("tide-and-maps/index.html", "Tide &amp; Maps", "<strong>careful</strong>",
				`src="./harbor.ogg"`, `href="./walk%20notes.txt"`, `href="../research.html"`)
			if strings.Count(read("tide-and-maps/index.html"), "<img ") != 1 {
				t.Error("article repeats its inline cover image")
			}
			assertContains("resource/index.html", `src="./library.webp"`)
			assertContains("tide-and-maps/walk notes.txt", "A map of the walk.")
			if read("assets/base.css") == "" || read("assets/style.css") == "" {
				t.Error("public CSS is empty")
			}
			switch item.ID {
			case "folio":
				assertContains("index.html", `src="./first/harbor.webp"`)
			case "notebook":
				assertContains("index.html", "Browse by topic")
			case "airwaves":
				assertContains("index.html", `src="./first/harbor.ogg"`, "00:00:05")
			case "linkshelf":
				assertContains("index.html", `href="https://example.org/maps?q=harbor&amp;view=1"`, `class="destination">example.org`)
			}
		})
	}
}

func TestKinpageTemplateVerificationChecksBothLayers(t *testing.T) {
	root, cleanup, err := ExtractBuiltin("gazette")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	for _, name := range []string{"template.json", "templates/base.html", "templates/home.html", "assets/base.css", "assets/style.css"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(root, filepath.FromSlash(name))
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Error(err)
				}
			})
			lineEndingVariant := bytes.ReplaceAll(normalizeTemplateText(data), []byte("\n"), []byte("\r\n"))
			if err := os.WriteFile(path, lineEndingVariant, 0600); err != nil {
				t.Fatal(err)
			}
			if err := Verify(root); err != nil {
				t.Fatalf("rejected the CRLF version: %v", err)
			}
			if err := os.WriteFile(path, append(data, ' '), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Verify(root); err == nil {
				t.Fatal("accepted modified template files")
			}
		})
	}
}
