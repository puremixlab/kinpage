package nativeui

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

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
