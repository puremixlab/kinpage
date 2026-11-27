package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"windows/internal/applog"
	"windows/internal/ipfs"
	"windows/internal/localapi"
	"windows/internal/services"
)

const (
	toolsTabAI       = 0
	toolsTabAPI      = 1
	toolsTabKeys     = 2
	toolsTabLogs     = 3
	toolsTabGeneral  = 4
	toolsTabFolders  = 5
	toolsTabOffline  = 6
	toolsTabTrash    = 7
	toolsTabTip      = 8
	toolsTabDataRepo = 9
)

type toolsModalState struct {
	open         bool
	tab          int
	search       string
	returnSearch string
	highlightID  string
	err          string
	msg          string

	// Secondary modal open states (Empty-State-First -> secondary sheet).
	aiModalOpen     bool
	apiModalOpen    bool
	keyModalOpen    bool
	folderModalOpen bool

	// Local AI chat (saved summary separated from modal draft).
	aiSavedBaseURL string
	aiSavedModel   string
	aiBaseURL      string
	aiModel        string
	aiToken        string
	aiProbe        services.AIProbeResult
	aiProbeMsg     string
	aiProbeErr     string
	aiPrompt       string
	aiHistory      []services.AIChatMessage

	// Local REST API.
	apiStatus   localapi.Status
	apiEnabled  bool
	apiPort     string
	apiAuthOn   bool
	apiUsername string
	apiPasscode string

	// IPNS keys.
	keys          []ipfs.KeyInfo
	importKeyName string

	// App logs.
	logCatIdx int
	logSearch string
	logs      []applog.Entry
}

var logCategories = []string{"", "Kinpage", "IPFS", "API"}

func (a *App) openToolsModal(tab int) {
	a.toolsModal.open = true
	a.toolsModal.tab = tab
	a.toolsModal.search = ""
	a.toolsModal.returnSearch = ""
	a.toolsModal.highlightID = ""
	a.toolsModal.err = ""
	a.toolsModal.msg = ""
	a.toolsModal.aiModalOpen = false
	a.toolsModal.apiModalOpen = false
	a.toolsModal.keyModalOpen = false
	a.toolsModal.folderModalOpen = false
	a.refreshToolsTab(tab)
}

func (a *App) refreshToolsTab(tab int) {
	m := &a.toolsModal
	if a.backend == nil {
		return
	}
	switch tab {
	case toolsTabAI:
		if cfg, err := a.backend.AIConfig(); err == nil {
			m.aiSavedBaseURL = cfg.BaseURL
			m.aiSavedModel = cfg.Model
			m.aiBaseURL = cfg.BaseURL
			m.aiModel = cfg.Model
		}
	case toolsTabAPI:
		if st, err := a.backend.LocalAPIStatus(); err == nil {
			m.apiStatus = st
			m.apiEnabled = st.Enabled
			m.apiPort = strconv.Itoa(st.Port)
			m.apiAuthOn = st.AuthEnabled
			m.apiUsername = st.Username
		}
	case toolsTabKeys:
		if keys, err := a.backend.ListIPNSKeys(); err != nil {
			m.err = err.Error()
		} else {
			m.keys = keys
		}
	case toolsTabLogs:
		cat := logCategories[m.logCatIdx]
		m.logs = a.backend.AppLogs(cat)
	case toolsTabFolders:
		if list, err := a.backend.PublishedFolders(); err != nil {
			m.err = err.Error()
		} else {
			a.foldersModal.folders = list
		}
	case toolsTabOffline:
		if usage, err := a.backend.OfflineMediaUsage(); err != nil {
			m.err = err.Error()
		} else {
			a.offlineModal.usage = usage
		}
	case toolsTabTrash:
		if list, err := a.backend.RemovedSites(); err != nil {
			m.err = err.Error()
		} else {
			a.removedSites.sites = list
		}
	}
}

