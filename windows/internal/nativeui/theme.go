package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"windows/internal/store"
)

const macPlanetFont = "Segoe UI"

type macPalette struct {
	dark            bool
	sidebar         ui.Color
	canvas          ui.Color
	subtle          ui.Color
	footer          ui.Color
	hover           ui.Color
	active          ui.Color
	sidebarSelected ui.Color
	searchBg        ui.Color
	line            ui.Color
	controlLine     ui.Color
	ink             ui.Color
	secondary       ui.Color
	muted           ui.Color
	accent          ui.Color
	accentHover     ui.Color
	accentActive    ui.Color
	listSelected    ui.Color
	listSelMuted    ui.Color
	selection       ui.Color
	orange          ui.Color
	yellow          ui.Color
	green           ui.Color
	amber           ui.Color
	danger          ui.Color
	pillBg          ui.Color
	bottomBg        ui.Color
	shadowColor     ui.Color
	popoverShadow   ui.Color
}

func paletteFor(dark bool) macPalette {
	if dark {
		return macPalette{
			dark:            true,
			sidebar:         ui.Hex("#232325"),
			canvas:          ui.Hex("#1c1c1e"),
			subtle:          ui.Hex("#2c2c2e"),
			footer:          ui.Hex("#242426"),
			hover:           ui.RGBA(255, 255, 255, 0.07),
			active:          ui.RGBA(255, 255, 255, 0.12),
			sidebarSelected: ui.RGBA(255, 255, 255, 0.13),
			searchBg:        ui.RGBA(255, 255, 255, 0.06),
			line:            ui.Hex("#343437"),
			controlLine:     ui.Hex("#48484a"),
			ink:             ui.Hex("#f5f5f7"),
			secondary:       ui.Hex("#c7c7cc"),
			muted:           ui.Hex("#8e8e93"),
			accent:          ui.Hex("#0a84ff"),
			accentHover:     ui.Hex("#3399ff"),
			accentActive:    ui.Hex("#0070e0"),
			listSelected:    ui.Hex("#0a66e8"),
			listSelMuted:    ui.RGBA(255, 255, 255, 0.82),
			selection:       ui.RGBA(10, 132, 255, 0.20),
			orange:          ui.Hex("#ff9500"),
			yellow:          ui.Hex("#f5b800"),
			green:           ui.Hex("#34c759"),
			amber:           ui.Hex("#f0b159"),
			danger:          ui.Hex("#ff453a"),
			pillBg:          ui.RGBA(255, 255, 255, 0.10),
			bottomBg:        ui.RGBA(255, 255, 255, 0.025),
			shadowColor:     ui.RGBA(0, 0, 0, 0.55),
			popoverShadow:   ui.RGBA(0, 0, 0, 0.45),
		}
	}
	return macPalette{
		dark:            false,
		sidebar:         ui.Hex("#ececec"),
		canvas:          ui.Hex("#ffffff"),
		subtle:          ui.Hex("#f5f5f7"),
		footer:          ui.Hex("#f7f7f9"),
		hover:           ui.RGBA(0, 0, 0, 0.05),
		active:          ui.RGBA(0, 0, 0, 0.09),
		sidebarSelected: ui.RGBA(0, 0, 0, 0.085),
		searchBg:        ui.RGBA(0, 0, 0, 0.045),
		line:            ui.Hex("#e2e2e5"),
		controlLine:     ui.Hex("#c7c7cc"),
		ink:             ui.Hex("#1d1d1f"),
		secondary:       ui.Hex("#515154"),
		muted:           ui.Hex("#86868b"),
		accent:          ui.Hex("#007aff"),
		accentHover:     ui.Hex("#0066d6"),
		accentActive:    ui.Hex("#0055b3"),
		listSelected:    ui.Hex("#0a66e8"),
		listSelMuted:    ui.RGBA(255, 255, 255, 0.82),
		selection:       ui.Hex("#e8f2ff"),
		orange:          ui.Hex("#ff9500"),
		yellow:          ui.Hex("#f5b800"),
		green:           ui.Hex("#34c759"),
		amber:           ui.Hex("#b25000"),
		danger:          ui.Hex("#ff3b30"),
		pillBg:          ui.RGBA(0, 0, 0, 0.065),
		bottomBg:        ui.RGBA(0, 0, 0, 0.025),
		shadowColor:     ui.RGBA(0, 0, 0, 0.14),
		popoverShadow:   ui.RGBA(0, 0, 0, 0.12),
	}
}

