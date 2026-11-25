package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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
	macSheetFrame(c, &m.open, 760, pal, func() {
		ui.Row(c).Width(760).Height(540).Background(pal.canvas).Children(func() {
			// Left 180px macOS Preferences sidebar navigation.
			ui.Column(c).Width(180).Shrink(0).AlignSelf(ui.Stretch).Background(pal.sidebar).Padding(14).Gap(4).Children(func() {
				ui.Text(c, "系统工具").Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink).PaddingX(8).PaddingY(6)
				tabs := []struct {
					idx   int
					label string
				}{
					{toolsTabAI, "AI 助手"},
					{toolsTabAPI, "本地 API"},
					{toolsTabKeys, "密钥管理"},
					{toolsTabLogs, "应用日志"},
				}
				for _, tab := range tabs {
					tIdx, tLbl := tab.idx, tab.label
					if sidebarNavRow(c, fmt.Sprintf("tools-nav-%d", tIdx), tLbl, m.tab == tIdx, 0, nil, pal) {
						if m.tab != tIdx {
							m.tab = tIdx
							m.err = ""
							m.msg = ""
							a.refreshToolsTab(m.tab)
						}
					}
				}
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.line)

			// Right form pane + footer.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(20).Gap(12).AlignSelf(ui.Stretch).Children(func() {
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
				hairlineH(c, pal)
				ui.Row(c).PaddingX(20).PaddingY(12).Gap(8).Justify(ui.End).Background(pal.footer).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					if macSecondaryButton(c, "tools-close", "关闭", pal) {
						m.open = false
					}
				})
			})
		})
	})
}

func (a *App) viewToolsAI(c *ui.Context) {
	t := c.Theme()
	m := &a.toolsModal
	ui.Form(c, func() {
		ui.Field(c, "Base URL", func() {
			ui.TextInput(c.Key("ai-url"), &m.aiBaseURL).Placeholder("http://127.0.0.1:11434/v1").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "模型名", func() {
			ui.TextInput(c.Key("ai-model"), &m.aiModel).Placeholder("llama3 / qwen2.5").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "API Key", func() {
			ui.TextInput(c.Key("ai-tok"), &m.aiToken).Password().Placeholder("本地模型可留空").AlignSelf(ui.Stretch)
		})
	})
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c.Key("ai-probe"), "探测本地服务与模型").Clicked() {
			if res, err := a.backend.ProbeAI(strings.TrimSpace(m.aiBaseURL)); err != nil {
				m.err = err.Error()
			} else {
				m.aiProbe = res
				m.err = ""
				m.msg = fmt.Sprintf("探测完成：发现 %d 个模型", len(res.Models))
				if m.aiModel == "" && len(res.Models) > 0 {
					m.aiModel = res.Models[0]
				}
			}
		}
		if ui.Button(c.Key("ai-save"), "保存 AI 配置").Clicked() {
			if _, err := a.backend.ConfigureAI(strings.TrimSpace(m.aiBaseURL), strings.TrimSpace(m.aiModel), strings.TrimSpace(m.aiToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "AI 配置已保存"
			}
		}
		if len(m.aiHistory) > 0 {
			if ui.Button(c.Key("ai-clear"), "清空对话").Clicked() {
				m.aiHistory = nil
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
				if ui.Button(c.Key("ai-mdl-"+name), name).Clicked() {
					m.aiModel = name
				}
			}
		})
	}

	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)

	// Conversation history rendered natively with Markdown.
	for i, msg := range m.aiHistory {
		roleLabel := "你"
		bg := t.Surface
		if msg.Role == "assistant" {
			roleLabel = "AI 助手"
			bg = t.Background
		}
		ui.Column(c.Key(fmt.Sprintf("ai-msg-%d", i))).Padding(10).Gap(4).Background(bg).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, roleLabel).Bold().FontSize(11).TextColor(t.TextMuted)
			if msg.Role == "assistant" {
				markdown.View(c, msg.Content, &a.markdownOpts)
			} else {
				ui.Text(c, msg.Content).FontSize(13)
			}
		})
	}

	ui.Row(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
		if ui.TextInput(c.Key("ai-prompt"), &m.aiPrompt).Placeholder("输入问题（自动附带当前文章上下文）…").Grow(1).Submitted() {
			a.sendAIPrompt()
		}
		if ui.PrimaryButton(c.Key("ai-send"), "发送").Clicked() {
			a.sendAIPrompt()
		}
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
	t := c.Theme()
	m := &a.toolsModal
	state := "已停止"
	if m.apiStatus.Running {
		state = fmt.Sprintf("运行中 (127.0.0.1:%d)", m.apiStatus.Port)
	}
	ui.Text(c, "服务状态："+state).Bold().FontSize(13)
	ui.Form(c, func() {
		ui.Field(c, "启用服务", func() { ui.Checkbox(c.Key("api-en"), &m.apiEnabled, "启动本地 REST API") })
		ui.Field(c, "监听端口", func() { ui.TextInput(c.Key("api-port"), &m.apiPort).Width(140) })
		ui.Field(c, "基础认证", func() { ui.Checkbox(c.Key("api-auth"), &m.apiAuthOn, "启用 HTTP Basic Auth") })
		ui.Field(c, "用户名", func() { ui.TextInput(c.Key("api-user"), &m.apiUsername).AlignSelf(ui.Stretch) })
		ui.Field(c, "密码", func() { ui.TextInput(c.Key("api-pass"), &m.apiPasscode).Password().AlignSelf(ui.Stretch) })
	})
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c.Key("api-save"), "保存 API 配置").Clicked() {
			port, err := strconv.Atoi(strings.TrimSpace(m.apiPort))
			if err != nil {
				m.err = "端口必须为整数"
				return
			}
			st, err := a.backend.ConfigureLocalAPI(m.apiEnabled, port, m.apiAuthOn, strings.TrimSpace(m.apiUsername), m.apiPasscode)
			if err != nil {
				m.err = err.Error()
			} else {
				m.apiStatus = st
				m.err = ""
				m.msg = "本地 API 配置已保存"
			}
		}
		if ui.Button(c.Key("api-clear-logs"), "清空请求日志").Clicked() {
			a.backend.ClearLocalAPILogs()
			a.refreshToolsTab(toolsTabAPI)
		}
	})
	if len(m.apiStatus.Logs) > 0 {
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
		for i, entry := range m.apiStatus.Logs {
			ui.Text(c.Key(fmt.Sprintf("api-log-%d", i)), fmt.Sprintf("%s %s %s -> %d (%dms)", entry.Time, entry.Method, entry.Path, entry.Status, entry.DurationMs)).FontSize(11).TextColor(t.TextMuted)
		}
	}
}

