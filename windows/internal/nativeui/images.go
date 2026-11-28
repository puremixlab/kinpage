package nativeui

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	_ "golang.org/x/image/webp"
	"windows/internal/preview"
)

const (
	maxMarkdownImageBytes = 10 << 20 // 10 MiB safety cap per remote image
	maxMarkdownImageDim   = 1600     // cap decoded bitmap longest edge
	imageErrorCacheTTL    = 30 * time.Second
	imageDownloadLimit    = 90 * time.Second
)

var errImageReadStalled = errors.New("image download stopped receiving data")

type cachedImage struct {
	src ui.ImageSource
	err error
	at  time.Time
}

// imageLoader safely fetches, decodes, and caches Markdown images off the UI
// goroutine for remote HTTP(S) URLs, data: URLs, and optional local files.
type imageLoader struct {
	client          *http.Client
	readIdleTimeout time.Duration

	mu    sync.Mutex
	cache map[string]cachedImage
}

func newImageHTTPTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Prefer HTTP/1.1 for remote image fetches to avoid HTTP/2 EOF issues
		// through local TUN/proxy setups and strict image CDNs.
		ForceAttemptHTTP2:     false,
		TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
		MaxIdleConns:          32,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func newImageLoader() *imageLoader {
	return &imageLoader{
		client:          &http.Client{Transport: newImageHTTPTransport()},
		readIdleTimeout: 15 * time.Second,
		cache:           map[string]cachedImage{},
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
		return l.fetchHTTPImage(rawURL, parsed, referer)
	default:
		return nil, fmt.Errorf("unsupported image scheme %q", scheme)
	}
}

type imageProgressReader struct {
	io.Reader
	timer *time.Timer
	idle  time.Duration
}

func (r imageProgressReader) Read(buf []byte) (int, error) {
	n, err := r.Reader.Read(buf)
	if n > 0 {
		r.timer.Reset(r.idle)
	}
	return n, err
}

func (l *imageLoader) doImageGet(parent context.Context, rawURL, referer string) ([]byte, string, int, error) {
	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/webp,image/png,image/jpeg,image/gif,image/svg+xml,image/*;q=0.8,*/*;q=0.5")
	if ref := strings.TrimSpace(referer); strings.HasPrefix(strings.ToLower(ref), "http://") || strings.HasPrefix(strings.ToLower(ref), "https://") {
		req.Header.Set("Referer", ref)
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", resp.StatusCode, fmt.Errorf("image http %d", resp.StatusCode)
	}
	if resp.ContentLength > maxMarkdownImageBytes {
		return nil, "", resp.StatusCode, errors.New("remote image exceeds size limit")
	}
	idle := time.AfterFunc(l.readIdleTimeout, func() { cancel(errImageReadStalled) })
	defer idle.Stop()
	reader := imageProgressReader{Reader: resp.Body, timer: idle, idle: l.readIdleTimeout}
	data, err := io.ReadAll(io.LimitReader(reader, maxMarkdownImageBytes+1))
	if err != nil {
		if cause := context.Cause(ctx); cause != nil {
			err = cause
		}
		return nil, "", resp.StatusCode, err
	}
	if len(data) > maxMarkdownImageBytes {
		return nil, "", resp.StatusCode, errors.New("remote image exceeds size limit")
	}
	return data, resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

func (l *imageLoader) fetchHTTPImage(rawURL string, parsed *url.URL, referer string) (ui.ImageSource, error) {
	ctx, cancel := context.WithTimeout(context.Background(), imageDownloadLimit)
	defer cancel()
	// Build a small ordered list of Referer strategies so both anti-hotlink CDNs
	// (requiring article/origin Referer) and strict CDNs (rejecting foreign Referer,
	// e.g. WeChat/Bilibili/Imgur) succeed on the first load.
	var attempts []string
	ref := strings.TrimSpace(referer)
	originRef := ""
	if parsed != nil && parsed.Host != "" {
		originRef = parsed.Scheme + "://" + parsed.Host + "/"
	}
	if ref != "" {
		attempts = append(attempts, ref, "")
		if originRef != "" && !strings.EqualFold(originRef, ref) {
			attempts = append(attempts, originRef)
		}
	} else {
		attempts = append(attempts, "")
		if originRef != "" {
			attempts = append(attempts, originRef)
		}
	}

	var lastErr error
	for i, tryRef := range attempts {
		data, ct, status, err := l.doImageGet(ctx, rawURL, tryRef)
		if err == nil {
			return decodeImageBytes(data, ct, parsed.Path)
		}
		lastErr = err
		// Only retry with alternate Referer on 400/401/403 responses.
		if i+1 < len(attempts) && (status == http.StatusForbidden || status == http.StatusUnauthorized || status == http.StatusBadRequest) {
			continue
		}
		break
	}
	return nil, lastErr
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

func (a *App) markdownAttachmentLoader(siteID, articleID, draftID string) func(string) (ui.ImageSource, error) {
	backend := a.backend
	if backend == nil || (articleID == "" && draftID == "") {
		return a.markdownImageLoader("", nil)
	}
	return a.markdownImageLoader("", func(name string) ([]byte, string, error) {
		return backend.ReadMarkdownAttachment(siteID, articleID, draftID, name)
	})
}

// markdownImageLoader resolves images against a subscription URL or a registered attachment.
func (a *App) markdownImageLoader(baseURL string, attachment func(string) ([]byte, string, error)) func(string) (ui.ImageSource, error) {
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
		if attachment != nil {
			name, err := preview.RelativeImageName(trimmed)
			if err != nil {
				return nil, err
			}
			data, mime, err := attachment(name)
			if err != nil {
				return nil, err
			}
			return decodeImageBytes(data, mime, name)
		}
		return nil, fmt.Errorf("unsupported image source %q", trimmed)
	}
}
