//go:build !windows

package services

// SetPreventSleep is a no-op on non-Windows platforms.
func SetPreventSleep(_ bool) error {
	return nil
}
