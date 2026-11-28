package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/store"
)

// smartFeedIcon renders a macOS-style 18px solid colored icon inside a 22px slot.
func smartFeedIcon(c *ui.Context, icon *ui.SVG, color ui.Color) {
	ui.Box(c).Size(22, 22).Center().Shrink(0).Children(func() {
		ui.Icon(c, icon).Size(17, 17).TextColor(color)
	})
}

// toolbarIconBtn renders a 28px borderless icon button with a native tooltip.
func toolbarIconBtn(c *ui.Context, key string, icon *ui.SVG, tooltip string) ui.Element {
	return action.ActionIcon(c.Key(key), icon, action.ActionIconProps{
		Variant:   action.ActionIconBorderless,
		BlockSize: 28,
		GlyphSize: 15,
		Title:     tooltip,
	}).Tooltip(tooltip)
}

// toolbarIconBtnColored renders a 28px borderless icon button with an explicit icon tint.
func toolbarIconBtnColored(c *ui.Context, key string, icon *ui.SVG, color ui.Color, tooltip string, pal macPalette) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Label(tooltip).Size(28, 28).Radius(6).
		Center().Cursor(ui.CursorPointer).Tooltip(tooltip).Transition(rowHoverTransition)
	if b.Pressed() {
		b.Background(pal.active)
	} else if b.Hovered() {
		b.Background(pal.hover)
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(color)
	})
	return b
}

type starTypeOption struct {
	id  int
	key string
}

var allStarTypes = []starTypeOption{
	{store.StarTypeDefault, "star.default"},
	{store.StarTypeTodo, "star.todo"},
	{store.StarTypeDone, "star.done"},
	{store.StarTypeSparkles, "star.sparkles"},
	{store.StarTypeHeart, "star.heart"},
	{store.StarTypePlan, "star.plan"},
}

func starTypeVisual(t int, pal macPalette) (*ui.SVG, ui.Color) {
	switch t {
	case store.StarTypeTodo:
		return lucide.Circle(), pal.orange
	case store.StarTypeDone:
		return lucide.CircleCheck(), pal.green
	case store.StarTypeSparkles:
		return lucide.Sparkles(), ui.Hex("#af52de")
	case store.StarTypeHeart:
		return lucide.Heart(), pal.danger
	case store.StarTypePlan:
		return lucide.Calendar(), pal.accent
	default:
		return iconStarSolid, pal.yellow
	}
}

type articleRowProps struct {
	key         string
	ariaLabel   string
	title       string
	summary     string
	dateText    string
	sourceLabel string
	selected    bool
	starred     bool
	starType    int
	unread      bool
	pinned      bool
	isDraft     bool
	isPage      bool
	navWeight   int // >0 when included in navigation
	attachments int
	locale      Locale
}

