package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"windows/internal/nativeui"
	"windows/internal/services"
	"windows/internal/store"
)

//go:embed resources/icon.png
var appIconPNG []byte

func main() {
	// MyGo CLI invokes the executable with MYGO_GENERATE set during builds.
	// Return early before touching user data directories or acquiring locks.
	if os.Getenv("MYGO_GENERATE") != "" {
		if err := mygo.App.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--planet-version" {
		fmt.Println(appVersion)
		return
	}
	if executable, err := os.Executable(); err != nil {
		showStartupError(err)
		log.Fatal(err)
	} else if err = checkUpdateReady(executable); err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	mygo.App.SetName("Kinpage")
	if mygo.IsDev() {
		path, err := mygo.App.Path(mygo.PathUserData)
		if err != nil {
			log.Fatal(err)
		}
		mygo.App.SetPath(mygo.PathUserData, filepath.Join(path, "development"))
	}
	defaultUserDataDir, _ := mygo.App.Path(mygo.PathUserData)
	if defaultUserDataDir != "" {
		if resolved := store.ResolveDataDir(defaultUserDataDir); resolved != "" && resolved != defaultUserDataDir {
			mygo.App.SetPath(mygo.PathUserData, resolved)
		}
	}
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	releaseUpdateLock, err := acquireUpdateLock()
	if err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	defer releaseUpdateLock()
	library := &Library{defaultDataDir: defaultUserDataDir}
	maintenanceCtx, cancelMaintenance := context.WithCancel(context.Background())
	var maintenance sync.WaitGroup
	defer func() {
		_ = services.SetPreventSleep(false)
		library.closeAPIServer()
		library.closeUpdates()
		cancelMaintenance()
		maintenance.Wait()
		library.closeNode()
		library.closeAudio()
		library.StopEditorMedia()
	}()
	mygo.App.WhenReady(func() {
		if root, err := mygo.App.Path(mygo.PathUserData); err == nil {
			_ = library.ensureAPIServer().Sync(root)
		}
		if library.PreventSleep() {
			_ = services.SetPreventSleep(true)
		}
		maintenance.Add(1)
		go func() { defer maintenance.Done(); library.maintainPublished(maintenanceCtx) }()
		nativeApp := nativeui.New(library)
		if s, err := library.open(); err == nil {
			maintenance.Add(1)
			go func() {
				defer maintenance.Done()
				s.WatchExternalChanges(maintenanceCtx, 2*time.Second, nativeApp.MarkDirtyExternal)
			}()
		}
		if root, err := mygo.App.Path(mygo.PathUserData); err == nil {
			nativeApp.EnablePrefsPersistence(filepath.Join(root, "nativeui-layout.json"))
		}
		opts := mygo.WindowOptions{
			Title: "邻页 · Kinpage", Width: 1280, Height: 820, MinWidth: 860, MinHeight: 600,
			BackgroundColor: "#f7f7f5", StateKey: "main",
			TitleBarStyle:  mygo.TitleBarHidden,
			TitleBarHeight: 44,
			Content:        ui.View(nativeApp.View),
		}
		win := mygo.NewWindow(opts)
		if len(appIconPNG) > 0 {
			_ = win.SetIcon(appIconPNG)
		}
		nativeApp.SetWebViewFactory(func() (nativeui.WebViewHost, error) {
			wv, err := win.NewWebView(mygo.WebViewOptions{
				Page: mygo.PageOptions{Untrusted: true},
			})
			if err != nil {
				return nil, err
			}
			return &mygoWebViewHost{wv: wv}, nil
		})

		showMainWindow := func() {
			if win.IsDestroyed() {
				return
			}
			if win.IsMinimized() {
				win.Restore()
			}
			win.Show()
			win.Focus()
		}

		var tray *mygo.Tray
		var quitting atomic.Bool
		var closing atomic.Bool

		quitApp := func() {
			if !closing.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer closing.Store(false)
				if library.ActivePublishCount() > 0 {
					showMainWindow()
					title, msg, detail, cancelBtn, quitBtn := trayQuitConfirmStrings(nativeApp.Locale())
					result, err := mygo.Dialog.Message(mygo.MessageOptions{
						Parent:        win,
						Type:          mygo.MessageQuestion,
						Title:         title,
						Message:       msg,
						Detail:        detail,
						Buttons:       []string{cancelBtn, quitBtn},
						DefaultButton: 0,
						CancelButton:  0,
					})
					if err != nil || result.Button != 1 {
						return
					}
				}
				quitting.Store(true)
				if tray != nil {
					tray.Destroy()
					tray = nil
				}
				nativeApp.Close()
				if !win.IsDestroyed() {
					win.Destroy()
				}
				mygo.App.Quit()
			}()
		}

		buildTrayMenu := func(loc nativeui.Locale) *mygo.Menu {
			showLabel, quitLabel := trayMenuStrings(loc)
			return mygo.NewMenu([]*mygo.MenuItem{
				{
					Label: showLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						showMainWindow()
					},
				},
				mygo.Separator(),
				{
					Label: quitLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						quitApp()
					},
				},
			})
		}

		if createdTray, err := mygo.NewTray(mygo.TrayOptions{
			ID:      "kinpage-tray",
			Icon:    appIconPNG,
			ToolTip: "邻页 · Kinpage",
			Menu:    buildTrayMenu(nativeApp.Locale()),
		}); err == nil {
			tray = createdTray
			tray.OnClick(showMainWindow)
			nativeApp.SetOnLocaleChange(func(loc nativeui.Locale) {
				if tray != nil {
					tray.SetMenu(buildTrayMenu(loc))
				}
			})
		} else {
			log.Printf("system tray unavailable: %v", err)
		}

		win.OnClose(func(event *mygo.CloseEvent) {
			if quitting.Load() {
				return
			}
			event.PreventDefault()
			// Hide to the Windows taskbar system tray so Planet continues running
			// in the background (IPFS node, Local API, and site maintenance).
			if tray != nil {
				win.Hide()
				return
			}
			// Fallback when system tray is unavailable: confirm and exit.
			if !closing.CompareAndSwap(false, true) {
				return
			}
			go func() {
				defer closing.Store(false)
				detail := "尚未保存的文章更改将会丢失。"
				if library.ActivePublishCount() > 0 {
					detail = "当前正在进行 IPNS 发布，强制关闭可能导致发布中断或站点无法解析。"
				}
				result, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: win, Type: mygo.MessageQuestion, Title: "关闭邻页", Message: "确定关闭窗口？", Detail: detail, Buttons: []string{"取消", "关闭"}, DefaultButton: 0, CancelButton: 0})
				if err == nil && result.Button == 1 {
					quitting.Store(true)
					nativeApp.Close()
					win.Destroy()
				}
			}()
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func trayMenuStrings(loc nativeui.Locale) (showLabel, quitLabel string) {
	switch loc {
	case nativeui.LocaleZhCN:
		return "显示邻页", "退出邻页"
	case nativeui.LocaleZhTW:
		return "顯示鄰頁", "結束鄰頁"
	default:
		return "Show Kinpage", "Quit Kinpage"
	}
}

