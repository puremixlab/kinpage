package sitegen

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"windows/internal/store"
)

type templateAttachment struct {
	Name string
	Path string
}

func addKinpageArticleFields(row map[string]any, article store.Article, rendered, hero string) {
	tokenizer := html.NewTokenizer(strings.NewReader(rendered))
	var text strings.Builder
	hasInlineHero := false
	for kind := tokenizer.Next(); kind != html.ErrorToken; kind = tokenizer.Next() {
		if kind == html.TextToken {
			text.Write(tokenizer.Text())
			continue
		}
		token := tokenizer.Token()
		// Keep inline formatting together and separate adjacent blocks.
		switch token.Data {
		case "p", "li", "div", "br", "h1", "h2", "h3", "h4", "h5", "h6", "td", "th", "pre", "blockquote":
			text.WriteByte(' ')
		case "img":
			for _, attr := range token.Attr {
				if attr.Key != "src" {
					continue
				}
				src, err := url.Parse(attr.Val)
				if err == nil && hero != "" && src.Scheme == "" && src.Host == "" && strings.EqualFold(strings.TrimPrefix(src.Path, "./"), hero) {
					hasInlineHero = true
				}
			}
		}
	}
	const excerptLimit = 200
	excerpt := []rune(strings.Join(strings.Fields(text.String()), " "))
	if len(excerpt) > excerptLimit {
		excerpt = append(excerpt[:excerptLimit-1], '…')
	}
	row["excerpt"] = string(excerpt)
	if hero != "" && !hasInlineHero {
		row["articleCover"] = row["heroImage"]
	}
	var files []templateAttachment
	for _, name := range article.Attachments {
		if name != hero && name != article.AudioFilename && name != article.VideoFilename {
			files = append(files, templateAttachment{Name: name, Path: url.PathEscape(name)})
		}
	}
	row["files"] = files
	if article.ExternalLink != "" {
		if target, err := url.Parse(article.ExternalLink); err == nil {
			row["linkHost"] = target.Hostname()
		}
	}
}
