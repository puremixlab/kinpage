package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
)

func (a *App) viewDeployConfigSheet(c *ui.Context, pal macPalette, m *integrationsModalState) {
	if m.deployKindIdx < 0 || m.deployKindIdx >= len(deployKinds) {
		m.deployKindIdx = 0
	}
	kind := deployKinds[m.deployKindIdx]
	label := deployLabels[m.deployKindIdx]
	existing := m.deployInfos[kind]
	isCloudflare := kind == "cloudflare"

	if macSheet(c, &m.deployModalOpen, macSheetProps{
		Title:        a.tf("int.deploy.modalTitle", label),
		Width:        500,
		PrimaryText:  a.t("int.deploy.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			nameField := a.t("int.deploy.nameField")
			serverField := a.t("int.deploy.serverField")
			if isCloudflare {
				nameField = a.t("int.deploy.cfProjectField")
				serverField = a.t("int.deploy.cfAccountField")
			} else {
				nameField = a.t("int.deploy.sshTargetField")
				serverField = a.t("int.deploy.sshKeyField")
			}
			ui.Row(c).Gap(12).AlignSelf(ui.Stretch).Children(func() {
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, nameField, "", pal, func() {
						ui.TextInput(c.Key("dep-name"), &m.deployName).Label(a.t("int.deploy.nameLabel")).AlignSelf(ui.Stretch)
					})
				})
				ui.Box(c).Grow(1).MinWidth(0).Children(func() {
					macField(c, serverField, "", pal, func() {
						ui.TextInput(c.Key("dep-server"), &m.deployServer).Label(a.t("int.deploy.serverLabel")).AlignSelf(ui.Stretch)
					})
				})
			})
			if isCloudflare {
				tokenHint := ""
				if existing.Configured {
					tokenHint = a.t("int.deploy.tokenKeepHint")
				}
				macField(c, a.t("int.deploy.tokenField"), tokenHint, pal, func() {
					ui.TextInput(c.Key("dep-token"), &m.deployToken).Password().Label(a.t("int.deploy.tokenLabel")).AlignSelf(ui.Stretch)
				})
			}
			ui.Row(c).Gap(20).Wrap().Children(func() {
				ui.Checkbox(c.Key("dep-auto"), &m.deployAuto, a.t("int.deploy.auto"))
				if !isCloudflare {
					ui.Checkbox(c.Key("dep-del"), &m.deployDelete, a.t("int.deploy.deleteRemote"))
				}
			})
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if err := a.backend.ConfigureDeployment(m.siteID, kind, strings.TrimSpace(m.deployName), strings.TrimSpace(m.deployServer), strings.TrimSpace(m.deployToken), m.deployAuto, m.deployDelete); err != nil {
			m.err = err.Error()
		} else {
			a.refreshDeploySummary(kind)
			m.err = ""
			m.msg = a.tf("int.deploy.saved", label)
			m.deployModalOpen = false
		}
	}
}
