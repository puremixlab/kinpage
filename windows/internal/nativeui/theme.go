package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"windows/internal/store"
)

const macPlanetFont = "Segoe UI"

var rowHoverTransition = ui.ElementTransition{
	Duration: 120 * time.Millisecond,
	Ease:     ui.CubicBezier(0.2, 0, 0, 1),
	Colors:   true,
}

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

	// Plan A (macOS Sequoia) Settings / Preferences layered tokens.
	prefsSidebar         ui.Color
	prefsCanvas          ui.Color
	prefsCard            ui.Color
	prefsCardBorder      ui.Color
	prefsCardShadow      ui.Color
	prefsFooter          ui.Color
	prefsNavActiveBg     ui.Color
	prefsNavActiveShadow ui.Color
	prefsInk             ui.Color
	prefsSecondary       ui.Color
	prefsMuted           ui.Color
}

func paletteFor(dark bool) macPalette {
	if dark {
		return macPalette{
			dark:            true,
			sidebar:         ui.Hex("#232325"),
			canvas:          ui.Hex("#18191c"),
			subtle:          ui.Hex("#23252a"),
			footer:          ui.Hex("#242426"),
			hover:           ui.RGBA(255, 255, 255, 0.07),
			active:          ui.RGBA(255, 255, 255, 0.12),
			sidebarSelected: ui.RGBA(255, 255, 255, 0.13),
			searchBg:        ui.RGBA(255, 255, 255, 0.06),
			line:            ui.Hex("#343437"),
			controlLine:     ui.Hex("#48484a"),
			ink:             ui.Hex("#f5f5f7"),
			secondary:       ui.Hex("#c7c7cc"),
			muted:           ui.Hex("#9ca3af"),
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
			danger:          ui.Hex("#ff5c5c"),
			pillBg:          ui.RGBA(255, 255, 255, 0.10),
			bottomBg:        ui.RGBA(255, 255, 255, 0.025),
			shadowColor:     ui.RGBA(0, 0, 0, 0.55),
			popoverShadow:   ui.RGBA(0, 0, 0, 0.45),

			prefsSidebar:         ui.Hex("#1e1f22"),
			prefsCanvas:          ui.Hex("#18191c"),
			prefsCard:            ui.Hex("#23252a"),
			prefsCardBorder:      ui.RGBA(255, 255, 255, 0.08),
			prefsCardShadow:      ui.RGBA(0, 0, 0, 0.28),
			prefsFooter:          ui.Hex("#1e1f22"),
			prefsNavActiveBg:     ui.RGBA(255, 255, 255, 0.09),
			prefsNavActiveShadow: ui.RGBA(0, 0, 0, 0.22),
			prefsInk:             ui.Hex("#f3f4f6"),
			prefsSecondary:       ui.Hex("#d1d5db"),
			prefsMuted:           ui.Hex("#9ca3af"),
		}
	}
	return macPalette{
		dark:            false,
		sidebar:         ui.Hex("#ececec"),
		canvas:          ui.Hex("#f7f8fa"),
		subtle:          ui.Hex("#ffffff"),
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
		danger:          ui.Hex("#e5484d"),
		pillBg:          ui.RGBA(0, 0, 0, 0.065),
		bottomBg:        ui.RGBA(0, 0, 0, 0.025),
		shadowColor:     ui.RGBA(0, 0, 0, 0.14),
		popoverShadow:   ui.RGBA(0, 0, 0, 0.12),

		prefsSidebar:         ui.Hex("#f1f2f5"),
		prefsCanvas:          ui.Hex("#f7f8fa"),
		prefsCard:            ui.Hex("#ffffff"),
		prefsCardBorder:      ui.RGBA(15, 23, 42, 0.08),
		prefsCardShadow:      ui.RGBA(15, 23, 42, 0.045),
		prefsFooter:          ui.Hex("#f1f2f5"),
		prefsNavActiveBg:     ui.Hex("#ffffff"),
		prefsNavActiveShadow: ui.RGBA(15, 23, 42, 0.07),
		prefsInk:             ui.Hex("#111827"),
		prefsSecondary:       ui.Hex("#4b5563"),
		prefsMuted:           ui.Hex("#6b7280"),
	}
}

