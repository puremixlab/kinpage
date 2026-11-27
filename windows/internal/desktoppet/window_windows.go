package desktoppet

import (
	"fmt"
	"image"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const Supported = true

const (
	wsPopup          = 0x80000000
	wsExToolWindow   = 0x00000080
	wsExLayered      = 0x00080000
	wsExNoActivate   = 0x08000000
	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoZOrder      = 0x0004
	swpNoActivate    = 0x0010
	swpShowWindow    = 0x0040
	wmEraseBkgnd     = 0x0014
	wmClose          = 0x0010
	wmSettingChange  = 0x001a
	wmCancelMode     = 0x001f
	wmMouseActivate  = 0x0021
	wmDisplayChange  = 0x007e
	wmNCDestroy      = 0x0082
	wmTimer          = 0x0113
	wmLButtonDown    = 0x0201
	wmLButtonUp      = 0x0202
	wmDPIChanged     = 0x02e0
	maNoActivate     = 3
	monitorPrimary   = 1
	monitorNearest   = 2
	petTimer         = 1
	petTickMS        = 32
	spiGetAnimations = 0x1042 // SPI_GETCLIENTAREAANIMATION
	smSwapButton     = 23
	vkLeftButton     = 0x01
	vkRightButton    = 0x02
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	registerClass        = user32.NewProc("RegisterClassExW")
	createWindow         = user32.NewProc("CreateWindowExW")
	destroyWindow        = user32.NewProc("DestroyWindow")
	defWindowProc        = user32.NewProc("DefWindowProcW")
	setWindowPos         = user32.NewProc("SetWindowPos")
	setTimer             = user32.NewProc("SetTimer")
	killTimer            = user32.NewProc("KillTimer")
	loadCursor           = user32.NewProc("LoadCursorW")
	getCursorPos         = user32.NewProc("GetCursorPos")
	getAsyncKeyState     = user32.NewProc("GetAsyncKeyState")
	getSystemMetrics     = user32.NewProc("GetSystemMetrics")
	monitorFromWindow    = user32.NewProc("MonitorFromWindow")
	monitorFromRect      = user32.NewProc("MonitorFromRect")
	getMonitorInfo       = user32.NewProc("GetMonitorInfoW")
	getDPIForWindow      = user32.NewProc("GetDpiForWindow")
	systemParametersInfo = user32.NewProc("SystemParametersInfoW")

	petClassName = windows.StringToUTF16Ptr("KinpageDesktopPet")
	petTitle     = windows.StringToUTF16Ptr("Kinpage Desktop Pet")
	petWindows   = make(map[uintptr]*Window) // UI thread only
	petClass     = sync.OnceValues(registerPetClass)
)

type winPoint struct{ x, y int32 }

func winPointOf(p image.Point) winPoint { return winPoint{int32(p.X), int32(p.Y)} }

type winRect struct{ left, top, right, bottom int32 }

type monitorInfo struct {
	size          uint32
	monitor, work winRect
	flags         uint32
}

type windowClass struct {
	size, style             uint32
	proc                    uintptr
	classExtra, windowExtra int32
	instance, icon, cursor  uintptr
	background, menu, name  uintptr
	smallIcon               uintptr
}

type pointerPress struct {
	cursor, origin image.Point
	key            uintptr
	dragged        bool
}

// Window is a native desktop overlay. It and its callback run on the UI thread.
type Window struct {
	handle       uintptr
	position     image.Point
	sprite       *sprite
	surface      *surface
	motion       motion
	lastPose     pose
	press        pointerPress
	layoutNeeded bool
	onMove       func(image.Point)
}

func registerPetClass() (windows.Handle, error) {
	var instance windows.Handle
	if err := windows.GetModuleHandleEx(windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT, nil, &instance); err != nil {
		return 0, err
	}
	const idcHand = 32649
	cursor, _, _ := loadCursor.Call(0, idcHand)
	class := windowClass{
		size: uint32(unsafe.Sizeof(windowClass{})), proc: windows.NewCallback(petWindowProc),
		instance: uintptr(instance), cursor: cursor, name: uintptr(unsafe.Pointer(petClassName)),
	}
	if atom, _, err := registerClass.Call(uintptr(unsafe.Pointer(&class))); atom == 0 {
		return 0, fmt.Errorf("RegisterClassExW: %w", err)
	}
	return instance, nil
}

// Open shows Amo near the bottom right of the anchor window's monitor, or at
// the saved desktop position. A completed drag reports the new position.
func Open(anchor uintptr, saved *image.Point, onMove func(image.Point)) (*Window, error) {
	instance, err := petClass()
	if err != nil {
		return nil, err
	}
	work, err := workArea(anchor, saved)
	if err != nil {
		return nil, err
	}
	start := work.Min
	if saved != nil {
		start = clampPosition(*saved, image.Pt(1, 1), work)
	}
	hwnd, _, err := createWindow.Call(wsExToolWindow|wsExLayered|wsExNoActivate,
		uintptr(unsafe.Pointer(petClassName)), uintptr(unsafe.Pointer(petTitle)), wsPopup,
		uintptr(start.X), uintptr(start.Y), 1, 1, 0, 0, uintptr(instance), 0)
	if hwnd == 0 {
		return nil, fmt.Errorf("CreateWindowExW: %w", err)
	}
	w := &Window{
		handle: hwnd, position: start, onMove: onMove,
		motion: motion{started: time.Now(), reduced: animationsDisabled()},
	}
	petWindows[hwnd] = w

	dpi := windowDPI(hwnd)
	margin := (18*dpi + 48) / 96
	position := work.Max.Sub(petSize(dpi, work)).Sub(image.Pt(margin, margin))
	if saved != nil {
		position = *saved
	}
	if err := w.place(position, dpi, work); err != nil {
		w.Close()
		return nil, err
	}
	if ok, _, err := setTimer.Call(hwnd, petTimer, petTickMS, 0); ok == 0 {
		w.Close()
		return nil, fmt.Errorf("SetTimer: %w", err)
	}
	const hwndTopmost = ^uintptr(0)
	if ok, _, err := setWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0,
		swpNoSize|swpNoMove|swpNoActivate|swpShowWindow); ok == 0 {
		w.Close()
		return nil, fmt.Errorf("show desktop pet: %w", err)
	}
	return w, nil
}