func (a *App) viewToolsModal(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	macPreferencesWindow(c, &m.open, 960, 680, pal, func() {
		ui.Row(c).Fill().Background(pal.prefsCanvas).Children(func() {
			// Left 232px grouped sidebar navigation with search bar.
			ui.Column(c).Width(232).Shrink(0).AlignSelf(ui.Stretch).Background(pal.prefsSidebar).Padding(16, 10, 14, 10).Gap(10).Children(func() {
				ui.Text(c, a.t("settings.globalTitle")).Font(macPlanetFont).FontSize(15).FontWeight(600).TextColor(pal.prefsInk).Padding(0, 8, 2, 8)
				prevSearch := m.search
				field.SearchBar(c.Key("settings-nav-search"), &m.search, field.SearchBarProps{
					Placeholder: a.t("settings.searchPlaceholder"),
					Variant:     field.InputFilled,
					Size:        theme.Small,
				})
				if m.search != prevSearch && strings.TrimSpace(m.search) != "" {
					m.returnSearch = ""
					m.highlightID = ""
				}
				if m.returnSearch != "" && strings.TrimSpace(m.search) == "" {
					if macSettingsReturnButton(c, "tools-return-search", a.t("settings.search.return"), pal) {
						m.search = m.returnSearch
						m.returnSearch = ""
						m.highlightID = ""
					}
				}

				type navSpec struct {
					idx   int
					label string
					icon  *ui.SVG
				}
				type navGroupSpec struct {
					heading string
					items   []navSpec
				}
				groups := []navGroupSpec{
					{
						heading: a.t("settings.groupGeneral"),
						items: []navSpec{
							{toolsTabGeneral, a.t("settings.tabGeneral"), lucide.SlidersHorizontal()},
						},
					},
					{
						heading: a.t("settings.groupData"),
						items: []navSpec{
							{toolsTabDataRepo, a.t("settings.tabDataRepo"), lucide.Database()},
							{toolsTabFolders, a.t("menu.publishedFolders"), lucide.FolderOpen()},
							{toolsTabOffline, a.t("menu.offlineStorage"), lucide.HardDrive()},
							{toolsTabTrash, a.t("menu.trash"), lucide.Trash2()},
						},
					},
					{
						heading: a.t("settings.groupTools"),
						items: []navSpec{
							{toolsTabAPI, a.t("menu.localAPI"), lucide.Terminal()},
							{toolsTabKeys, a.t("menu.keyManager"), lucide.KeyRound()},
							{toolsTabLogs, a.t("menu.appLogs"), lucide.FileText()},
						},
					},
					{
						heading: a.t("settings.groupExtensions"),
						items: []navSpec{
							{toolsTabAI, a.t("nav.aiAssistant"), lucide.Sparkles()},
							{toolsTabTip, a.t("settings.tabTip"), lucide.Heart()},
						},
					},
				}

				searching := strings.TrimSpace(m.search) != ""
				var searchMatches []settingsSearchItem
				matchedTabs := map[int]bool{}
				if searching {
					searchMatches = a.matchSettingsSearch(m.search, globalSettingsSearchItems())
					for _, sm := range searchMatches {
						matchedTabs[sm.tab] = true
					}
				}

				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
						for _, grp := range groups {
							var visible []navSpec
							for _, tab := range grp.items {
								if !searching || matchedTabs[tab.idx] {
									visible = append(visible, tab)
								}
							}
							if len(visible) == 0 {
								continue
							}
							ui.Column(c).Gap(2).AlignSelf(ui.Stretch).Children(func() {
								ui.Text(c, grp.heading).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.prefsMuted).Padding(0, 10, 4, 10)
								for _, tab := range visible {
									tIdx, tLbl := tab.idx, tab.label
									selected := !searching && m.tab == tIdx
									if prefsNavRow(c, fmt.Sprintf("tools-nav-%d", tIdx), tLbl, selected, tab.icon, pal.prefsInk, pal) {
										m.search = ""
										m.returnSearch = ""
										m.highlightID = ""
										if m.tab != tIdx {
											m.tab = tIdx
											m.err = ""
											m.msg = ""
											a.refreshToolsTab(m.tab)
										}
									}
								}
							})
						}
					})
				})
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)

			// Right form pane + header + footer.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.prefsCanvas).Children(func() {
				tabTitles := map[int]string{
					toolsTabGeneral:  a.t("settings.tabGeneral"),
					toolsTabDataRepo: a.t("settings.tabDataRepo"),
					toolsTabFolders:  a.t("menu.publishedFolders"),
					toolsTabOffline:  a.t("menu.offlineStorage"),
					toolsTabTrash:    a.t("menu.trash"),
					toolsTabAI:       a.t("nav.aiAssistant"),
					toolsTabAPI:      a.t("menu.localAPI"),
					toolsTabKeys:     a.t("menu.keyManager"),
					toolsTabLogs:     a.t("menu.appLogs"),
					toolsTabTip:      a.t("settings.tabTip"),
				}
				searching := strings.TrimSpace(m.search) != ""
				headerTitle := tabTitles[m.tab]
				if searching {
					headerTitle = a.t("settings.search.heading")
				}
				ui.Row(c).Padding(20, 20, 12, 28).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, headerTitle).Font(macPlanetFont).FontSize(18).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.prefsInk)
					if toolbarIconBtn(c, "tools-close-x", lucide.X(), a.t("common.close")).Clicked() {
						m.open = false
					}
				})
				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(4, 28, 24, 28).Gap(14).AlignSelf(ui.Stretch).Children(func() {
						if searching {
							matches := a.matchSettingsSearch(m.search, globalSettingsSearchItems())
							suggestions := []string{
								a.t("settings.search.sugSleep"),
								a.t("settings.search.sugPort"),
								a.t("settings.search.sugDataRepo"),
								a.t("nav.aiAssistant"),
							}
							a.viewSettingsSearchResults(c, "global", m.search, matches, suggestions, func(it settingsSearchItem) {
								m.returnSearch = strings.TrimSpace(m.search)
								m.search = ""
								m.tab = it.tab
								m.highlightID = it.targetID
								m.err = ""
								m.msg = ""
								a.refreshToolsTab(m.tab)
								switch it.opensModal {
								case "api":
									a.openAPIConfigSheet()
								case "ai":
									a.openAIConfigSheet()
								}
							}, func(sug string) {
								m.search = sug
							})
							return
						}
						switch m.tab {
						case toolsTabGeneral:
							a.viewToolsGeneral(c)
						case toolsTabDataRepo:
							a.viewToolsDataRepo(c)
						case toolsTabFolders:
							a.viewToolsFolders(c)
						case toolsTabOffline:
							a.viewToolsOffline(c)
						case toolsTabTrash:
							a.viewToolsTrash(c)
						case toolsTabAI:
							a.viewToolsAI(c)
						case toolsTabAPI:
							a.viewToolsLocalAPI(c)
						case toolsTabKeys:
							a.viewToolsKeys(c)
						case toolsTabLogs:
							a.viewToolsLogs(c)
						case toolsTabTip:
							a.viewToolsTip(c)
						}
						if m.err != "" {
							ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
						} else if m.msg != "" {
							ui.Text(c, m.msg).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
						}
					})
				})
				prefsHairlineH(c, pal)
				ui.Row(c).Padding(12, 28, 12, 28).Gap(8).Justify(ui.End).Background(pal.prefsFooter).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					if macSecondaryButton(c, "tools-close", a.t("common.close"), pal) {
						m.open = false
					}
				})
			})
		})
	})

	// Secondary configuration modals rendered above the Settings window so
	// canceling or pressing Esc only closes the secondary sheet.
	a.viewToolsAIConfigSheet(c)
	a.viewToolsAPIConfigSheet(c)
	a.viewToolsKeyImportSheet(c)
	a.viewToolsFolderAddSheet(c)
}

