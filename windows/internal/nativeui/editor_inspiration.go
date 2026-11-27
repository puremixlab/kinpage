package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

type inspirationPhase uint8

const (
	inspirationClosed inspirationPhase = iota
	inspirationTree
	inspirationPrompt
)

const inspirationInset = 10

type editorInspiration struct {
	phase         inspirationPhase
	picks         uint64
	bufferVersion uint64
	promptKey     string
	body          ui.Handle
	pick          ui.Handle
}

func (a *App) viewInspirationToggle(c *ui.Context, compact bool) {
	inspiration := &a.editorInspiration
	label := a.t("editor.inspiration")
	open := inspiration.phase != inspirationClosed
	pal := a.pal
	button := ui.ButtonBase(c.Key("editor-inspiration-toggle")).Label(label).Tooltip(label).
		Expanded(open).Height(28).PaddingX(7).Radius(6).Center().Shrink(0).Cursor(ui.CursorPointer)
	if open || button.Hovered() {
		button.Background(pal.hover)
	}
	button.Children(func() {
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, lucide.Apple()).Size(15, 15).TextColor(pal.secondary).Shrink(0)
			if !compact {
				ui.Text(c, label).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).NoWrap()
			}
		})
	})
	button.OnClick(func() {
		if inspiration.phase != inspirationClosed {
			a.resumeInspiredWriting(c)
			return
		}
		inspiration.phase = inspirationTree
		inspiration.bufferVersion = a.editorBuffer.Version()
		inspiration.pick.Focus(c)
	})
	if open {
		button.OnShortcut(0, ui.KeyEscape, func() { a.resumeInspiredWriting(c) })
	}
}

func (a *App) viewEditorBody(c *ui.Context) {
	inspiration := &a.editorInspiration
	ui.Box(c.Key("editor-body")).Fill().MinWidth(0).MinHeight(0).Clip().Children(func() {
		editor.View(c, a.editorState).Fill().Bind(&inspiration.body)
		if inspiration.phase != inspirationClosed {
			a.viewInspiration(c, inspiration.body.Bounds(c))
		}
	})
}

func (a *App) viewInspiration(c *ui.Context, body ui.Rect) {
	width := min(270, body.W-2*inspirationInset)
	if width <= 0 || body.H <= 0 {
		return
	}
	inspiration := &a.editorInspiration
	pal := a.pal
	controls := ui.Column(c.Key("editor-inspiration")).Absolute().Right(inspirationInset).Top(inspirationInset).
		ZIndex(2).PassThrough().Width(width).AlignItems(ui.End).Gap(7)
	controls.OnShortcut(0, ui.KeyEscape, func() { a.resumeInspiredWriting(c) })
	controls.Children(func() {
		label := a.t("editor.inspirationPick")
		if inspiration.phase == inspirationPrompt {
			label = a.t("editor.inspirationAgain")
		}
		treeWidth := min(112, width)
		tree := ui.ButtonBase(c.Key("editor-inspiration-pick")).Bind(&inspiration.pick).
			Label(label).Width(treeWidth).PaddingY(4).Radius(8).Background(pal.canvas).Cursor(ui.CursorPointer)
		if tree.Hovered() || tree.Pressed() {
			tree.Background(pal.hover)
		}
		tree.Children(func() {
			ui.Column(c).Width(treeWidth).AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Box(c).Width(treeWidth).Height(76).Draw(func(p *ui.Painter, r ui.Rect) {
					drawInspirationTree(p, r, pal.dark)
				})
				ui.Text(c, label).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).TextAlign(ui.Center)
			})
		})
		tree.OnClick(func() { a.pickInspiration(c) })
		if inspiration.phase == inspirationPrompt {
			ui.Column(c.Key("editor-inspiration-note")).Width(width).Padding(12).Gap(8).Radius(8).
				Background(pal.subtle).Border(0.5, pal.line).Children(func() {
				ui.Text(c, a.t(inspiration.promptKey)).Font(macPlanetFont).FontSize(13).TextColor(pal.ink)
				ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
					if macSecondaryButton(c, "editor-inspiration-resume", a.t("editor.inspirationResume"), pal) {
						a.resumeInspiredWriting(c)
					}
				})
			})
		}
	})
}

func (a *App) pickInspiration(c *ui.Context) {
	inspiration := &a.editorInspiration
	prompts := [...]string{"editor.inspirationContinue1", "editor.inspirationContinue2", "editor.inspirationContinue3"}
	if strings.TrimSpace(a.editorBuffer.String()) == "" {
		prompts = [...]string{"editor.inspirationOpen1", "editor.inspirationOpen2", "editor.inspirationOpen3"}
	}
	inspiration.promptKey = prompts[inspiration.picks%uint64(len(prompts))]
	inspiration.picks++
	inspiration.phase = inspirationPrompt
	c.Announce(a.t(inspiration.promptKey))
}

func (a *App) resumeInspiredWriting(c *ui.Context) {
	a.editorInspiration.phase = inspirationClosed
	a.editorInspiration.body.Focus(c)
	c.Invalidate()
}

func drawInspirationTree(p *ui.Painter, r ui.Rect, dark bool) {
	scale := min(r.W/96, r.H/76)
	x, y := r.X+(r.W-96*scale)/2, r.Y+(r.H-76*scale)/2
	leaf, leafTop, leafShade := ui.Hex("#7f926e"), ui.Hex("#a1b28b"), ui.Hex("#607655")
	bark, apple := ui.Hex("#79664f"), ui.Hex("#b84233")
	if dark {
		leaf, leafTop, leafShade = ui.Hex("#6f865e"), ui.Hex("#94a879"), ui.Hex("#526c48")
		bark, apple = ui.Hex("#a28565"), ui.Hex("#e56d5d")
	}
	fill := func(left, top, width, height, radius float32, color ui.Color) {
		p.Fill(ui.Rect{X: x + left*scale, Y: y + top*scale, W: width * scale, H: height * scale}, color, radius*scale)
	}
	fill(12, 71, 72, 2, 1, leafShade.Alpha(0.42))
	trunk := (&ui.Path{}).MoveTo(x+44*scale, y+71*scale).
		LineTo(x+46*scale, y+49*scale).LineTo(x+34*scale, y+38*scale).
		LineTo(x+40*scale, y+39*scale).LineTo(x+50*scale, y+46*scale).
		LineTo(x+56*scale, y+40*scale).LineTo(x+63*scale, y+38*scale).
		LineTo(x+53*scale, y+51*scale).LineTo(x+55*scale, y+71*scale).Close()
	p.FillPath(trunk, bark)
	fill(4, 21, 54, 36, 18, leafShade)
	fill(40, 22, 52, 34, 17, leaf)
	fill(22, 6, 56, 45, 22, leafTop)
	fill(62, 28, 2, 7, 1, bark)
	fill(63, 27, 6, 3, 1.5, leafShade)
	fill(55, 32, 16, 15, 7.5, apple)
}
