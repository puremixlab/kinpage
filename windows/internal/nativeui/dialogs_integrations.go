package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

const (
	intTabTemplates = 0
	intTabDeploy    = 1
	intTabServices  = 2
	intTabX         = 3
	intTabTelegram  = 4
	intTabAggregate = 5
	intTabWeb3      = 6
)

var deployKinds = []string{"cloudflare", "rsync", "dstack"}

type integrationsModalState struct {
	open     bool
	tab      int
	siteID   string
	siteName string
	err      string
	msg      string

	// Templates.
	templateName   string
	templateValues map[string]string

	// Deployment.
	deployKindIdx int
	deployName    string
	deployServer  string
	deployToken   string
	deployAuto    bool
	deployDelete  bool

	// External services (Plausible / Filebase / Pinnable).
	plausibleDomain   string
	plausibleToken    string
	plausibleScriptID string
	plausibleServer   string
	filebaseBucket    string
	filebaseToken     string
	pinnableEndpoint  string

	// X sync.
	xAPIKey       string
	xAPISecret    string
	xAccessToken  string
	xAccessSecret string
	xAutoSync     bool

	// Telegram.
	tgChatID string
	tgToken  string

	// Aggregation.
	aggSources string

	// Web3.
	web3Donate     bool
	web3Message    string
	web3ETH        string
	web3Juicebox   bool
	web3JuiceboxID string
}

func (a *App) openIntegrationsModal(site store.Site) {
	vals := map[string]string{}
	if loaded, err := a.backend.TemplateSettings(site.ID); err == nil && loaded != nil {
		vals = loaded
	}
	a.integrations = integrationsModalState{
		open:           true,
		tab:            intTabTemplates,
		siteID:         site.ID,
		siteName:       fallbackTitle(site.Name),
		templateName:   site.TemplateName,
		templateValues: vals,
		aggSources:     strings.Join(site.Aggregation, "\n"),
		web3Donate:     site.AcceptsDonation,
		web3Message:    site.AcceptsDonationMessage,
		web3ETH:        site.AcceptsDonationETHAddress,
		web3Juicebox:   site.JuiceboxEnabled,
		web3JuiceboxID: strconv.Itoa(site.JuiceboxProjectID),
	}
}

func (a *App) viewIntegrationsModal(c *ui.Context) {
	t := c.Theme()
	m := &a.integrations
	ui.Modal(c, &m.open, func() {
		ui.Column(c).Width(640).Gap(12).Children(func() {
			ui.Text(c, "模板与扩展 · "+m.siteName).Bold().FontSize(16)
			ui.Segmented(c.Key("int-tab"), &m.tab, "模板", "部署", "服务", "X 同步", "Telegram", "聚合", "Web3").Label("扩展分类")

			ui.Scroll(c).Height(340).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(10).Padding(4).AlignSelf(ui.Stretch).Children(func() {
					switch m.tab {
					case intTabTemplates:
						a.viewIntTemplates(c)
					case intTabDeploy:
						a.viewIntDeploy(c)
					case intTabServices:
						a.viewIntServices(c)
					case intTabX:
						a.viewIntX(c)
					case intTabTelegram:
						a.viewIntTelegram(c)
					case intTabAggregate:
						a.viewIntAggregate(c)
					case intTabWeb3:
						a.viewIntWeb3(c)
					}
				})
			})

			if m.err != "" {
				ui.Text(c, m.err).FontSize(12).TextColor(t.Danger)
			} else if m.msg != "" {
				ui.Text(c, m.msg).FontSize(12).TextColor(t.TextMuted)
			}

			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("int-close"), "关闭").Clicked() {
					m.open = false
				}
			})
		})
	})
}