func (a *App) viewToolsGeneral(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal

	// Language switcher card (immediate effect).
	macHighlightCard(c, pal, m.highlightID == "global-language", func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, a.t("settings.languageTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
				ui.Text(c, a.t("settings.languageDesc")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			})
			locales := SupportedLocales()
			labels := make([]string, len(locales))
			curIdx := 0
			for i, loc := range locales {
				labels[i] = LocaleDisplayName(loc)
				if a.Locale() == loc {
					curIdx = i
				}
			}
			prevIdx := curIdx
			macSegmented(c, "settings-locale-seg", &curIdx, labels, pal)
			if curIdx != prevIdx && curIdx >= 0 && curIdx < len(locales) {
				chosen := locales[curIdx]
				a.SetLocale(chosen)
				a.setStatus(a.tf("status.languageSwitched", LocaleDisplayName(chosen)))
			}
		})
	})

	if a.applyAppIcon != nil {
		a.viewAppIcons(c)
	}

	// System behavior card (immediate effect).
	macHighlightCard(c, pal, m.highlightID == "global-sleep", func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, a.t("settings.systemTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
				ui.Text(c, a.t("settings.search.sleepSummary")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			})
			preventSleep := a.backend != nil && a.backend.PreventSleep()
			prev := preventSleep
			ui.Checkbox(c.Key("settings-prevent-sleep"), &preventSleep, a.t("menu.preventSleep"))
			if preventSleep != prev && a.backend != nil {
				if err := a.backend.SetPreventSleep(preventSleep); err != nil {
					m.err = a.tf("status.preventSleepErr", err.Error())
				} else {
					m.err = ""
					if preventSleep {
						a.setStatus(a.t("status.preventSleepOn"))
					} else {
						a.setStatus(a.t("status.preventSleepOff"))
					}
				}
			}
		})
		if a.DesktopPetAvailable() {
			prefsHairlineH(c, pal)
			active := a.DesktopPetEnabled()
			enabled := active
			ui.Checkbox(c.Key("settings-desktop-pet"), &enabled, a.DesktopPetLabel())
			if enabled != active {
				a.SetDesktopPetEnabled(enabled)
			}
		}
	})
}

