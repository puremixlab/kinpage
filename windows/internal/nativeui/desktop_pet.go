package nativeui

import (
	"image"

	"windows/internal/desktoppet"
)

type desktopPetPrefs struct {
	Enabled    bool  `json:"enabled,omitempty"`
	X          int32 `json:"x,omitempty"`
	Y          int32 `json:"y,omitempty"`
	Positioned bool  `json:"positioned,omitempty"`
}

// RestoreDesktopPet restores the saved visibility once the native main window
// is available. The pet lives independently of the main window's visibility.
func (a *App) RestoreDesktopPet(anchor uintptr) {
	a.desktopPetAnchor = anchor
	if a.DesktopPetAvailable() && a.desktopPetPrefs.Enabled {
		if !a.SetDesktopPetEnabled(true) {
			a.desktopPetPrefs.Enabled = false
			a.maybeSaveLayoutPrefs()
		}
	}
}

func (a *App) DesktopPetAvailable() bool {
	return desktoppet.Supported && a.desktopPetAnchor != 0
}

func (a *App) DesktopPetEnabled() bool { return a.desktopPet != nil }

func (a *App) DesktopPetLabel() string { return a.t("settings.desktopPet") }

func (a *App) SetOnDesktopPetChange(fn func()) { a.onDesktopPetChange = fn }

// SetDesktopPetEnabled applies and saves a visibility change on the UI thread.
func (a *App) SetDesktopPetEnabled(enabled bool) bool {
	if enabled == a.DesktopPetEnabled() {
		return true
	}
	if enabled {
		if !a.DesktopPetAvailable() {
			return false
		}
		var position *image.Point
		if a.desktopPetPrefs.Positioned {
			saved := image.Pt(int(a.desktopPetPrefs.X), int(a.desktopPetPrefs.Y))
			position = &saved
		}
		pet, err := desktoppet.Open(a.desktopPetAnchor, position, a.rememberDesktopPetPosition)
		if err != nil {
			a.setError(a.tf("status.desktopPetError", err.Error()))
			return false
		}
		a.desktopPet = pet
	} else {
		a.CloseDesktopPet()
	}
	a.desktopPetPrefs.Enabled = enabled
	a.maybeSaveLayoutPrefs()
	if a.onDesktopPetChange != nil {
		a.onDesktopPetChange()
	}
	return true
}

func (a *App) rememberDesktopPetPosition(point image.Point) {
	a.desktopPetPrefs.X = int32(point.X)
	a.desktopPetPrefs.Y = int32(point.Y)
	a.desktopPetPrefs.Positioned = true
	a.maybeSaveLayoutPrefs()
}

// CloseDesktopPet tears down the overlay on the UI thread without changing
// the saved visibility, which is restored when the application starts again.
func (a *App) CloseDesktopPet() {
	if a.desktopPet != nil {
		a.desktopPet.Close()
		a.desktopPet = nil
	}
}