func (a *App) viewIntTemplates(c *ui.Context) {
	t := c.Theme()
	m := &a.integrations
	curName := m.templateName
	if curName == "" {
		curName = "默认模板"
	}
	ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, "当前模板："+curName).Bold().FontSize(13)
		if ui.Button(c.Key("tpl-import"), "导入本地模板目录").Clicked() {
			if site, err := a.backend.SelectTemplate(context.Background(), m.siteID); err != nil {
				m.err = err.Error()
			} else if site.ID != "" {
				m.templateName = site.TemplateName
				m.err = ""
				m.msg = "已导入模板：" + site.TemplateName
				a.Reload()
			}
		}
	})

	for _, b := range a.backend.BuiltinTemplates() {
		tpl := b
		ui.Row(c.Key("tpl-row-"+tpl.ID)).Padding(8).Gap(8).AlignItems(ui.Center).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, tpl.Name).Bold().FontSize(13)
				ui.Text(c, tpl.Description).FontSize(11).TextColor(t.TextMuted)
			})
			if ui.PrimaryButton(c.Key("tpl-apply-"+tpl.ID), "应用").Clicked() {
				if site, err := a.backend.ApplyBuiltinTemplate(m.siteID, tpl.ID); err != nil {
					m.err = err.Error()
				} else {
					m.templateName = site.TemplateName
					m.err = ""
					m.msg = "已应用模板：" + tpl.Name
					if loaded, err := a.backend.TemplateSettings(m.siteID); err == nil && loaded != nil {
						m.templateValues = loaded
					}
					a.Reload()
				}
			}
		})
	}

	if len(m.templateValues) > 0 {
		ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
		ui.Text(c, "模板参数设置").Bold().FontSize(13)
		keys := make([]string, 0, len(m.templateValues))
		for k := range m.templateValues {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		ui.Form(c, func() {
			for _, k := range keys {
				key := k
				val := m.templateValues[key]
				ui.Field(c, key, func() {
					ui.TextInput(c.Key("tpl-val-"+key), &val).Label(key).AlignSelf(ui.Stretch)
				})
				m.templateValues[key] = val
			}
		})
		if ui.PrimaryButton(c.Key("tpl-save-vals"), "保存模板设置").Clicked() {
			raw, _ := json.Marshal(m.templateValues)
			if _, err := a.backend.SaveTemplateSettings(m.siteID, string(raw)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "模板设置已保存"
				a.Reload()
			}
		}
	}
}

