package resources

import (
	"embed"
	"slices"
)

// AppIcon identifies an icon for the window, tray, and desktop shortcut.
type AppIcon string

const ClassicIcon AppIcon = "classic"

var appIcons = [...]AppIcon{ClassicIcon, "dawn", "tide", "orbit"}

//go:embed icon.png icon.ico app-icons/*.png app-icons/*.ico
var appIconFiles embed.FS

func AppIcons() []AppIcon { return slices.Clone(appIcons[:]) }

func NormalizeAppIcon(icon AppIcon) AppIcon {
	for _, candidate := range appIcons {
		if icon == candidate {
			return icon
		}
	}
	return ClassicIcon
}

func (icon AppIcon) PNG() []byte { return icon.read(".png") }
func (icon AppIcon) ICO() []byte { return icon.read(".ico") }

func (icon AppIcon) read(ext string) []byte {
	icon = NormalizeAppIcon(icon)
	name := "icon"
	if icon != ClassicIcon {
		name = "app-icons/" + string(icon)
	}
	data, err := appIconFiles.ReadFile(name + ext)
	if err != nil {
		panic(err)
	}
	return data
}
