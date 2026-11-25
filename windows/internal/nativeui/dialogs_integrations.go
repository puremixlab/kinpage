package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
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
	dwebDomain        string
	dwebToken         string

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
	web3Farcaster  string
}

type templatePreviewState struct {
	open     bool
	name     string
	htmlPath string
	htmlDoc  string
	wv       WebViewHost
}

func (a *App) loadIntegrationsState(site store.Site) {
	vals := map[string]string{}
	if a.backend != nil {
		if loaded, err := a.backend.TemplateSettings(site.ID); err == nil && loaded != nil {
			vals = loaded
		}
	}
	a.integrations = integrationsModalState{
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

func (a *App) openIntegrationsModal(site store.Site) {
	a.openSiteSettings(site)
	a.siteSettings.tab = siteSettingsTabTemplates
}

func (a *App) viewIntTemplates(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	curName := m.templateName
	if curName == "" {
		curName = "默认模板"
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "当前使用模板").Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
				ui.Text(c, curName).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink)
			})
			if macSecondaryButton(c, "tpl-import", "导入本地模板目录", pal) {
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
	})

	macInsetCard(c, pal, func() {
		ui.Text(c, "内置模板库").Font(macPlanetFont).FontSize(12.5).FontWeight(600).TextColor(pal.ink)
		for i, b := range a.backend.BuiltinTemplates() {
			tpl := b
			if i > 0 {
				hairlineH(c, pal)
			}
			ui.Row(c.Key("tpl-row-" + tpl.ID)).PaddingY(4).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, tpl.Name).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
					ui.Text(c, tpl.Description).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
				})
				ui.Row(c).Gap(6).Shrink(0).Children(func() {
					if macSecondaryButton(c, "tpl-preview-"+tpl.ID, "预览", pal) {
						a.previewBuiltinTemplate(tpl.ID, tpl.Name)
					}
					if macSecondaryButton(c, "tpl-apply-"+tpl.ID, "应用", pal) {
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
			})
		}
	})

	if len(m.templateValues) > 0 {
		macInsetCard(c, pal, func() {
			ui.Text(c, "模板参数设置").Font(macPlanetFont).FontSize(12.5).FontWeight(600).TextColor(pal.ink)
			keys := make([]string, 0, len(m.templateValues))
			for k := range m.templateValues {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				key := k
				val := m.templateValues[key]
				macField(c, key, "", pal, func() {
					ui.TextInput(c.Key("tpl-val-"+key), &val).Label(key).AlignSelf(ui.Stretch)
				})
				m.templateValues[key] = val
			}
			ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
				if macPrimaryButtonSmall(c, "tpl-save-vals", "保存模板设置", pal) {
					raw, _ := json.Marshal(m.templateValues)
					if _, err := a.backend.SaveTemplateSettings(m.siteID, string(raw)); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						m.msg = "模板设置已保存"
						a.Reload()
					}
				}
			})
		})
	}
}

func (a *App) previewBuiltinTemplate(id, name string) {
	m := &a.integrations
	htmlDoc, err := a.backend.BuiltinTemplatePreview(id)
	if err != nil {
		m.err = "生成模板预览失败：" + err.Error()
		return
	}
	dir := filepath.Join(os.TempDir(), "planet-template-preview")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		m.err = err.Error()
		return
	}
	target := filepath.Join(dir, id+".html")
	if err := os.WriteFile(target, []byte(htmlDoc), 0o600); err != nil {
		m.err = err.Error()
		return
	}

	// Prefer in-app sandboxed WebView preview when host window supports it.
	if a.newWebView != nil {
		wv := a.templatePreview.wv
		if wv == nil {
			created, wvErr := a.newWebView()
			if wvErr == nil && created != nil {
				wv = created
			}
		}
		if wv != nil {
			wv.LoadHTML(htmlDoc, "")
			a.templatePreview = templatePreviewState{
				open:     true,
				name:     name,
				htmlPath: target,
				htmlDoc:  htmlDoc,
				wv:       wv,
			}
			m.err = ""
			m.msg = ""
			return
		}
	}

	// Fallback to external browser in headless/unsupported environments.
	opener := a.openExternal
	if opener == nil {
		opener = openBrowserFile
	}
	if err := opener(target); err != nil {
		m.err = "打开浏览器预览失败：" + err.Error()
		return
	}
	m.err = ""
	m.msg = "已在系统浏览器中打开模板预览：" + name
}

