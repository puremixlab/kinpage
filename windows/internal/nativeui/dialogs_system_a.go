package nativeui

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"windows/internal/ipfs"
	"windows/internal/offline"
	"windows/internal/services"
	"windows/internal/store"
)

type ipfsModalState struct {
	open     bool
	status   ipfs.KuboStatus
	repoSize int64
	err      string
	msg      string
}

type offlineModalState struct {
	open  bool
	usage offline.Usage
	err   string
	msg   string
}

type foldersModalState struct {
	open        bool
	folders     []store.PublishedFolder
	newName     string
	autoPublish bool
	err         string
}

type removedSitesState struct {
	open  bool
	sites []store.Site
	err   string
}

func (a *App) openTipModal(recipient string) {
	a.tipRecipient = strings.TrimSpace(recipient)
	if a.tipAmount == "" {
		a.tipAmount = "0.005"
	}
	a.tipErr = ""
	a.tipModalOpen = true
}

func (a *App) viewTipCreatorSheet(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.tipModalOpen, macSheetProps{
		Title:        a.t("dialog.tip.title"),
		Width:        460,
		PrimaryText:  a.t("dialog.tip.primary"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("dialog.tip.recipientField"), "", pal, func() {
				ui.TextInput(c.Key("tip-recipient"), &a.tipRecipient).Placeholder(a.t("dialog.tip.recipientPlaceholder")).Label(a.t("dialog.tip.recipientLabel")).AlignSelf(ui.Stretch)
			})
			macField(c, a.t("dialog.tip.amountField"), "", pal, func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0).Children(func() {
						ui.TextInput(c.Key("tip-amount"), &a.tipAmount).Placeholder("0.005").Label(a.t("dialog.tip.amountLabel")).AlignSelf(ui.Stretch)
					})
					for _, preset := range []string{"0.001", "0.005", "0.01"} {
						p := preset
						if macSecondaryButton(c, "tip-preset-"+p, p+" ETH", pal) {
							a.tipAmount = p
						}
					}
				})
			})
		})
		if uri, err := services.BuildTipPaymentURI(a.tipRecipient, a.tipAmount); err == nil {
			ui.Text(c, "EIP-681: "+uri).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).MaxLines(2).Ellipsis("…")
		}
		if a.tipErr != "" {
			ui.Text(c, a.tipErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		uri, err := services.BuildTipPaymentURI(a.tipRecipient, a.tipAmount)
		if err != nil {
			a.tipErr = err.Error()
			return
		}
		if a.openExternal != nil {
			if err := a.openExternal(uri); err != nil {
				a.tipErr = a.tf("status.tipOpenErr", err.Error())
				return
			}
		}
		a.tipModalOpen = false
		a.setStatus(a.tf("status.tipOpened", a.tipRecipient))
	}
}

func (a *App) openIPFSResourceModal() {
	a.openIPFSResInput = ""
	a.openIPFSResErr = ""
	a.openIPFSResOpen = true
}

