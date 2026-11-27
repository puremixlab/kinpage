package nativeui

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"windows/internal/mediainfo"
	"windows/internal/tagname"
)

func (a *App) viewEditorMetadata(c *ui.Context) {
	pal := a.pal
	ui.Column(c).Padding(12, 14, 12, 14).Gap(12).Background(pal.canvas).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
		// Tags section.
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.t("editor.tagsHeading")).Font(macPlanetFont).FontSize(11.5).FontWeight(600).TextColor(pal.muted)
			keys := sortedTagKeys(a.editorTags)
			if len(keys) > 0 {
				ui.Row(c).Gap(6).Wrap().AlignSelf(ui.Stretch).Children(func() {
					for _, k := range keys {
						label := a.editorTags[k]
						b := ui.ButtonBase(c.Key("rm-tag-"+k)).Label(label+" ×").
							Padding(3, 8, 3, 8).Radius(6).Background(pal.subtle).Border(0.5, pal.line).
							Cursor(ui.CursorPointer).Transition(rowHoverTransition)
						if b.Hovered() {
							b.Background(pal.hover)
						}
						b.Children(func() {
							ui.Text(c, label+" ×").Font(macPlanetFont).FontSize(12).TextColor(pal.ink).NoWrap()
						})
						if b.Clicked() {
							delete(a.editorTags, k)
							a.editorTagsDirty = true
						}
					}
				})
			}
			ui.Row(c).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				if ui.TextInput(c.Key("add-tag-input"), &a.newTagInput).Placeholder(a.t("editor.addTagPlaceholder")).Label(a.t("editor.addTagLabel")).Grow(1).Submitted() {
					a.addEditorTag()
				}
				if macSecondaryButton(c, "add-tag-btn", a.t("editor.addTagBtn"), pal) {
					a.addEditorTag()
				}
			})
		})

		hairlineH(c, pal)

		// Attachments & HeroImage section.
		ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, a.t("editor.attachmentsHeading")).Font(macPlanetFont).FontSize(11.5).FontWeight(600).TextColor(pal.muted)
				if macSecondaryButton(c, "add-attachment-btn", a.t("editor.addAttachmentBtn"), pal) {
					a.addDraftAttachment()
				}
			})
			if len(a.editorAttachments) == 0 {
				ui.Text(c, a.t("editor.noAttachments")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
			}
			for _, name := range a.editorAttachments {
				isHero := a.editorHeroImage == name
				ui.Row(c.Key("att-row-"+name)).Gap(6).AlignItems(ui.Center).Padding(6, 8, 6, 8).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
					if isHero {
						ui.Icon(c, iconStarSolid).Size(12, 12).TextColor(pal.yellow).Shrink(0)
					}
					ui.Text(c, name).Font(macPlanetFont).FontSize(12).TextColor(pal.ink).MaxLines(1).Ellipsis("…").Grow(1)
					if macSecondaryButton(c, "att-insert-"+name, a.t("editor.insertBtn"), pal) {
						a.insertAttachmentMarkdown(name)
					}
					if isImageFilename(name) {
						heroLabel := a.t("editor.setHeroBtn")
						if isHero {
							heroLabel = a.t("editor.unsetHeroBtn")
						}
						if macSecondaryButton(c, "att-hero-"+name, heroLabel, pal) {
							nextHero := name
							if isHero {
								nextHero = ""
							}
							a.setDraftHeroImage(nextHero)
						}
					}
					if isVideoFilename(name) {
						btn := ui.ButtonBase(c.Key("att-compress-"+name)).Label(a.t("editor.compressVideo")).
							Height(26).PaddingX(8).Radius(6).Background(pal.canvas).Border(0.5, pal.controlLine).
							Center().Cursor(ui.CursorPointer)
						if btn.Hovered() {
							btn.Background(pal.hover)
						}
						btn.Children(func() {
							ui.Text(c, a.t("editor.compressBtn")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.ink).NoWrap()
						})
						btn.Menu(func(m *ui.Menu) {
							presets := []struct{ id, label string }{
								{mediainfo.VideoPresetH2641080p, "H.264 1080p"},
								{mediainfo.VideoPresetH264720p, "H.264 720p"},
								{mediainfo.VideoPresetHEVC1080p, "HEVC 1080p"},
								{mediainfo.VideoPresetHEVC720p, "HEVC 720p"},
							}
							for _, p := range presets {
								if m.Item(p.label).Chosen() {
									if a.backend != nil && a.selectedSiteID != "" && a.activeDraftID != "" {
										if _, err := a.backend.CompressDraftVideo(a.selectedSiteID, a.activeDraftID, name, p.id); err != nil {
											a.setError(a.tf("status.compressErr", err.Error()))
										} else {
											a.setStatus(a.tf("status.compressDone", name, p.label))
										}
									}
								}
							}
						})
					}
					if macSecondaryButton(c, "att-rm-"+name, "×", pal) {
						a.removeDraftAttachment(name)
					}
				})
			}
		})
	})
}