func (a *App) viewTemplatePreviewSheet(c *ui.Context) {
	pal := a.pal
	tp := &a.templatePreview
	if !tp.open {
		if tp.wv != nil {
			a.destroyTemplatePreviewWV()
		}
		return
	}
	macPreferencesWindow(c, &tp.open, 1020, 720, pal, func() {
		ui.Column(c).Fill().Background(pal.canvas).Children(func() {
			ui.Row(c).Height(48).Padding(0, 14, 0, 20).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.prefsSidebar).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, fallbackTitle(tp.name)).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.prefsInk)
					ui.Box(c).Padding(2, 7, 2, 7).Radius(5).Background(pal.accent.Alpha(0.12)).Children(func() {
						ui.Text(c, "模板预览").Font(macPlanetFont).FontSize(11).FontWeight(500).TextColor(pal.accent)
					})
				})
				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					if tp.htmlPath != "" {
						if toolbarIconBtn(c, "tpl-preview-ext", lucide.ExternalLink(), "在浏览器中打开").Clicked() {
							opener := a.openExternal
							if opener == nil {
								opener = openBrowserFile
							}
							if err := opener(tp.htmlPath); err != nil {
								a.setError("打开外部浏览器失败：" + err.Error())
							}
						}
					}
					if toolbarIconBtn(c, "tpl-preview-close", lucide.X(), "关闭预览").Clicked() {
						tp.open = false
					}
				})
			})
			prefsHairlineH(c, pal)
			ui.Box(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
				if tp.wv != nil && tp.wv.Native() != nil {
					ui.WebView(c, tp.wv.Native()).Fill()
				} else {
					ui.Column(c).Fill().Center().Children(func() {
						ui.Text(c, "正在加载模板预览…").Font(macPlanetFont).FontSize(13).TextColor(pal.muted)
					})
				}
			})
		})
	})
	if !tp.open && tp.wv != nil {
		a.destroyTemplatePreviewWV()
	}
}

