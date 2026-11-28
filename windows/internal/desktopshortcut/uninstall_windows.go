package desktopshortcut

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"windows/resources"
)

const maxShortcutSize = 1 << 20

// RemoveInstalled removes the current user's Kinpage shortcut if both its
// executable and its managed icon identify the desktop installation.
func RemoveInstalled(exe string) error {
	cleanup, err := initCOM()
	if err != nil {
		return err
	}
	defer cleanup()
	desktop, err := windows.KnownFolderPath(windows.FOLDERID_Desktop, windows.KF_FLAG_DONT_VERIFY)
	if err != nil {
		return err
	}
	local, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, windows.KF_FLAG_DEFAULT)
	if err != nil {
		return err
	}
	return removeDesktopShortcutAt(filepath.Join(desktop, shortcutName), filepath.Join(local, "Kinpage", "DesktopIcons"), exe)
}

func removeDesktopShortcutAt(path, iconDir, exe string) error {
	hash, err := installedShortcutHash(path, iconDir, exe)
	if err != nil || hash == "" {
		return err
	}
	return RemoveIfHashMatches(path, hash)
}

func installedShortcutHash(path, iconDir, exe string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxShortcutSize {
		return "", nil
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	f := os.NewFile(uintptr(h), path)
	if f == nil {
		windows.CloseHandle(h)
		return "", errors.New("cannot wrap shortcut handle")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) || opened.Size() > maxShortcutSize {
		return "", err
	}
	// Keep the same file in place while COM reads its target and icon path.
	link, persist, err := newShellLink()
	if err != nil {
		return "", err
	}
	defer link.close()
	defer persist.close()
	if err := persist.load(path); err != nil || !link.matches(shortcutTarget{exe: exe}) {
		return "", nil
	}
	var iconPath [32768]uint16
	var index int32
	hr, _, _ := syscall.SyscallN(link.vt.getIconLocation, uintptr(unsafe.Pointer(link)),
		uintptr(unsafe.Pointer(&iconPath[0])), uintptr(len(iconPath)), uintptr(unsafe.Pointer(&index)))
	if hr != 0 || index != 0 || !managedIcon(windows.UTF16ToString(iconPath[:]), iconDir) {
		return "", nil
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func managedIcon(path, dir string) bool {
	if !sameExecutablePath(filepath.Dir(path), dir) {
		return false
	}
	name, ok := strings.CutSuffix(strings.ToLower(filepath.Base(path)), ".ico")
	if !ok {
		return false
	}
	variant, digest, ok := strings.Cut(name, "-")
	icon := resources.AppIcon(variant)
	if !ok || resources.NormalizeAppIcon(icon) != icon || len(digest) != 16 {
		return false
	}
	// Recognize both the current icon and an intact icon from an older build.
	current := sha256.Sum256(icon.ICO())
	if hex.EncodeToString(current[:8]) == digest {
		return true
	}
	const maxIconSize = 4 << 20
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxIconSize {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return false
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, maxIconSize+1))
	return err == nil && n == info.Size() && hex.EncodeToString(h.Sum(nil)[:8]) == digest
}

// RemoveIfHashMatches hashes and deletes through the same exclusive handle.
func RemoveIfHashMatches(path, want string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.DELETE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return nil
	}
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(h), path)
	if f == nil {
		windows.CloseHandle(h)
		return errors.New("cannot wrap shortcut handle")
	}
	defer f.Close()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return err
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 ||
		info.FileSizeHigh != 0 || info.FileSizeLow > maxShortcutSize {
		return nil
	}
	digest := sha256.New()
	if _, err := io.Copy(digest, f); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(digest.Sum(nil)), want) {
		return nil
	}
	flag := byte(1)
	return windows.SetFileInformationByHandle(h, windows.FileDispositionInfo, &flag, 1)
}