func buildLobeTokens(pal macPalette) theme.Tokens {
	var tk theme.Tokens
	if pal.dark {
		tk = *theme.Dark()
	} else {
		tk = *theme.Light()
	}
	// Map Planet Windows Mac-Planet WebView palette directly onto lobe-mygo tokens.
	tk.BgBase = pal.canvas
	tk.BgLayout = pal.subtle
	tk.BgContainer = pal.canvas
	tk.BgContainerSecondary = pal.subtle
	tk.BgElevated = pal.canvas
	tk.Border = pal.controlLine
	tk.BorderSecondary = pal.line
	tk.Fill = pal.active
	tk.FillSecondary = pal.hover
	tk.FillTertiary = pal.subtle
	tk.FillQuaternary = pal.line
	tk.Text = pal.ink
	tk.TextSecondary = pal.secondary
	tk.TextTertiary = pal.muted
	tk.TextQuaternary = pal.muted
	tk.TextPlaceholder = pal.muted
	tk.Icon = pal.secondary
	tk.IconHover = pal.ink
	tk.Primary = pal.accent
	tk.PrimaryHover = pal.accentHover
	tk.PrimaryActive = pal.accentActive
	tk.PrimaryForeground = ui.Hex("#ffffff")
	tk.PrimaryBg = pal.selection
	tk.PrimaryBorder = pal.accent
	tk.Info = pal.accent
	tk.Success = pal.green
	tk.Warning = pal.amber
	tk.Error = pal.danger
	tk.BorderRadius = 6
	tk.BorderRadiusSM = 6
	tk.BorderRadiusLG = 8
	tk.FontSize = 13
	return tk
}

var planetLobeTheme = theme.NewConfig(buildLobeTokens(paletteFor(false)), buildLobeTokens(paletteFor(true)))

func applyMacPlanetTheme(c *ui.Context) macPalette {
	planetLobeTheme.Apply(c)
	pal := paletteFor(c.Theme().Dark)
	// Keep ui.Theme's background/surface pure canvas so scroll areas and inputs stay crisp.
	t := *c.Theme()
	t.Background = pal.canvas
	t.Surface = pal.canvas
	t.SurfaceHover = pal.subtle
	t.SurfacePressed = pal.sidebarSelected
	t.Border = pal.line
	t.Selection = pal.selection
	t.Focus = pal.accent
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

// interactiveRow creates a truly borderless, left-aligned full-width clickable row
// using ui.ButtonBase (zero unwanted default button chrome or centering).
func interactiveRow(c *ui.Context, key, ariaLabel string, restBg, hoverBg, activeBg ui.Color) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Label(ariaLabel).AlignSelf(ui.Stretch).
		Justify(ui.Start).AlignItems(ui.Center).
		Cursor(ui.CursorPointer)
	bg := restBg
	switch {
	case b.Pressed():
		bg = activeBg
	case b.Hovered():
		bg = hoverBg
	}
	b.Background(bg)
	return b
}

