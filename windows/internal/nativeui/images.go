package nativeui

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	maxMarkdownImageBytes = 10 << 20 // 10 MiB safety cap per remote image
	maxMarkdownImageDim   = 1600     // cap decoded bitmap longest edge
)

type cachedImage struct {
	bmp *ui.Bitmap
	err error
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
		client: &http.Client{Timeout: 12 * time.Second},
		cache:  map[string]cachedImage{},
	}
}

func (l *imageLoader) Load(rawURL string) (ui.ImageSource, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("empty image url")
	}

	l.mu.Lock()
	if entry, ok := l.cache[rawURL]; ok {
		l.mu.Unlock()
		return entry.bmp, entry.err
	}
	l.mu.Unlock()

	bmp, err := l.fetchAndDecode(rawURL)

	l.mu.Lock()
	if len(l.cache) > 256 {
		l.cache = map[string]cachedImage{}
	}
	l.cache[rawURL] = cachedImage{bmp: bmp, err: err}
	l.mu.Unlock()

	return bmp, err
}

func (l *imageLoader) fetchAndDecode(rawURL string) (*ui.Bitmap, error) {
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
		return ui.DecodeBitmapScaled(data, maxMarkdownImageDim)
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
		req.Header.Set("User-Agent", "Planet-Windows-NativeUI/1.0")
		req.Header.Set("Accept", "image/*,*/*;q=0.8")
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
		return ui.DecodeBitmapScaled(data, maxMarkdownImageDim)
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
			return loader.Load(resolved)
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
			return ui.DecodeBitmapScaled(data, maxMarkdownImageDim)
		}
		return nil, fmt.Errorf("unsupported image source %q", trimmed)
	}
}