func (a *App) viewIntDeploy(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macSegmented(c, "dep-kind", &m.deployKindIdx, []string{"Cloudflare", "Rsync", "dStack"}, pal)
	kind := deployKinds[m.deployKindIdx]
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "项目 / 目标名", "", pal, func() {
					ui.TextInput(c.Key("dep-name"), &m.deployName).Label("项目名称").AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "服务器 / 账号", "", pal, func() {
					ui.TextInput(c.Key("dep-server"), &m.deployServer).Label("服务器地址").AlignSelf(ui.Stretch)
				})
			})
		})
		macField(c, "API Token / 密钥", "", pal, func() {
			ui.TextInput(c.Key("dep-token"), &m.deployToken).Password().Label("部署令牌").AlignSelf(ui.Stretch)
		})
		ui.Row(c).Gap(20).Wrap().Children(func() {
			ui.Checkbox(c.Key("dep-auto"), &m.deployAuto, "发布后自动部署")
			ui.Checkbox(c.Key("dep-del"), &m.deployDelete, "清理远端多余文件")
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "dep-save", "保存部署配置", pal) {
			if err := a.backend.ConfigureDeployment(m.siteID, kind, strings.TrimSpace(m.deployName), strings.TrimSpace(m.deployServer), strings.TrimSpace(m.deployToken), m.deployAuto, m.deployDelete); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 " + kind + " 部署配置"
			}
		}
		if macSecondaryButton(c, "dep-run", "立即部署", pal) {
			if msg, err := a.backend.DeploySite(m.siteID, kind); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "部署完成：" + msg
			}
		}
		if macSecondaryButton(c, "dep-disable", "停用部署", pal) {
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
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		ui.Text(c, "Plausible 统计").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "域名", "", pal, func() { ui.TextInput(c.Key("srv-pl-dom"), &m.plausibleDomain).AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Script ID", "", pal, func() { ui.TextInput(c.Key("srv-pl-sid"), &m.plausibleScriptID).AlignSelf(ui.Stretch) })
			})
		})
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "API Key", "", pal, func() { ui.TextInput(c.Key("srv-pl-tok"), &m.plausibleToken).Password().AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "服务器", "", pal, func() {
					ui.TextInput(c.Key("srv-pl-srv"), &m.plausibleServer).Placeholder("https://plausible.io").AlignSelf(ui.Stretch)
				})
			})
		})
		ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "srv-pl-save", "保存 Plausible", pal) {
				if err := a.backend.ConfigurePlausibleAt(m.siteID, strings.TrimSpace(m.plausibleDomain), strings.TrimSpace(m.plausibleToken), strings.TrimSpace(m.plausibleScriptID), strings.TrimSpace(m.plausibleServer)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = "已保存 Plausible 配置"
				}
			}
		})
	})

	macInsetCard(c, pal, func() {
		ui.Text(c, "Filebase & Pinnable 固定服务").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Filebase Pin 名", "", pal, func() { ui.TextInput(c.Key("srv-fb-name"), &m.filebaseBucket).AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Filebase Token", "", pal, func() { ui.TextInput(c.Key("srv-fb-tok"), &m.filebaseToken).Password().AlignSelf(ui.Stretch) })
			})
		})
		macField(c, "Pinnable Endpoint", "", pal, func() { ui.TextInput(c.Key("srv-pin-ep"), &m.pinnableEndpoint).AlignSelf(ui.Stretch) })
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if macSecondaryButton(c, "srv-fb-save", "保存 Filebase", pal) {
				if err := a.backend.ConfigureService(m.siteID, "filebase", strings.TrimSpace(m.filebaseBucket), strings.TrimSpace(m.filebaseToken)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = "已保存 Filebase 配置"
				}
			}
			if macSecondaryButton(c, "srv-fb-pin", "触发 Filebase Pin", pal) {
				if err := a.backend.RequestFilebasePin(m.siteID); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = "已请求 Filebase 固定"
				}
			}
			if macSecondaryButton(c, "srv-pin-save", "保存 Pinnable", pal) {
				if err := a.backend.ConfigurePinnable(m.siteID, strings.TrimSpace(m.pinnableEndpoint)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = "已保存 Pinnable 配置"
				}
			}
			if macSecondaryButton(c, "srv-pin-req", "触发 Pinnable Pin", pal) {
				if err := a.backend.RequestPinnablePin(m.siteID); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = "已请求 Pinnable 固定"
				}
			}
		})
	})
	macInsetCard(c, pal, func() {
		ui.Text(c, "dWebServices.xyz 托管 IPNS").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "域名", "", pal, func() {
					ui.TextInput(c.Key("srv-dweb-dom"), &m.dwebDomain).Placeholder("example.eth").AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "API Key", "", pal, func() {
					ui.TextInput(c.Key("srv-dweb-tok"), &m.dwebToken).Password().AlignSelf(ui.Stretch)
				})
			})
		})
		if macSecondaryButton(c, "srv-dweb-save", "保存 dWebServices", pal) {
			if err := a.backend.ConfigureService(m.siteID, "dwebservices", strings.TrimSpace(m.dwebDomain), strings.TrimSpace(m.dwebToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 dWebServices.xyz 配置"
			}
		}
	})
}