func applyMacPlanetTheme(c *ui.Context) macPalette {
	base := c.Theme()
	pal := paletteFor(base.Dark)
	t := *base
	t.Background = pal.canvas
	t.Surface = pal.canvas
	t.SurfaceHover = pal.subtle
	t.SurfacePressed = pal.sidebarSelected
	t.Border = pal.line
	t.Text = pal.ink
	t.TextMuted = pal.muted
	t.Accent = pal.accent
	t.AccentHover = pal.accentHover
	t.AccentPressed = pal.accentActive
	t.AccentText = ui.Hex("#ffffff")
	t.Selection = pal.selection
	t.Focus = pal.accent
	t.Success = pal.green
	t.Warning = pal.amber
	t.Danger = pal.danger
	t.Radius = 6
	c.SetTheme(&t)
	return pal
}

func paperEditorTheme(c *ui.Context, pal macPalette) *editor.Theme {
	ct := editor.DefaultTheme(c.Theme())
	ct.Background = pal.canvas
	ct.Text = pal.ink
	ct.Cursor = pal.accent
	ct.Selection = pal.selection
	ct.ActiveLine = ui.Color{}
	return ct
}

// Exact 6-color Mac Planet site avatar cycle from mac-planet.css:350-356.
var macSiteAvatarColors = []ui.Color{
	ui.Hex("#6b7de0"),
	ui.Hex("#e26d5a"),
	ui.Hex("#4da89b"),
	ui.Hex("#e09f3e"),
	ui.Hex("#9a72c8"),
	ui.Hex("#4aa3b5"),
}

func firstRuneLabel(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "P"
	}
	for _, r := range trimmed {
		return strings.ToUpper(string(r))
	}
	return "P"
}

func avatarCircleIndexed(c *ui.Context, name string, index int, size float32) {
	col := macSiteAvatarColors[index%len(macSiteAvatarColors)]
	fontSize := size * 0.5
	if fontSize < 10 {
		fontSize = 10
	}
	ui.Box(c).Size(size, size).Radius(size / 2).Background(col).Center().Shrink(0).Children(func() {
		ui.Text(c, firstRuneLabel(name)).Font(macPlanetFont).FontSize(fontSize).FontWeight(600).TextColor(ui.Hex("#ffffff")).NoWrap()
	})
}

func (a *App) siteAvatar(c *ui.Context, site store.Site, index int, size float32) {
	if bmp := a.bitmapForDataURL(site.AvatarPNG); bmp != nil {
		ui.Box(c).Size(size, size).Radius(size / 2).Clip().Shrink(0).Children(func() {
			ui.Image(c, bmp).Size(size, size)
		})
		return
	}
	avatarCircleIndexed(c, site.Name, index, size)
}

func (a *App) subAvatar(c *ui.Context, sub store.Subscription, size float32, pal macPalette) {
	if bmp := a.bitmapForDataURL(sub.Feed.IconPNG); bmp != nil {
		ui.Box(c).Size(size, size).Radius(5).Clip().Shrink(0).Children(func() {
			ui.Image(c, bmp).Size(size, size)
		})
		return
	}
	ui.Box(c).Size(size, size).Radius(5).Background(pal.pillBg).Center().Shrink(0).Children(func() {
		ui.Icon(c, iconGlobe).Size(size*0.64, size*0.64).TextColor(pal.secondary)
	})
}

