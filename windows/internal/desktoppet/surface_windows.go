package desktoppet

import (
	"fmt"
	"image"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gdi32              = windows.NewLazySystemDLL("gdi32.dll")
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	selectObject       = gdi32.NewProc("SelectObject")
	deleteObject       = gdi32.NewProc("DeleteObject")
	deleteDC           = gdi32.NewProc("DeleteDC")
	updateLayered      = user32.NewProc("UpdateLayeredWindow")
)

type bitmapInfoHeader struct {
	size, width, height   int32
	planes, bitCount      uint16
	compression, imageLen uint32
	xPixelsPerMeter       int32
	yPixelsPerMeter       int32
	colors, colorsUsed    uint32
}

type surface struct {
	size              image.Point
	dc, bitmap, saved uintptr
	pixels            []byte
}

func newSurface(size image.Point) (*surface, error) {
	dc, _, err := createCompatibleDC.Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %w", err)
	}
	header := bitmapInfoHeader{
		size: int32(unsafe.Sizeof(bitmapInfoHeader{})), width: int32(size.X), height: -int32(size.Y),
		planes: 1, bitCount: 32,
	}
	var bits *byte
	bitmap, _, err := createDIBSection.Call(dc, uintptr(unsafe.Pointer(&header)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bitmap == 0 || bits == nil {
		if bitmap != 0 {
			deleteObject.Call(bitmap)
		}
		deleteDC.Call(dc)
		return nil, fmt.Errorf("CreateDIBSection: %w", err)
	}
	saved, _, err := selectObject.Call(dc, bitmap)
	if saved == 0 || saved == ^uintptr(0) {
		deleteObject.Call(bitmap)
		deleteDC.Call(dc)
		return nil, fmt.Errorf("SelectObject: %w", err)
	}
	return &surface{
		size: size, dc: dc, bitmap: bitmap, saved: saved,
		pixels: unsafe.Slice(bits, size.X*size.Y*4),
	}, nil
}

func (s *surface) paint(hwnd uintptr, position image.Point, frame *image.RGBA) error {
	// image.RGBA is already premultiplied. A 32-bit Win32 DIB uses BGRA order.
	for i := 0; i < len(s.pixels); i += 4 {
		s.pixels[i+0] = frame.Pix[i+2]
		s.pixels[i+1] = frame.Pix[i+1]
		s.pixels[i+2] = frame.Pix[i+0]
		s.pixels[i+3] = frame.Pix[i+3]
	}
	dst, size, src := winPointOf(position), winPointOf(s.size), winPoint{}
	// BLENDFUNCTION: AC_SRC_OVER, reserved, full opacity, AC_SRC_ALPHA.
	blend := [4]byte{0, 0, 255, 1}
	const ulwAlpha = 2
	ok, _, err := updateLayered.Call(hwnd, 0, uintptr(unsafe.Pointer(&dst)), uintptr(unsafe.Pointer(&size)),
		s.dc, uintptr(unsafe.Pointer(&src)), 0, uintptr(unsafe.Pointer(&blend)), ulwAlpha)
	if ok == 0 {
		return fmt.Errorf("UpdateLayeredWindow: %w", err)
	}
	return nil
}

func (s *surface) close() {
	selectObject.Call(s.dc, s.saved)
	deleteObject.Call(s.bitmap)
	deleteDC.Call(s.dc)
}
