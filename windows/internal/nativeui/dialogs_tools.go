package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/applog"
	"windows/internal/ipfs"
	"windows/internal/localapi"
	"windows/internal/services"
)

const (
	toolsTabAI   = 0
	toolsTabAPI  = 1
	toolsTabKeys = 2
	toolsTabLogs = 3
)

type toolsModalState struct {
	open bool
	tab  int
	err  string
	msg  string

	// Local AI chat.
	aiBaseURL string
	aiModel   string
	aiToken   string
	aiProbe   services.AIProbeResult
	aiPrompt  string
	aiHistory []services.AIChatMessage

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
	logs      []applog.Entry
}

var logCategories = []string{"", "Planet", "IPFS", "API"}

func (a *App) openToolsModal(tab int) {
	a.toolsModal.open = true
	a.toolsModal.tab = tab
	a.toolsModal.err = ""
	a.toolsModal.msg = ""
	a.refreshToolsTab(tab)
}

func (a *App) refreshToolsTab(tab int) {
	m := &a.toolsModal
	switch tab {
	case toolsTabAI:
		if cfg, err := a.backend.AIConfig(); err == nil {
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
	}
}

func (a *App) viewToolsModal(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	macPreferencesWindow(c, &m.open, 920, 660, pal, func() {
		ui.Row(c).Fill().Background(pal.prefsCanvas).Children(func() {
			// Left 216px macOS Sequoia sidebar navigation with tinted icons.
			ui.Column(c).Width(216).Shrink(0).AlignSelf(ui.Stretch).Background(pal.prefsSidebar).Padding(18, 10, 14, 10).Gap(3).Children(func() {
				ui.Text(c, a.t("tools.sidebarTitle")).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.prefsMuted).Padding(0, 10, 6, 10)
				tabs := []struct {
					idx   int
					label string
					icon  *ui.SVG
					tint  ui.Color
				}{
					{toolsTabAI, a.t("nav.aiAssistant"), lucide.Sparkles(), ui.Hex("#7c3aed")},
					{toolsTabAPI, a.t("menu.localAPI"), lucide.Terminal(), ui.Hex("#0891b2")},
					{toolsTabKeys, a.t("menu.keyManager"), lucide.KeyRound(), ui.Hex("#d97706")},
					{toolsTabLogs, a.t("menu.appLogs"), lucide.FileText(), ui.Hex("#64748b")},
				}
				for _, tab := range tabs {
					tIdx, tLbl := tab.idx, tab.label
					if prefsNavRow(c, fmt.Sprintf("tools-nav-%d", tIdx), tLbl, m.tab == tIdx, tab.icon, tab.tint, pal) {
						if m.tab != tIdx {
							m.tab = tIdx
							m.err = ""
							m.msg = ""
							a.refreshToolsTab(m.tab)
						}
					}
				}
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)

			// Right form pane + header + footer.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.prefsCanvas).Children(func() {
				tabTitles := map[int]string{
					toolsTabAI:   a.t("nav.aiAssistant"),
					toolsTabAPI:  a.t("menu.localAPI"),
					toolsTabKeys: a.t("menu.keyManager"),
					toolsTabLogs: a.t("menu.appLogs"),
				}
				ui.Row(c).Padding(20, 20, 12, 28).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, tabTitles[m.tab]).Font(macPlanetFont).FontSize(18).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.prefsInk)
					if toolbarIconBtn(c, "tools-close-x", lucide.X(), a.t("common.close")).Clicked() {
						m.open = false
					}
				})
				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(4, 28, 24, 28).Gap(14).AlignSelf(ui.Stretch).Children(func() {
						switch m.tab {
						case toolsTabAI:
							a.viewToolsAI(c)
						case toolsTabAPI:
							a.viewToolsLocalAPI(c)
						case toolsTabKeys:
							a.viewToolsKeys(c)
						case toolsTabLogs:
							a.viewToolsLogs(c)
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
}

func (a *App) viewToolsAI(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
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
		ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if macSecondaryButton(c, "ai-probe", a.t("tools.ai.probeBtn"), pal) {
					if res, err := a.backend.ProbeAI(strings.TrimSpace(m.aiBaseURL)); err != nil {
						m.err = err.Error()
					} else {
						m.aiProbe = res
						m.err = ""
						m.msg = a.tf("tools.ai.probeDone", len(res.Models))
						if m.aiModel == "" && len(res.Models) > 0 {
							m.aiModel = res.Models[0]
						}
					}
				}
				if len(m.aiHistory) > 0 {
					if macSecondaryButton(c, "ai-clear", a.t("tools.ai.clearChat"), pal) {
						m.aiHistory = nil
					}
				}
			})
			if macPrimaryButtonSmall(c, "ai-save", a.t("tools.ai.saveBtn"), pal) {
				if _, err := a.backend.ConfigureAI(strings.TrimSpace(m.aiBaseURL), strings.TrimSpace(m.aiModel), strings.TrimSpace(m.aiToken)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("tools.ai.savedMsg")
				}
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

func (a *App) viewToolsLocalAPI(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	state := a.t("tools.api.stateStopped")
	dotCol := pal.muted
	if m.apiStatus.Running {
		state = a.tf("tools.api.stateRunning", m.apiStatus.Port)
		dotCol = pal.green
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Box(c).Size(8, 8).Radius(4).Background(dotCol)
			ui.Text(c, a.tf("tools.api.statusPrefix", state)).Font(macPlanetFont).FontWeight(600).FontSize(13).TextColor(pal.ink)
		})
		ui.Row(c).Gap(16).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.Checkbox(c.Key("api-en"), &m.apiEnabled, a.t("tools.api.enableCheck"))
			ui.Box(c).Width(140).Children(func() {
				ui.TextInput(c.Key("api-port"), &m.apiPort).Placeholder(a.t("tools.api.portPlaceholder")).AlignSelf(ui.Stretch)
			})
		})
		ui.Checkbox(c.Key("api-auth"), &m.apiAuthOn, a.t("tools.api.authCheck"))
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("tools.api.userField"), "", pal, func() { ui.TextInput(c.Key("api-user"), &m.apiUsername).AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("tools.api.passField"), "", pal, func() { ui.TextInput(c.Key("api-pass"), &m.apiPasscode).Password().AlignSelf(ui.Stretch) })
			})
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "api-save", a.t("tools.api.saveBtn"), pal) {
			port, err := strconv.Atoi(strings.TrimSpace(m.apiPort))
			if err != nil {
				m.err = a.t("tools.api.portIntErr")
				return
			}
			st, err := a.backend.ConfigureLocalAPI(m.apiEnabled, port, m.apiAuthOn, strings.TrimSpace(m.apiUsername), m.apiPasscode)
			if err != nil {
				m.err = err.Error()
			} else {
				m.apiStatus = st
				m.err = ""
				m.msg = a.t("tools.api.savedMsg")
			}
		}
		if macSecondaryButton(c, "api-clear-logs", a.t("tools.api.clearLogs"), pal) {
			a.backend.ClearLocalAPILogs()
			a.refreshToolsTab(toolsTabAPI)
		}
	})
	if len(m.apiStatus.Logs) > 0 {
		macInsetCard(c, pal, func() {
			for i, entry := range m.apiStatus.Logs {
				ui.Text(c.Key(fmt.Sprintf("api-log-%d", i)), fmt.Sprintf("%s %s %s -> %d (%dms)", entry.Time, entry.Method, entry.Path, entry.Status, entry.DurationMs)).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
			}
		})
	}
}