func (a *App) viewToolsDataRepo(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	dataDir := ""
	if a.backend != nil {
		dataDir = a.backend.DataDir()
	}

	macHighlightCard(c, pal, m.highlightID == "global-data-repo", func() {
		ui.Column(c).Gap(4).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.t("settings.dataDirTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
			ui.Text(c, a.t("settings.dataDirDesc")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			if dataDir != "" {
				ui.Box(c).Padding(6, 10, 6, 10).Radius(6).Background(pal.hover).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, dataDir).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsSecondary).MaxLines(2).Ellipsis("…")
				})
			}
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if macSecondaryButton(c, "settings-move-data", a.t("menu.moveDataDir"), pal) {
				if a.backend != nil {
					if dst, err := a.backend.MoveDataDir(context.Background()); err != nil {
						m.err = a.tf("status.moveDataDirErr", err.Error())
					} else if dst != "" {
						m.err = ""
						m.msg = a.tf("status.moveDataDirDone", dst)
					}
				}
			}
			if macSecondaryButton(c, "settings-refresh-data", a.t("menu.refreshData"), pal) {
				a.Reload()
				m.err = ""
				m.msg = a.t("status.dataRefreshed")
			}
		})
	})
}

func (a *App) isIPFSOnline() bool {
	if a.backend == nil {
		return false
	}
	st, err := a.backend.IPFSStatus()
	return err == nil && st.Online
}

func (a *App) viewToolsFolders(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	folders := a.foldersModal.folders
	ipfsOnline := a.isIPFSOnline()

	if !ipfsOnline {
		settingsPrerequisiteNote(c, pal, a.t("settings.prereq.foldersPublishIPFS"))
	}

	if len(folders) == 0 {
		if settingsEmptyState(c, "folders-empty-add", lucide.FolderOpen(), a.t("settings.folders.emptyDesc"), a.t("common.add"), pal) {
			a.foldersModal.newName = ""
			a.foldersModal.autoPublish = false
			a.foldersModal.err = ""
			m.folderModalOpen = true
		}
		return
	}

	ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
		if macPrimaryButtonSmall(c, "folders-add-btn", a.t("dialog.folders.addBtn"), pal) {
			a.foldersModal.newName = ""
			a.foldersModal.autoPublish = false
			a.foldersModal.err = ""
			m.folderModalOpen = true
		}
	})
	macInsetCard(c, pal, func() {
		for i, f := range folders {
			fid := f.ID
			if i > 0 {
				prefsHairlineH(c, pal)
			}
			ui.Row(c.Key("settings-folder-row-" + fid)).PaddingY(4).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, a.fallbackTitle(f.Name)).Font(macPlanetFont).FontWeight(600).FontSize(13).TextColor(pal.prefsInk)
					ui.Text(c, f.Path).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).MaxLines(1).Ellipsis("…")
				})
				if macSecondaryButton(c, "settings-folder-pub-"+fid, a.t("common.publish"), pal) {
					if _, err := a.backend.PublishFolder(fid); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						a.refreshToolsTab(toolsTabFolders)
					}
				}
				if macSecondaryButton(c, "settings-folder-rm-"+fid, a.t("common.remove"), pal) {
					if err := a.backend.RemovePublishedFolder(fid); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						a.refreshToolsTab(toolsTabFolders)
					}
				}
			})
		}
	})
}

func (a *App) viewToolsFolderAddSheet(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	fm := &a.foldersModal
	if macSheet(c, &m.folderModalOpen, macSheetProps{
		Title:        a.t("settings.folders.modalTitle"),
		Width:        440,
		PrimaryText:  a.t("common.add"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("dialog.folders.nameLabel"), "", pal, func() {
				ui.TextInput(c.Key("folder-sheet-name"), &fm.newName).Placeholder(a.t("dialog.folders.namePlaceholder")).Label(a.t("dialog.folders.nameLabel")).AutoFocus().AlignSelf(ui.Stretch)
			})
			ui.Checkbox(c.Key("folder-sheet-auto"), &fm.autoPublish, a.t("dialog.folders.autoPublish"))
		})
		if fm.err != "" {
			ui.Text(c, fm.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if _, err := a.backend.AddPublishedFolder(context.Background(), strings.TrimSpace(fm.newName), fm.autoPublish); err != nil {
			fm.err = err.Error()
			return
		}
		fm.newName = ""
		fm.err = ""
		m.folderModalOpen = false
		a.refreshToolsTab(toolsTabFolders)
	}
}

func (a *App) viewToolsOffline(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	usage := a.offlineModal.usage
	macInsetCard(c, pal, func() {
		ui.Text(c, a.tf("dialog.offline.managedFiles", usage.ManagedFiles, formatBytes(usage.TotalBytes))).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.prefsInk)
		ui.Text(c, a.tf("dialog.offline.reclaimableSpace", usage.ReclaimableFiles, formatBytes(usage.ReclaimableBytes))).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
		ui.Row(c).Gap(8).Children(func() {
			if macSecondaryButton(c, "settings-offline-clean", a.t("dialog.offline.cleanBtn"), pal) {
				next, err := a.backend.CleanupOfflineMedia()
				if err != nil {
					m.err = err.Error()
				} else {
					a.offlineModal.usage = next
					m.err = ""
					m.msg = a.tf("dialog.offline.reclaimedMsg", formatBytes(next.ReclaimedBytes))
				}
			}
			if macSecondaryButton(c, "settings-offline-refresh", a.t("common.refresh"), pal) {
				a.refreshToolsTab(toolsTabOffline)
			}
		})
	})
}

