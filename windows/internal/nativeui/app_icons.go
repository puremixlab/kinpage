package nativeui

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/desktopshortcut"
	"windows/resources"
)

func (a *App) AppIcon() resources.AppIcon {
	return resources.NormalizeAppIcon(a.appIcon)
}

func (a *App) SetAppIconHandlers(apply func(resources.AppIcon) error, shortcut func(resources.AppIcon, bool) (bool, error)) {
	a.applyAppIcon = apply
	a.applyShortcut = shortcut
}

func (a *App) viewAppIcons(c *ui.Context) {
	pal := a.pal
	icons := resources.AppIcons()
	if a.appIconPreviews == nil {
		a.appIconPreviews = make(map[resources.AppIcon]*ui.Bitmap)
		for _, icon := range icons {
			a.appIconPreviews[icon], _ = ui.DecodeBitmap(icon.PNG())
		}
	}
	macHighlightCard(c, pal, a.toolsModal.highlightID == "global-app-icon", func() {
		title := a.t("settings.appIconTitle")
		ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
		current := a.AppIcon()
		choice := current
		ui.Row(c).Role(ui.RoleRadioGroup).Label(title).Wrap().Gap(8).AlignSelf(ui.Stretch).Children(func() {
			for _, icon := range icons {
				label := a.t("settings.appIcon." + string(icon))
				tile := ui.RadioBase(c.Key("settings-app-icon-"+string(icon)), &choice, icon).Label(label).
					MinWidth(88).Padding(10, 12, 8, 12).Radius(8).Center().Border(0.5, pal.prefsCardBorder).
					Cursor(ui.CursorPointer)
				selected := current == icon
				if selected {
					tile.Background(pal.active).Border(0.5, pal.prefsSecondary)
				} else if tile.Hovered() {
					tile.Background(pal.hover)
				}
				tile.Children(func() {
					ui.Column(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Image(c, a.appIconPreviews[icon]).Size(46, 46)
						ui.Text(c, label).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsInk).NoWrap()
					})
					if selected {
						ui.Icon(c, lucide.Check()).Absolute().Top(5).Right(5).Size(12, 12).TextColor(pal.prefsInk)
					}
				})
			}
		})
		if choice != current {
			a.changeAppIcon(choice)
		}
		if a.applyShortcut != nil {
			ui.Row(c).Children(func() {
				if macSecondaryButton(c, "settings-desktop-shortcut", a.t("settings.desktopShortcut"), pal) {
					a.syncDesktopShortcut(true)
				}
			})
		}
	})
}

func (a *App) changeAppIcon(icon resources.AppIcon) {
	previous := a.AppIcon()
	err := a.applyAppIcon(icon)
	if err == nil {
		a.appIcon = icon
		err = a.maybeSaveLayoutPrefs()
	}
	if err != nil {
		a.appIcon = previous
		err = errors.Join(err, a.applyAppIcon(previous))
		a.setError(a.tf("settings.appIconFailed", err.Error()))
		return
	}
	a.syncDesktopShortcut(false)
}

func (a *App) syncDesktopShortcut(create bool) {
	if a.applyShortcut == nil {
		return
	}
	_, err := a.applyShortcut(a.AppIcon(), create)
	switch {
	case errors.Is(err, desktopshortcut.ErrOccupied):
		a.setError(a.t("settings.desktopShortcutOccupied"))
	case err != nil:
		a.setError(a.tf("settings.desktopShortcutFailed", err.Error()))
	case create:
		a.setStatus(a.t("settings.desktopShortcutReady"))
	}
}