func (a *App) viewIntX(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "API Key", "", pal, func() { ui.TextInput(c.Key("x-ak"), &m.xAPIKey).Password().AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "API Secret", "", pal, func() { ui.TextInput(c.Key("x-as"), &m.xAPISecret).Password().AlignSelf(ui.Stretch) })
			})
		})
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Access Token", "", pal, func() { ui.TextInput(c.Key("x-at"), &m.xAccessToken).Password().AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Access Secret", "", pal, func() { ui.TextInput(c.Key("x-ats"), &m.xAccessSecret).Password().AlignSelf(ui.Stretch) })
			})
		})
		ui.Checkbox(c.Key("x-auto"), &m.xAutoSync, "发布文章时自动同步到 X")
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "x-save", "保存 X 配置", pal) {
			if err := a.backend.ConfigureXSync(m.siteID, m.xAPIKey, m.xAPISecret, m.xAccessToken, m.xAccessSecret, m.xAutoSync); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 X 同步配置"
			}
		}
		if macSecondaryButton(c, "x-disable", "停用 X 同步", pal) {
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
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		macField(c, "频道 / Chat ID", "", pal, func() {
			ui.TextInput(c.Key("tg-chat"), &m.tgChatID).Placeholder("@channel 或数字 ID").AlignSelf(ui.Stretch)
		})
		macField(c, "Bot Token", "", pal, func() { ui.TextInput(c.Key("tg-token"), &m.tgToken).Password().AlignSelf(ui.Stretch) })
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "tg-save", "保存 Telegram", pal) {
			if err := a.backend.ConfigureTelegram(m.siteID, strings.TrimSpace(m.tgChatID), strings.TrimSpace(m.tgToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存 Telegram 同步配置"
			}
		}
		if macSecondaryButton(c, "tg-retry", "重试未同步文章", pal) {
			if n, err := a.backend.RetryTelegram(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = fmt.Sprintf("已同步 %d 篇文章到 Telegram", n)
			}
		}
		if macSecondaryButton(c, "tg-disable", "停用 Telegram", pal) {
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
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		macField(c, "聚合源列表", "每行一个 RSS / Atom / IPNS 地址", pal, func() {
			ui.TextArea(c.Key("agg-sources"), &m.aggSources).Lines(6, 10).Label("聚合源列表").AlignSelf(ui.Stretch)
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "agg-save", "保存聚合源", pal) {
			if _, err := a.backend.UpdateSiteAggregation(m.siteID, m.aggSources); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = "已保存聚合源配置"
				a.Reload()
			}
		}
		if macSecondaryButton(c, "agg-sync", "立即抓取同步", pal) {
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
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		ui.Checkbox(c.Key("w3-donate"), &m.web3Donate, "接受以太坊捐赠")
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "ETH / ENS 地址", "", pal, func() {
					ui.TextInput(c.Key("w3-eth"), &m.web3ETH).Placeholder("0x… 或 name.eth").AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "捐赠说明", "", pal, func() { ui.TextInput(c.Key("w3-msg"), &m.web3Message).AlignSelf(ui.Stretch) })
			})
		})
	})
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(16).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.Checkbox(c.Key("w3-jb"), &m.web3Juicebox, "启用 Juicebox 项目支持")
			ui.Box(c).Width(160).Children(func() {
				ui.TextInput(c.Key("w3-jbid"), &m.web3JuiceboxID).Placeholder("Juicebox ID").AlignSelf(ui.Stretch)
			})
		})
	})
	macInsetCard(c, pal, func() {
		macField(c, "Farcaster Mini App (.well-known/farcaster.json)", "", pal, func() {
			ui.TextArea(c.Key("w3-farcaster"), &m.web3Farcaster).Placeholder(`{"accountAssociation":{...}}`).Lines(3, 6).AlignSelf(ui.Stretch)
		})
	})
	if macPrimaryButtonSmall(c, "w3-save", "保存 Web3 设置", pal) {
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
			FarcasterManifestJSON:     strings.TrimSpace(m.web3Farcaster),
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