func trayQuitConfirmStrings(loc nativeui.Locale) (title, msg, detail, cancelBtn, quitBtn string) {
	switch loc {
	case nativeui.LocaleZhCN:
		return "退出邻页", "确定退出邻页？", "当前正在进行 IPNS 发布，强制退出可能导致发布中断或站点无法解析。", "取消", "退出"
	case nativeui.LocaleZhTW:
		return "結束鄰頁", "確定要結束鄰頁？", "目前正在進行 IPNS 發布，強制結束可能導致發布中斷或站點無法解析。", "取消", "結束"
	default:
		return "Quit Kinpage", "Quit Kinpage?", "An IPNS publish is currently in progress. Quitting now may interrupt publishing.", "Cancel", "Quit"
	}
}

type mygoWebViewHost struct {
	wv *mygo.WebView
}

func (h *mygoWebViewHost) Native() ui.NativeWebView {
	if h == nil || h.wv == nil {
		return nil
	}
	return h.wv
}

func (h *mygoWebViewHost) LoadHTML(html, baseURL string) {
	if h != nil && h.wv != nil && !h.wv.IsDestroyed() {
		h.wv.Page().LoadHTML(html, baseURL)
	}
}

func (h *mygoWebViewHost) LoadURL(rawURL string) error {
	if h == nil || h.wv == nil || h.wv.IsDestroyed() {
		return nil
	}
	return h.wv.Page().LoadURL(rawURL)
}

func (h *mygoWebViewHost) Destroy() {
	if h != nil && h.wv != nil && !h.wv.IsDestroyed() {
		h.wv.Destroy()
	}
}
