package nativeui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/webp"
)

const (
	maxMarkdownImageBytes = 10 << 20 // 10 MiB safety cap per remote image
	maxMarkdownImageDim   = 1600     // cap decoded bitmap longest edge
	imageErrorCacheTTL    = 30 * time.Second
)

type cachedImage struct {
	src ui.ImageSource
	err error
	at  time.Time
}

// imageLoader safely fetches, decodes, and caches Markdown images off the UI
// goroutine for remote HTTP(S) URLs, data: URLs, and optional local files.
type imageLoader struct {
	client *http.Client

	mu    sync.Mutex
	cache map[string]cachedImage
}

func newImageLoader() *imageLoader {
	return &imageLoader{
		client: &http.Client{Timeout: 15 * time.Second},
		cache:  map[string]cachedImage{},
	}
}

func (l *imageLoader) Load(rawURL string) (ui.ImageSource, error) {
	return l.LoadWithReferer(rawURL, "")
}

func (l *imageLoader) LoadWithReferer(rawURL, referer string) (ui.ImageSource, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("empty image url")
	}

	l.mu.Lock()
	if entry, ok := l.cache[rawURL]; ok {
		if entry.err == nil && entry.src != nil {
			l.mu.Unlock()
			return entry.src, nil
		}
		if entry.err != nil && time.Since(entry.at) < imageErrorCacheTTL {
			l.mu.Unlock()
			return nil, entry.err
		}
	}
	l.mu.Unlock()

	src, err := l.fetchAndDecode(rawURL, referer)

	l.mu.Lock()
	if len(l.cache) > 256 {
		l.cache = map[string]cachedImage{}
	}
	l.cache[rawURL] = cachedImage{src: src, err: err, at: time.Now()}
	l.mu.Unlock()

	if err != nil || src == nil {
		return nil, err
	}
	return src, nil
}

func decodeImageBytes(data []byte, contentType, rawURL string) (ui.ImageSource, error) {
	ct := strings.ToLower(contentType)
	lowerURL := strings.ToLower(rawURL)
	trimmed := bytes.TrimSpace(data)
	isSVG := strings.Contains(ct, "image/svg") ||
		strings.HasSuffix(lowerURL, ".svg") ||
		bytes.HasPrefix(trimmed, []byte("<svg")) ||
		(bytes.HasPrefix(trimmed, []byte("<?xml")) && bytes.Contains(trimmed[:min(len(trimmed), 256)], []byte("<svg")))
	if isSVG {
		svg, err := ui.ParseSVG(data)
		if err != nil || svg == nil {
			return nil, err
		}
		return svg, nil
	}
	bmp, err := ui.DecodeBitmapScaled(data, maxMarkdownImageDim)
	if err != nil || bmp == nil {
		return nil, err
	}
	return bmp, nil
}

func (l *imageLoader) fetchAndDecode(rawURL, referer string) (ui.ImageSource, error) {
	if strings.HasPrefix(strings.ToLower(rawURL), "data:") {
		idx := strings.Index(rawURL, ",")
		if idx < 0 {
			return nil, errors.New("malformed data url")
		}
		meta := strings.ToLower(rawURL[:idx])
		if !strings.Contains(meta, ";base64") {
			return nil, errors.New("unsupported non-base64 data url")
		}
		payload := rawURL[idx+1:]
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			data, err = base64.RawStdEncoding.DecodeString(payload)
			if err != nil {
				return nil, err
			}
		}
		if len(data) > maxMarkdownImageBytes {
			return nil, errors.New("data image too large")
		}
		return decodeImageBytes(data, meta, "")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	scheme := strings.ToLower(parsed.Scheme)
	switch scheme {
	case "http", "https":
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "image/webp,image/png,image/jpeg,image/gif,image/svg+xml,image/*;q=0.8,*/*;q=0.5")
		if ref := strings.TrimSpace(referer); strings.HasPrefix(strings.ToLower(ref), "http://") || strings.HasPrefix(strings.ToLower(ref), "https://") {
			req.Header.Set("Referer", ref)
		}
		resp, err := l.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("image http %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxMarkdownImageBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > maxMarkdownImageBytes {
			return nil, errors.New("remote image exceeds size limit")
		}
		return decodeImageBytes(data, resp.Header.Get("Content-Type"), parsed.Path)
	default:
		return nil, fmt.Errorf("unsupported image scheme %q", scheme)
	}
}

// resolveRelativeURL resolves a potentially relative image/link URL against baseURL.
func resolveRelativeURL(raw, baseURL string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:") {
		return raw
	}
	if strings.HasPrefix(raw, "//") {
		return "https:" + raw
	}
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") {
		return raw
	}
	ref, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return base.ResolveReference(ref).String()
}

// markdownImageLoader returns a LoadImage callback for lobe-mygo Markdown views,
// resolving relative paths against baseURL (or localDir if provided).
func (a *App) markdownImageLoader(baseURL, localDir string) func(string) (ui.ImageSource, error) {
	if a.imgLoader == nil {
		a.imgLoader = newImageLoader()
	}
	loader := a.imgLoader
	return func(raw string) (ui.ImageSource, error) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return nil, errors.New("empty image url")
		}
		resolved := resolveRelativeURL(trimmed, baseURL)
		lower := strings.ToLower(resolved)
		if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:") {
			return loader.LoadWithReferer(resolved, baseURL)
		}
		if localDir != "" && !filepath.IsAbs(trimmed) && !strings.Contains(trimmed, "..") {
			candidate := filepath.Join(localDir, filepath.FromSlash(trimmed))
			data, err := os.ReadFile(candidate)
			if err != nil {
				return nil, err
			}
			if len(data) > maxMarkdownImageBytes {
				return nil, errors.New("local image exceeds size limit")
			}
			return decodeImageBytes(data, "", candidate)
		}
		return nil, fmt.Errorf("unsupported image source %q", trimmed)
	}
}