func sidebarNavRow(c *ui.Context, key, label string, selected bool, badge int, leading func(), pal macPalette) bool {
	bg := ui.Color{}
	if selected {
		bg = pal.sidebarSelected
	}
	btn := ui.Button(c.Key(key), "").Label(label).AlignSelf(ui.Stretch).MinHeight(30).
		Background(bg).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(5).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).AlignSelf(ui.Stretch).Children(func() {
				if leading != nil {
					leading()
				}
				txt := ui.Text(c, label).Font(macPlanetFont).FontSize(13).TextColor(pal.ink).Grow(1).NoWrap()
				if selected {
					txt.FontWeight(500)
				}
				if badge > 0 {
					ui.Box(c).MinWidth(20).PaddingX(6).PaddingY(1).Radius(10).Background(pal.pillBg).Center().Shrink(0).Children(func() {
						ui.Text(c, fmt.Sprintf("%d", badge)).Font(macPlanetFont).FontSize(11).FontWeight(500).TextColor(pal.secondary).NoWrap()
					})
				}
			})
		})
	return btn.Clicked()
}

func macIconButton(c *ui.Context, key string, icon *ui.SVG, boxSize, iconSize float32, tint ui.Color, ariaLabel string) bool {
	return ui.Button(c.Key(key), "").Label(ariaLabel).Size(boxSize, boxSize).
		Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).Center().
		Children(func() {
			ui.Icon(c, icon).Size(iconSize, iconSize).TextColor(tint)
		}).Clicked()
}

func ghostIconButton(c *ui.Context, key, text, ariaLabel string, pal macPalette) bool {
	return ui.Button(c.Key(key), "").Label(ariaLabel).MinHeight(28).
		Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(4).
		Children(func() {
			ui.Text(c, text).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).NoWrap()
		}).Clicked()
}

func macPrimaryButton(c *ui.Context, key, label string, pal macPalette) bool {
	return ui.Button(c.Key(key), "").Label(label).MinHeight(30).
		Background(pal.accent).Border(0, ui.Color{}).Radius(6).PaddingX(14).PaddingY(5).
		Shadow(0, 0.5, 1, 0, ui.RGBA(0, 122, 255, 0.25)).
		Children(func() {
			ui.Text(c, label).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(ui.Hex("#ffffff")).NoWrap()
		}).Clicked()
}

func macSecondaryButton(c *ui.Context, key, label string, pal macPalette) bool {
	return ui.Button(c.Key(key), "").Label(label).MinHeight(30).
		Background(pal.canvas).Border(0.5, pal.controlLine).Radius(6).PaddingX(12).PaddingY(5).
		Shadow(0, 0.5, 1, 0, ui.RGBA(0, 0, 0, 0.05)).
		Children(func() {
			ui.Text(c, label).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.ink).NoWrap()
		}).Clicked()
}

// macSegmented mirrors .view-switch in mac-planet.css:870-886:
// subtle track, 0.5px border, 7px radius, pure white shadowed active pill.
func macSegmented(c *ui.Context, key string, selected *int, labels []string, pal macPalette) {
	ui.Row(c.Key(key)).Padding(2).Gap(2).Background(pal.subtle).Border(0.5, pal.line).Radius(7).Shrink(0).Children(func() {
		for i, lbl := range labels {
			idx := i
			active := *selected == idx
			bg := ui.Color{}
			fg := pal.muted
			btn := ui.Button(c.Key(fmt.Sprintf("%s-%d", key, idx)), "").Label(lbl).
				Border(0, ui.Color{}).Radius(5).PaddingX(10).PaddingY(4)
			if active {
				bg = pal.canvas
				fg = pal.ink
				btn.Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.10))
			}
			btn.Background(bg).Children(func() {
				txt := ui.Text(c, lbl).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(fg).NoWrap()
				_ = txt
			})
			if btn.Clicked() {
				*selected = idx
			}
		}
	})
}

func hairlineH(c *ui.Context, pal macPalette) {
	ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.line)
}

func excerptText(md string, maxRunes int) string {
	var lines []string
	for _, raw := range strings.Split(md, "\n") {
		s := strings.TrimSpace(raw)
		s = strings.TrimLeft(s, "#>-* ")
		s = strings.ReplaceAll(s, "**", "")
		s = strings.ReplaceAll(s, "`", "")
		if s != "" {
			lines = append(lines, s)
		}
		if len(lines) >= 2 {
			break
		}
	}
	joined := strings.Join(lines, " ")
	runes := []rune(joined)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return joined
}

func formatShortDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	lt := t.Local()
	return fmt.Sprintf("%d月%d日", int(lt.Month()), lt.Day())
}

func formatSlashDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006/01/02")
}
