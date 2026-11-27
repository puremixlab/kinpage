package nativeui

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

type inspirationPhase uint8

const (
	inspirationClosed inspirationPhase = iota
	inspirationTree
	inspirationFalling
	inspirationPrompt
)

const (
	inspirationInset    = 10
	inspirationMinWidth = 148
)

type editorInspiration struct {
	phase         inspirationPhase
	picks         uint64
	bufferVersion uint64
	promptKey     string
	body          ui.Handle
	pick          ui.Handle
	art           ui.Handle
	drop          inspirationDrop
}

type inspirationDrop struct {
	apple      ui.Rect
	landing    ui.Point
	paneWidth  float32
	paneHeight float32
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
	if inspiration.phase == inspirationFalling {
		a.viewAppleDrop(c, body)
	}
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
			Label(label).Width(treeWidth).PaddingY(4).Radius(8).
			Background(pal.canvas).Border(0.5, pal.line).Cursor(ui.CursorPointer)
		if tree.Hovered() || tree.Pressed() {
			tree.Background(pal.subtle)
		}
		tree.Children(func() {
			ui.Column(c).Width(treeWidth - 1).AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Box(c.Key("editor-inspiration-art")).AlignSelf(ui.Stretch).Height(76).Bind(&inspiration.art).Draw(func(p *ui.Painter, r ui.Rect) {
					drawInspirationTree(p, r, pal.dark, inspiration.phase != inspirationFalling)
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
	if inspiration.phase == inspirationFalling || inspiration.phase == inspirationClosed {
		return
	}
	prompts := [...]string{"editor.inspirationContinue1", "editor.inspirationContinue2", "editor.inspirationContinue3"}
	if strings.TrimSpace(a.editorBuffer.String()) == "" {
		prompts = [...]string{"editor.inspirationOpen1", "editor.inspirationOpen2", "editor.inspirationOpen3"}
	}
	inspiration.promptKey = prompts[inspiration.picks%uint64(len(prompts))]
	inspiration.picks++

	pane, art := inspiration.body.Bounds(c), inspiration.art.Bounds(c)
	if c.Preferences().ReduceMotion || art.W == 0 || pane.H == 0 {
		a.revealInspiration(c)
		return
	}
	apple := inspirationAppleBounds(art)
	apple.X -= pane.X
	apple.Y -= pane.Y
	inspiration.drop = inspirationDrop{
		apple:      apple,
		landing:    a.appleLanding(pane, apple),
		paneWidth:  pane.W,
		paneHeight: pane.H,
	}
	inspiration.phase = inspirationFalling
}

func (a *App) appleLanding(pane, apple ui.Rect) ui.Point {
	// LineTop refers to the Markdown line even when the editor wraps it.
	state := a.editorState
	line := state.CursorPos().Line
	lineY := float32(state.LineTop(line) - state.ScrollY())
	lineHeight := min(float32(state.LineTop(line+1)-state.LineTop(line)), state.Font.Size*state.LineHeight)
	bottom := pane.H - apple.H/2 - 14
	top := min(apple.Y+apple.H/2+56, bottom)
	y := min(top+25, bottom)
	if lineY >= 0 && lineY < float32(state.ViewportHeight()) {
		y = min(max(lineY+lineHeight/2, top), bottom)
	}
	x := max(inspirationInset+apple.W/2, apple.X+apple.W/2-min(90, pane.W*0.3))
	return ui.Point{X: x, Y: y}
}

func (a *App) viewAppleDrop(c *ui.Context, pane ui.Rect) {
	inspiration := &a.editorInspiration
	drop := inspiration.drop
	// The captured trajectory is in the editing pane's coordinates.
	if pane.W != drop.paneWidth || pane.H != drop.paneHeight {
		a.revealInspiration(c)
		return
	}
	const fall = 320 * time.Millisecond
	const rise = 70 * time.Millisecond
	const settle = 100 * time.Millisecond
	apple := ui.Box(c.Key("editor-inspiration-drop")).Absolute().
		Left(drop.apple.X).Top(drop.apple.Y).Size(drop.apple.W, drop.apple.H).
		PassThrough().AccessHidden(true).ZIndex(3).Draw(func(p *ui.Painter, r ui.Rect) {
		drawInspirationApple(p, r, a.pal.dark)
	})
	timeline := apple.Timeline("fall", ui.TimelineOptions{Run: inspiration.picks})
	dx := drop.landing.X - drop.apple.X - drop.apple.W/2
	dy := drop.landing.Y - drop.apple.Y - drop.apple.H/2
	x := timeline.Tween(ui.Tween{From: 0, To: dx, Duration: fall, Ease: ui.EaseInOut}, ui.AtTime(0))
	y := timeline.Tween(ui.Tween{From: 0, To: dy, Duration: fall, Ease: ui.EaseIn}, ui.AtTime(0))
	y += timeline.Tween(ui.Tween{From: 0, To: -8, Duration: rise}, ui.AtTime(fall))
	y += timeline.Tween(ui.Tween{From: 0, To: 8, Duration: settle, Ease: ui.EaseIn}, ui.AtTime(fall+rise))
	impact := timeline.Tween(ui.Tween{From: 0.56, To: 0, Duration: rise + settle}, ui.AtTime(fall))
	if timeline.Settled() {
		apple.Opacity(0)
		a.revealInspiration(c)
		return
	}
	apple.Translate(x, y)
	if timeline.Time() >= fall {
		ui.Box(c.Key("editor-inspiration-impact")).Absolute().Left(drop.landing.X-13).Top(drop.landing.Y+3).
			Size(26, 9).Radius(5).Border(1, a.pal.danger).Opacity(impact).
			PassThrough().AccessHidden(true).ZIndex(3)
	}
}

func (a *App) revealInspiration(c *ui.Context) {
	a.editorInspiration.phase = inspirationPrompt
	c.Announce(a.t(a.editorInspiration.promptKey))
}

func (a *App) resumeInspiredWriting(c *ui.Context) {
	a.editorInspiration.phase = inspirationClosed
	a.editorInspiration.body.Focus(c)
	c.Invalidate()
}

func inspirationTreeLayout(r ui.Rect) (x, y, scale float32) {
	scale = min(r.W/96, r.H/76)
	return r.X + (r.W-96*scale)/2, r.Y + (r.H-76*scale)/2, scale
}

func inspirationAppleBounds(tree ui.Rect) ui.Rect {
	x, y, scale := inspirationTreeLayout(tree)
	return ui.Rect{X: x + 55*scale, Y: y + 27*scale, W: 16 * scale, H: 20 * scale}
}

func drawInspirationTree(p *ui.Painter, r ui.Rect, dark, showApple bool) {
	x, y, scale := inspirationTreeLayout(r)
	leaf, leafTop, leafShade := ui.Hex("#7f926e"), ui.Hex("#a1b28b"), ui.Hex("#607655")
	bark := ui.Hex("#79664f")
	if dark {
		leaf, leafTop, leafShade = ui.Hex("#6f865e"), ui.Hex("#94a879"), ui.Hex("#526c48")
		bark = ui.Hex("#a28565")
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
	if showApple {
		drawInspirationApple(p, inspirationAppleBounds(r), dark)
	}
}

func drawInspirationApple(p *ui.Painter, r ui.Rect, dark bool) {
	bark, leaf, fruit := ui.Hex("#79664f"), ui.Hex("#607655"), ui.Hex("#b84233")
	if dark {
		bark, leaf, fruit = ui.Hex("#a28565"), ui.Hex("#526c48"), ui.Hex("#e56d5d")
	}
	scale := r.W / 16
	p.Fill(ui.Rect{X: r.X + 7*scale, Y: r.Y + scale, W: 2 * scale, H: 7 * scale}, bark, scale)
	p.Fill(ui.Rect{X: r.X + 8*scale, Y: r.Y, W: 6 * scale, H: 3 * scale}, leaf, 1.5*scale)
	p.Fill(ui.Rect{X: r.X, Y: r.Y + 5*scale, W: r.W, H: 15 * scale}, fruit, 7.5*scale)
	p.Fill(ui.Rect{X: r.X + 3*scale, Y: r.Y + 7*scale, W: 4 * scale, H: 3 * scale}, ui.RGBA(255, 255, 255, 0.23), 1.5*scale)
}
