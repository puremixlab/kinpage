package nativeui

import (
	"context"
	"encoding/json"
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
		siteName:       a.fallbackTitle(site.Name),
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
		curName = a.t("int.tpl.default")
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, a.t("int.tpl.current")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
				ui.Text(c, curName).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.ink)
			})
			if macSecondaryButton(c, "tpl-import", a.t("int.tpl.importLocal"), pal) {
				if site, err := a.backend.SelectTemplate(context.Background(), m.siteID); err != nil {
					m.err = err.Error()
				} else if site.ID != "" {
					m.templateName = site.TemplateName
					m.err = ""
					m.msg = a.tf("int.tpl.imported", site.TemplateName)
					a.Reload()
				}
			}
		})
	})

	macInsetCard(c, pal, func() {
		ui.Text(c, a.t("int.tpl.builtinLib")).Font(macPlanetFont).FontSize(12.5).FontWeight(600).TextColor(pal.ink)
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
					if macSecondaryButton(c, "tpl-preview-"+tpl.ID, a.t("common.preview"), pal) {
						a.previewBuiltinTemplate(tpl.ID, tpl.Name)
					}
					if macSecondaryButton(c, "tpl-apply-"+tpl.ID, a.t("common.apply"), pal) {
						if site, err := a.backend.ApplyBuiltinTemplate(m.siteID, tpl.ID); err != nil {
							m.err = err.Error()
						} else {
							m.templateName = site.TemplateName
							m.err = ""
							m.msg = a.tf("int.tpl.applied", tpl.Name)
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
			ui.Text(c, a.t("int.tpl.params")).Font(macPlanetFont).FontSize(12.5).FontWeight(600).TextColor(pal.ink)
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
				if macPrimaryButtonSmall(c, "tpl-save-vals", a.t("int.tpl.saveParams"), pal) {
					raw, _ := json.Marshal(m.templateValues)
					if _, err := a.backend.SaveTemplateSettings(m.siteID, string(raw)); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						m.msg = a.t("int.tpl.paramsSaved")
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
		m.err = a.tf("int.tpl.previewGenErr", err.Error())
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
		m.err = a.tf("int.tpl.previewBrowserErr", err.Error())
		return
	}
	m.err = ""
	m.msg = a.tf("int.tpl.previewBrowserOpened", name)
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
					ui.Text(c, a.fallbackTitle(tp.name)).Font(macPlanetFont).FontSize(14).FontWeight(600).TextColor(pal.prefsInk)
					ui.Box(c).Padding(2, 7, 2, 7).Radius(5).Background(pal.accent.Alpha(0.12)).Children(func() {
						ui.Text(c, a.t("int.tpl.previewBadge")).Font(macPlanetFont).FontSize(11).FontWeight(500).TextColor(pal.accent)
					})
				})
				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					if tp.htmlPath != "" {
						if toolbarIconBtn(c, "tpl-preview-ext", lucide.ExternalLink(), a.t("menu.openInBrowser")).Clicked() {
							opener := a.openExternal
							if opener == nil {
								opener = openBrowserFile
							}
							if err := opener(tp.htmlPath); err != nil {
								a.setError(a.tf("status.openBrowserErr", err.Error()))
							}
						}
					}
					if toolbarIconBtn(c, "tpl-preview-close", lucide.X(), a.t("int.tpl.previewClose")).Clicked() {
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
						ui.Text(c, a.t("int.tpl.previewLoading")).Font(macPlanetFont).FontSize(13).TextColor(pal.muted)
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
				macField(c, a.t("int.deploy.nameField"), "", pal, func() {
					ui.TextInput(c.Key("dep-name"), &m.deployName).Label(a.t("int.deploy.nameLabel")).AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.deploy.serverField"), "", pal, func() {
					ui.TextInput(c.Key("dep-server"), &m.deployServer).Label(a.t("int.deploy.serverLabel")).AlignSelf(ui.Stretch)
				})
			})
		})
		macField(c, a.t("int.deploy.tokenField"), "", pal, func() {
			ui.TextInput(c.Key("dep-token"), &m.deployToken).Password().Label(a.t("int.deploy.tokenLabel")).AlignSelf(ui.Stretch)
		})
		ui.Row(c).Gap(20).Wrap().Children(func() {
			ui.Checkbox(c.Key("dep-auto"), &m.deployAuto, a.t("int.deploy.auto"))
			ui.Checkbox(c.Key("dep-del"), &m.deployDelete, a.t("int.deploy.deleteRemote"))
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "dep-save", a.t("int.deploy.save"), pal) {
			if err := a.backend.ConfigureDeployment(m.siteID, kind, strings.TrimSpace(m.deployName), strings.TrimSpace(m.deployServer), strings.TrimSpace(m.deployToken), m.deployAuto, m.deployDelete); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.tf("int.deploy.saved", kind)
			}
		}
		if macSecondaryButton(c, "dep-run", a.t("int.deploy.run"), pal) {
			if msg, err := a.backend.DeploySite(m.siteID, kind); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.tf("int.deploy.runDone", msg)
			}
		}
		if macSecondaryButton(c, "dep-disable", a.t("int.deploy.disable"), pal) {
			if err := a.backend.DisableDeployment(m.siteID, kind); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.tf("int.deploy.disabled", kind)
			}
		}
	})
}

