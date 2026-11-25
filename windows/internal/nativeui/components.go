package nativeui

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/action"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
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

type articleRowProps struct {
	key         string
	ariaLabel   string
	title       string
	summary     string
	dateText    string
	sourceLabel string
	selected    bool
	starred     bool
	unread      bool
	pinned      bool
	isDraft     bool
	isPage      bool
	navWeight   int // >0 when included in navigation
	attachments int
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
		restBg, hoverBg = pal.listSelected, pal.listSelected
		titleColor = ui.Hex("#ffffff")
		excerptColor = pal.listSelMuted
		metaColor = pal.listSelMuted
		badgeBorder = ui.RGBA(255, 255, 255, 0.35)
	}

	aria := p.ariaLabel
	if aria == "" {
		aria = p.title
	}

	row := interactiveRow(c, p.key, aria, restBg, hoverBg, pal.active).
		MinHeight(84).Radius(7).Padding(9, 12, 9, 4).
		Children(func() {
			ui.Row(c).AlignItems(ui.Start).Gap(2).Grow(1).MinWidth(0).Children(func() {
				// Left 18px gutter for star / unread dot so all titles stay vertically aligned.
				ui.Box(c).Width(18).PaddingY(3).Center().Shrink(0).Children(func() {
					switch {
					case p.starred:
						starCol := pal.yellow
						if p.selected {
							starCol = ui.Hex("#ffffff")
						}
						ui.Icon(c, iconStarSolid).Size(12, 12).TextColor(starCol)
					case p.unread:
						dotCol := pal.accent
						if p.selected {
							dotCol = ui.Hex("#ffffff")
						}
						ui.Box(c).Size(7, 7).Radius(3.5).Background(dotCol)
					}
				})

				ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
					// Row 1: Pin icon + single-line Title + right-aligned Date.
					ui.Row(c).AlignItems(ui.Center).Gap(6).AlignSelf(ui.Stretch).Children(func() {
						if p.pinned {
							pinCol := pal.accent
							if p.selected {
								pinCol = ui.Hex("#ffffff")
							}
							ui.Icon(c, lucide.Pin()).Size(12, 12).TextColor(pinCol).Shrink(0)
						}
						ui.Text(c, p.title).Font(macPlanetFont).FontSize(13).FontWeight(600).
							TextColor(titleColor).MaxLines(1).Ellipsis("…").Grow(1)
						if p.dateText != "" {
							ui.Text(c, p.dateText).Font(macPlanetFont).FontSize(12).
								TextColor(metaColor).NoWrap().Shrink(0)
						}
					})

					// Row 2: Summary excerpt (up to 2 lines).
					if p.summary != "" {
						ui.Text(c, p.summary).Font(macPlanetFont).FontSize(12.5).
							TextColor(excerptColor).MaxLines(2).Ellipsis("…")
					}

					// Row 3: Compact metadata / media / source tags.
					hasFooter := p.isDraft || p.isPage || p.navWeight > 0 || p.attachments > 0 || p.sourceLabel != ""
					if hasFooter {
						ui.Row(c).AlignItems(ui.Center).Gap(6).AlignSelf(ui.Stretch).Children(func() {
							outlinedPill := func(text string) {
								ui.Box(c).Padding(1, 5, 1, 5).Radius(4).Border(0.5, badgeBorder).Shrink(0).Children(func() {
									ui.Text(c, text).Font(macPlanetFont).FontSize(10.5).TextColor(metaColor).NoWrap()
								})
							}
							if p.isDraft {
								outlinedPill("草稿")
							}
							if p.isPage {
								outlinedPill("页面")
							}
							if p.navWeight > 0 {
								outlinedPill(fmt.Sprintf("导航 %d", p.navWeight))
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
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02 15:04:05",
		"2006-01-02",
		"2006/01/02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return formatRelativeDate(t)
		}
	}
	return s
}
