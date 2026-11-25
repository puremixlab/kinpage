package nativeui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

func (a *App) guardUnsaved(action func()) {
	if !a.isEditorDirty() {
		action()
		return
	}
	a.pendingNavAction = action
	a.unsavedDialogOpen = true
}

func (a *App) viewUnsavedDialog(c *ui.Context) {
	switch macThreeActionSheet(c, &a.unsavedDialogOpen,
		"有未保存的草稿更改",
		"是否先保存当前草稿再离开？",
		"取消", "放弃更改", "保存并继续", a.pal) {
	case 1:
		a.editorBuffer.MarkSaved()
		a.savedTitle = a.editorTitle
		a.editorTagsDirty = false
		act := a.pendingNavAction
		a.pendingNavAction = nil
		if act != nil {
			act()
		}
	case 2:
		if a.saveCurrentDraft() {
			act := a.pendingNavAction
			a.pendingNavAction = nil
			if act != nil {
				act()
			}
		}
	default:
		if !a.unsavedDialogOpen {
			a.pendingNavAction = nil
		}
	}
}

const articleDateInputLayout = "2006-01-02 15:04"

func (a *App) openArticleSettings(art store.Article) {
	a.articleSettingsID = art.ID
	a.articleSettingsSlug = art.Slug
	a.articleSettingsType = art.ArticleType
	a.articleSettingsExtLink = art.ExternalLink
	a.articleSettingsInNav = art.IsIncludedInNavigation
	a.articleSettingsWeight = strconv.Itoa(art.NavigationWeight)
	dateStr := ""
	if !art.Created.IsZero() {
		dateStr = art.Created.Local().Format(articleDateInputLayout)
	}
	a.articleSettingsDate = dateStr
	a.articleSettingsOrigDate = dateStr
	a.articleSettingsErr = ""
	a.articleSettingsOpen = true
}

func (a *App) viewArticleSettingsModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.articleSettingsOpen, macSheetProps{
		Title:       "文章设置",
		Width:       480,
		PrimaryText: "保存设置",
		CancelText:  "取消",
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, "发布日期", "", pal, func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0).Children(func() {
						ui.TextInput(c.Key("art-set-date"), &a.articleSettingsDate).Placeholder("YYYY-MM-DD HH:MM").Label("发布日期").AlignSelf(ui.Stretch)
					})
					if macSecondaryButton(c, "art-set-date-now", "设为当前时间", pal) {
						a.articleSettingsDate = time.Now().Format(articleDateInputLayout)
					}
				})
			})
			macField(c, "自定义路径 (Slug)", "", pal, func() {
				ui.TextInput(c.Key("art-set-slug"), &a.articleSettingsSlug).Placeholder("留空使用默认路径").Label("自定义路径").AlignSelf(ui.Stretch)
			})
			macField(c, "类型", "", pal, func() {
				macSegmented(c, "art-set-type", &a.articleSettingsType, []string{"博客文章", "独立页面"}, pal)
			})
			macField(c, "外部链接", "", pal, func() {
				ui.TextInput(c.Key("art-set-ext"), &a.articleSettingsExtLink).Placeholder("https://…（可选）").Label("外部链接").AlignSelf(ui.Stretch)
			})
		})
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Checkbox(c.Key("art-set-nav"), &a.articleSettingsInNav, "显示在站点导航栏")
				ui.Box(c).Width(110).Children(func() {
					ui.TextInput(c.Key("art-set-weight"), &a.articleSettingsWeight).Placeholder("权重 0").Label("导航权重").AlignSelf(ui.Stretch)
				})
			})
		})
		if a.articleSettingsErr != "" {
			ui.Text(c, a.articleSettingsErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		weight := 0
		if w := strings.TrimSpace(a.articleSettingsWeight); w != "" {
			parsed, err := strconv.Atoi(w)
			if err != nil {
				a.articleSettingsErr = "导航权重必须为整数"
				return
			}
			weight = parsed
		}
		var newCreated time.Time
		dateChanged := strings.TrimSpace(a.articleSettingsDate) != strings.TrimSpace(a.articleSettingsOrigDate)
		if dateChanged {
			rawDate := strings.TrimSpace(a.articleSettingsDate)
			parsedDate, err := time.ParseInLocation(articleDateInputLayout, rawDate, time.Local)
			if err != nil {
				parsedDate, err = time.ParseInLocation("2006-01-02", rawDate, time.Local)
				if err != nil {
					a.articleSettingsErr = "发布日期格式应为 YYYY-MM-DD HH:MM"
					return
				}
			}
			newCreated = parsedDate.UTC()
		}
		_, err := a.backend.UpdateArticleSettings(
			a.selectedSiteID,
			a.articleSettingsID,
			strings.TrimSpace(a.articleSettingsSlug),
			a.articleSettingsType,
			strings.TrimSpace(a.articleSettingsExtLink),
			a.articleSettingsInNav,
			weight,
		)
		if err != nil {
			a.articleSettingsErr = err.Error()
			return
		}
		if dateChanged {
			if _, err := a.backend.UpdateArticleDate(a.selectedSiteID, a.articleSettingsID, newCreated); err != nil {
				a.articleSettingsErr = err.Error()
				return
			}
		}
		a.articleSettingsOpen = false
		a.Reload()
		a.setStatus("文章设置已更新")
	}
}

