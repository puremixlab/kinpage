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
	deployKindIdx    int
	deployName       string
	deployServer     string
	deployToken      string
	deployAuto       bool
	deployDelete     bool
	deployConfigured map[string]bool
	deployModalOpen  bool

	// External services (Plausible / Filebase / Pinnable / dWebServices).
	plausibleDomain     string
	plausibleToken      string
	plausibleScriptID   string
	plausibleServer     string
	plausibleConfigured bool
	plausibleModalOpen  bool
	filebaseBucket      string
	filebaseToken       string
	filebaseConfigured  bool
	filebaseModalOpen   bool
	pinnableEndpoint    string
	pinnableConfigured  bool
	pinnableModalOpen   bool
	dwebDomain          string
	dwebToken           string
	dwebConfigured      bool
	dwebModalOpen       bool

	// X sync.
	xAPIKey       string
	xAPISecret    string
	xAccessToken  string
	xAccessSecret string
	xAutoSync     bool
	xConfigured   bool
	xModalOpen    bool

	// Telegram.
	tgChatID     string
	tgToken      string
	tgConfigured bool
	tgModalOpen  bool

	// Aggregation.
	aggSources   string
	aggModalOpen bool

	// Web3.
	web3Donate     bool
	web3Message    string
	web3ETH        string
	web3Juicebox   bool
	web3JuiceboxID string
	web3Farcaster  string
	web3ModalOpen  bool
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
		siteID:           site.ID,
		siteName:         a.fallbackTitle(site.Name),
		templateName:     site.TemplateName,
		templateValues:   vals,
		deployConfigured: map[string]bool{},
		aggSources:       strings.Join(site.Aggregation, "\n"),
		web3Donate:       site.AcceptsDonation,
		web3Message:      site.AcceptsDonationMessage,
		web3ETH:          site.AcceptsDonationETHAddress,
		web3Juicebox:     site.JuiceboxEnabled,
		web3JuiceboxID:   strconv.Itoa(site.JuiceboxProjectID),
		web3Farcaster:    site.FarcasterManifestJSON,
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

var deployLabels = []string{"Cloudflare", "Rsync", "dStack"}

func (a *App) viewIntDeploy(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	macSegmented(c, "dep-kind", &m.deployKindIdx, deployLabels, pal)
	kind := deployKinds[m.deployKindIdx]
	label := deployLabels[m.deployKindIdx]
	configured := m.deployConfigured != nil && m.deployConfigured[kind]

	if !configured {
		if settingsEmptyState(c, "dep-empty-add-"+kind, lucide.Upload(), a.tf("int.deploy.emptyDesc", label), a.t("common.add"), pal) {
			m.err = ""
			m.deployModalOpen = true
		}
		return
	}

	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
					title := label
					if strings.TrimSpace(m.deployName) != "" {
						title += " · " + strings.TrimSpace(m.deployName)
					}
					ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
				})
				if strings.TrimSpace(m.deployServer) != "" {
					ui.Text(c, m.deployServer).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).MaxLines(1).Ellipsis("…")
				}
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				if macSecondaryButton(c, "dep-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.deployModalOpen = true
				}
				if macPrimaryButtonSmall(c, "dep-run", a.t("int.deploy.run"), pal) {
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
						if m.deployConfigured != nil {
							delete(m.deployConfigured, kind)
						}
						m.err = ""
						m.msg = a.tf("int.deploy.disabled", kind)
					}
				}
			})
		})
	})
}