// Close destroys the overlay and releases its timer and drawing resources.
func (w *Window) Close() {
	if w.handle != 0 {
		destroyWindow.Call(w.handle)
	}
}

func petWindowProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	if w := petWindows[hwnd]; w != nil {
		switch msg {
		case wmMouseActivate:
			return maNoActivate
		case wmEraseBkgnd:
			return 1
		case wmClose:
			return 0 // Visibility is controlled by the app preference.
		case wmDisplayChange, wmDPIChanged:
			w.layoutNeeded = true
		case wmSettingChange:
			w.motion.reduced = animationsDisabled()
			w.layoutNeeded = true
		case wmTimer:
			if wp == petTimer {
				w.tick(time.Now())
				return 0
			}
		case wmLButtonDown:
			key := uintptr(vkLeftButton)
			if swapped, _, _ := getSystemMetrics.Call(smSwapButton); swapped != 0 {
				key = vkRightButton
			}
			w.press = pointerPress{cursor: w.screenPoint(lp), origin: w.position, key: key}
			return 0
		case wmLButtonUp:
			w.release(w.screenPoint(lp), time.Now())
			return 0
		case wmCancelMode:
			w.finishDrag()
			w.press = pointerPress{}
		case wmNCDestroy:
			delete(petWindows, hwnd)
			killTimer.Call(hwnd, petTimer)
			if w.surface != nil {
				w.surface.close()
				w.surface = nil
			}
			w.handle = 0
		}
	}
	result, _, _ := defWindowProc.Call(hwnd, uintptr(msg), wp, lp)
	return result
}

func (w *Window) screenPoint(lparam uintptr) image.Point {
	return w.position.Add(image.Pt(int(int16(lparam)), int(int16(lparam>>16))))
}

func (w *Window) tick(now time.Time) {
	var point winPoint
	ok, _, _ := getCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	cursor := image.Pt(int(point.x), int(point.y))
	if ok != 0 && w.press.key != 0 {
		// The overlay leaves focus and capture with the foreground app. Poll
		// the pressed button so a drag continues off the pet's visible pixels.
		if state, _, _ := getAsyncKeyState.Call(w.press.key); state&0x8000 != 0 {
			w.drag(cursor)
		} else {
			w.release(cursor, now)
		}
	}
	if w.layoutNeeded && w.press.key == 0 {
		w.layoutNeeded = false
		if work, err := workArea(w.handle, nil); err == nil {
			_ = w.place(w.position, windowDPI(w.handle), work)
		}
	}
	target := gazeRest
	if ok != 0 {
		target = gazeToward(cursor, image.Rectangle{Min: w.position, Max: w.position.Add(w.surface.size)})
	}
	next := w.motion.frame(now, target)
	if next != w.lastPose {
		if err := w.surface.paint(w.handle, w.position, w.sprite.paint(next)); err == nil {
			w.lastPose = next
		}
	}
}