func sidebarNavRowElem(c *ui.Context, key, label string, selected bool, badge int, leading func(), pal macPalette) ui.Element {
	rest := ui.Transparent
	hover := pal.hover
	active := pal.active
	if selected {
		rest = pal.sidebarSelected
		hover = pal.sidebarSelected
	}
	return interactiveRow(c, key, label, rest, hover, active).
		MinHeight(30).Radius(6).PaddingX(8).PaddingY(5).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Grow(1).MinWidth(0).Children(func() {
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
}

func sidebarNavRow(c *ui.Context, key, label string, selected bool, badge int, leading func(), pal macPalette) bool {
	return sidebarNavRowElem(c, key, label, selected, badge, leading, pal).Clicked()
}

// macIconButton wraps lobe-mygo's action.ActionIcon for crisp hover/press transitions.
func macIconButton(c *ui.Context, key string, icon *ui.SVG, boxSize, iconSize float32, _ ui.Color, ariaLabel string) bool {
	return action.ActionIcon(c.Key(key), icon, action.ActionIconProps{
		Variant:   action.ActionIconBorderless,
		BlockSize: boxSize,
		GlyphSize: iconSize,
		Title:     ariaLabel,
	}).Clicked()
}

func ghostIconButton(c *ui.Context, key, text, ariaLabel string, _ macPalette) bool {
	return action.Button(c.Key(key), text, action.ButtonProps{
		Variant: action.Text,
		Size:    theme.Small,
	}).Label(ariaLabel).Clicked()
}

func macPrimaryButton(c *ui.Context, key, label string, pal macPalette) bool {
	return macPrimaryButtonSized(c, key, label, 32, 14, 13, pal)
}

func macPrimaryButtonSmall(c *ui.Context, key, label string, pal macPalette) bool {
	return macPrimaryButtonSized(c, key, label, 28, 12, 12, pal)
}

func macPrimaryButtonSized(c *ui.Context, key, label string, height, padX, fontSize float32, pal macPalette) bool {
	b := ui.ButtonBase(c.Key(key)).Label(label).Height(height).PaddingX(padX).Radius(6).
		Center().Cursor(ui.CursorPointer)
	bg := pal.prefsInk
	switch {
	case b.Pressed():
		bg = pal.prefsInk.Alpha(0.76)
	case b.Hovered():
		bg = pal.prefsInk.Alpha(0.86)
	}
	b.Background(bg).Children(func() {
		ui.Text(c, label).Font(macPlanetFont).FontSize(fontSize).FontWeight(500).TextColor(pal.prefsCanvas).NoWrap()
	})
	return b.Clicked()
}

func macSecondaryButton(c *ui.Context, key, label string, pal macPalette) bool {
	bg := ui.Hex("#ffffff")
	hoverBg := ui.Hex("#f4f4f5")
	activeBg := ui.Hex("#e5e7eb")
	border := ui.RGBA(15, 23, 42, 0.12)
	if pal.dark {
		bg = ui.RGBA(255, 255, 255, 0.08)
		hoverBg = ui.RGBA(255, 255, 255, 0.13)
		activeBg = ui.RGBA(255, 255, 255, 0.18)
		border = ui.RGBA(255, 255, 255, 0.11)
	}
	b := ui.ButtonBase(c.Key(key)).Label(label).Height(28).PaddingX(12).Radius(6).
		Border(0.5, border).Center().Cursor(ui.CursorPointer)
	switch {
	case b.Pressed():
		b.Background(activeBg)
	case b.Hovered():
		b.Background(hoverBg)
	default:
		b.Background(bg)
	}
	b.Children(func() {
		ui.Text(c, label).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.prefsInk).NoWrap()
	})
	return b.Clicked()
}

// macSegmented renders a crisp pill segmented control with balanced light & dark tokens.
func macSegmented(c *ui.Context, key string, selected *int, options []string, pal macPalette) {
	trackBg := ui.RGBA(15, 23, 42, 0.055)
	pillBg := ui.Hex("#ffffff")
	pillShadow := ui.RGBA(15, 23, 42, 0.08)
	if pal.dark {
		trackBg = ui.RGBA(255, 255, 255, 0.06)
		pillBg = ui.RGBA(255, 255, 255, 0.14)
		pillShadow = ui.RGBA(0, 0, 0, 0.24)
	}
	ui.Row(c.Key(key)).Padding(2.5).Radius(7).Background(trackBg).Gap(2).AlignItems(ui.Center).Children(func() {
		for i, opt := range options {
			idx := i
			active := selected != nil && *selected == idx
			fg := pal.prefsSecondary
			if active {
				fg = pal.prefsInk
			}
			btn := ui.ButtonBase(c.Key(fmt.Sprintf("%s-%d", key, idx))).Label(opt).
				Padding(4, 11, 4, 11).Radius(5).Cursor(ui.CursorPointer)
			if active {
				btn.Background(pillBg).Shadow(0, 1, 2, 0, pillShadow)
			} else if btn.Hovered() {
				btn.Background(pal.hover)
			}
			btn.Children(func() {
				t := ui.Text(c, opt).Font(macPlanetFont).FontSize(12).TextColor(fg).NoWrap()
				if active {
					t.FontWeight(500)
				}
			})
			if btn.Clicked() && selected != nil {
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