func (a *App) viewIntServices(c *ui.Context) {
	pal := a.pal
	m := &a.integrations

	// 1. Plausible Analytics
	if !m.plausibleConfigured {
		if settingsEmptyState(c, "srv-pl-empty-add", lucide.Activity(), a.t("int.srv.plausibleEmpty"), a.t("common.add"), pal) {
			m.err = ""
			m.plausibleModalOpen = true
		}
	} else {
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
						ui.Text(c, a.t("int.srv.plausibleTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
					})
					if m.plausibleDomain != "" {
						ui.Text(c, m.plausibleDomain).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
					}
				})
				if macSecondaryButton(c, "srv-pl-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.plausibleModalOpen = true
				}
			})
		})
	}

	// 2. Filebase Pinning
	if !m.filebaseConfigured {
		if settingsEmptyState(c, "srv-fb-empty-add", lucide.HardDrive(), a.t("int.srv.filebaseEmpty"), a.t("common.add"), pal) {
			m.err = ""
			m.filebaseModalOpen = true
		}
	} else {
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
						ui.Text(c, "Filebase").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
					})
					if m.filebaseBucket != "" {
						ui.Text(c, m.filebaseBucket).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
					}
				})
				ui.Row(c).Gap(6).Shrink(0).Children(func() {
					if macSecondaryButton(c, "srv-fb-pin", a.t("int.srv.reqFilebasePin"), pal) {
						if err := a.backend.RequestFilebasePin(m.siteID); err != nil {
							m.err = err.Error()
						} else {
							m.err = ""
							m.msg = a.t("int.srv.reqFilebaseDone")
						}
					}
					if macSecondaryButton(c, "srv-fb-edit", a.t("common.edit"), pal) {
						m.err = ""
						m.filebaseModalOpen = true
					}
				})
			})
		})
	}

	// 3. Pinnable Pinning
	if !m.pinnableConfigured {
		if settingsEmptyState(c, "srv-pin-empty-add", lucide.Pin(), a.t("int.srv.pinnableEmpty"), a.t("common.add"), pal) {
			m.err = ""
			m.pinnableModalOpen = true
		}
	} else {
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
						ui.Text(c, "Pinnable").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
					})
					if m.pinnableEndpoint != "" {
						ui.Text(c, m.pinnableEndpoint).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).MaxLines(1).Ellipsis("…")
					}
				})
				ui.Row(c).Gap(6).Shrink(0).Children(func() {
					if macSecondaryButton(c, "srv-pin-req", a.t("int.srv.reqPinnablePin"), pal) {
						if err := a.backend.RequestPinnablePin(m.siteID); err != nil {
							m.err = err.Error()
						} else {
							m.err = ""
							m.msg = a.t("int.srv.reqPinnableDone")
						}
					}
					if macSecondaryButton(c, "srv-pin-edit", a.t("common.edit"), pal) {
						m.err = ""
						m.pinnableModalOpen = true
					}
				})
			})
		})
	}

	// 4. dWebServices.xyz
	if !m.dwebConfigured {
		if settingsEmptyState(c, "srv-dweb-empty-add", lucide.Globe(), a.t("int.srv.dwebEmpty"), a.t("common.add"), pal) {
			m.err = ""
			m.dwebModalOpen = true
		}
	} else {
		macInsetCard(c, pal, func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
						ui.Text(c, a.t("int.srv.dwebTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
					})
					if m.dwebDomain != "" {
						ui.Text(c, m.dwebDomain).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
					}
				})
				if macSecondaryButton(c, "srv-dweb-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.dwebModalOpen = true
				}
			})
		})
	}
}

func (a *App) viewIntX(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	if !m.xConfigured {
		if settingsEmptyState(c, "x-empty-add", lucide.Share2(), a.t("int.x.emptyDesc"), a.t("common.add"), pal) {
			m.err = ""
			m.xModalOpen = true
		}
		return
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
					ui.Text(c, a.t("int.x.configuredSummary")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
				})
				mode := a.t("int.x.autoSyncOff")
				if m.xAutoSync {
					mode = a.t("int.x.autoSyncOn")
				}
				ui.Text(c, mode).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				if macSecondaryButton(c, "x-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.xModalOpen = true
				}
				if macSecondaryButton(c, "x-disable", a.t("int.x.disable"), pal) {
					if err := a.backend.DisableXSync(m.siteID); err != nil {
						m.err = err.Error()
					} else {
						m.xConfigured = false
						m.err = ""
						m.msg = a.t("int.x.disabled")
					}
				}
			})
		})
	})
}

func (a *App) viewIntTelegram(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	if !m.tgConfigured {
		if settingsEmptyState(c, "tg-empty-add", lucide.Send(), a.t("int.tg.emptyDesc"), a.t("common.add"), pal) {
			m.err = ""
			m.tgModalOpen = true
		}
		return
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(3).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Box(c).Size(7, 7).Radius(4).Background(pal.green).Shrink(0)
					ui.Text(c, "Telegram").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
				})
				if strings.TrimSpace(m.tgChatID) != "" {
					ui.Text(c, m.tgChatID).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
				}
			})
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				if macSecondaryButton(c, "tg-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.tgModalOpen = true
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
						m.tgConfigured = false
						m.err = ""
						m.msg = a.t("int.tg.disabled")
					}
				}
			})
		})
	})
}