func (a *App) hasOtherActiveSite(excludeSiteID string) bool {
	for _, s := range a.sites {
		if !s.Archived && s.ID != excludeSiteID {
			return true
		}
	}
	return false
}

func (a *App) openMoveArticleSheet(art store.Article) {
	a.moveArticleID = art.ID
	a.moveArticleTitle = art.Title
	a.moveArticleFromSiteID = art.SiteID
	a.moveArticleTargetSiteID = ""
	for _, s := range a.sites {
		if !s.Archived && s.ID != art.SiteID {
			a.moveArticleTargetSiteID = s.ID
			break
		}
	}
	a.moveArticleErr = ""
	a.moveArticleOpen = true
}

func (a *App) viewMoveArticleSheet(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.moveArticleOpen, macSheetProps{
		Title:       "移动文章",
		Width:       420,
		PrimaryText: "移动",
		CancelText:  "取消",
	}, pal, func() {
		ui.Text(c, fmt.Sprintf("将“%s”移动到目标站点：", fallbackTitle(a.moveArticleTitle))).
			Font(macPlanetFont).FontSize(13).TextColor(pal.secondary)
		macInsetCard(c, pal, func() {
			for _, s := range a.sites {
				if s.Archived || s.ID == a.moveArticleFromSiteID {
					continue
				}
				targetID := s.ID
				active := a.moveArticleTargetSiteID == targetID
				bg, fg := ui.Transparent, pal.ink
				if active {
					bg, fg = pal.accent, ui.Hex("#ffffff")
				}
				btn := ui.ButtonBase(c.Key("move-target-"+targetID)).Label(fallbackTitle(s.Name)).
					AlignSelf(ui.Stretch).Padding(8, 10, 8, 10).Radius(6).Background(bg).
					Cursor(ui.CursorPointer).Transition(rowHoverTransition)
				if !active && btn.Hovered() {
					btn.Background(pal.hover)
				}
				btn.Children(func() {
					ui.Text(c, fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(fg)
				})
				if btn.Clicked() {
					a.moveArticleTargetSiteID = targetID
				}
			}
		})
		if a.moveArticleErr != "" {
			ui.Text(c, a.moveArticleErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if a.moveArticleTargetSiteID == "" {
			a.moveArticleErr = "请选择目标站点"
			return
		}
		if a.backend == nil {
			return
		}
		if _, err := a.backend.MoveArticle(a.moveArticleFromSiteID, a.moveArticleID, a.moveArticleTargetSiteID); err != nil {
			a.moveArticleErr = err.Error()
			return
		}
		targetName := a.moveArticleTargetSiteID
		for _, s := range a.sites {
			if s.ID == a.moveArticleTargetSiteID {
				targetName = fallbackTitle(s.Name)
				break
			}
		}
		if a.selectedReadID == a.moveArticleID {
			a.selectedReadID = ""
		}
		a.moveArticleOpen = false
		a.Reload()
		a.setStatus("文章已移动至「" + targetName + "」")
	}
}

func (a *App) viewDeleteArticleDialog(c *ui.Context) {
	if macConfirmSheet(c, &a.deleteArticleOpen,
		"删除文章？",
		fmt.Sprintf("确定要删除“%s”吗？此操作不可撤销。", fallbackTitle(a.deleteArticleTitle)),
		"取消", "删除", true, a.pal) {
		if a.backend != nil && a.selectedSiteID != "" && a.deleteArticleID != "" {
			if err := a.backend.DeleteArticle(a.selectedSiteID, a.deleteArticleID); err != nil {
				a.setError("删除文章失败：" + err.Error())
			} else {
				if a.selectedReadID == a.deleteArticleID {
					a.selectedReadID = ""
				}
				a.Reload()
				a.setStatus("文章已删除")
			}
		}
		a.deleteArticleID = ""
		a.deleteArticleTitle = ""
	} else if !a.deleteArticleOpen {
		a.deleteArticleID = ""
		a.deleteArticleTitle = ""
	}
}

func (a *App) openQuickPost() {
	var activeSites []store.Site
	for _, s := range a.sites {
		if !s.Archived {
			activeSites = append(activeSites, s)
		}
	}
	if len(activeSites) == 0 {
		a.createSiteOpen = true
		return
	}
	validSelected := false
	for _, s := range activeSites {
		if s.ID == a.quickPostSiteID {
			validSelected = true
			break
		}
	}
	if !validSelected {
		a.quickPostSiteID = activeSites[0].ID
		for _, s := range activeSites {
			if s.ID == a.selectedSiteID {
				a.quickPostSiteID = s.ID
				break
			}
		}
	}
	a.quickPostTitle = ""
	a.quickPostContent = ""
	a.quickPostAttachmentName = ""
	a.quickPostAttachmentB64 = ""
	a.quickPostDiscardConfirm = false
	a.quickPostErr = ""
	a.quickPostOpen = true
}

func (a *App) viewQuickPostModal(c *ui.Context) {
	pal := a.pal
	if a.quickPostDiscardConfirm {
		if macConfirmSheet(c, &a.quickPostDiscardConfirm,
			"放弃未发布的快捷发文？",
			"已输入的内容与附件将被清空。",
			"继续编辑", "放弃", true, pal) {
			a.quickPostOpen = false
			a.quickPostTitle = ""
			a.quickPostContent = ""
			a.quickPostAttachmentName = ""
			a.quickPostAttachmentB64 = ""
		}
		return
	}
	wasOpen := a.quickPostOpen
	var activeSites []store.Site
	for _, s := range a.sites {
		if !s.Archived {
			activeSites = append(activeSites, s)
		}
	}
	if macSheet(c, &a.quickPostOpen, macSheetProps{
		Title:       "快捷发文",
		Width:       500,
		PrimaryText: "立即发布",
		CancelText:  "取消",
	}, pal, func() {
		if len(activeSites) > 1 {
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "发布到：").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				for _, s := range activeSites {
					active := a.quickPostSiteID == s.ID
					bg, fg := pal.subtle, pal.ink
					if active {
						bg, fg = pal.accent, ui.Hex("#ffffff")
					}
					chip := ui.ButtonBase(c.Key("qp-site-"+s.ID)).Label(fallbackTitle(s.Name)).
						Padding(4, 10, 4, 10).Radius(6).Background(bg).Border(0.5, pal.line).
						Cursor(ui.CursorPointer).Transition(rowHoverTransition)
					if !active && chip.Hovered() {
						chip.Background(pal.hover)
					}
					chip.Children(func() {
						ui.Text(c, fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(fg).NoWrap()
					})
					if chip.Clicked() {
						a.quickPostSiteID = s.ID
					}
				}
			})
		}
		ui.TextInput(c.Key("qp-title"), &a.quickPostTitle).Placeholder("标题（可选）").Label("快捷发文标题").AutoFocus().AlignSelf(ui.Stretch)
		ui.TextArea(c.Key("qp-content"), &a.quickPostContent).Placeholder("输入 Markdown 内容…").Label("快捷发文正文").Lines(5, 12).AlignSelf(ui.Stretch)
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "qp-pick-att", "+ 添加媒体附件", pal) {
				if a.backend != nil {
					name, b64, err := a.backend.PickQuickPostAttachment(context.Background())
					if err != nil {
						a.quickPostErr = err.Error()
					} else if name != "" {
						a.quickPostAttachmentName = name
						a.quickPostAttachmentB64 = b64
						a.quickPostErr = ""
					}
				}
			}
			if a.quickPostAttachmentName != "" {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, a.quickPostAttachmentName).Font(macPlanetFont).FontSize(12).TextColor(pal.secondary).MaxLines(1).Ellipsis("…")
					if macSecondaryButton(c, "qp-rm-att", "移除", pal) {
						a.quickPostAttachmentName = ""
						a.quickPostAttachmentB64 = ""
					}
				})
			}
		})
		if a.quickPostErr != "" {
			ui.Text(c, a.quickPostErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if strings.TrimSpace(a.quickPostTitle) == "" && strings.TrimSpace(a.quickPostContent) == "" && a.quickPostAttachmentName == "" {
			a.quickPostErr = "请输入标题、正文或添加附件"
			return
		}
		art, err := a.backend.QuickPost(a.quickPostSiteID, strings.TrimSpace(a.quickPostTitle), a.quickPostContent, a.quickPostAttachmentName, a.quickPostAttachmentB64)
		if err != nil {
			a.quickPostErr = err.Error()
			return
		}
		a.quickPostOpen = false
		a.quickPostAttachmentName = ""
		a.quickPostAttachmentB64 = ""
		a.Reload()
		a.openArticleForRead(a.quickPostSiteID, art.ID)
		a.setStatus("快捷发文已发布：" + fallbackTitle(art.Title))
	} else if wasOpen && !a.quickPostOpen {
		hasDraft := strings.TrimSpace(a.quickPostTitle) != "" || strings.TrimSpace(a.quickPostContent) != "" || a.quickPostAttachmentName != ""
		if hasDraft {
			a.quickPostOpen = true
			a.quickPostDiscardConfirm = true
		}
	}
}

func (a *App) toggleArticlePinned(art store.Article) {
	if a.backend == nil || a.selectedSiteID == "" {
		return
	}
	nextPinned := art.Pinned == nil
	if _, err := a.backend.SetArticlePinned(a.selectedSiteID, art.ID, nextPinned); err != nil {
		a.setError("更新置顶状态失败：" + err.Error())
		return
	}
	a.Reload()
	if nextPinned {
		a.setStatus("已置顶文章：" + fallbackTitle(art.Title))
	} else {
		a.setStatus("已取消置顶：" + fallbackTitle(art.Title))
	}
}
