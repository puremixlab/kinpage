package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

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
