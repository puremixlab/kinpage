//go:build !windows

package services

// SendDesktopNotification is a no-op on non-Windows platforms.
func SendDesktopNotification(_, _ string) {}