func (a *App) viewIntServices(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		ui.Text(c, a.t("int.srv.plausibleTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.srv.domain"), "", pal, func() { ui.TextInput(c.Key("srv-pl-dom"), &m.plausibleDomain).AlignSelf(ui.Stretch) })
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
				macField(c, a.t("int.srv.server"), "", pal, func() {
					ui.TextInput(c.Key("srv-pl-srv"), &m.plausibleServer).Placeholder("https://plausible.io").AlignSelf(ui.Stretch)
				})
			})
		})
		ui.Row(c).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "srv-pl-save", a.t("int.srv.savePlausible"), pal) {
				if err := a.backend.ConfigurePlausibleAt(m.siteID, strings.TrimSpace(m.plausibleDomain), strings.TrimSpace(m.plausibleToken), strings.TrimSpace(m.plausibleScriptID), strings.TrimSpace(m.plausibleServer)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("int.srv.savedPlausible")
				}
			}
		})
	})

	macInsetCard(c, pal, func() {
		ui.Text(c, a.t("int.srv.pinningTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.srv.filebasePinName"), "", pal, func() { ui.TextInput(c.Key("srv-fb-name"), &m.filebaseBucket).AlignSelf(ui.Stretch) })
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "Filebase Token", "", pal, func() { ui.TextInput(c.Key("srv-fb-tok"), &m.filebaseToken).Password().AlignSelf(ui.Stretch) })
			})
		})
		macField(c, "Pinnable Endpoint", "", pal, func() { ui.TextInput(c.Key("srv-pin-ep"), &m.pinnableEndpoint).AlignSelf(ui.Stretch) })
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if macSecondaryButton(c, "srv-fb-save", a.t("int.srv.saveFilebase"), pal) {
				if err := a.backend.ConfigureService(m.siteID, "filebase", strings.TrimSpace(m.filebaseBucket), strings.TrimSpace(m.filebaseToken)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("int.srv.savedFilebase")
				}
			}
			if macSecondaryButton(c, "srv-fb-pin", a.t("int.srv.reqFilebasePin"), pal) {
				if err := a.backend.RequestFilebasePin(m.siteID); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("int.srv.reqFilebaseDone")
				}
			}
			if macSecondaryButton(c, "srv-pin-save", a.t("int.srv.savePinnable"), pal) {
				if err := a.backend.ConfigurePinnable(m.siteID, strings.TrimSpace(m.pinnableEndpoint)); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("int.srv.savedPinnable")
				}
			}
			if macSecondaryButton(c, "srv-pin-req", a.t("int.srv.reqPinnablePin"), pal) {
				if err := a.backend.RequestPinnablePin(m.siteID); err != nil {
					m.err = err.Error()
				} else {
					m.err = ""
					m.msg = a.t("int.srv.reqPinnableDone")
				}
			}
		})
	})
	macInsetCard(c, pal, func() {
		ui.Text(c, a.t("int.srv.dwebTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.srv.domain"), "", pal, func() {
					ui.TextInput(c.Key("srv-dweb-dom"), &m.dwebDomain).Placeholder("example.eth").AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, "API Key", "", pal, func() {
					ui.TextInput(c.Key("srv-dweb-tok"), &m.dwebToken).Password().AlignSelf(ui.Stretch)
				})
			})
		})
		if macSecondaryButton(c, "srv-dweb-save", a.t("int.srv.saveDweb"), pal) {
			if err := a.backend.ConfigureService(m.siteID, "dwebservices", strings.TrimSpace(m.dwebDomain), strings.TrimSpace(m.dwebToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.srv.savedDweb")
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
		ui.Checkbox(c.Key("x-auto"), &m.xAutoSync, a.t("int.x.autoSync"))
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "x-save", a.t("int.x.save"), pal) {
			if err := a.backend.ConfigureXSync(m.siteID, m.xAPIKey, m.xAPISecret, m.xAccessToken, m.xAccessSecret, m.xAutoSync); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.x.saved")
			}
		}
		if macSecondaryButton(c, "x-disable", a.t("int.x.disable"), pal) {
			if err := a.backend.DisableXSync(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.x.disabled")
			}
		}
	})
}

