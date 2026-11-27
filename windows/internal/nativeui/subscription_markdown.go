package nativeui

import (
	"strings"

	"golang.org/x/net/html"
	"windows/internal/feeds"
)

// htmlToReadableMarkdown converts simple RSS/Atom HTML fragments into clean
// Markdown suitable for native ui/markdown.View rendering without a WebView.
func htmlToReadableMarkdown(raw string) string {
	return htmlToReadableMarkdownWithBase(raw, "")
}

func formatMarkdownImage(alt, rawURL string) string {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return ""
	}
	cleanAlt := strings.Join(strings.Fields(alt), " ")
	cleanAlt = strings.ReplaceAll(cleanAlt, "]", "\\]")
	if strings.ContainsAny(u, " \t\r\n()") {
		return "![" + cleanAlt + "](<" + u + ">)"
	}
	return "![" + cleanAlt + "](" + u + ")"
}

func firstSrcsetURL(srcset string) string {
	for _, entry := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(entry))
		if len(fields) > 0 && fields[0] != "" {
			return fields[0]
		}
	}
	return ""
}

func nodeContainsImage(n *html.Node) bool {
	if n == nil {
		return false
	}
	if n.Type == html.ElementNode {
		tag := strings.ToLower(n.Data)
		if tag == "img" || tag == "picture" {
			return true
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if nodeContainsImage(ch) {
			return true
		}
	}
	return false
}

func htmlToReadableMarkdownWithBase(raw, baseURL string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.Contains(trimmed, "<") {
		return trimmed
	}
	doc, err := html.Parse(strings.NewReader(trimmed))
	if err != nil {
		return trimmed
	}
	var out strings.Builder
	extractImgURL := func(n *html.Node, pictureFallback string) (string, string) {
		src := ""
		lazySrc := ""
		srcset := ""
		alt := ""
		for _, attr := range n.Attr {
			key := strings.ToLower(attr.Key)
			val := strings.TrimSpace(attr.Val)
			if val == "" {
				continue
			}
			switch key {
			case "src":
				if !strings.HasPrefix(strings.ToLower(val), "data:image/svg+xml") {
					src = val
				}
			case "data-src", "data-original", "data-lazy-src", "data-actualsrc", "data-url":
				if lazySrc == "" {
					lazySrc = val
				}
			case "srcset", "data-srcset":
				if srcset == "" {
					srcset = val
				}
			case "alt":
				alt = val
			}
		}
		// Prefer lazy-load attributes when src is missing or a tiny data: placeholder.
		chosen := src
		if lazySrc != "" && (chosen == "" || strings.HasPrefix(strings.ToLower(chosen), "data:")) {
			chosen = lazySrc
		}
		if chosen == "" && srcset != "" {
			chosen = firstSrcsetURL(srcset)
		}
		if chosen == "" && pictureFallback != "" {
			chosen = pictureFallback
		}
		return chosen, alt
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			out.WriteString(n.Data)
		case html.ElementNode:
			tag := strings.ToLower(n.Data)
			switch tag {
			case "script", "style", "noscript":
				return
			case "br":
				out.WriteString("\n")
				return
			case "picture":
				sourceFallback := ""
				var imgNode *html.Node
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					if ch.Type != html.ElementNode {
						continue
					}
					chTag := strings.ToLower(ch.Data)
					if chTag == "source" && sourceFallback == "" {
						for _, attr := range ch.Attr {
							k := strings.ToLower(attr.Key)
							if k == "srcset" || k == "data-srcset" || k == "src" {
								if u := firstSrcsetURL(attr.Val); u != "" {
									sourceFallback = u
									break
								}
							}
						}
					} else if chTag == "img" && imgNode == nil {
						imgNode = ch
					}
				}
				if imgNode != nil {
					chosen, alt := extractImgURL(imgNode, sourceFallback)
					if chosen != "" {
						resolved := resolveRelativeURL(chosen, baseURL)
						out.WriteString("\n\n" + formatMarkdownImage(alt, resolved) + "\n\n")
					}
				} else if sourceFallback != "" {
					resolved := resolveRelativeURL(sourceFallback, baseURL)
					out.WriteString("\n\n" + formatMarkdownImage("", resolved) + "\n\n")
				}
				return
			case "img":
				chosen, alt := extractImgURL(n, "")
				if chosen != "" {
					resolved := resolveRelativeURL(chosen, baseURL)
					out.WriteString("\n\n" + formatMarkdownImage(alt, resolved) + "\n\n")
				}
				return
			case "figure":
				out.WriteString("\n")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n")
				return
			case "figcaption":
				var capText strings.Builder
				var collectCap func(*html.Node)
				collectCap = func(c *html.Node) {
					if c.Type == html.TextNode {
						capText.WriteString(c.Data)
					}
					for ch := c.FirstChild; ch != nil; ch = ch.NextSibling {
						collectCap(ch)
					}
				}
				collectCap(n)
				if s := strings.TrimSpace(capText.String()); s != "" {
					out.WriteString("\n*" + s + "*\n\n")
				}
				return
			case "h1", "h2", "h3", "h4":
				level := int(tag[1] - '0')
				out.WriteString("\n" + strings.Repeat("#", level) + " ")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n\n")
				return
			case "p", "div", "section", "article", "blockquote":
				if tag == "blockquote" {
					out.WriteString("\n> ")
				} else {
					out.WriteString("\n")
				}
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("\n\n")
				return
			case "li":
				out.WriteString("\n- ")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				return
			case "strong", "b":
				out.WriteString("**")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("**")
				return
			case "em", "i":
				out.WriteString("*")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("*")
				return
			case "code":
				out.WriteString("`")
				for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				out.WriteString("`")
				return
			case "a":
				// RSS/Atom feeds frequently wrap images inside <a href="..."><img src="..."></a>.
				// Preserve nested images by walking children normally instead of dropping non-text nodes.
				if nodeContainsImage(n) {
					for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
						walk(ch)
					}
					return
				}
				href := ""
				for _, attr := range n.Attr {
					if strings.EqualFold(attr.Key, "href") {
						href = strings.TrimSpace(attr.Val)
						break
					}
				}
				var label strings.Builder
				var collect func(*html.Node)
				collect = func(c *html.Node) {
					if c.Type == html.TextNode {
						label.WriteString(c.Data)
					}
					for ch := c.FirstChild; ch != nil; ch = ch.NextSibling {
						collect(ch)
					}
				}
				collect(n)
				text := strings.TrimSpace(label.String())
				if href != "" && text != "" {
					out.WriteString("[" + text + "](" + href + ")")
				} else if text != "" {
					out.WriteString(text)
				}
				return
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	return strings.TrimSpace(out.String())
}

func isSubImageFilename(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// promoteMarkdownImages moves linked images ([![alt](img)](href)) and inline
// images mixed with text onto standalone paragraphs because mygo's Markdown
// renderer only loads and paints images in image-only paragraphs.
func promoteMarkdownImages(md string) string {
	if !strings.Contains(md, "![") {
		return md
	}
	lines := strings.Split(md, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if inFence || !strings.Contains(line, "![") {
			out = append(out, line)
			continue
		}
		out = append(out, promoteLineMarkdownImages(line))
	}
	return strings.Join(out, "\n")
}

func promoteLineMarkdownImages(line string) string {
	var b strings.Builder
	i := 0
	inCode := false
	for i < len(line) {
		if line[i] == '`' {
			inCode = !inCode
			b.WriteByte(line[i])
			i++
			continue
		}
		if inCode {
			b.WriteByte(line[i])
			i++
			continue
		}
		// Match linked image: [![alt](dest)](href)
		if line[i] == '[' && i+2 < len(line) && line[i+1] == '!' && line[i+2] == '[' {
			if imgEnd, ok := matchMarkdownImageEnd(line, i+1); ok && imgEnd < len(line) && line[imgEnd] == ']' {
				if imgEnd+1 < len(line) && line[imgEnd+1] == '(' {
					if linkEnd, ok2 := matchParenEnd(line, imgEnd+1); ok2 {
						imgExpr := line[i+1 : imgEnd]
						b.WriteString("\n\n" + imgExpr + "\n\n")
						i = linkEnd
						continue
					}
				}
			}
		}
		// Match inline image: ![alt](dest)
		if line[i] == '!' && i+1 < len(line) && line[i+1] == '[' {
			if imgEnd, ok := matchMarkdownImageEnd(line, i); ok {
				imgExpr := line[i:imgEnd]
				// Only add paragraph breaks if there is non-whitespace text on the same line.
				if strings.TrimSpace(line) != imgExpr {
					b.WriteString("\n\n" + imgExpr + "\n\n")
				} else {
					b.WriteString(imgExpr)
				}
				i = imgEnd
				continue
			}
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

func matchMarkdownImageEnd(s string, start int) (int, bool) {
	if start+1 >= len(s) || s[start] != '!' || s[start+1] != '[' {
		return 0, false
	}
	j := start + 2
	for j < len(s) {
		if s[j] == '\\' && j+1 < len(s) {
			j += 2
			continue
		}
		if s[j] == ']' {
			break
		}
		j++
	}
	if j >= len(s) || s[j] != ']' || j+1 >= len(s) || s[j+1] != '(' {
		return 0, false
	}
	return matchParenEnd(s, j+1)
}

func matchParenEnd(s string, openPos int) (int, bool) {
	if openPos >= len(s) || s[openPos] != '(' {
		return 0, false
	}
	j := openPos + 1
	if j < len(s) && s[j] == '<' {
		for j < len(s) && s[j] != '>' {
			j++
		}
		if j >= len(s) {
			return 0, false
		}
		j++ // past '>'
		for j < len(s) && s[j] != ')' {
			j++
		}
		if j < len(s) && s[j] == ')' {
			return j + 1, true
		}
		return 0, false
	}
	depth := 1
	for j < len(s) {
		if s[j] == '\\' && j+1 < len(s) {
			j += 2
			continue
		}
		if s[j] == '(' {
			depth++
		} else if s[j] == ')' {
			depth--
			if depth == 0 {
				return j + 1, true
			}
		}
		j++
	}
	return 0, false
}

// buildSubscriptionMarkdown converts HTML (if present) to readable Markdown and
// appends unreferenced Planet image attachments or RSS media image enclosures.
func buildSubscriptionMarkdown(body string, item feeds.Item, baseURL string) string {
	md := promoteMarkdownImages(htmlToReadableMarkdownWithBase(body, baseURL))
	var extras []string
	for _, name := range item.Attachments {
		name = strings.TrimSpace(name)
		if name == "" || !isSubImageFilename(name) {
			continue
		}
		resolved := resolveRelativeURL(name, baseURL)
		if strings.Contains(md, name) || strings.Contains(md, resolved) {
			continue
		}
		extras = append(extras, formatMarkdownImage(name, resolved))
	}
	for _, m := range item.Media {
		u := strings.TrimSpace(m.URL)
		if u == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(m.Type)), "image/") && !isSubImageFilename(u) {
			continue
		}
		resolved := resolveRelativeURL(u, baseURL)
		if strings.Contains(md, u) || strings.Contains(md, resolved) {
			continue
		}
		extras = append(extras, formatMarkdownImage("", resolved))
	}
	if len(extras) > 0 {
		if strings.TrimSpace(md) != "" {
			md += "\n\n"
		}
		md += strings.Join(extras, "\n\n")
	}
	return md
}