func (a *App) viewToolsTrash(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	sites := a.removedSites.sites
	if len(sites) == 0 {
		settingsEmptyState(c, "trash-empty-state", lucide.Trash2(), a.t("dialog.removedSites.empty"), "", pal)
		return
	}
	macInsetCard(c, pal, func() {
		for i, s := range sites {
			sid := s.ID
			if i > 0 {
				prefsHairlineH(c, pal)
			}
			ui.Row(c.Key("settings-rm-site-" + sid)).PaddingY(4).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, a.fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.prefsInk).Grow(1)
				if macPrimaryButtonSmall(c, "settings-restore-site-"+sid, a.t("common.restore"), pal) {
					if _, err := a.backend.RestoreSite(sid); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						a.Reload()
						a.refreshToolsTab(toolsTabTrash)
						a.setStatus(a.t("status.siteRestored"))
					}
				}
				if macSecondaryButton(c, "settings-purge-site-"+sid, a.t("dialog.removedSites.purgeBtn"), pal) {
					if err := a.backend.PermanentlyDeleteSite(sid); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						a.refreshToolsTab(toolsTabTrash)
					}
				}
			})
		}
	})
}

func (a *App) viewToolsTip(c *ui.Context) {
	pal := a.pal
	if settingsEmptyState(c, "settings-tip-open", lucide.Heart(), a.t("settings.tip.desc"), a.t("settings.tip.openBtn"), pal) {
		a.openTipModal("")
	}
}

func (a *App) openAIConfigSheet() {
	m := &a.toolsModal
	m.aiBaseURL = m.aiSavedBaseURL
	m.aiModel = m.aiSavedModel
	m.aiToken = ""
	m.err = ""
	m.aiModalOpen = true
}

func (a *App) probeAIService(baseURL string) {
	m := &a.toolsModal
	if res, err := a.backend.ProbeAI(strings.TrimSpace(baseURL)); err != nil {
		m.aiProbeErr = err.Error()
		m.aiProbeMsg = ""
		m.err = err.Error()
	} else {
		m.aiProbe = res
		m.aiProbeErr = ""
		m.err = ""
		if len(res.Models) == 0 {
			m.aiProbeMsg = a.t("tools.ai.probeZero")
		} else {
			m.aiProbeMsg = a.tf("tools.ai.probeDone", len(res.Models))
		}
		m.msg = m.aiProbeMsg
		if m.aiModel == "" && len(res.Models) > 0 {
			m.aiModel = res.Models[0]
		}
	}
}