func (a *App) viewIntAggregate(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	var lines []string
	for _, line := range strings.Split(m.aggSources, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			lines = append(lines, s)
		}
	}
	if len(lines) == 0 {
		if settingsEmptyState(c, "agg-empty-add", lucide.Rss(), a.t("int.agg.emptyDesc"), a.t("common.add"), pal) {
			m.err = ""
			m.aggModalOpen = true
		}
		return
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.tf("int.agg.sourceCount", len(lines))).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
			ui.Row(c).Gap(6).Shrink(0).Children(func() {
				if macSecondaryButton(c, "agg-edit", a.t("common.edit"), pal) {
					m.err = ""
					m.aggModalOpen = true
				}
				if macPrimaryButtonSmall(c, "agg-sync", a.t("int.agg.sync"), pal) {
					if n, err := a.backend.SyncSiteAggregation(m.siteID); err != nil {
						m.err = err.Error()
					} else {
						m.err = ""
						m.msg = a.tf("int.agg.syncDone", n)
						a.Reload()
					}
				}
			})
		})
		prefsHairlineH(c, pal)
		for _, src := range lines {
			ui.Text(c, src).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsSecondary).MaxLines(1).Ellipsis("…")
		}
	})
}

func (a *App) viewIntWeb3(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	configured := m.web3Donate || m.web3Juicebox || strings.TrimSpace(m.web3Farcaster) != ""
	if !configured {
		if settingsEmptyState(c, "w3-empty-add", lucide.Wallet(), a.t("int.web3.emptyDesc"), a.t("common.configure"), pal) {
			m.err = ""
			m.web3ModalOpen = true
		}
		return
	}
	macInsetCard(c, pal, func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				if m.web3Donate {
					line := a.t("int.web3.donateEnabled")
					if strings.TrimSpace(m.web3ETH) != "" {
						line += " · " + strings.TrimSpace(m.web3ETH)
					}
					ui.Text(c, line).Font(macPlanetFont).FontSize(13).FontWeight(500).TextColor(pal.prefsInk)
				}
				if m.web3Juicebox {
					ui.Text(c, a.tf("int.web3.jbEnabled", strings.TrimSpace(m.web3JuiceboxID))).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsSecondary)
				}
				if strings.TrimSpace(m.web3Farcaster) != "" {
					ui.Text(c, a.t("int.web3.farcasterEnabled")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsSecondary)
				}
			})
			if macSecondaryButton(c, "w3-edit", a.t("common.edit"), pal) {
				m.err = ""
				m.web3ModalOpen = true
			}
		})
	})
}

func (a *App) viewIntegrationsSecondarySheets(c *ui.Context) {
	pal := a.pal
	m := &a.integrations
	a.viewDeployConfigSheet(c, pal, m)
	a.viewPlausibleConfigSheet(c, pal, m)
	a.viewFilebaseConfigSheet(c, pal, m)
	a.viewPinnableConfigSheet(c, pal, m)
	a.viewDWebServicesConfigSheet(c, pal, m)
	a.viewXSyncConfigSheet(c, pal, m)
	a.viewTelegramConfigSheet(c, pal, m)
	a.viewAggregationConfigSheet(c, pal, m)
	a.viewWeb3ConfigSheet(c, pal, m)
}

