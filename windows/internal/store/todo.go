package store

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

// TodoItem represents a Markdown GFM task list item inside an article.
type TodoItem struct {
	LineIndex int    `json:"lineIndex"`
	Checked   bool   `json:"checked"`
	Text      string `json:"text"`
}

var markdownTodoLineRe = regexp.MustCompile(`^(\s*[-*+]\s+\[)([ xX])(\]\s+)(.*)$`)

// ParseMarkdownTodos extracts GFM task list items outside fenced code blocks.
func ParseMarkdownTodos(md string) []TodoItem {
	lines := strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n")
	var out []TodoItem
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		m := markdownTodoLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, TodoItem{
			LineIndex: i,
			Checked:   m[2] == "x" || m[2] == "X",
			Text:      strings.TrimSpace(m[4]),
		})
	}
	return out
}

// ToggleMarkdownTodo flips the checkbox on lineIndex and returns the updated
// Markdown string.
func ToggleMarkdownTodo(md string, lineIndex int) (string, bool) {
	useCRLF := strings.Contains(md, "\r\n")
	normalized := strings.ReplaceAll(md, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if lineIndex < 0 || lineIndex >= len(lines) {
		return md, false
	}
	m := markdownTodoLineRe.FindStringSubmatch(lines[lineIndex])
	if m == nil {
		return md, false
	}
	nextMark := "x"
	if m[2] == "x" || m[2] == "X" {
		nextMark = " "
	}
	lines[lineIndex] = m[1] + nextMark + m[3] + m[4]
	joined := strings.Join(lines, "\n")
	if useCRLF {
		joined = strings.ReplaceAll(joined, "\n", "\r\n")
	}
	return joined, true
}

// ToggleArticleTodo toggles a GFM task checkbox inside a saved local article.
func (s *Store) ToggleArticleTodo(siteID, articleID string, lineIndex int) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	next := s.data
	next.Articles = append([]Article(nil), s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
		updated, ok := ToggleMarkdownTodo(article.Content, lineIndex)
		if !ok {
			return Article{}, errors.New("待办项不存在")
		}
		article.Content = updated
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, siteID, true)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}
