package desktopshortcut

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type shortcutTarget struct {
	exe   string
	appID string
}

type shellLink struct {
	vt *struct {
		queryInterface, addRef, release              uintptr
		getPath, getIDList, setIDList                uintptr
		getDescription, setDescription               uintptr
		getWorkingDirectory, setWorkingDirectory     uintptr
		getArguments, setArguments                   uintptr
		getHotkey, setHotkey, getShowCmd, setShowCmd uintptr
		getIconLocation, setIconLocation             uintptr
		setRelativePath, resolve, setPath            uintptr
	}
}

type persistFile struct {
	vt *struct {
		queryInterface, addRef, release, getClassID, isDirty uintptr
		load, save, saveCompleted, getCurFile                uintptr
	}
}

var (
	coCreateInstance   = windows.NewLazySystemDLL("ole32.dll").NewProc("CoCreateInstance")
	shParseDisplayName = shell32.NewProc("SHParseDisplayName")
	ilIsEqual          = shell32.NewProc("ILIsEqual")
	clsidShellLink     = windows.GUID{Data1: 0x00021401, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidShellLinkW      = windows.GUID{Data1: 0x000214f9, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPersistFile     = windows.GUID{Data1: 0x0000010b, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
)

func newShellLink() (*shellLink, *persistFile, error) {
	var link *shellLink
	const clsctxInprocServer = 1
	hr, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&link)))
	if err := checkHRESULT("create ShellLink", hr); err != nil {
		return nil, nil, err
	}
	var persist *persistFile
	hr, _, _ = syscall.SyscallN(link.vt.queryInterface, uintptr(unsafe.Pointer(link)),
		uintptr(unsafe.Pointer(&iidPersistFile)), uintptr(unsafe.Pointer(&persist)))
	if err := checkHRESULT("get IPersistFile", hr); err != nil {
		link.close()
		return nil, nil, err
	}
	return link, persist, nil
}

func (link *shellLink) close() { syscall.SyscallN(link.vt.release, uintptr(unsafe.Pointer(link))) }
func (p *persistFile) close()  { syscall.SyscallN(p.vt.release, uintptr(unsafe.Pointer(p))) }

func (link *shellLink) setTarget(target shortcutTarget) error {
	if target.appID != "" {
		pidl, err := appsFolderItem(target.appID)
		if err != nil {
			return err
		}
		defer windows.CoTaskMemFree(pidl)
		hr, _, _ := syscall.SyscallN(link.vt.setIDList, uintptr(unsafe.Pointer(link)), uintptr(pidl))
		return checkHRESULT("set shortcut app", hr)
	}
	exe, err := windows.UTF16PtrFromString(target.exe)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(target.exe))
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(link.vt.setPath, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(exe)))
	if err := checkHRESULT("set shortcut target", hr); err != nil {
		return err
	}
	hr, _, _ = syscall.SyscallN(link.vt.setWorkingDirectory, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(dir)))
	return checkHRESULT("set shortcut working directory", hr)
}

func (link *shellLink) matches(target shortcutTarget) bool {
	if target.appID != "" {
		var actual unsafe.Pointer
		hr, _, _ := syscall.SyscallN(link.vt.getIDList, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(&actual)))
		if hr != 0 || actual == nil {
			return false
		}
		defer windows.CoTaskMemFree(actual)
		expected, err := appsFolderItem(target.appID)
		if err != nil {
			return false
		}
		defer windows.CoTaskMemFree(expected)
		equal, _, _ := ilIsEqual.Call(uintptr(actual), uintptr(expected))
		return equal != 0
	}
	var path [32768]uint16
	hr, _, _ := syscall.SyscallN(link.vt.getPath, uintptr(unsafe.Pointer(link)),
		uintptr(unsafe.Pointer(&path[0])), uintptr(len(path)), 0, 0)
	return hr == 0 && sameExecutablePath(windows.UTF16ToString(path[:]), target.exe)
}

func sameExecutablePath(actual, expected string) bool {
	actual, expected = filepath.Clean(actual), filepath.Clean(expected)
	if !filepath.IsAbs(actual) || !filepath.IsAbs(expected) || !strings.EqualFold(filepath.VolumeName(actual), filepath.VolumeName(expected)) {
		return false
	}
	if strings.EqualFold(actual, expected) {
		return true
	}
	// os.Executable and the shell can spell the same local path differently
	// when Windows provides a short alias for one or more components.
	actualLong, expectedLong := longPath(actual), longPath(expected)
	return actualLong != "" && expectedLong != "" && strings.EqualFold(actualLong, expectedLong)
}

func longPath(path string) string {
	value, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ""
	}
	var buffer [32768]uint16
	n, err := windows.GetLongPathName(value, &buffer[0], uint32(len(buffer)))
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		return ""
	}
	return filepath.Clean(windows.UTF16ToString(buffer[:n]))
}

func appsFolderItem(appID string) (unsafe.Pointer, error) {
	name, err := windows.UTF16PtrFromString(`shell:AppsFolder\` + appID)
	if err != nil {
		return nil, err
	}
	var pidl unsafe.Pointer
	hr, _, _ := shParseDisplayName.Call(uintptr(unsafe.Pointer(name)), 0, uintptr(unsafe.Pointer(&pidl)), 0, 0)
	return pidl, checkHRESULT("find app in AppsFolder", hr)
}

func (link *shellLink) setIcon(path string) error {
	icon, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(link.vt.setIconLocation, uintptr(unsafe.Pointer(link)), uintptr(unsafe.Pointer(icon)), 0)
	return checkHRESULT("set shortcut icon", hr)
}

func (p *persistFile) load(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(p.vt.load, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(name)), 0)
	return checkHRESULT("load shortcut", hr)
}

func (p *persistFile) save(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	hr, _, _ := syscall.SyscallN(p.vt.save, uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(name)), 0)
	return checkHRESULT("save shortcut", hr)
}

func checkHRESULT(operation string, hr uintptr) error {
	if int32(hr) < 0 {
		return fmt.Errorf("%s: HRESULT 0x%08x", operation, uint32(hr))
	}
	return nil
}