func (a *App) viewIntTelegram(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		macField(c, a.t("int.tg.chatIDField"), "", pal, func() {
			ui.TextInput(c.Key("tg-chat"), &m.tgChatID).Placeholder(a.t("int.tg.chatIDPlaceholder")).AlignSelf(ui.Stretch)
		})
		macField(c, "Bot Token", "", pal, func() { ui.TextInput(c.Key("tg-token"), &m.tgToken).Password().AlignSelf(ui.Stretch) })
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "tg-save", a.t("int.tg.save"), pal) {
			if err := a.backend.ConfigureTelegram(m.siteID, strings.TrimSpace(m.tgChatID), strings.TrimSpace(m.tgToken)); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.tg.saved")
			}
		}
		if macSecondaryButton(c, "tg-retry", a.t("int.tg.retry"), pal) {
			if n, err := a.backend.RetryTelegram(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.tf("int.tg.retryDone", n)
			}
		}
		if macSecondaryButton(c, "tg-disable", a.t("int.tg.disable"), pal) {
			if err := a.backend.DisableTelegram(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.tg.disabled")
			}
		}
	})
}

func (a *App) viewIntAggregate(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		macField(c, a.t("int.agg.sourcesField"), a.t("int.agg.sourcesHint"), pal, func() {
			ui.TextArea(c.Key("agg-sources"), &m.aggSources).Lines(6, 10).Label(a.t("int.agg.sourcesField")).AlignSelf(ui.Stretch)
		})
	})
	ui.Row(c).Gap(8).Children(func() {
		if macPrimaryButtonSmall(c, "agg-save", a.t("int.agg.save"), pal) {
			if _, err := a.backend.UpdateSiteAggregation(m.siteID, m.aggSources); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.t("int.agg.saved")
				a.Reload()
			}
		}
		if macSecondaryButton(c, "agg-sync", a.t("int.agg.sync"), pal) {
			if n, err := a.backend.SyncSiteAggregation(m.siteID); err != nil {
				m.err = err.Error()
			} else {
				m.err = ""
				m.msg = a.tf("int.agg.syncDone", n)
				a.Reload()
			}
		}
	})
}

func (a *App) viewIntWeb3(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macInsetCard(c, pal, func() {
		ui.Checkbox(c.Key("w3-donate"), &m.web3Donate, a.t("int.web3.donateCheck"))
		ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.web3.ethField"), "", pal, func() {
					ui.TextInput(c.Key("w3-eth"), &m.web3ETH).Placeholder(a.t("int.web3.ethPlaceholder")).AlignSelf(ui.Stretch)
				})
			})
			ui.Box(c).Grow(1).MinWidth(0).Children(func() {
				macField(c, a.t("int.web3.donateMsgField"), "", pal, func() { ui.TextInput(c.Key("w3-msg"), &m.web3Message).AlignSelf(ui.Stretch) })
			})
		})
	})
	macInsetCard(c, pal, func() {
		ui.Row(c).Gap(16).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.Checkbox(c.Key("w3-jb"), &m.web3Juicebox, a.t("int.web3.juiceboxCheck"))
			ui.Box(c).Width(160).Children(func() {
				ui.TextInput(c.Key("w3-jbid"), &m.web3JuiceboxID).Placeholder("Juicebox ID").AlignSelf(ui.Stretch)
			})
		})
	})
	macInsetCard(c, pal, func() {
		macField(c, a.t("int.web3.farcasterField"), "", pal, func() {
			ui.TextArea(c.Key("w3-farcaster"), &m.web3Farcaster).Placeholder(`{"accountAssociation":{...}}`).Lines(3, 6).AlignSelf(ui.Stretch)
		})
	})
	if macPrimaryButtonSmall(c, "w3-save", a.t("int.web3.save"), pal) {
		jbID := 0
		if s := strings.TrimSpace(m.web3JuiceboxID); s != "" {
			parsed, err := strconv.Atoi(s)
			if err != nil {
				m.err = a.t("int.web3.jbIntErr")
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
			m.msg = a.t("int.web3.saved")
			a.Reload()
		}
	}
}
