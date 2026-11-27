//go:build !windows

package desktoppet

import (
	"errors"
	"image"
)

const Supported = false

type Window struct{}

func Open(uintptr, *image.Point, func(image.Point)) (*Window, error) {
	return nil, errors.New("desktop pet requires Windows")
}

func (*Window) Close() {}