func (a *App) addEditorTag() {
	label := strings.TrimSpace(a.newTagInput)
	if label == "" {
		return
	}
	key := makeTagKey(label)
	if !tagname.Valid(key, label) {
		a.setError(a.t("status.invalidTagErr"))
		return
	}
	if a.editorTags == nil {
		a.editorTags = map[string]string{}
	}
	a.editorTags[key] = label
	a.editorTagsDirty = true
	a.newTagInput = ""
	a.clearStatus()
}

func (a *App) addDraftAttachment() {
	if a.backend == nil || a.selectedSiteID == "" {
		return
	}
	if a.activeDraftID == "" {
		if !a.saveCurrentDraft() {
			return
		}
	}
	draft, err := a.backend.AddDraftAttachment(context.Background(), a.selectedSiteID, a.activeDraftID)
	if err != nil {
		a.setError(a.tf("status.addAttErr", err.Error()))
		return
	}
	if draft.ID == "" {
		return
	}
	a.editorAttachments = append([]string(nil), draft.Attachments...)
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	a.setStatus(a.t("status.attUpdated"))
}

func (a *App) removeDraftAttachment(name string) {
	if a.backend == nil || a.selectedSiteID == "" || a.activeDraftID == "" {
		return
	}
	draft, err := a.backend.RemoveDraftAttachment(a.selectedSiteID, a.activeDraftID, name)
	if err != nil {
		a.setError(a.tf("status.removeAttErr", err.Error()))
		return
	}
	a.editorAttachments = append([]string(nil), draft.Attachments...)
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	a.setStatus(a.tf("status.attRemoved", name))
}

func (a *App) setDraftHeroImage(name string) {
	if a.backend == nil || a.selectedSiteID == "" || a.activeDraftID == "" {
		return
	}
	draft, err := a.backend.SetDraftHeroImage(a.selectedSiteID, a.activeDraftID, name)
	if err != nil {
		a.setError(a.tf("status.setHeroErr", err.Error()))
		return
	}
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	if name == "" {
		a.setStatus(a.t("status.heroUnset"))
	} else {
		a.setStatus(a.tf("status.heroSet", name))
	}
}

func (a *App) insertAttachmentMarkdown(name string) {
	snippet := "[" + name + "](" + name + ")"
	if isImageFilename(name) {
		snippet = "![" + name + "](" + name + ")"
	}
	pos := a.editorState.CursorPos()
	off := a.editorBuffer.Offset(pos)
	a.editorBuffer.Edit(editor.Edit{Start: off, End: off, Text: snippet})
	a.setStatus(a.tf("status.attInserted", name))
}

func (a *App) marshalEditorTags() (string, error) {
	tags := a.editorTags
	if tags == nil {
		tags = map[string]string{}
	}
	b, err := json.Marshal(tags)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func sortedTagKeys(tags map[string]string) []string {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func cloneTagsMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func makeTagKey(label string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) || r == '_' {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func isImageFilename(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".svg":
		return true
	default:
		return false
	}
}

func isVideoFilename(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv":
		return true
	default:
		return false
	}
}
