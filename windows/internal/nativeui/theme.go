package nativeui

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type macPalette struct {
	dark            bool
	sidebar         ui.Color
	canvas          ui.Color
	subtle          ui.Color
	hover           ui.Color
	sidebarSelected ui.Color
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
	green           ui.Color
	amber           ui.Color
	danger          ui.Color
	pillBg          ui.Color
}

func paletteFor(dark bool) macPalette {
	if dark {
		return macPalette{
			dark:            true,
			sidebar:         ui.Hex("#232325"),
			canvas:          ui.Hex("#1c1c1e"),
			subtle:          ui.Hex("#2c2c2e"),
			hover:           ui.RGBA(255, 255, 255, 0.07),
			sidebarSelected: ui.RGBA(255, 255, 255, 0.13),
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
			green:           ui.Hex("#34c759"),
			amber:           ui.Hex("#f0b159"),
			danger:          ui.Hex("#ff453a"),
			pillBg:          ui.RGBA(255, 255, 255, 0.12),
		}
	}
	return macPalette{
		dark:            false,
		sidebar:         ui.Hex("#ececec"),
		canvas:          ui.Hex("#ffffff"),
		subtle:          ui.Hex("#f5f5f7"),
		hover:           ui.RGBA(0, 0, 0, 0.05),
		sidebarSelected: ui.RGBA(0, 0, 0, 0.085),
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
		green:           ui.Hex("#248a3d"),
		amber:           ui.Hex("#b25000"),
		danger:          ui.Hex("#ff3b30"),
		pillBg:          ui.RGBA(0, 0, 0, 0.07),
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

var avatarColors = []ui.Color{
	ui.Hex("#4f7cff"),
	ui.Hex("#e2645a"),
	ui.Hex("#34b371"),
	ui.Hex("#f59e0b"),
	ui.Hex("#8b5cf6"),
	ui.Hex("#0ea5e9"),
	ui.Hex("#ec4899"),
}

func avatarColorFor(name string) ui.Color {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return avatarColors[int(h.Sum32())%len(avatarColors)]
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

func avatarCircle(c *ui.Context, name string, size float32) {
	fontSize := size * 0.5
	if fontSize < 10 {
		fontSize = 10
	}
	ui.Box(c).Size(size, size).Radius(size / 2).Background(avatarColorFor(name)).Center().Shrink(0).Children(func() {
		ui.Text(c, firstRuneLabel(name)).FontSize(fontSize).Bold().TextColor(ui.Hex("#ffffff")).NoWrap()
	})
}

func sidebarNavRow(c *ui.Context, key, label string, selected bool, badge int, leading func(), pal macPalette) bool {
	bg := ui.Color{}
	if selected {
		bg = pal.sidebarSelected
	}
	btn := ui.Button(c.Key(key), "").Label(label).AlignSelf(ui.Stretch).
		Background(bg).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(6).
		Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).AlignSelf(ui.Stretch).Children(func() {
				if leading != nil {
					leading()
				}
				txt := ui.Text(c, label).FontSize(13).TextColor(pal.ink).Grow(1).NoWrap()
				if selected {
					txt.Bold()
				}
				if badge > 0 {
					ui.Box(c).PaddingX(6).PaddingY(1).Radius(9).Background(pal.pillBg).Shrink(0).Children(func() {
						ui.Text(c, fmt.Sprintf("%d", badge)).FontSize(11).TextColor(pal.secondary).NoWrap()
					})
				}
			})
		})
	return btn.Clicked()
}

func ghostIconButton(c *ui.Context, key, text, ariaLabel string, pal macPalette) bool {
	return ui.Button(c.Key(key), "").Label(ariaLabel).
		Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(6).PaddingY(4).
		Children(func() {
			ui.Text(c, text).FontSize(13).TextColor(pal.secondary).NoWrap()
		}).Clicked()
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
