//go:build windows

package services

import (
	"fmt"
	"syscall"
)

const (
	esContinuous     uintptr = 0x80000000
	esSystemRequired uintptr = 0x00000001
)

var (
	kernel32DLL             = syscall.NewLazyDLL("kernel32.dll")
	setThreadExecutionState = kernel32DLL.NewProc("SetThreadExecutionState")
)

// SetPreventSleep requests or releases the Windows system sleep inhibition
// state via SetThreadExecutionState so Planet can stay online as an IPFS node.
func SetPreventSleep(enabled bool) error {
	flags := esContinuous
	if enabled {
		flags |= esSystemRequired
	}
	ret, _, err := setThreadExecutionState.Call(flags)
	if ret == 0 {
		if err != nil && err != syscall.Errno(0) {
			return fmt.Errorf("设置防休眠状态失败：%w", err)
		}
		return fmt.Errorf("设置防休眠状态失败")
	}
	return nil
}