// articleRow renders a unified macOS Planet-style article list row (90px rhythm):
// - Left 20px indicator gutter (star > unread dot > empty)
// - Row 1: optional pin icon + single-line title + right-aligned date
// - Row 2: 2-line summary excerpt
// - Row 3: compact outlined badges / media indicators / source label
func articleRow(c *ui.Context, p articleRowProps, pal macPalette) ui.Element {
	restBg, hoverBg := ui.Transparent, pal.hover
	titleColor := pal.ink
	excerptColor := pal.secondary
	metaColor := pal.muted
	badgeBorder := pal.line
	if p.selected {
		if pal.dark {
			restBg = ui.RGBA(255, 255, 255, 0.08)
			hoverBg = ui.RGBA(255, 255, 255, 0.10)
		} else {
			restBg = ui.RGBA(15, 23, 42, 0.06)
			hoverBg = ui.RGBA(15, 23, 42, 0.08)
		}
	}

	aria := p.ariaLabel
	if aria == "" {
		aria = p.title
	}

	row := interactiveRow(c, p.key, aria, restBg, hoverBg, pal.active).
		MinHeight(78).Radius(7).Padding(9, 12, 9, 4).
		Children(func() {
			ui.Row(c).AlignItems(ui.Stretch).Gap(2).Grow(1).MinWidth(0).Children(func() {
				// Subtle 2.5px left accent bar when selected (calm monochrome identity).
				if p.selected {
					ui.Box(c).Width(2.5).Radius(2).Background(pal.prefsInk).Margin(2, 2, 2, 0).Shrink(0)
				}
				// Left 18px gutter for star / unread dot / draft pen so all titles stay vertically aligned.
				ui.Box(c).Width(18).PaddingY(3).Center().Shrink(0).Children(func() {
					switch {
					case p.starred:
						starIcon, starCol := starTypeVisual(p.starType, pal)
						ui.Icon(c, starIcon).Size(12, 12).TextColor(starCol)
					case p.unread:
						ui.Box(c).Size(7, 7).Radius(3.5).Background(pal.accent)
					case p.isDraft:
						ui.Icon(c, lucide.PenLine()).Size(12, 12).TextColor(pal.muted)
					}
				})

				ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
					// Row 1: Pin icon + single-line Title + right-aligned Date.
					ui.Row(c).AlignItems(ui.Center).Gap(6).AlignSelf(ui.Stretch).Children(func() {
						if p.pinned {
							ui.Icon(c, lucide.Pin()).Size(12, 12).TextColor(pal.secondary).Shrink(0)
						}
						ui.Text(c, p.title).Font(macPlanetFont).FontSize(13).FontWeight(600).
							TextColor(titleColor).MaxLines(1).Ellipsis("…").Grow(1)
						if p.dateText != "" {
							ui.Text(c, p.dateText).Font(macPlanetFont).FontSize(11.5).
								TextColor(metaColor).NoWrap().Shrink(0)
						}
					})

					// Row 2: Summary excerpt (up to 2 lines).
					if p.summary != "" {
						ui.Text(c, p.summary).Font(macPlanetFont).FontSize(12.5).
							TextColor(excerptColor).MaxLines(2).Ellipsis("…")
					}

					// Row 3: Compact metadata / media / source tags (no redundant "草稿" badge).
					hasFooter := p.isPage || p.navWeight > 0 || p.attachments > 0 || p.sourceLabel != ""
					if hasFooter {
						ui.Row(c).AlignItems(ui.Center).Gap(6).AlignSelf(ui.Stretch).Children(func() {
							outlinedPill := func(text string) {
								ui.Box(c).Padding(1, 5, 1, 5).Radius(4).Border(0.5, badgeBorder).Shrink(0).Children(func() {
									ui.Text(c, text).Font(macPlanetFont).FontSize(10.5).TextColor(metaColor).NoWrap()
								})
							}
							loc := p.locale
							if loc == "" {
								loc = LocaleZhCN
							}
							if p.isPage {
								outlinedPill(translate(loc, "list.badgePage"))
							}
							if p.navWeight > 0 {
								outlinedPill(fmt.Sprintf(translate(loc, "list.badgeNav"), p.navWeight))
							}
							if p.attachments > 0 {
								ui.Row(c).AlignItems(ui.Center).Gap(2).Shrink(0).Children(func() {
									ui.Icon(c, lucide.Paperclip()).Size(11, 11).TextColor(metaColor)
									ui.Text(c, fmt.Sprintf("%d", p.attachments)).Font(macPlanetFont).FontSize(11).TextColor(metaColor).NoWrap()
								})
							}
							if p.sourceLabel != "" {
								ui.Text(c, p.sourceLabel).Font(macPlanetFont).FontSize(11).
									TextColor(metaColor).MaxLines(1).Ellipsis("…").Grow(1)
							}
						})
					}
				})
			})
		})

	return row
}

// formatRelativeDate formats a timestamp in concise macOS Planet list style.
func formatRelativeDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now().Local()
	lt := t.Local()
	ny, nm, nd := now.Date()
	ly, lm, ld := lt.Date()
	if ny == ly && nm == lm && nd == ld {
		return lt.Format("15:04")
	}
	yesterday := now.AddDate(0, 0, -1)
	yy, ym, yd := yesterday.Date()
	if yy == ly && ym == lm && yd == ld {
		return "昨天"
	}
	if ny == ly {
		return fmt.Sprintf("%d月%d日", int(lm), ld)
	}
	return lt.Format("2006/01/02")
}

// formatReaderDate formats a full date/time string for the article reader header.
func formatReaderDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006年1月2日 15:04")
}

// formatFeedDate converts a feed Published string to a short date when possible.
func formatFeedDate(raw string) string {
	if date, ok := parseFeedDate(raw, time.Local); ok {
		return formatRelativeDate(date)
	}
	return strings.TrimSpace(raw)
}

type macSheetProps struct {
	Title        string
	Width        float32
	PrimaryText  string
	CancelText   string
	CloseTooltip string
	HideFooter   bool
	Danger       bool
}

// macPreferencesWindow renders a spacious, viewport-responsive macOS Sequoia
// Preferences window (centered, up to 960x680, adapting cleanly to smaller viewports).
func macPreferencesWindow(c *ui.Context, open *bool, maxWidth, maxHeight float32, pal macPalette, content func()) {
	if open == nil || !*open {
		return
	}
	if maxWidth <= 0 {
		maxWidth = 960
	}
	if maxHeight <= 0 {
		maxHeight = 680
	}
	if _, winH := c.Size(); winH > 200 && maxHeight > winH-56 {
		maxHeight = winH - 56
	}
	dim := ui.RGBA(0, 0, 0, 0.18)
	if pal.dark {
		dim = ui.RGBA(0, 0, 0, 0.45)
	}
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Center().Padding(24).Background(dim)
		panel.WidthPercent(100).MaxWidth(maxWidth).HeightPercent(100).MaxHeight(maxHeight).
			Radius(14).Background(pal.prefsCanvas).Border(0.5, pal.prefsCardBorder).
			Shadow(0, 18, 44, 0, pal.shadowColor).Clip()
		content()
	})
}

