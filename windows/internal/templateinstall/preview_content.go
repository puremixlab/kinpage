package templateinstall

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"

	"windows/internal/store"
)

//go:embed previewmedia/*.webp previewmedia/*.ogg
var previewMedia embed.FS

func kinpagePreviewContent(id string, site store.Site) (store.Site, []store.Article) {
	profiles := map[string]struct{ name, about string }{
		"gazette":   {"Low Tide Gazette", "Stories from the waterfront and the streets just beyond it."},
		"folio":     {"Tide & Timber", "A collection of photographs and small design studies from the coast."},
		"notebook":  {"Harbor Notes", "An open notebook on coastal places, public spaces, and field recording."},
		"dispatch":  {"The Saturday Letter", "A weekend letter about paying attention to the places we live."},
		"revision":  {"Atlas Changelog", "Release notes for a field notebook that works offline."},
		"airwaves":  {"Low Tide Radio", "Field recordings and short essays from the edge of the water."},
		"waypoint":  {"Along the Coast", "A walking journal from the ferry landing to the headland."},
		"linkshelf": {"The Coast Index", "Annotated links for planning walks and getting to know the shoreline."},
	}
	profile := profiles[id]
	site.Name, site.About = profile.name, profile.about
	site.AuthorName = "Alex Rivera"

	posts := []store.Article{
		{
			Title:       "The hour before the ferries",
			Content:     "Before the first crossing, the harbor belongs to deck crews and gulls. I took the long way down to the pier and stayed to watch the light reach the far shore.",
			Attachments: []string{"harbor.webp"},
			Tags:        map[string]string{"field-notes": "Field notes", "photography": "Photography"},
		},
		{
			Title:       "A library in the rain",
			Content:     "A single lit window changes the whole approach to a building. On a wet afternoon, the reading room was visible from the path above the water.",
			Attachments: []string{"library.webp"},
			Tags:        map[string]string{"places": "Places", "photography": "Photography"},
		},
		{
			Title:   "A map drawn from memory",
			Content: "Drawing the route back from the pier revealed which parts of the walk I actually remembered: a steep stair, the old seawall, and a stand of wind-bent trees.",
			Tags:    map[string]string{"field-notes": "Field notes"},
		},
		{
			Title:   "What the tide leaves behind",
			Content: "After the water recedes, the beach becomes a record of the night. Each visit turns up a different line of shells, glass, and driftwood.",
			Tags:    map[string]string{"field-notes": "Field notes"},
		},
	}
	switch id {
	case "airwaves":
		duration := 5
		posts[0].Attachments = append(posts[0].Attachments, "harbor.ogg")
		posts[0].AudioFilename, posts[0].AudioDuration = "harbor.ogg", &duration
	case "revision":
		posts = []store.Article{
			{Title: "1.4 · Offline maps", Content: "Save a route before leaving home and open the area map without a connection.", Tags: map[string]string{"feature": "Feature"}},
			{Title: "1.3 · Faster notes", Content: "Start a note from any saved waypoint and pick up where you left off on your next visit.", Tags: map[string]string{"improvement": "Improvement"}},
			{Title: "1.2 · Stable links", Content: "Fixed shared links to long notes with repeated headings.", Tags: map[string]string{"fix": "Fix"}},
		}
	case "linkshelf":
		posts = []store.Article{
			{Title: "Harbor tide tables", Content: "Check the next low tide before following a route that crosses the beach.", ExternalLink: "https://example.org/tide-tables", Tags: map[string]string{"planning": "Planning"}},
			{Title: "Shoreline map archive", Content: "Compare older maps with the current access paths to see where the shoreline has moved.", ExternalLink: "https://example.net/shoreline-maps", Tags: map[string]string{"maps": "Maps"}},
			{Title: "A note on the headland loop", Content: "The inland path is an easier return if the wind picks up. Bring a printed copy of the route.", Tags: map[string]string{"planning": "Planning"}},
		}
	}
	posts = append(posts, store.Article{
		Title: "About", Slug: "about", ArticleType: 1, IsIncludedInNavigation: true,
		Content: site.About,
	})
	for i := range posts {
		posts[i].ID = fmt.Sprintf("00000000-0000-4000-8000-%012d", i+2)
		posts[i].SiteID = site.ID
		posts[i].Created = site.Created.AddDate(0, 0, -3*i)
		posts[i].Modified = site.Created
	}
	return site, posts
}

func embedPreviewMedia(page string, posts []store.Article) (string, error) {
	for _, post := range posts {
		for _, name := range post.Attachments {
			src := `src="./` + post.ID + "/" + name + `"`
			if !strings.Contains(page, src) {
				continue
			}
			b, err := previewMedia.ReadFile("previewmedia/" + name)
			if err != nil {
				return "", err
			}
			mediaType := "image/webp"
			if name == post.AudioFilename {
				mediaType = "audio/ogg"
			}
			data := `src="data:` + mediaType + ";base64," + base64.StdEncoding.EncodeToString(b) + `"`
			page = strings.ReplaceAll(page, src, data)
		}
	}
	return page, nil
}
