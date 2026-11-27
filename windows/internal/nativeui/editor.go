package nativeui

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobemd "github.com/lobehub/lobe-mygo/pkg/markdown"
	"github.com/lobehub/lobe-mygo/pkg/overlay"
)

func (a *App) viewEditor(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		// Single 760px rail matching mac-planet.css:837-886 & writing-editor.png
		// (save state, 保存草稿, 发布, 删除 live in the unified top titlebar).
		ui.Column(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).AlignItems(ui.Center).Background(pal.canvas).Children(func() {
			maxW := float32(740)
			if a.editorPaneMode == editorPaneSplit {
				maxW = 1440
			}
			colW := a.detailColumnWidth(c, maxW)
			ui.Column(c).Grow(1).MinHeight(0).Width(colW).PaddingY(24).Gap(12).Children(func() {
				// 26px SemiBold truly borderless paper title input aligned flush with editor body (+8px).
				ui.Box(c).AlignSelf(ui.Stretch).PaddingX(8).Children(func() {
					ui.TextInputBase(c.Key("editor-title"), &a.editorTitle).
						Placeholder(a.t("editor.titlePlaceholder")).Label(a.t("editor.titleLabel")).
						Font(macPlanetFont).FontSize(26).FontWeight(600).TextColor(pal.ink).
						Background(ui.Transparent).Border(0, ui.Transparent).Padding(0).FocusRing(false).
						AlignSelf(ui.Stretch)
				})

				// Metadata floating popover trigger + crisp Segmented switch aligned flush with editor body.
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).PaddingX(8).Gap(8).Shrink(0).Children(func() {
					metaSummary := a.tf("editor.metaSummary", len(a.editorTags), len(a.editorAttachments))
					metaBg := pal.subtle
					if a.editorMetaOpen {
						metaBg = pal.selection
					}
					metaAnchor := ui.ButtonBase(c.Key("editor-meta-toggle")).Label(a.t("editor.metaLabel")).
						Background(metaBg).Border(0.5, pal.line).Radius(6).PaddingX(10).PaddingY(5).MinWidth(0).
						Cursor(ui.CursorPointer).Transition(rowHoverTransition)
					if !a.editorMetaOpen && metaAnchor.Hovered() {
						metaAnchor.Background(pal.hover)
					}
					metaAnchor.Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Text(c, metaSummary).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).MinWidth(0).MaxLines(1).Ellipsis("…")
							ui.Icon(c, iconChevronDown).Size(12, 12).TextColor(pal.muted).Shrink(0)
						})
					})
					overlay.Popover(c, metaAnchor, &a.editorMetaOpen, overlay.PopoverProps{Placement: overlay.Bottom}, func() {
						ui.Column(c).Width(360).Children(func() {
							a.viewEditorMetadata(c)
						})
					})

					ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
						previousMode := a.editorPaneMode
						macSegmented(c, "editor-mode", &a.editorPaneMode, []string{a.t("editor.modeWrite"), a.t("editor.modeSplit"), a.t("editor.modePreview")}, pal)
						if a.editorPaneMode != previousMode || a.editorPaneMode == editorPaneView {
							a.editorInspiration.phase = inspirationClosed
						}
						if a.editorPaneMode != editorPaneView {
							a.viewInspirationToggle(c, colW < 500)
						}
					})
				})

				ui.Box(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					switch a.editorPaneMode {
					case editorPaneWrite:
						a.viewEditorBody(c)
					case editorPaneView:
						a.viewMarkdownPreview(c, "", a.editorBuffer.String(), &a.previewScroll)
					default:
						ui.SplitPercent(c.Key("editor-split"), &a.editorSplitRatio,
							func() { a.viewEditorBody(c) },
							func() { a.viewMarkdownPreview(c, "", a.editorBuffer.String(), &a.previewScroll) },
						).SplitReset(50).Fill()
					}
				})
			})
		})

		// Minimal footer aligned on the 760px rail matching writing-editor.png.
		hairlineH(c, pal)
		pos := a.editorState.CursorPos()
		chars := utf8.RuneCountInString(a.editorBuffer.String())
		ui.Row(c).MinHeight(36).PaddingX(32).Gap(16).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.canvas).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.t("editor.footerLocalOnly")).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
			ui.Text(c, a.tf("editor.footerCursor", pos.Line+1, pos.Col+1, chars)).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
		})
	})
}

func (a *App) viewMarkdownPreview(c *ui.Context, title, content string, scroll *ui.ScrollState) {
	pal := a.pal
	ui.Scroll(c).TrackScroll(scroll).Fill().Background(pal.canvas).Children(func() {
		ui.Column(c).PaddingX(8).PaddingY(8).Gap(14).AlignSelf(ui.Stretch).Children(func() {
			if strings.TrimSpace(title) != "" {
				ui.Text(c, title).Font(macPlanetFont).FontSize(28).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.ink)
			}
			if strings.TrimSpace(content) == "" {
				display.Empty(c, display.EmptyProps{
					Icon:        lucide.Eye(),
					Title:       a.t("editor.previewEmptyTitle"),
					Description: a.t("editor.previewEmptyDesc"),
				})
			} else {
				lobemd.Markdown(c, content, lobemd.MarkdownProps{
					FontSize:              15,
					LineHeight:            1.8,
					HeaderMultiple:        0.65,
					MarginMultiple:        1.2,
					FullFeaturedCodeBlock: true,
					LoadImage:             a.markdownImageLoader("", ""),
				})
			}
		})
	})
}
