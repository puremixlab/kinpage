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
	"windows/internal/tagname"
)

func (a *App) viewEditorMetadata(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).PaddingX(14).PaddingY(10).Gap(10).Background(t.Surface).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
		// Tags row.
		ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "标签").FontSize(12).Bold().TextColor(t.TextMuted)
			keys := sortedTagKeys(a.editorTags)
			for _, k := range keys {
				label := a.editorTags[k]
				if ui.Button(c.Key("rm-tag-"+k), label+" ×").Clicked() {
					delete(a.editorTags, k)
					a.editorTagsDirty = true
				}
			}
			if ui.TextInput(c.Key("add-tag-input"), &a.newTagInput).Placeholder("添加标签并回车…").Label("添加标签").Width(160).Submitted() {
				a.addEditorTag()
			}
			if ui.Button(c.Key("add-tag-btn"), "+ 标签").Clicked() {
				a.addEditorTag()
			}
		})

		// Attachments & HeroImage row.
		ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, "附件").FontSize(12).Bold().TextColor(t.TextMuted)
			if ui.Button(c.Key("add-attachment-btn"), "+ 添加附件").Clicked() {
				a.addDraftAttachment()
			}
			if len(a.editorAttachments) == 0 {
				ui.Text(c, "暂无附件").FontSize(12).TextColor(t.TextMuted)
			}
			for _, name := range a.editorAttachments {
				isHero := a.editorHeroImage == name
				ui.Row(c.Key("att-row-"+name)).Gap(4).AlignItems(ui.Center).PaddingX(6).PaddingY(3).Background(t.Background).Border(1, t.Border).Radius(6).Children(func() {
					display := name
					if isHero {
						display = "★ " + name
					}
					ui.Text(c, display).FontSize(12)
					if ui.Button(c.Key("att-insert-"+name), "插入").Clicked() {
						a.insertAttachmentMarkdown(name)
					}
					if isImageFilename(name) {
						heroLabel := "设为封面"
						if isHero {
							heroLabel = "取消封面"
						}
						if ui.Button(c.Key("att-hero-"+name), heroLabel).Clicked() {
							nextHero := name
							if isHero {
								nextHero = ""
							}
							a.setDraftHeroImage(nextHero)
						}
					}
					if ui.Button(c.Key("att-rm-"+name), "×").Clicked() {
						a.removeDraftAttachment(name)
					}
				})
			}
		})
	})
	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
}

func (a *App) addEditorTag() {
	label := strings.TrimSpace(a.newTagInput)
	if label == "" {
		return
	}
	key := makeTagKey(label)
	if !tagname.Valid(key, label) {
		a.setError("标签名称无效（不能使用保留词或特殊路径字符）")
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
		a.setError("添加附件失败：" + err.Error())
		return
	}
	if draft.ID == "" {
		return
	}
	a.editorAttachments = append([]string(nil), draft.Attachments...)
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	a.setStatus("已更新草稿附件")
}

func (a *App) removeDraftAttachment(name string) {
	if a.backend == nil || a.selectedSiteID == "" || a.activeDraftID == "" {
		return
	}
	draft, err := a.backend.RemoveDraftAttachment(a.selectedSiteID, a.activeDraftID, name)
	if err != nil {
		a.setError("移除附件失败：" + err.Error())
		return
	}
	a.editorAttachments = append([]string(nil), draft.Attachments...)
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	a.setStatus("已移除附件：" + name)
}

func (a *App) setDraftHeroImage(name string) {
	if a.backend == nil || a.selectedSiteID == "" || a.activeDraftID == "" {
		return
	}
	draft, err := a.backend.SetDraftHeroImage(a.selectedSiteID, a.activeDraftID, name)
	if err != nil {
		a.setError("设置封面图失败：" + err.Error())
		return
	}
	a.editorHeroImage = draft.HeroImage
	a.Reload()
	if name == "" {
		a.setStatus("已取消封面图")
	} else {
		a.setStatus("已设为封面图：" + name)
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
	a.setStatus("已插入引用：" + name)
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
