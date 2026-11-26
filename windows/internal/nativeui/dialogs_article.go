package nativeui

import (
	"context"
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
		a.t("dialog.unsaved.title"),
		a.t("dialog.unsaved.message"),
		a.t("common.cancel"), a.t("dialog.unsaved.discard"), a.t("dialog.unsaved.saveAndContinue"), a.pal) {
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
		Title:        a.t("dialog.articleSettings.title"),
		Width:        480,
		PrimaryText:  a.t("dialog.articleSettings.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("dialog.articleSettings.dateField"), "", pal, func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0).Children(func() {
						ui.TextInput(c.Key("art-set-date"), &a.articleSettingsDate).Placeholder("YYYY-MM-DD HH:MM").Label(a.t("dialog.articleSettings.dateField")).AlignSelf(ui.Stretch)
					})
					if macSecondaryButton(c, "art-set-date-now", a.t("dialog.articleSettings.setNow"), pal) {
						a.articleSettingsDate = time.Now().Format(articleDateInputLayout)
					}
				})
			})
			macField(c, a.t("dialog.articleSettings.slugField"), "", pal, func() {
				ui.TextInput(c.Key("art-set-slug"), &a.articleSettingsSlug).Placeholder(a.t("dialog.articleSettings.slugPlaceholder")).Label(a.t("dialog.articleSettings.slugLabel")).AlignSelf(ui.Stretch)
			})
			macField(c, a.t("dialog.articleSettings.typeField"), "", pal, func() {
				macSegmented(c, "art-set-type", &a.articleSettingsType, []string{a.t("dialog.articleSettings.typeBlog"), a.t("dialog.articleSettings.typePage")}, pal)
			})
			macField(c, a.t("dialog.articleSettings.extLinkField"), "", pal, func() {
				ui.TextInput(c.Key("art-set-ext"), &a.articleSettingsExtLink).Placeholder(a.t("dialog.articleSettings.extLinkPlaceholder")).Label(a.t("dialog.articleSettings.extLinkField")).AlignSelf(ui.Stretch)
			})
		})
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Checkbox(c.Key("art-set-nav"), &a.articleSettingsInNav, a.t("dialog.articleSettings.showInNav"))
				ui.Box(c).Width(110).Children(func() {
					ui.TextInput(c.Key("art-set-weight"), &a.articleSettingsWeight).Placeholder(a.t("dialog.articleSettings.weightPlaceholder")).Label(a.t("dialog.articleSettings.weightLabel")).AlignSelf(ui.Stretch)
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
				a.articleSettingsErr = a.t("dialog.articleSettings.weightIntErr")
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
					a.articleSettingsErr = a.t("dialog.articleSettings.dateFormatErr")
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
		a.setStatus(a.t("status.artSettingsUpdated"))
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
		Title:        a.t("dialog.moveArticle.title"),
		Width:        420,
		PrimaryText:  a.t("common.move"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Text(c, a.tf("dialog.moveArticle.prompt", a.fallbackTitle(a.moveArticleTitle))).
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
				btn := ui.ButtonBase(c.Key("move-target-"+targetID)).Label(a.fallbackTitle(s.Name)).
					AlignSelf(ui.Stretch).Padding(8, 10, 8, 10).Radius(6).Background(bg).
					Cursor(ui.CursorPointer).Transition(rowHoverTransition)
				if !active && btn.Hovered() {
					btn.Background(pal.hover)
				}
				btn.Children(func() {
					ui.Text(c, a.fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(fg)
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
			a.moveArticleErr = a.t("dialog.moveArticle.targetRequired")
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
				targetName = a.fallbackTitle(s.Name)
				break
			}
		}
		if a.selectedReadID == a.moveArticleID {
			a.selectedReadID = ""
		}
		a.moveArticleOpen = false
		a.Reload()
		a.setStatus(a.tf("status.artMoved", targetName))
	}
}

func (a *App) viewDeleteArticleDialog(c *ui.Context) {
	if macConfirmSheet(c, &a.deleteArticleOpen,
		a.t("dialog.deleteArticle.title"),
		a.tf("dialog.deleteArticle.message", a.fallbackTitle(a.deleteArticleTitle)),
		a.t("common.cancel"), a.t("common.delete"), true, a.pal) {
		if a.backend != nil && a.selectedSiteID != "" && a.deleteArticleID != "" {
			if err := a.backend.DeleteArticle(a.selectedSiteID, a.deleteArticleID); err != nil {
				a.setError(a.tf("status.deleteArtErr", err.Error()))
			} else {
				if a.selectedReadID == a.deleteArticleID {
					a.selectedReadID = ""
				}
				a.Reload()
				a.setStatus(a.t("status.artDeleted"))
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
			a.t("dialog.quickPost.discardTitle"),
			a.t("dialog.quickPost.discardMsg"),
			a.t("dialog.quickPost.keepEditing"), a.t("dialog.quickPost.discard"), true, pal) {
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
		Title:        a.t("dialog.quickPost.title"),
		Width:        500,
		PrimaryText:  a.t("dialog.quickPost.primary"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		if len(activeSites) > 1 {
			ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
				ui.Text(c, a.t("dialog.quickPost.publishTo")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				for _, s := range activeSites {
					active := a.quickPostSiteID == s.ID
					bg, fg := pal.subtle, pal.ink
					if active {
						bg, fg = pal.accent, ui.Hex("#ffffff")
					}
					chip := ui.ButtonBase(c.Key("qp-site-"+s.ID)).Label(a.fallbackTitle(s.Name)).
						Padding(4, 10, 4, 10).Radius(6).Background(bg).Border(0.5, pal.line).
						Cursor(ui.CursorPointer).Transition(rowHoverTransition)
					if !active && chip.Hovered() {
						chip.Background(pal.hover)
					}
					chip.Children(func() {
						ui.Text(c, a.fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(fg).NoWrap()
					})
					if chip.Clicked() {
						a.quickPostSiteID = s.ID
					}
				}
			})
		}
		ui.TextInput(c.Key("qp-title"), &a.quickPostTitle).Placeholder(a.t("dialog.quickPost.titlePlaceholder")).Label(a.t("dialog.quickPost.titleLabel")).AutoFocus().AlignSelf(ui.Stretch)
		ui.TextArea(c.Key("qp-content"), &a.quickPostContent).Placeholder(a.t("dialog.quickPost.contentPlaceholder")).Label(a.t("dialog.quickPost.contentLabel")).Lines(5, 12).AlignSelf(ui.Stretch)
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "qp-pick-att", a.t("dialog.quickPost.addMedia"), pal) {
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
					if macSecondaryButton(c, "qp-rm-att", a.t("common.remove"), pal) {
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
			a.quickPostErr = a.t("dialog.quickPost.emptyErr")
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
		a.setStatus(a.tf("status.quickPostPublished", a.fallbackTitle(art.Title)))
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
		a.setError(a.tf("status.pinErr", err.Error()))
		return
	}
	a.Reload()
	if nextPinned {
		a.setStatus(a.tf("status.pinnedArt", a.fallbackTitle(art.Title)))
	} else {
		a.setStatus(a.tf("status.unpinnedArt", a.fallbackTitle(art.Title)))
	}
}