func (a *App) viewIntDeploy(c *ui.Context) {
	m := &a.integrations
	ui.Segmented(c.Key("dep-kind"), &m.deployKindIdx, "Cloudflare", "Rsync", "dStack").Label("部署目标")
	kind := deployKinds[m.deployKindIdx]
	ui.Form(c, func() {
		ui.Field(c, "项目 / 目标名", func() {
			ui.TextInput(c.Key("dep-name"), &m.deployName).Label("项目名称").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "服务器 / 账号", func() {
			ui.TextInput(c.Key("dep-server"), &m.deployServer).Label("服务器地址").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "API Token / 密钥", func() {
			ui.TextInput(c.Key("dep-token"), &m.deployToken).Password().Label("部署令牌").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "选项", func() {
			ui.Row(c).Gap(12).Children(func() {
				ui.Checkbox(c.Key("dep-auto"), &m.deployAuto, "发布后自动部署")
				ui.Checkbox(c.Key("dep-del"), &m.deployDelete, "清理远端多余文件")
			})
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c.Key("dep-save"), "保存部署配置").Clicked() {
			if err := a.backend.ConfigureDeployment(m.siteID, kind, strings.TrimSpace(m.deployName), strings.TrimSpace(m.deployServer), strings.TrimSpace(m.deployToken), m.deployAuto, m.deployDelete); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 " + kind + " 部署配置"
			}
		}
		if ui.Button(c.Key("dep-run"), "立即部署").Clicked() {
			if msg, err := a.backend.DeploySite(m.siteID, kind); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "部署完成：" + msg
			}
		}
		if ui.Button(c.Key("dep-disable"), "停用部署").Clicked() {
			if err := a.backend.DisableDeployment(m.siteID, kind); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已停用 " + kind + " 部署"
			}
		}
	})
}

func (a *App) viewIntServices(c *ui.Context) {
	t := c.Theme()
	m := &a.integrations
	ui.Text(c, "Plausible 统计").Bold().FontSize(13)
	ui.Form(c, func() {
		ui.Field(c, "域名", func() { ui.TextInput(c.Key("srv-pl-dom"), &m.plausibleDomain).AlignSelf(ui.Stretch) })
		ui.Field(c, "API Key", func() { ui.TextInput(c.Key("srv-pl-tok"), &m.plausibleToken).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "Script ID", func() { ui.TextInput(c.Key("srv-pl-sid"), &m.plausibleScriptID).AlignSelf(ui.Stretch) })
		ui.Field(c, "服务器", func() {
			ui.TextInput(c.Key("srv-pl-srv"), &m.plausibleServer).Placeholder("https://plausible.io").AlignSelf(ui.Stretch)
		})
	})
	if ui.Button(c.Key("srv-pl-save"), "保存 Plausible").Clicked() {
		if err := a.backend.ConfigurePlausibleAt(m.siteID, strings.TrimSpace(m.plausibleDomain), strings.TrimSpace(m.plausibleToken), strings.TrimSpace(m.plausibleScriptID), strings.TrimSpace(m.plausibleServer)); err != nil {
			m.err = err.Error()
		} else {
			m.err = ""
			m.msg = "已保存 Plausible 配置"
		}
	}

	ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
	ui.Text(c, "Filebase & Pinnable 固定服务").Bold().FontSize(13)
	ui.Form(c, func() {
		ui.Field(c, "Filebase Pin 名", func() { ui.TextInput(c.Key("srv-fb-name"), &m.filebaseBucket).AlignSelf(ui.Stretch) })
		ui.Field(c, "Filebase Token", func() { ui.TextInput(c.Key("srv-fb-tok"), &m.filebaseToken).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "Pinnable Endpoint", func() { ui.TextInput(c.Key("srv-pin-ep"), &m.pinnableEndpoint).AlignSelf(ui.Stretch) })
	})
	ui.Row(c).Gap(8).Wrap().Children(func() {
		if ui.Button(c.Key("srv-fb-save"), "保存 Filebase").Clicked() {
			if err := a.backend.ConfigureService(m.siteID, "filebase", strings.TrimSpace(m.filebaseBucket), strings.TrimSpace(m.filebaseToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 Filebase 配置"
			}
		}
		if ui.Button(c.Key("srv-fb-pin"), "触发 Filebase Pin").Clicked() {
			if err := a.backend.RequestFilebasePin(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已请求 Filebase 固定"
			}
		}
		if ui.Button(c.Key("srv-pin-save"), "保存 Pinnable").Clicked() {
			if err := a.backend.ConfigurePinnable(m.siteID, strings.TrimSpace(m.pinnableEndpoint)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 Pinnable 配置"
			}
		}
		if ui.Button(c.Key("srv-pin-req"), "触发 Pinnable Pin").Clicked() {
			if err := a.backend.RequestPinnablePin(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已请求 Pinnable 固定"
			}
		}
	})
}

func (a *App) viewIntX(c *ui.Context) {
	m := &a.integrations
	ui.Form(c, func() {
		ui.Field(c, "API Key", func() { ui.TextInput(c.Key("x-ak"), &m.xAPIKey).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "API Secret", func() { ui.TextInput(c.Key("x-as"), &m.xAPISecret).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "Access Token", func() { ui.TextInput(c.Key("x-at"), &m.xAccessToken).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "Access Secret", func() { ui.TextInput(c.Key("x-ats"), &m.xAccessSecret).Password().AlignSelf(ui.Stretch) })
		ui.Field(c, "同步方式", func() { ui.Checkbox(c.Key("x-auto"), &m.xAutoSync, "发布文章时自动同步到 X") })
	})
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c.Key("x-save"), "保存 X 配置").Clicked() {
			if err := a.backend.ConfigureXSync(m.siteID, m.xAPIKey, m.xAPISecret, m.xAccessToken, m.xAccessSecret, m.xAutoSync); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 X 同步配置"
			}
		}
		if ui.Button(c.Key("x-disable"), "停用 X 同步").Clicked() {
			if err := a.backend.DisableXSync(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已停用 X 同步"
			}
		}
	})
}

func (a *App) viewIntTelegram(c *ui.Context) {
	m := &a.integrations
	ui.Form(c, func() {
		ui.Field(c, "频道 / Chat ID", func() {
			ui.TextInput(c.Key("tg-chat"), &m.tgChatID).Placeholder("@channel 或数字 ID").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "Bot Token", func() { ui.TextInput(c.Key("tg-token"), &m.tgToken).Password().AlignSelf(ui.Stretch) })
	})
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c.Key("tg-save"), "保存 Telegram").Clicked() {
			if err := a.backend.ConfigureTelegram(m.siteID, strings.TrimSpace(m.tgChatID), strings.TrimSpace(m.tgToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 Telegram 同步配置"
			}
		}
		if ui.Button(c.Key("tg-retry"), "重试未同步文章").Clicked() {
			if n, err := a.backend.RetryTelegram(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = fmt.Sprintf("已同步 %d 篇文章到 Telegram", n)
			}
		}
		if ui.Button(c.Key("tg-disable"), "停用 Telegram").Clicked() {
			if err := a.backend.DisableTelegram(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已停用 Telegram 同步"
			}
		}
	})
}

func (a *App) viewIntAggregate(c *ui.Context) {
	m := &a.integrations
	ui.Text(c, "每行一个 RSS / Atom / IPNS 聚合源地址：").FontSize(12).TextColor(c.Theme().TextMuted)
	ui.TextArea(c.Key("agg-sources"), &m.aggSources).Lines(6, 10).Label("聚合源列表").AlignSelf(ui.Stretch)
	ui.Row(c).Gap(8).Children(func() {
		if ui.PrimaryButton(c.Key("agg-save"), "保存聚合源").Clicked() {
			if _, err := a.backend.UpdateSiteAggregation(m.siteID, m.aggSources); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存聚合源配置"
				a.Reload()
			}
		}
		if ui.Button(c.Key("agg-sync"), "立即抓取同步").Clicked() {
			if n, err := a.backend.SyncSiteAggregation(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = fmt.Sprintf("已聚合新增 %d 篇文章", n)
				a.Reload()
			}
		}
	})
}

func (a *App) viewIntWeb3(c *ui.Context) {
	m := &a.integrations
	ui.Form(c, func() {
		ui.Field(c, "以太坊捐赠", func() { ui.Checkbox(c.Key("w3-donate"), &m.web3Donate, "接受以太坊捐赠") })
		ui.Field(c, "ETH / ENS 地址", func() {
			ui.TextInput(c.Key("w3-eth"), &m.web3ETH).Placeholder("0x… 或 name.eth").AlignSelf(ui.Stretch)
		})
		ui.Field(c, "捐赠说明", func() { ui.TextInput(c.Key("w3-msg"), &m.web3Message).AlignSelf(ui.Stretch) })
		ui.Field(c, "Juicebox", func() { ui.Checkbox(c.Key("w3-jb"), &m.web3Juicebox, "启用 Juicebox 项目支持") })
		ui.Field(c, "Juicebox ID", func() { ui.TextInput(c.Key("w3-jbid"), &m.web3JuiceboxID).Width(140) })
	})
	if ui.PrimaryButton(c.Key("w3-save"), "保存 Web3 设置").Clicked() {
		jbID := 0
		if s := strings.TrimSpace(m.web3JuiceboxID); s != "" {
			parsed, err := strconv.Atoi(s)
			if err != nil {
				m.err = "Juicebox 项目 ID 必须为整数"
				return
			}
			jbID = parsed
		}
		raw, _ := json.Marshal(store.SiteWeb3Settings{
			AcceptsDonation:           m.web3Donate,
			AcceptsDonationMessage:    strings.TrimSpace(m.web3Message),
			AcceptsDonationETHAddress: strings.TrimSpace(m.web3ETH),
			JuiceboxEnabled:           m.web3Juicebox,
			JuiceboxProjectID:         jbID,
		})
		if _, err := a.backend.UpdateSiteWeb3(m.siteID, string(raw)); err != nil {
			m.err = err.Error()
		} else {
			m.err = ""
			m.msg = "Web3 设置已保存"
			a.Reload()
		}
	}
}