func (a *App) viewToolsAI(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	configured := strings.TrimSpace(m.aiSavedBaseURL) != "" || strings.TrimSpace(m.aiSavedModel) != ""

	if !configured {
		if settingsEmptyState(c, "ai-empty-add", lucide.Sparkles(), a.t("settings.ai.emptyDesc"), a.t("tools.ai.configureBtn"), pal) {
			a.openAIConfigSheet()
		}
		return
	}

	// Configured AI summary card (saved config separated from live connection test).
	macHighlightCard(c, pal, m.highlightID == "global-ai-service", func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
					title := m.aiSavedModel
					if title == "" {
						title = a.t("nav.aiAssistant")
					}
					ui.Text(c, title).Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.prefsInk)
					ui.Box(c).Padding(1, 6, 1, 6).Radius(4).Background(pal.hover).Children(func() {
						ui.Text(c, a.t("tools.ai.savedTag")).Font(macPlanetFont).FontSize(10.5).TextColor(pal.prefsSecondary).NoWrap()
					})
				})
				if m.aiSavedBaseURL != "" {
					ui.Text(c, m.aiSavedBaseURL).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).MaxLines(1).Ellipsis("…")
				}
				if m.aiProbeErr != "" {
					ui.Text(c, m.aiProbeErr).Font(macPlanetFont).FontSize(11.5).TextColor(pal.danger)
				} else if m.aiProbeMsg != "" {
					ui.Text(c, m.aiProbeMsg).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsSecondary)
				}
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				if macSecondaryButton(c, "ai-test-conn", a.t("tools.ai.testBtn"), pal) {
					a.probeAIService(m.aiSavedBaseURL)
				}
				if len(m.aiHistory) > 0 {
					if macSecondaryButton(c, "ai-clear", a.t("tools.ai.clearChat"), pal) {
						m.aiHistory = nil
					}
				}
				if macSecondaryButton(c, "ai-edit-btn", a.t("common.edit"), pal) {
					a.openAIConfigSheet()
				}
			})
		})
	})

	// Conversation card clearly separated from configuration above.
	macInsetCard(c, pal, func() {
		ui.Text(c, a.t("tools.ai.chatTitle")).Font(macPlanetFont).FontSize(12.5).FontWeight(600).TextColor(pal.prefsInk)
		for i, msg := range m.aiHistory {
			roleLabel := a.t("tools.ai.roleUser")
			bg := pal.prefsCanvas
			if msg.Role == "assistant" {
				roleLabel = a.t("nav.aiAssistant")
				bg = pal.prefsCard
			}
			ui.Column(c.Key(fmt.Sprintf("ai-msg-%d", i))).Padding(12).Gap(4).Background(bg).Border(0.5, pal.prefsCardBorder).Radius(8).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, roleLabel).Font(macPlanetFont).FontWeight(600).FontSize(11).TextColor(pal.prefsMuted)
				if msg.Role == "assistant" {
					markdown.View(c, msg.Content, &a.markdownOpts)
				} else {
					ui.Text(c, msg.Content).Font(macPlanetFont).FontSize(13).TextColor(pal.prefsInk)
				}
			})
		}
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			if ui.TextInput(c.Key("ai-prompt"), &m.aiPrompt).Placeholder(a.t("tools.ai.promptPlaceholder")).Grow(1).Submitted() {
				a.sendAIPrompt()
			}
			if macSecondaryButton(c, "ai-send", a.t("tools.ai.sendBtn"), pal) {
				a.sendAIPrompt()
			}
		})
	})
}

func (a *App) viewToolsAIConfigSheet(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	if macSheet(c, &m.aiModalOpen, macSheetProps{
		Title:        a.t("settings.ai.modalTitle"),
		Width:        500,
		PrimaryText:  a.t("tools.ai.saveBtn"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "Base URL", "", pal, func() {
						ui.TextInput(c.Key("ai-url"), &m.aiBaseURL).Placeholder("http://127.0.0.1:11434/v1").AlignSelf(ui.Stretch)
					})
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, a.t("tools.ai.modelField"), "", pal, func() {
						ui.TextInput(c.Key("ai-model"), &m.aiModel).Placeholder("llama3 / qwen2.5").AlignSelf(ui.Stretch)
					})
				})
			})
			macField(c, "API Key", "", pal, func() {
				ui.TextInput(c.Key("ai-tok"), &m.aiToken).Password().Placeholder(a.t("tools.ai.apiKeyPlaceholder")).AlignSelf(ui.Stretch)
			})
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if macSecondaryButton(c, "ai-probe", a.t("tools.ai.probeBtn"), pal) {
					a.probeAIService(m.aiBaseURL)
				}
			})
			if len(m.aiProbe.Models) > 0 {
				ui.Row(c).Gap(6).Wrap().Children(func() {
					for i, mdl := range m.aiProbe.Models {
						if i >= 6 {
							break
						}
						name := mdl
						if macSecondaryButton(c, "ai-mdl-"+name, name, pal) {
							m.aiModel = name
						}
					}
				})
			}
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		} else if m.aiProbeMsg != "" {
			ui.Text(c, m.aiProbeMsg).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsSecondary)
		}
	}) {
		cfg, err := a.backend.ConfigureAI(strings.TrimSpace(m.aiBaseURL), strings.TrimSpace(m.aiModel), strings.TrimSpace(m.aiToken))
		if err != nil {
			m.err = err.Error()
			return
		}
		m.aiSavedBaseURL = cfg.BaseURL
		m.aiSavedModel = cfg.Model
		m.aiBaseURL = cfg.BaseURL
		m.aiModel = cfg.Model
		m.aiToken = ""
		m.aiProbeErr = ""
		m.err = ""
		m.msg = a.t("tools.ai.savedMsg")
		m.aiModalOpen = false
	}
}