func (a *App) viewDeployConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	kind := deployKinds[m.deployKindIdx]
	label := deployLabels[m.deployKindIdx]
	if macSheet(c, &m.deployModalOpen, macSheetProps{
		Title:        a.tf("int.deploy.modalTitle", label),
		Width:        500,
		PrimaryText:  a.t("int.deploy.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
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
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureDeployment(m.siteID, kind, strings.TrimSpace(m.deployName), strings.TrimSpace(m.deployServer), strings.TrimSpace(m.deployToken), m.deployAuto, m.deployDelete); err != nil {
			m.err = err.Error()
		} else {
			if m.deployConfigured == nil {
				m.deployConfigured = map[string]bool{}
			}
			m.deployConfigured[kind] = true
			m.err = ""
			m.msg = a.tf("int.deploy.saved", kind)
			m.deployModalOpen = false
		}
	}
}

func (a *App) viewPlausibleConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.plausibleModalOpen, macSheetProps{
		Title:        a.t("int.srv.plausibleModal"),
		Width:        480,
		PrimaryText:  a.t("int.srv.savePlausible"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, a.t("int.srv.domain"), "", pal, func() { ui.TextInput(c.Key("srv-pl-dom"), &m.plausibleDomain).AlignSelf(ui.Stretch) })
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "Script ID", "", pal, func() { ui.TextInput(c.Key("srv-pl-sid"), &m.plausibleScriptID).AlignSelf(ui.Stretch) })
				})
			})
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "API Key", "", pal, func() { ui.TextInput(c.Key("srv-pl-tok"), &m.plausibleToken).Password().AlignSelf(ui.Stretch) })
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, a.t("int.srv.server"), "", pal, func() {
						ui.TextInput(c.Key("srv-pl-srv"), &m.plausibleServer).Placeholder("https://plausible.io").AlignSelf(ui.Stretch)
					})
				})
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigurePlausibleAt(m.siteID, strings.TrimSpace(m.plausibleDomain), strings.TrimSpace(m.plausibleToken), strings.TrimSpace(m.plausibleScriptID), strings.TrimSpace(m.plausibleServer)); err != nil {
			m.err = err.Error()
		} else {
			m.plausibleConfigured = true
			m.err = ""
			m.msg = a.t("int.srv.savedPlausible")
			m.plausibleModalOpen = false
		}
	}
}

func (a *App) viewFilebaseConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.filebaseModalOpen, macSheetProps{
		Title:        a.t("int.srv.filebaseModal"),
		Width:        460,
		PrimaryText:  a.t("int.srv.saveFilebase"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("int.srv.filebasePinName"), "", pal, func() { ui.TextInput(c.Key("srv-fb-name"), &m.filebaseBucket).AlignSelf(ui.Stretch) })
			macField(c, "Filebase Token", "", pal, func() { ui.TextInput(c.Key("srv-fb-tok"), &m.filebaseToken).Password().AlignSelf(ui.Stretch) })
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureService(m.siteID, "filebase", strings.TrimSpace(m.filebaseBucket), strings.TrimSpace(m.filebaseToken)); err != nil {
			m.err = err.Error()
		} else {
			m.filebaseConfigured = true
			m.err = ""
			m.msg = a.t("int.srv.savedFilebase")
			m.filebaseModalOpen = false
		}
	}
}

func (a *App) viewPinnableConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.pinnableModalOpen, macSheetProps{
		Title:        a.t("int.srv.pinnableModal"),
		Width:        460,
		PrimaryText:  a.t("int.srv.savePinnable"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, "Pinnable Endpoint", "", pal, func() { ui.TextInput(c.Key("srv-pin-ep"), &m.pinnableEndpoint).AlignSelf(ui.Stretch) })
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigurePinnable(m.siteID, strings.TrimSpace(m.pinnableEndpoint)); err != nil {
			m.err = err.Error()
		} else {
			m.pinnableConfigured = true
			m.err = ""
			m.msg = a.t("int.srv.savedPinnable")
			m.pinnableModalOpen = false
		}
	}
}