// macInsetCard renders an elevated macOS Sequoia Inset Grouped settings card
// (#ffffff floating on #f7f8fa canvas in light mode, #23252a on #18191c in dark mode).
func macInsetCard(c *ui.Context, pal macPalette, content func()) {
	ui.Column(c).Padding(16, 18, 16, 18).Gap(12).AlignSelf(ui.Stretch).
		Background(pal.prefsCard).Border(0.5, pal.prefsCardBorder).Radius(10).
		Shadow(0, 1, 3, 0, pal.prefsCardShadow).Children(content)
}

// settingsEmptyState renders a clean ChatGPT-style empty state card inside a
// settings pane (icon + concise description + primary pill action button),
// avoiding raw empty forms until the user explicitly chooses to add/configure.
func settingsEmptyState(c *ui.Context, key string, icon *ui.SVG, desc, actionLabel string, pal macPalette) bool {
	clicked := false
	macInsetCard(c, pal, func() {
		ui.Column(c).PaddingY(24).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			if icon != nil {
				ui.Icon(c, icon).Size(24, 24).TextColor(pal.prefsSecondary)
			}
			ui.Text(c, desc).Font(macPlanetFont).FontSize(12.5).TextColor(pal.prefsMuted)
			if actionLabel != "" {
				btn := ui.ButtonBase(c.Key(key)).Label(actionLabel).
					Height(30).PaddingX(16).Radius(15).Background(pal.prefsInk).
					Center().Cursor(ui.CursorPointer)
				if btn.Hovered() {
					btn.Background(pal.prefsInk.Alpha(0.86))
				}
				btn.Children(func() {
					ui.Text(c, actionLabel).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.prefsCanvas).NoWrap()
				})
				if btn.Clicked() {
					clicked = true
				}
			}
		})
	})
	return clicked
}

// macField renders a top-aligned settings field with clear label and optional hint.
func macField(c *ui.Context, label, hint string, pal macPalette, input func()) {
	ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, label).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.prefsInk)
			if hint != "" {
				ui.Text(c, hint).Font(macPlanetFont).FontSize(11).TextColor(pal.prefsMuted)
			}
		})
		input()
	})
}

// prefsNavRow renders a macOS System Settings sidebar item with a neutral
// monochrome 20x20 rounded-square badge icon and an elevated active pill.
func prefsNavRow(c *ui.Context, key, label string, selected bool, icon *ui.SVG, _ ui.Color, pal macPalette) bool {
	rest := ui.Transparent
	hover := pal.hover
	active := pal.active
	if selected {
		rest = pal.prefsNavActiveBg
		hover = pal.prefsNavActiveBg
	}
	badgeBg := ui.RGBA(15, 23, 42, 0.06)
	if pal.dark {
		badgeBg = ui.RGBA(255, 255, 255, 0.08)
	}
	iconFg := pal.prefsSecondary
	if selected {
		iconFg = pal.prefsInk
	}
	b := interactiveRow(c, key, label, rest, hover, active).
		MinHeight(32).Radius(7).Padding(5, 8, 5, 8)
	if selected {
		b.Border(0.5, pal.prefsCardBorder).Shadow(0, 1, 2, 0, pal.prefsNavActiveShadow)
	}
	b.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(9).Grow(1).MinWidth(0).Children(func() {
			if icon != nil {
				ui.Box(c).Size(20, 20).Radius(5).Background(badgeBg).Center().Shrink(0).Children(func() {
					ui.Icon(c, icon).Size(12, 12).TextColor(iconFg)
				})
			}
			txt := ui.Text(c, label).Font(macPlanetFont).FontSize(13).TextColor(pal.prefsInk).Grow(1).NoWrap()
			if selected {
				txt.FontWeight(600)
			}
		})
	})
	return b.Clicked()
}

// prefsHairlineH renders a subtle 0.5px separator matching the Preferences palette.
func prefsHairlineH(c *ui.Context, pal macPalette) {
	ui.Box(c).Height(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)
}

