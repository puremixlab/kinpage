package preview

import (
	"strings"
	"testing"
)

func TestYouTubeAutoEmbed(t *testing.T) {
	html, err := Markdown("Intro\n\nhttps://www.youtube.com/watch?v=dQw4w9WgXcQ\n\nOutro")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Fatalf("expected youtube-nocookie embed iframe, got: %s", html)
	}

	shortHTML, err := Markdown("https://youtu.be/dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shortHTML, "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ") {
		t.Fatalf("expected youtu.be embed iframe, got: %s", shortHTML)
	}

	// Non-YouTube or invalid ID must NOT become an iframe.
	safeHTML, err := Markdown("https://www.youtube.com/watch?v=invalid!id\n\nhttps://example.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(safeHTML, "<iframe") {
		t.Fatalf("unexpected iframe for invalid/non-youtube URL: %s", safeHTML)
	}
}
