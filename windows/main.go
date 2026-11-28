package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"windows/internal/desktopshortcut"
	"windows/internal/nativeui"
	"windows/internal/services"
	"windows/internal/store"
	"windows/resources"
)

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
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	releaseUpdateLock, err := acquireUpdateLock()
	if err != nil {
		showStartupError(err)
		log.Fatal(err)
	}
	defer releaseUpdateLock()
	if defaultUserDataDir != "" {
		resolved, err := store.ApplyScheduledDataDirMigration(defaultUserDataDir)
		if err != nil {
			if resolved == "" {
				showStartupError(err)
				return
			}
			showStartupError(fmt.Errorf("无法应用数据仓库位置，当前使用 %s：%w", resolved, err))
		}
		if resolved != defaultUserDataDir {
			mygo.App.SetPath(mygo.PathUserData, resolved)
		}
	}
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
		nativeApp.SetBackgroundContext(maintenanceCtx, win.Invalidate)
		nativeApp.CheckUpdatesOnStartup()
		maintenance.Add(1)
		go func() {
			defer maintenance.Done()
			library.maintainSubscriptions(maintenanceCtx, func() {
				nativeApp.MarkSubscriptionsDirty()
				win.Invalidate()
			})
		}()
		appIconPNG := nativeApp.AppIcon().PNG()
		if err := win.SetIcon(appIconPNG); err != nil {
			log.Printf("window icon unavailable: %v", err)
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
		nativeApp.RestoreDesktopPet(win.NativeHandle())
		win.OnClosed(nativeApp.CloseDesktopPet)

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

		finishQuit := func() {
			if !quitting.CompareAndSwap(false, true) {
				return
			}
			cancelMaintenance()
			mygo.RunOnMain(func() {
				if tray != nil {
					tray.Destroy()
					tray = nil
				}
				nativeApp.Close()
				if !win.IsDestroyed() {
					win.Destroy()
				}
				mygo.App.Quit()
			})
		}

		quitApp := func() {
			if quitting.Load() || !closing.CompareAndSwap(false, true) {
				return
			}
			win.Update(func() {
				showMainWindow()
				requestEditorExit := func() {
					closing.Store(false)
					nativeApp.RequestExit(func() { go finishQuit() })
				}
				if library.ActivePublishCount() == 0 && !nativeApp.HasPendingPublications() {
					requestEditorExit()
					return
				}
				title, msg, detail, cancelBtn, quitBtn := trayQuitConfirmStrings(nativeApp.Locale())
				go func() {
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
						closing.Store(false)
						return
					}
					win.Update(requestEditorExit)
				}()
			})
		}

		buildTrayMenu := func(loc nativeui.Locale) *mygo.Menu {
			showLabel, quitLabel := trayMenuStrings(loc)
			items := []*mygo.MenuItem{
				{
					Label: showLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						showMainWindow()
					},
				},
			}
			if nativeApp.DesktopPetAvailable() {
				items = append(items, &mygo.MenuItem{
					Label:   nativeApp.DesktopPetLabel(),
					Type:    mygo.MenuItemCheckbox,
					Checked: nativeApp.DesktopPetEnabled(),
					Click: func(item *mygo.MenuItem, _ *mygo.Window) {
						if quitting.Load() {
							return
						}
						if !nativeApp.SetDesktopPetEnabled(item.IsChecked()) {
							item.SetChecked(nativeApp.DesktopPetEnabled())
							showMainWindow()
						}
						win.Invalidate()
					},
				})
			}
			return mygo.NewMenu(append(items, mygo.Separator(),
				&mygo.MenuItem{
					Label: quitLabel,
					Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
						quitApp()
					},
				},
			))
		}

		if createdTray, err := mygo.NewTray(mygo.TrayOptions{
			ID:      "kinpage-tray",
			Icon:    appIconPNG,
			ToolTip: "邻页 · Kinpage",
			Menu:    buildTrayMenu(nativeApp.Locale()),
		}); err == nil {
			tray = createdTray
			tray.OnClick(showMainWindow)
			refreshTrayMenu := func(loc nativeui.Locale) {
				if tray != nil {
					tray.SetMenu(buildTrayMenu(loc))
				}
			}
			nativeApp.SetOnLocaleChange(refreshTrayMenu)
			nativeApp.SetOnDesktopPetChange(func() {
				refreshTrayMenu(nativeApp.Locale())
			})
		} else {
			log.Printf("system tray unavailable: %v", err)
		}
		nativeApp.SetAppIconHandlers(func(icon resources.AppIcon) error {
			png := icon.PNG()
			if err := win.SetIcon(png); err != nil {
				return err
			}
			if tray != nil {
				return tray.SetIcon(png, false)
			}
			return nil
		}, desktopshortcut.Apply)

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
			quitApp()
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