func (a *App) viewToolsKeys(c *ui.Context) {
	t := c.Theme()
	m := &a.toolsModal
	ui.Row(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
		ui.TextInput(c.Key("key-imp-name"), &m.importKeyName).Placeholder("导入后的密钥名称").Grow(1)
		if ui.PrimaryButton(c.Key("key-imp-btn"), "导入密钥").Clicked() {
			name := strings.TrimSpace(m.importKeyName)
			if name == "" {
				m.err = "请输入导入密钥名称"
				return
			}
			if id, err := a.backend.ImportIPNSKey(context.Background(), name); err != nil {
				m.err = err.Error()
			} else {
				m.importKeyName = ""
				m.err = ""
				m.msg = "已导入密钥：" + id
				a.refreshToolsTab(toolsTabKeys)
			}
		}
	})
	if len(m.keys) == 0 {
		ui.Text(c, "暂无 IPNS 密钥（需先启动 IPFS 节点）").FontSize(12).TextColor(t.TextMuted)
	}
	for _, k := range m.keys {
		name := k.Name
		ui.Row(c.Key("key-row-"+name)).Padding(8).Gap(8).AlignItems(ui.Center).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, name).Bold().FontSize(12)
				ui.Text(c, k.ID).FontSize(11).TextColor(t.TextMuted)
			})
			if ui.Button(c.Key("key-exp-"+name), "导出").Clicked() {
				if path, err := a.backend.ExportIPNSKey(context.Background(), name); err != nil {
					m.err = err.Error()
				} else if path != "" {
					m.err = ""
					m.msg = "已导出至：" + path
				}
			}
		})
	}
}

func (a *App) viewToolsLogs(c *ui.Context) {
	t := c.Theme()
	m := &a.toolsModal
	ui.Row(c).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
		prev := m.logCatIdx
		ui.Segmented(c.Key("log-cat"), &m.logCatIdx, "全部", "Planet", "IPFS", "API").Label("日志分类")
		if m.logCatIdx != prev {
			a.refreshToolsTab(toolsTabLogs)
		}
		ui.Row(c).Gap(6).Children(func() {
			if ui.Button(c.Key("log-refresh"), "刷新").Clicked() {
				a.refreshToolsTab(toolsTabLogs)
			}
			if ui.Button(c.Key("log-clear"), "清空日志").Clicked() {
				a.backend.ClearAppLogs(logCategories[m.logCatIdx])
				a.refreshToolsTab(toolsTabLogs)
			}
		})
	})
	if len(m.logs) == 0 {
		ui.Text(c, "暂无日志记录").FontSize(12).TextColor(t.TextMuted)
	}
	for i, entry := range m.logs {
		ui.Text(c.Key(fmt.Sprintf("app-log-%d", i)), fmt.Sprintf("[%s] [%s] %s", entry.Time, entry.Category, entry.Message)).FontSize(11).TextColor(t.TextMuted)
	}
}
