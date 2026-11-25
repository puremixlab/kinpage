package nativeui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/feeds"
	"windows/internal/store"
)

type fakeWebViewHost struct {
	loadedHTML string
	loadedURL  string
	destroyed  bool
}

func (f *fakeWebViewHost) Native() ui.NativeWebView    { return nil }
func (f *fakeWebViewHost) LoadHTML(html, _ string)     { f.loadedHTML = html }
func (f *fakeWebViewHost) LoadURL(rawURL string) error { f.loadedURL = rawURL; return nil }
func (f *fakeWebViewHost) Destroy()                    { f.destroyed = true }

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 10, G: 120, B: 240, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode tiny png: %v", err)
	}
	return buf.Bytes()
}

func TestHTMLToReadableMarkdownImagesAndFigures(t *testing.T) {
	raw := `<article><p>开头段落</p><figure><img src="/assets/photo.jpg" alt="河岸风景"><figcaption>傍晚的芦苇</figcaption></figure><p><img data-src="images/lazy.png" alt="延迟加载"></p></article>`
	md := htmlToReadableMarkdownWithBase(raw, "https://river.example/posts/autumn")
	for _, want := range []string{
		"![河岸风景](https://river.example/assets/photo.jpg)",
		"*傍晚的芦苇*",
		"![延迟加载](https://river.example/posts/images/lazy.png)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in converted markdown:\n%s", want, md)
		}
	}
}

func TestImageLoaderHTTPCacheAndSafety(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	loader := newImageLoader()
	src1, err := loader.Load(srv.URL + "/pic.png")
	if err != nil || src1 == nil {
		t.Fatalf("expected first remote load to succeed, src=%v err=%v", src1, err)
	}
	src2, err := loader.Load(srv.URL + "/pic.png")
	if err != nil || src2 == nil {
		t.Fatalf("expected cached remote load to succeed, src=%v err=%v", src2, err)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 HTTP request due to cache, got %d", hits.Load())
	}

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	if src, err := loader.Load(dataURL); err != nil || src == nil {
		t.Fatalf("expected data URL image load to succeed, src=%v err=%v", src, err)
	}

	for _, bad := range []string{"javascript:alert(1)", "ftp://example.com/a.png", "file:///C:/Windows/win.ini"} {
		src, err := loader.Load(bad)
		if err == nil {
			t.Fatalf("expected unsafe scheme %q to be rejected", bad)
		}
		if src != nil {
			t.Fatalf("expected untyped nil ImageSource on error for %q, got %T", bad, src)
		}
		// Second call hits the error cache and must also return untyped nil.
		if cachedSrc, cachedErr := loader.Load(bad); cachedErr == nil || cachedSrc != nil {
			t.Fatalf("expected cached untyped nil ImageSource on error for %q, got src=%T err=%v", bad, cachedSrc, cachedErr)
		}
	}
}

func TestInAppTemplatePreviewAndSubscriptionWebViewToggle(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "预览站", Created: now}},
		subscriptions: []store.Subscription{{
			URL: "https://example.com/rss",
			Feed: feeds.Feed{
				Title: "外部周刊",
				Items: []feeds.Item{{
					Key:     "item-1",
					Title:   "带图文章",
					Link:    "https://example.com/posts/1",
					Content: `<p>正文</p><img src="/img/cover.png" alt="封面">`,
				}},
			},
		}},
	}

	var created []*fakeWebViewHost
	app := New(backend)
	app.SetWebViewFactory(func() (WebViewHost, error) {
		f := &fakeWebViewHost{}
		created = append(created, f)
		return f, nil
	})

	tt := ui.NewTester(app.View, 1360, 860)
	tt.Frame()

	// 1. In-app template preview inside Site Settings.
	if err := tt.Click("预览站"); err != nil {
		t.Fatalf("select site: %v", err)
	}
	if err := tt.Click("站点设置"); err != nil {
		t.Fatalf("open site settings: %v", err)
	}
	if err := tt.Click("模板"); err != nil {
		t.Fatalf("switch to templates tab: %v", err)
	}
	if err := tt.Click("预览"); err != nil {
		t.Fatalf("click template preview: %v", err)
	}
	if !app.templatePreview.open || len(created) != 1 || !strings.Contains(created[0].loadedHTML, "<h1>plain</h1>") {
		t.Fatalf("expected in-app template preview WebView loaded, open=%v created=%d html=%q",
			app.templatePreview.open, len(created), func() string {
				if len(created) > 0 {
					return created[0].loadedHTML
				}
				return ""
			}())
	}
	if err := tt.Click("关闭预览"); err != nil {
		t.Fatalf("close template preview: %v", err)
	}
	if app.templatePreview.open || !created[0].destroyed {
		t.Fatalf("expected template preview closed and WebView destroyed, open=%v destroyed=%v",
			app.templatePreview.open, created[0].destroyed)
	}
	app.siteSettings.open = false
	tt.Frame()

	// 2. Subscription Reader <-> Original WebView toggle.
	if err := tt.Click("外部周刊"); err != nil {
		t.Fatalf("select subscription: %v", err)
	}
	if err := tt.Click("查看原网页"); err != nil {
		t.Fatalf("switch to original webpage: %v", err)
	}
	if !app.subShowWebView || len(created) != 2 || created[1].loadedURL != "https://example.com/posts/1" {
		t.Fatalf("expected subscription WebView loaded with article URL, show=%v created=%d", app.subShowWebView, len(created))
	}
	if err := tt.Click("切换阅读模式"); err != nil {
		t.Fatalf("switch back to reader view: %v", err)
	}
	if app.subShowWebView || !created[1].destroyed {
		t.Fatalf("expected reader mode restored and WebView destroyed, show=%v destroyed=%v",
			app.subShowWebView, created[1].destroyed)
	}
}
