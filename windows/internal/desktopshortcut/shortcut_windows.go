package desktopshortcut

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"windows/resources"
)

const shortcutName = "Kinpage.lnk"

var (
	getCurrentPackageFamilyName = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentPackageFamilyName")
	shell32                     = windows.NewLazySystemDLL("shell32.dll")
	shChangeNotify              = shell32.NewProc("SHChangeNotify")
)

// Apply updates the current user's desktop shortcut. If create is false, a
// missing shortcut is left alone and Apply reports false.
func Apply(icon resources.AppIcon, create bool) (bool, error) {
	cleanup, err := initCOM()
	if err != nil {
		return false, err
	}
	defer cleanup()
	desktop, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, windows.KF_FLAG_DONT_VERIFY)
	if err != nil {
		return false, err
	}
	path := filepath.Join(desktop, shortcutName)
	if !create {
		if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
	}
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return false, err
	}
	var name [128]uint16
	length := uint32(len(name))
	code, _, _ := getCurrentPackageFamilyName.Call(uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&name[0])))
	var target shortcutTarget
	iconDir := filepath.Join(local, "Kinpage", "DesktopIcons")
	switch syscall.Errno(code) {
	case 0:
		family := windows.UTF16ToString(name[:])
		// The app ID comes from msix/AppxManifest.template.xml. The family
		// and ID survive updates to the package's versioned install path.
		target.appID = family + "!Kinpage"
		iconDir = filepath.Join(local, "Packages", family, "LocalState", "DesktopIcons")
	case windows.APPMODEL_ERROR_NO_PACKAGE:
		target.exe, err = os.Executable()
		if err != nil {
			return false, err
		}
	default:
		return false, fmt.Errorf("GetCurrentPackageFamilyName: %w", syscall.Errno(code))
	}
	return applyAt(path, iconDir, target, icon, create)
}

func initCOM() (func(), error) {
	runtime.LockOSThread()
	err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED)
	switch err {
	case nil, syscall.Errno(1): // S_FALSE also requires CoUninitialize.
		return func() {
			windows.CoUninitialize()
			runtime.UnlockOSThread()
		}, nil
	case syscall.Errno(0x80010106): // This thread already has a COM apartment.
		return runtime.UnlockOSThread, nil
	default:
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("CoInitializeEx: %w", err)
	}
}

func applyAt(path, iconDir string, target shortcutTarget, icon resources.AppIcon, create bool) (bool, error) {
	before, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if !create {
			return false, nil
		}
	} else if err != nil {
		return false, err
	} else if !before.Mode().IsRegular() {
		return false, ErrOccupied
	}

	link, persist, err := newShellLink()
	if err != nil {
		return false, err
	}
	defer link.close()
	defer persist.close()
	if before == nil {
		if err := link.setTarget(target); err != nil {
			return false, err
		}
	} else {
		if err := persist.load(path); err != nil {
			return false, err
		}
		if !link.matches(target) {
			return false, ErrOccupied
		}
	}
	iconPath, err := ensureIconFile(iconDir, icon)
	if err != nil {
		return false, err
	}
	if err := link.setIcon(iconPath); err != nil {
		return false, err
	}
	if err := publishShortcut(path, persist, before); err != nil {
		return false, err
	}
	return true, nil
}

func ensureIconFile(dir string, icon resources.AppIcon) (string, error) {
	icon = resources.NormalizeAppIcon(icon)
	data := icon.ICO()
	sum := sha256.Sum256(data)
	// Explorer caches by path. A new build of an icon gets a new filename.
	path := filepath.Join(dir, fmt.Sprintf("%s-%x.ico", icon, sum[:8]))
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) {
		return path, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, ".icon-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return "", err
	}
	return path, os.Rename(f.Name(), path)
}

func publishShortcut(path string, persist *persistFile, before os.FileInfo) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".kinpage-*.lnk")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Close(); err != nil {
		return err
	}
	if err := persist.save(f.Name()); err != nil {
		return err
	}
	var flags uint32
	event := uintptr(0x00000002) // SHCNE_CREATE
	if before != nil {
		current, err := os.Lstat(path)
		if err != nil || !os.SameFile(before, current) || !before.ModTime().Equal(current.ModTime()) || before.Size() != current.Size() {
			return ErrOccupied
		}
		flags = windows.MOVEFILE_REPLACE_EXISTING
		event = 0x00002000 // SHCNE_UPDATEITEM
	}
	from, err := windows.UTF16PtrFromString(f.Name())
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, flags); err != nil {
		if flags == 0 && (errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS)) {
			return ErrOccupied
		}
		return err
	}
	const shcnfPathW = 0x0005
	shChangeNotify.Call(event, shcnfPathW, uintptr(unsafe.Pointer(to)), 0)
	return nil
}