func (a *App) sendAIPrompt() {
	m := &a.toolsModal
	prompt := strings.TrimSpace(m.aiPrompt)
	if prompt == "" {
		return
	}
	ctxTitle, ctxBody := "", ""
	if a.editing {
		ctxTitle, ctxBody = a.editorTitle, a.editorBuffer.String()
	} else if art := a.currentArticle(); art != nil {
		ctxTitle, ctxBody = art.Title, art.Content
	}
	histJSON, _ := json.Marshal(m.aiHistory)
	reply, err := a.backend.AskAI(prompt, ctxTitle, ctxBody, string(histJSON))
	if err != nil {
		m.err = err.Error()
		return
	}
	m.err = ""
	m.aiPrompt = ""
	m.aiHistory = append(m.aiHistory,
		services.AIChatMessage{Role: "user", Content: prompt},
		services.AIChatMessage{Role: "assistant", Content: reply},
	)
}

func (a *App) openAPIConfigSheet() {
	m := &a.toolsModal
	m.apiEnabled = m.apiStatus.Enabled
	if !m.apiStatus.Enabled && !m.apiStatus.Running {
		m.apiEnabled = true
	}
	if m.apiStatus.Port > 0 {
		m.apiPort = strconv.Itoa(m.apiStatus.Port)
	} else {
		m.apiPort = "8086"
	}
	m.apiAuthOn = m.apiStatus.AuthEnabled
	m.apiUsername = m.apiStatus.Username
	m.apiPasscode = ""
	m.err = ""
	m.apiModalOpen = true
}

func (a *App) viewToolsLocalAPI(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	configured := m.apiStatus.Enabled || m.apiStatus.Running

	if !configured {
		if settingsEmptyState(c, "api-empty-add", lucide.Terminal(), a.t("settings.api.emptyDesc"), a.t("common.configure"), pal) {
			a.openAPIConfigSheet()
		}
		return
	}

	state := a.t("tools.api.stateStopped")
	dotCol := pal.muted
	if m.apiStatus.Running {
		state = a.tf("tools.api.stateRunning", m.apiStatus.Port)
		dotCol = pal.green
	}
	macHighlightCard(c, pal, m.highlightID == "global-api-port", func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Box(c).Size(8, 8).Radius(4).Background(dotCol)
				ui.Text(c, a.tf("tools.api.statusPrefix", state)).Font(macPlanetFont).FontWeight(600).FontSize(13).TextColor(pal.prefsInk)
			})
			ui.Row(c).Gap(6).Children(func() {
				if macSecondaryButton(c, "api-edit-btn", a.t("common.edit"), pal) {
					a.openAPIConfigSheet()
				}
				if macSecondaryButton(c, "api-clear-logs", a.t("tools.api.clearLogs"), pal) {
					a.backend.ClearLocalAPILogs()
					a.refreshToolsTab(toolsTabAPI)
				}
			})
		})
	})
	if len(m.apiStatus.Logs) > 0 {
		macInsetCard(c, pal, func() {
			for i, entry := range m.apiStatus.Logs {
				ui.Text(c.Key(fmt.Sprintf("api-log-%d", i)), fmt.Sprintf("%s %s %s -> %d (%dms)", entry.Time, entry.Method, entry.Path, entry.Status, entry.DurationMs)).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
			}
		})
	}
}

func (a *App) viewToolsAPIConfigSheet(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	primaryText := a.t("tools.api.saveBtn")
	if m.apiEnabled {
		primaryText = a.t("tools.api.saveAndStartBtn")
	}
	if macSheet(c, &m.apiModalOpen, macSheetProps{
		Title:        a.t("settings.api.modalTitle"),
		Width:        480,
		PrimaryText:  primaryText,
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Row(c).Gap(16).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Checkbox(c.Key("api-en"), &m.apiEnabled, a.t("tools.api.enableCheck"))
				ui.Box(c).Width(140).Children(func() {
					ui.TextInput(c.Key("api-port"), &m.apiPort).Placeholder(a.t("tools.api.portPlaceholder")).AutoFocus().AlignSelf(ui.Stretch)
				})
			})
			ui.Checkbox(c.Key("api-auth"), &m.apiAuthOn, a.t("tools.api.authCheck"))
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, a.t("tools.api.userField"), "", pal, func() { ui.TextInput(c.Key("api-user"), &m.apiUsername).AlignSelf(ui.Stretch) })
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, a.t("tools.api.passField"), "", pal, func() { ui.TextInput(c.Key("api-pass"), &m.apiPasscode).Password().AlignSelf(ui.Stretch) })
				})
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		port, err := strconv.Atoi(strings.TrimSpace(m.apiPort))
		if err != nil {
			m.err = a.t("tools.api.portIntErr")
			return
		}
		st, err := a.backend.ConfigureLocalAPI(m.apiEnabled, port, m.apiAuthOn, strings.TrimSpace(m.apiUsername), m.apiPasscode)
		if err != nil {
			if st.Port > 0 || st.Enabled {
				m.apiStatus = st
			}
			m.err = err.Error()
			return
		}
		m.apiStatus = st
		m.err = ""
		m.msg = a.t("tools.api.savedMsg")
		m.apiModalOpen = false
	}
}

