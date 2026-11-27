//go:build !windows

package desktopshortcut

import (
	"errors"
	"windows/resources"
)

func Apply(resources.AppIcon, bool) (bool, error) {
	return false, errors.New("desktop shortcuts require Windows")
}