func (a *App) viewToolsKeys(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.TextInput(c.Key("key-imp-name"), &m.importKeyName).Placeholder(a.t("tools.keys.importPlaceholder")).Grow(1)
			if macPrimaryButtonSmall(c, "key-imp-btn", a.t("tools.keys.importBtn"), pal) {
				name := strings.TrimSpace(m.importKeyName)
				if name == "" {
					m.err = a.t("tools.keys.nameRequired")
					return
				}
				if id, err := a.backend.ImportIPNSKey(context.Background(), name); err != nil {
					m.err = err.Error()
				} else {
					m.importKeyName = ""
					m.err = ""
					m.msg = a.tf("tools.keys.importedMsg", id)
					a.refreshToolsTab(toolsTabKeys)
				}
			}
		})
	})
	if len(m.keys) == 0 {
		ui.Text(c, a.t("tools.keys.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
	} else {
		macInsetCard(c, pal, func() {
			for i, k := range m.keys {
				name := k.Name
				if i > 0 {
					hairlineH(c, pal)
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
}

func (a *App) viewToolsLogs(c *ui.Context) {
	pal := a.pal
	m := &a.toolsModal
	ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
		prev := m.logCatIdx
		macSegmented(c, "log-cat", &m.logCatIdx, []string{a.t("common.all"), "Planet", "IPFS", "API"}, pal)
		if m.logCatIdx != prev {
			a.refreshToolsTab(toolsTabLogs)
		}
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
	macInsetCard(c, pal, func() {
		if len(m.logs) == 0 {
			ui.Text(c, a.t("tools.logs.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		}
		for i, entry := range m.logs {
			ui.Text(c.Key(fmt.Sprintf("app-log-%d", i)), fmt.Sprintf("[%s] [%s] %s", entry.Time, entry.Category, entry.Message)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.secondary)
		}
	})
}