func (a *App) viewToolsKeys(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	if len(m.keys) == 0 {
		if settingsEmptyState(c, "keys-empty-import", lucide.KeyRound(), a.t("tools.keys.empty"), a.t("tools.keys.importBtn"), pal) {
			m.importKeyName = ""
			m.err = ""
			m.keyModalOpen = true
		}
		return
	}

	ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
		if macPrimaryButtonSmall(c, "key-imp-open", a.t("tools.keys.importBtn"), pal) {
			m.importKeyName = ""
			m.err = ""
			m.keyModalOpen = true
		}
	})
	macInsetCard(c, pal, func() {
		for i, k := range m.keys {
			name := k.Name
			if i > 0 {
				prefsHairlineH(c, pal)
			}
			ui.Row(c.Key("key-row-" + name)).PaddingY(4).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, name).Font(macPlanetFont).FontWeight(600).FontSize(12.5).TextColor(pal.ink)
					ui.Text(c, k.ID).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
				})
				if macSecondaryButton(c, "key-exp-"+name, a.t("common.export"), pal) {
					if path, err := a.backend.ExportIPNSKey(context.Background(), name); err != nil {
						m.err = err.Error()
					} else if path != "" {
						m.err = ""
						m.msg = a.tf("tools.keys.exportedMsg", path)
					}
				}
			})
		}
	})
}

func (a *App) viewToolsKeyImportSheet(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	if macSheet(c, &m.keyModalOpen, macSheetProps{
		Title:        a.t("settings.keys.modalTitle"),
		Width:        440,
		PrimaryText:  a.t("tools.keys.importBtn"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("tools.keys.importPlaceholder"), "", pal, func() {
				ui.TextInput(c.Key("key-imp-name"), &m.importKeyName).Placeholder(a.t("tools.keys.importPlaceholder")).AutoFocus().AlignSelf(ui.Stretch)
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		name := strings.TrimSpace(m.importKeyName)
		if name == "" {
			m.err = a.t("tools.keys.nameRequired")
			return
		}
		id, err := a.backend.ImportIPNSKey(context.Background(), name)
		if err != nil {
			m.err = err.Error()
			return
		}
		m.importKeyName = ""
		m.err = ""
		m.msg = a.tf("tools.keys.importedMsg", id)
		m.keyModalOpen = false
		a.refreshToolsTab(toolsTabKeys)
	}
}

func (a *App) viewToolsLogs(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Wrap().AlignSelf(ui.Stretch).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			prev := m.logCatIdx
			macSegmented(c, "log-cat", &m.logCatIdx, []string{a.t("common.all"), "Kinpage", "IPFS", "API"}, pal)
			if m.logCatIdx != prev {
				a.refreshToolsTab(toolsTabLogs)
			}
			ui.Box(c).Width(180).Children(func() {
				field.SearchBar(c.Key("log-filter-search"), &m.logSearch, field.SearchBarProps{
					Placeholder: a.t("tools.logs.searchPlaceholder"),
					Variant:     field.InputFilled,
					Size:        theme.Small,
				})
			})
		})
		ui.Row(c).Gap(6).Children(func() {
			if macSecondaryButton(c, "log-refresh", a.t("common.refresh"), pal) {
				a.refreshToolsTab(toolsTabLogs)
			}
			if macSecondaryButton(c, "log-clear", a.t("tools.logs.clearBtn"), pal) {
				a.backend.ClearAppLogs(logCategories[m.logCatIdx])
				a.refreshToolsTab(toolsTabLogs)
			}
		})
	})

	q := strings.ToLower(strings.TrimSpace(m.logSearch))
	var filtered []applog.Entry
	for _, entry := range m.logs {
		if q == "" ||
			strings.Contains(strings.ToLower(entry.Message), q) ||
			strings.Contains(strings.ToLower(entry.Category), q) ||
			strings.Contains(strings.ToLower(entry.Time), q) {
			filtered = append(filtered, entry)
		}
	}

	macInsetCard(c, pal, func() {
		if len(m.logs) == 0 {
			ui.Text(c, a.t("tools.logs.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
			return
		}
		if len(filtered) == 0 {
			ui.Text(c, a.t("tools.logs.noMatch")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
			return
		}
		for i, entry := range filtered {
			ui.Text(c.Key(fmt.Sprintf("app-log-%d", i)), fmt.Sprintf("[%s] [%s] %s", entry.Time, entry.Category, entry.Message)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.secondary)
		}
	})
}
