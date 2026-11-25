package nativeui

import "github.com/egoist/mygo/ui"

// WebViewHost abstracts an embedded sandboxed WebView2 instance created by
// mygo.Window.NewWebView so nativeui stays testable in headless ui.Tester runs.
type WebViewHost interface {
	Native() ui.NativeWebView
	LoadHTML(html, baseURL string)
	LoadURL(rawURL string) error
	Destroy()
}

// WebViewFactory constructs a new untrusted/sandboxed WebViewHost bound to the
// native host window.
type WebViewFactory func() (WebViewHost, error)

// SetWebViewFactory injects the host window's WebView creator for in-app
// template previews and subscription original-webpage viewing.
func (a *App) SetWebViewFactory(fn WebViewFactory) {
	a.newWebView = fn
}

func (a *App) destroyTemplatePreviewWV() {
	if a.templatePreview.wv != nil {
		a.templatePreview.wv.Destroy()
		a.templatePreview.wv = nil
	}
}

func (a *App) destroySubWebView() {
	if a.subWebView != nil {
		a.subWebView.Destroy()
		a.subWebView = nil
	}
	a.subWebViewURL = ""
}