func (a *App) viewDWebServicesConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.dwebModalOpen, macSheetProps{
		Title:        a.t("int.srv.dwebModal"),
		Width:        460,
		PrimaryText:  a.t("int.srv.saveDweb"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("int.srv.domain"), "", pal, func() {
				ui.TextInput(c.Key("srv-dweb-dom"), &m.dwebDomain).Placeholder("example.eth").AlignSelf(ui.Stretch)
			})
			macField(c, "API Key", "", pal, func() {
				ui.TextInput(c.Key("srv-dweb-tok"), &m.dwebToken).Password().AlignSelf(ui.Stretch)
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureService(m.siteID, "dwebservices", strings.TrimSpace(m.dwebDomain), strings.TrimSpace(m.dwebToken)); err != nil {
			m.err = err.Error()
		} else {
			m.dwebConfigured = true
			m.err = ""
			m.msg = a.t("int.srv.savedDweb")
			m.dwebModalOpen = false
		}
	}
}

func (a *App) viewXSyncConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.xModalOpen, macSheetProps{
		Title:        a.t("int.x.modalTitle"),
		Width:        500,
		PrimaryText:  a.t("int.x.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "API Key", "", pal, func() { ui.TextInput(c.Key("x-ak"), &m.xAPIKey).Password().AlignSelf(ui.Stretch) })
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "API Secret", "", pal, func() { ui.TextInput(c.Key("x-as"), &m.xAPISecret).Password().AlignSelf(ui.Stretch) })
				})
			})
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "Access Token", "", pal, func() { ui.TextInput(c.Key("x-at"), &m.xAccessToken).Password().AlignSelf(ui.Stretch) })
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, "Access Secret", "", pal, func() { ui.TextInput(c.Key("x-ats"), &m.xAccessSecret).Password().AlignSelf(ui.Stretch) })
				})
			})
			ui.Checkbox(c.Key("x-auto"), &m.xAutoSync, a.t("int.x.autoSync"))
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureXSync(m.siteID, m.xAPIKey, m.xAPISecret, m.xAccessToken, m.xAccessSecret, m.xAutoSync); err != nil {
			m.err = err.Error()
		} else {
			m.xConfigured = true
			m.err = ""
			m.msg = a.t("int.x.saved")
			m.xModalOpen = false
		}
	}
}

func (a *App) viewTelegramConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.tgModalOpen, macSheetProps{
		Title:        a.t("int.tg.modalTitle"),
		Width:        460,
		PrimaryText:  a.t("int.tg.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("int.tg.chatIDField"), "", pal, func() {
				ui.TextInput(c.Key("tg-chat"), &m.tgChatID).Placeholder(a.t("int.tg.chatIDPlaceholder")).AlignSelf(ui.Stretch)
			})
			macField(c, "Bot Token", "", pal, func() { ui.TextInput(c.Key("tg-token"), &m.tgToken).Password().AlignSelf(ui.Stretch) })
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureTelegram(m.siteID, strings.TrimSpace(m.tgChatID), strings.TrimSpace(m.tgToken)); err != nil {
			m.err = err.Error()
		} else {
			m.tgConfigured = true
			m.err = ""
			m.msg = a.t("int.tg.saved")
			m.tgModalOpen = false
		}
	}
}

func (a *App) viewAggregationConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.aggModalOpen, macSheetProps{
		Title:        a.t("int.agg.modalTitle"),
		Width:        500,
		PrimaryText:  a.t("int.agg.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("int.agg.sourcesField"), a.t("int.agg.sourcesHint"), pal, func() {
				ui.TextArea(c.Key("agg-sources"), &m.aggSources).Lines(6, 10).Label(a.t("int.agg.sourcesField")).AlignSelf(ui.Stretch)
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if _, err := a.backend.UpdateSiteAggregation(m.siteID, m.aggSources); err != nil {
			m.err = err.Error()
		} else {
			m.err = ""
			m.msg = a.t("int.agg.saved")
			m.aggModalOpen = false
			a.Reload()
		}
	}
}

func (a *App) viewWeb3ConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if macSheet(c, &m.web3ModalOpen, macSheetProps{
		Title:        a.t("int.web3.modalTitle"),
		Width:        520,
		PrimaryText:  a.t("int.web3.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			ui.Checkbox(c.Key("w3-donate"), &m.web3Donate, a.t("int.web3.donateCheck"))
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
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
			ui.Row(c).Gap(12).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.Checkbox(c.Key("w3-jb"), &m.web3Juicebox, a.t("int.web3.juiceboxCheck"))
				ui.Box(c).Width(160).Children(func() {
					ui.TextInput(c.Key("w3-jbid"), &m.web3JuiceboxID).Placeholder("Juicebox ID").AlignSelf(ui.Stretch)
				})
			})
		})
		macInsetCard(c, pal, func() {
			macField(c, a.t("int.web3.farcasterField"), "", pal, func() {
				ui.TextArea(c.Key("w3-farcaster"), &m.web3Farcaster).Placeholder(`{"accountAssociation":{...}}`).Lines(3, 5).AlignSelf(ui.Stretch)
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
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
			m.web3ModalOpen = false
			a.Reload()
		}
	}
}
