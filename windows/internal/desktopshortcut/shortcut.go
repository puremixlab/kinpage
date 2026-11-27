package desktopshortcut

import "errors"

// ErrOccupied means that the desktop already contains a different Kinpage.lnk.
var ErrOccupied = errors.New("Kinpage.lnk is already in use")
