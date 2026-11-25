package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMarkdownTodoParseAndToggle(t *testing.T) {
	md := "# Tasks\n- [ ] Write docs\n- [x] Ship code\n```\n- [ ] Inside fence\n```\n* [ ] Celebrate"
	items := ParseMarkdownTodos(md)
	if len(items) != 3 {
		t.Fatalf("expected 3 todos (ignoring fenced code), got %d: %+v", len(items), items)
	}
	if items[0].Checked || items[0].Text != "Write docs" || !items[1].Checked {
		t.Fatalf("unexpected parsed todos: %+v", items)
	}

	s, err := Open(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, _ := s.CreateSite("Blog", "")
	art, _ := s.SaveArticle(site.ID, "", "Checklist", md)

	updated, err := s.ToggleArticleTodo(site.ID, art.ID, items[0].LineIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.Content, "- [x] Write docs") {
		t.Fatalf("expected toggled checkbox, got:\n%s", updated.Content)
	}
}
