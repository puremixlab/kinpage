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
	raw := `<article>` +
		`<p>开头段落</p>` +
		`<figure><a href="https://river.example/full"><img src="/assets/photo.jpg" alt="河岸风景"></a><figcaption>傍晚的芦苇</figcaption></figure>` +
		`<p><img src="data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7" data-src="images/lazy.png" alt="延迟加载"></p>` +
		`<p><img srcset="/img/hero-1x.webp 1x, /img/hero-2x.webp 2x" alt="高清封面"></p>` +
		`<picture><source srcset="https://cdn.example/pic-1.webp 1x, https://cdn.example/pic-2.webp 2x"><img alt="画册"></picture>` +
		`</article>`
	md := htmlToReadableMarkdownWithBase(raw, "https://river.example/posts/autumn")
	for _, want := range []string{
		"![河岸风景](https://river.example/assets/photo.jpg)",
		"*傍晚的芦苇*",
		"![延迟加载](https://river.example/posts/images/lazy.png)",
		"![高清封面](https://river.example/img/hero-1x.webp)",
		"![画册](https://cdn.example/pic-1.webp)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in converted markdown:\n%s", want, md)
		}
	}
}

func TestPromoteMarkdownImages(t *testing.T) {
	raw := "前言 [![封面](https://cdn.example/a.jpg)](https://example.com) 后记\n\n`![code](https://ignore.example/x.png)`\n\n文字 ![插图](https://cdn.example/b.png) 续文"
	got := promoteMarkdownImages(raw)
	if !strings.Contains(got, "\n\n![封面](https://cdn.example/a.jpg)\n\n") {
		t.Fatalf("expected linked image promoted to standalone paragraph:\n%s", got)
	}
	if !strings.Contains(got, "\n\n![插图](https://cdn.example/b.png)\n\n") {
		t.Fatalf("expected inline image promoted to standalone paragraph:\n%s", got)
	}
	if !strings.Contains(got, "`![code](https://ignore.example/x.png)`") {
		t.Fatalf("expected inline code image to stay untouched:\n%s", got)
	}
}

func TestImageLoaderReferer403Fallback(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Referer") != "" {
			http.Error(w, "hotlink blocked", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	loader := newImageLoader()
	src, err := loader.LoadWithReferer(srv.URL+"/wechat.png", "https://blog.example/post")
	if err != nil || src == nil {
		t.Fatalf("expected retry without Referer to succeed, src=%v err=%v", src, err)
	}
	if hits.Load() != 2 {
		t.Fatalf("expected 2 requests (with Referer -> 403, then without Referer -> 200), got %d", hits.Load())
	}
}

func TestSubscriptionReaderLoadsHTTPImageEndToEnd(t *testing.T) {
	pngBytes := tinyPNGBytes(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer srv.Close()

	imgURL := srv.URL + "/article-photo.png"
	backend := &fakeBackend{
		subscriptions: []store.Subscription{{
			URL: "https://example.com/feed.xml",
			Feed: feeds.Feed{
				Title: "图片测试源",
				Items: []feeds.Item{{
					Key:     "img-1",
					Title:   "带外链图的订阅文章",
					Link:    "https://example.com/posts/img-1",
					Content: `<p>开头</p><p><a href="https://example.com/full"><img src="` + imgURL + `" alt="远端图片"></a></p>`,
				}},
			},
		}},
	}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 800)
	tt.Frame()
	if err := tt.Click("图片测试源"); err != nil {
		t.Fatalf("select subscription: %v", err)
	}
	// Wait briefly for the async LoadImage goroutine to fetch and decode the HTTP image.
	var cached cachedImage
	var found bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if app.imgLoader != nil {
			app.imgLoader.mu.Lock()
			cached, found = app.imgLoader.cache[imgURL]
			app.imgLoader.mu.Unlock()
			if found {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if hits.Load() == 0 {
		t.Fatalf("expected subscription reader to fetch HTTP image %q, got 0 requests", imgURL)
	}
	if !found || cached.err != nil || cached.src == nil {
		t.Fatalf("expected HTTP image to decode and cache cleanly, found=%v src=%v err=%v", found, cached.src, cached.err)
	}
}

func TestSubscriptionBaseURLAndAttachments(t *testing.T) {
	app := New(&fakeBackend{})
	app.ipfsModal.status.Running = true
	app.ipfsModal.status.Gateway = "http://127.0.0.1:8080"
	cur := subItemEntry{
		subURL: "olivida.eth",
		item: feeds.Item{
			ContentCID:  "bafybeigdyrzt",
			ArticleID:   "art-42",
			Content:     "一段正文",
			Attachments: []string{"cover.jpg"},
			Media:       []feeds.RemoteMedia{{URL: "https://cdn.example/pic.webp", Type: "image/webp"}},
		},
	}
	base := app.subscriptionBaseURL(cur)
	wantBase := "http://127.0.0.1:8080/ipfs/bafybeigdyrzt/art-42/"
	if base != wantBase {
		t.Fatalf("subscriptionBaseURL = %q, want %q", base, wantBase)
	}
	md := buildSubscriptionMarkdown(cur.item.Content, cur.item, base)
	for _, want := range []string{
		"![cover.jpg](http://127.0.0.1:8080/ipfs/bafybeigdyrzt/art-42/cover.jpg)",
		"![](https://cdn.example/pic.webp)",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in built markdown:\n%s", want, md)
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
