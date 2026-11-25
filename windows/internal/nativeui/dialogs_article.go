package nativeui

import (
	"fmt"
	"strconv"
	"strings"

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

func (a *App) openArticleSettings(art store.Article) {
	a.articleSettingsID = art.ID
	a.articleSettingsSlug = art.Slug
	a.articleSettingsType = art.ArticleType
	a.articleSettingsExtLink = art.ExternalLink
	a.articleSettingsInNav = art.IsIncludedInNavigation
	a.articleSettingsWeight = strconv.Itoa(art.NavigationWeight)
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
			macField(c, "自定义路径 (Slug)", "留空使用默认路径", pal, func() {
				ui.TextInput(c.Key("art-set-slug"), &a.articleSettingsSlug).Placeholder("留空使用默认路径").Label("自定义路径").AlignSelf(ui.Stretch)
			})
			macField(c, "类型", "", pal, func() {
				macSegmented(c, "art-set-type", &a.articleSettingsType, []string{"博客文章", "独立页面"}, pal)
			})
			macField(c, "外部链接", "可选", pal, func() {
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
		a.articleSettingsOpen = false
		a.Reload()
		a.setStatus("文章设置已更新")
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
	if len(a.sites) == 0 {
		a.createSiteOpen = true
		return
	}
	if a.quickPostSiteID == "" {
		if a.selectedSiteID != "" {
			a.quickPostSiteID = a.selectedSiteID
		} else {
			a.quickPostSiteID = a.sites[0].ID
		}
	}
	a.quickPostTitle = ""
	a.quickPostContent = ""
	a.quickPostErr = ""
	a.quickPostOpen = true
}

func (a *App) viewQuickPostModal(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.quickPostOpen, macSheetProps{
		Title:       "快捷发文",
		Width:       480,
		PrimaryText: "立即发布",
		CancelText:  "取消",
	}, pal, func() {
		if len(a.sites) > 1 {
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
				ui.Text(c, "发布到：").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				for _, s := range a.sites {
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
		ui.TextArea(c.Key("qp-content"), &a.quickPostContent).Placeholder("输入 Markdown 内容…").Label("快捷发文正文").Lines(5, 10).AlignSelf(ui.Stretch)
		if a.quickPostErr != "" {
			ui.Text(c, a.quickPostErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if strings.TrimSpace(a.quickPostTitle) == "" && strings.TrimSpace(a.quickPostContent) == "" {
			a.quickPostErr = "请输入标题或正文内容"
			return
		}
		art, err := a.backend.QuickPost(a.quickPostSiteID, strings.TrimSpace(a.quickPostTitle), a.quickPostContent, "", "")
		if err != nil {
			a.quickPostErr = err.Error()
			return
		}
		a.quickPostOpen = false
		a.Reload()
		a.openArticleForRead(a.quickPostSiteID, art.ID)
		a.setStatus("快捷发文已发布：" + fallbackTitle(art.Title))
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