// macSheetFrame renders a bare macOS-style top-attached sheet frame (no inner
// padding or footer), ideal for two-column Preferences sheets.
func macSheetFrame(c *ui.Context, open *bool, width float32, pal macPalette, content func()) {
	if open == nil || !*open {
		return
	}
	if width <= 0 {
		width = 520
	}
	dim := ui.RGBA(0, 0, 0, 0.18)
	if pal.dark {
		dim = ui.RGBA(0, 0, 0, 0.45)
	}
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Justify(ui.Start).AlignItems(ui.Center).Padding(44, 16, 24, 16).Background(dim)
		panel.Width(width).MaxWidthPercent(100).MaxHeightPercent(100).
			Radius(10).Background(pal.canvas).Border(0.5, pal.line).
			Shadow(0, 12, 32, 0, pal.shadowColor).Clip()
		content()
	})
}

// macConfirmSheet renders a compact macOS-style confirmation sheet.
// Returns true on the frame the confirm button is clicked (and closes *open).
func macConfirmSheet(c *ui.Context, open *bool, title, message, cancelText, confirmText string, danger bool, pal macPalette) bool {
	confirmed := macSheet(c, open, macSheetProps{
		Title:       title,
		Width:       380,
		PrimaryText: confirmText,
		CancelText:  cancelText,
		Danger:      danger,
	}, pal, func() {
		if message != "" {
			ui.Text(c, message).Font(macPlanetFont).FontSize(13).TextColor(pal.secondary)
		}
	})
	if confirmed {
		*open = false
	}
	return confirmed
}

// macThreeActionSheet renders a macOS-style 3-action sheet (cancel, secondary, primary).
// Returns 0 (none/cancelled), 1 (secondary), or 2 (primary).
func macThreeActionSheet(c *ui.Context, open *bool, title, message, cancelText, secondaryText, primaryText string, pal macPalette) int {
	result := 0
	macSheet(c, open, macSheetProps{
		Title:      title,
		Width:      420,
		HideFooter: true,
	}, pal, func() {
		if message != "" {
			ui.Text(c, message).Font(macPlanetFont).FontSize(13).TextColor(pal.secondary)
		}
		ui.Row(c).Padding(8, 0, 0, 0).Gap(8).AlignItems(ui.Center).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "sheet3-cancel-"+title, cancelText, pal) {
				*open = false
				result = 0
			}
			if macSecondaryButton(c, "sheet3-sec-"+title, secondaryText, pal) {
				*open = false
				result = 1
			}
			if macPrimaryButtonSized(c, "sheet3-pri-"+title, primaryText, 30, 14, 13, pal) {
				*open = false
				result = 2
			}
		})
	})
	return result
}

// macSheet renders a macOS-style top-attached sheet hanging directly beneath
// the 44px unified titlebar. Returns true on the frame the primary button is clicked.
func macSheet(c *ui.Context, open *bool, p macSheetProps, pal macPalette, body func()) bool {
	if open == nil || !*open {
		return false
	}
	width := p.Width
	if width <= 0 {
		width = 480
	}
	submitted := false
	macSheetFrame(c, open, width, pal, func() {
		ui.Column(c).AlignSelf(ui.Stretch).Children(func() {
			if p.Title != "" {
				ui.Row(c).MinHeight(44).Padding(10, 16, 10, 16).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, p.Title).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink)
					closeTip := p.CloseTooltip
					if closeTip == "" {
						closeTip = translate(LocaleZhCN, "common.close")
					}
					if toolbarIconBtn(c, "sheet-close-"+p.Title, lucide.X(), closeTip).Clicked() {
						*open = false
					}
				})
				hairlineH(c, pal)
			}
			ui.Scroll(c).Shrink(1).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Padding(16).Gap(12).AlignSelf(ui.Stretch).Children(body)
			})
			if !p.HideFooter {
				hairlineH(c, pal)
				ui.Row(c).Padding(12, 16, 12, 16).Gap(8).AlignItems(ui.Center).Justify(ui.End).Background(pal.footer).AlignSelf(ui.Stretch).Children(func() {
					if p.CancelText != "" {
						if macSecondaryButton(c, "sheet-cancel-"+p.Title, p.CancelText, pal) {
							*open = false
						}
					}
					if p.PrimaryText != "" {
						if p.Danger {
							b := ui.ButtonBase(c.Key("sheet-ok-" + p.Title)).Label(p.PrimaryText).Height(30).PaddingX(14).Radius(6).
								Center().Cursor(ui.CursorPointer).Background(pal.danger).Children(func() {
								ui.Text(c, p.PrimaryText).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(ui.Hex("#ffffff")).NoWrap()
							})
							if b.Clicked() {
								submitted = true
							}
						} else if macPrimaryButtonSized(c, "sheet-ok-"+p.Title, p.PrimaryText, 30, 14, 13, pal) {
							submitted = true
						}
					}
				})
			}
		})
	})
	return submitted
}