func (w *Window) drag(cursor image.Point) {
	delta := cursor.Sub(w.press.cursor)
	threshold := max(3, w.surface.size.X/38)
	if !w.press.dragged && delta.In(image.Rect(1-threshold, 1-threshold, threshold, threshold)) {
		return
	}
	w.press.dragged = true
	_ = w.move(w.press.origin.Add(delta))
}

func (w *Window) release(cursor image.Point, now time.Time) {
	if w.press.key == 0 {
		return
	}
	w.drag(cursor)
	if w.press.dragged {
		w.finishDrag()
	} else {
		local := cursor.Sub(w.position)
		if w.sprite.frame.RGBAAt(local.X, local.Y).A != 0 {
			w.motion.tapped = now
		}
	}
	w.press = pointerPress{}
}

func (w *Window) finishDrag() {
	if !w.press.dragged {
		return
	}
	if work, err := workArea(w.handle, nil); err == nil {
		_ = w.place(w.position, windowDPI(w.handle), work)
	}
	if w.onMove != nil && w.position != w.press.origin {
		w.onMove(w.position)
	}
}

func (w *Window) place(position image.Point, dpi int, work image.Rectangle) error {
	size := petSize(dpi, work)
	position = clampPosition(position, size, work)
	if w.surface != nil && w.surface.size == size {
		return w.move(position)
	}
	spr, err := newSprite(size)
	if err != nil {
		return err
	}
	buf, err := newSurface(size)
	if err != nil {
		return err
	}
	if err := buf.paint(w.handle, position, spr.paint(w.lastPose)); err != nil {
		buf.close()
		return err
	}
	if w.surface != nil {
		w.surface.close()
	}
	w.sprite, w.surface, w.position = spr, buf, position
	return nil
}

func (w *Window) move(position image.Point) error {
	if position == w.position {
		return nil
	}
	ok, _, err := setWindowPos.Call(w.handle, 0, uintptr(position.X), uintptr(position.Y), 0, 0,
		swpNoActivate|swpNoZOrder|swpNoSize)
	if ok == 0 {
		return fmt.Errorf("SetWindowPos: %w", err)
	}
	w.position = position
	return nil
}

func windowDPI(hwnd uintptr) int {
	if getDPIForWindow.Find() == nil {
		if dpi, _, _ := getDPIForWindow.Call(hwnd); dpi != 0 {
			return int(dpi)
		}
	}
	return 96
}

func petSize(dpi int, work image.Rectangle) image.Point {
	width := min((152*dpi+48)/96, work.Dx(), work.Dy()*spriteCrop.Dx()/spriteCrop.Dy())
	return image.Pt(width, (width*spriteCrop.Dy()+spriteCrop.Dx()/2)/spriteCrop.Dx())
}

func workArea(hwnd uintptr, position *image.Point) (image.Rectangle, error) {
	var monitor uintptr
	if position == nil {
		fallback := uintptr(monitorNearest)
		if hwnd == 0 {
			fallback = monitorPrimary
		}
		monitor, _, _ = monitorFromWindow.Call(hwnd, fallback)
	} else {
		x, y := int32(position.X), int32(position.Y)
		rect := winRect{x, y, x, y}
		monitor, _, _ = monitorFromRect.Call(uintptr(unsafe.Pointer(&rect)), monitorNearest)
	}
	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if ok, _, err := getMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); ok == 0 {
		return image.Rectangle{}, fmt.Errorf("GetMonitorInfoW: %w", err)
	}
	work := image.Rect(int(info.work.left), int(info.work.top), int(info.work.right), int(info.work.bottom))
	if work.Empty() {
		return image.Rectangle{}, fmt.Errorf("monitor work area is empty")
	}
	return work, nil
}

func animationsDisabled() bool {
	var enabled uint32
	ok, _, _ := systemParametersInfo.Call(spiGetAnimations, 0, uintptr(unsafe.Pointer(&enabled)), 0)
	return ok != 0 && enabled == 0
}