func (a *App) viewOpenIPFSResourceSheet(c *ui.Context) {
	pal := a.pal
	if macSheet(c, &a.openIPFSResOpen, macSheetProps{
		Title:        a.t("ipfs.openResourceTitle"),
		Width:        480,
		PrimaryText:  a.t("ipfs.openResourcePrimary"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, a.t("ipfs.openResourceField"), "", pal, func() {
				ui.TextInput(c.Key("open-ipfs-res-input"), &a.openIPFSResInput).
					Placeholder(a.t("ipfs.openResourceHint")).
					Label(a.t("ipfs.openResourceLabel")).AutoFocus().AlignSelf(ui.Stretch)
			})
		})
		if a.openIPFSResErr != "" {
			ui.Text(c, a.openIPFSResErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		if a.backend == nil {
			return
		}
		targetURL, err := a.backend.ResolveIPFSResourceURL(a.openIPFSResInput)
		if err != nil {
			a.openIPFSResErr = err.Error()
			return
		}
		if a.openExternal != nil {
			if err := a.openExternal(targetURL); err != nil {
				a.openIPFSResErr = a.tf("ipfs.openGatewayErr", err.Error())
				return
			}
		}
		a.openIPFSResOpen = false
		a.setStatus(a.tf("ipfs.openedGatewayStatus", targetURL))
	}
}

func (a *App) openIPFSModal() {
	a.ipfsModal.open = true
	a.refreshIPFSModal()
}

func (a *App) refreshIPFSModal() {
	st, err := a.backend.IPFSStatus()
	if err != nil {
		a.ipfsModal.err = err.Error()
	} else {
		a.ipfsModal.status = st
		a.ipfsModal.err = ""
	}
	if size, err := a.backend.IPFSRepoSize(); err == nil {
		a.ipfsModal.repoSize = size
	}
}

func (a *App) viewIPFSPopoverContent(c *ui.Context) {
	pal := a.pal
	m := &a.ipfsModal
	ui.Column(c).Width(320).Padding(4).Gap(10).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.t("ipfs.statusTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
			if toolbarIconBtn(c, "ipfs-close", lucide.X(), a.t("common.close")).Clicked() {
				m.open = false
			}
		})

		stateText := a.t("ipfs.stateNotRunning")
		if m.status.Disabled {
			stateText = a.t("ipfs.stateDisabled")
		} else if m.status.Running && m.status.Online {
			stateText = a.t("ipfs.stateOnline")
		} else if m.status.Running {
			stateText = a.t("ipfs.stateLocal")
		}

		ui.Column(c).Padding(10).Gap(5).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.tf("ipfs.statusLine", stateText, m.status.Peers)).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
			if m.status.Version != "" {
				ui.Text(c, a.tf("ipfs.kuboVersion", m.status.Version)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
			}
			if m.status.Gateway != "" {
				ui.Text(c, a.tf("ipfs.localGateway", m.status.Gateway)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
			}
			ui.Text(c, a.tf("ipfs.repoSizeLine", formatBytes(m.repoSize), m.status.RateIn, m.status.RateOut)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
		})

		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(11.5).TextColor(pal.danger)
		} else if m.msg != "" {
			ui.Text(c, m.msg).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
		}

		ui.Row(c).Gap(6).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "ipfs-gc", a.t("ipfs.gcBtn"), pal) {
				if err := a.backend.IPFSGarbageCollect(); err != nil {
					m.err = err.Error()
				} else {
					m.msg = a.t("ipfs.gcDone")
					a.refreshIPFSModal()
				}
			}
			if macSecondaryButton(c, "ipfs-refresh", a.t("common.refresh"), pal) {
				a.refreshIPFSModal()
			}
			isRunning := !m.status.Disabled && m.status.Running
			toggleLabel := a.t("ipfs.startNode")
			if isRunning {
				toggleLabel = a.t("ipfs.stopNode")
			}
			if macPrimaryButtonSmall(c, "ipfs-toggle", toggleLabel, pal) {
				if err := a.backend.SetIPFSEnabled(!isRunning); err != nil {
					m.err = err.Error()
				} else {
					m.msg = a.t("ipfs.stateUpdated")
					a.refreshIPFSModal()
				}
			}
		})
	})
}

func (a *App) openOfflineModal() {
	a.offlineModal = offlineModalState{open: true}
	usage, err := a.backend.OfflineMediaUsage()
	if err != nil {
		a.offlineModal.err = err.Error()
	} else {
		a.offlineModal.usage = usage
	}
}

func (a *App) viewOfflineModal(c *ui.Context) {
	pal := a.pal
	m := &a.offlineModal
	macSheet(c, &m.open, macSheetProps{
		Title:        a.t("dialog.offline.title"),
		Width:        440,
		HideFooter:   true,
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Column(c).Padding(12).Gap(6).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.tf("dialog.offline.managedFiles", m.usage.ManagedFiles, formatBytes(m.usage.TotalBytes))).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
			ui.Text(c, a.tf("dialog.offline.reclaimableSpace", m.usage.ReclaimableFiles, formatBytes(m.usage.ReclaimableBytes))).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		} else if m.msg != "" {
			ui.Text(c, m.msg).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		}
		ui.Row(c).Gap(8).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "offline-clean", a.t("dialog.offline.cleanBtn"), pal) {
				usage, err := a.backend.CleanupOfflineMedia()
				if err != nil {
					m.err = err.Error()
				} else {
					m.usage = usage
					m.msg = a.tf("dialog.offline.reclaimedMsg", formatBytes(usage.ReclaimedBytes))
				}
			}
			if macPrimaryButtonSmall(c, "offline-close", a.t("common.close"), pal) {
				m.open = false
			}
		})
	})
}

func (a *App) openFoldersModal() {
	a.foldersModal.open = true
	a.foldersModal.err = ""
	if list, err := a.backend.PublishedFolders(); err != nil {
		a.foldersModal.err = err.Error()
	} else {
		a.foldersModal.folders = list
	}
}

func (a *App) viewFoldersModal(c *ui.Context) {
	pal := a.pal
	m := &a.foldersModal
	macSheet(c, &m.open, macSheetProps{
		Title:        a.t("dialog.folders.title"),
		Width:        520,
		CancelText:   a.t("common.close"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.TextInput(c.Key("folder-new-name"), &m.newName).Placeholder(a.t("dialog.folders.namePlaceholder")).Label(a.t("dialog.folders.nameLabel")).Grow(1)
			ui.Checkbox(c.Key("folder-auto"), &m.autoPublish, a.t("dialog.folders.autoPublish"))
			if macPrimaryButtonSmall(c, "folder-add", a.t("dialog.folders.addBtn"), pal) {
				if _, err := a.backend.AddPublishedFolder(context.Background(), strings.TrimSpace(m.newName), m.autoPublish); err != nil {
					m.err = err.Error()
				} else {
					m.newName = ""
					a.openFoldersModal()
				}
			}
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(220).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				if len(m.folders) == 0 {
					ui.Text(c, a.t("dialog.folders.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for _, f := range m.folders {
					fid := f.ID
					ui.Row(c.Key("folder-row-"+fid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, a.fallbackTitle(f.Name)).Font(macPlanetFont).FontWeight(600).FontSize(12).TextColor(pal.ink)
							ui.Text(c, f.Path).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
						})
						if macSecondaryButton(c, "folder-pub-"+fid, a.t("common.publish"), pal) {
							if _, err := a.backend.PublishFolder(fid); err != nil {
								m.err = err.Error()
							} else {
								a.openFoldersModal()
							}
						}
						if macSecondaryButton(c, "folder-rm-"+fid, a.t("common.remove"), pal) {
							if err := a.backend.RemovePublishedFolder(fid); err != nil {
								m.err = err.Error()
							} else {
								a.openFoldersModal()
							}
						}
					})
				}
			})
		})
	})
}

func (a *App) openRemovedSitesModal() {
	a.removedSites.open = true
	a.removedSites.err = ""
	if list, err := a.backend.RemovedSites(); err != nil {
		a.removedSites.err = err.Error()
	} else {
		a.removedSites.sites = list
	}
}

func (a *App) viewRemovedSitesModal(c *ui.Context) {
	pal := a.pal
	m := &a.removedSites
	macSheet(c, &m.open, macSheetProps{
		Title:        a.t("dialog.removedSites.title"),
		Width:        480,
		CancelText:   a.t("common.close"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(220).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				if len(m.sites) == 0 {
					ui.Text(c, a.t("dialog.removedSites.empty")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for _, s := range m.sites {
					sid := s.ID
					ui.Row(c.Key("rm-site-"+sid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, a.fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(13).TextColor(pal.ink).Grow(1)
						if macPrimaryButtonSmall(c, "restore-site-"+sid, a.t("common.restore"), pal) {
							if _, err := a.backend.RestoreSite(sid); err != nil {
								m.err = err.Error()
							} else {
								a.Reload()
								a.openRemovedSitesModal()
								a.setStatus(a.t("status.siteRestored"))
							}
						}
						if macSecondaryButton(c, "purge-site-"+sid, a.t("dialog.removedSites.purgeBtn"), pal) {
							if err := a.backend.PermanentlyDeleteSite(sid); err != nil {
								m.err = err.Error()
							} else {
								a.openRemovedSitesModal()
							}
						}
					})
				}
			})
		})
	})
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.2f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.2f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
