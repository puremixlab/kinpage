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
		Title:       "打赏创作者 (ETH)",
		Width:       460,
		PrimaryText: "在钱包中打开支付链接",
		CancelText:  "取消",
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, "收款地址 (0x… 或 ENS)", "", pal, func() {
				ui.TextInput(c.Key("tip-recipient"), &a.tipRecipient).Placeholder("例如 planetable.eth 或 0x…").Label("收款地址").AlignSelf(ui.Stretch)
			})
			macField(c, "打赏金额 (ETH)", "", pal, func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
					ui.Box(c).Grow(1).MinWidth(0).Children(func() {
						ui.TextInput(c.Key("tip-amount"), &a.tipAmount).Placeholder("0.005").Label("打赏金额").AlignSelf(ui.Stretch)
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
				a.tipErr = "无法唤起钱包应用：" + err.Error()
				return
			}
		}
		a.tipModalOpen = false
		a.setStatus("已唤起钱包打赏链接：" + a.tipRecipient)
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
		Title:       "打开 IPFS 资源",
		Width:       480,
		PrimaryText: "在本地网关打开",
		CancelText:  "取消",
	}, pal, func() {
		macInsetCard(c, pal, func() {
			macField(c, "CID / IPNS / 去中心化域名", "", pal, func() {
				ui.TextInput(c.Key("open-ipfs-res-input"), &a.openIPFSResInput).
					Placeholder("例如 Qm… / bafy… / k51… / planetable.eth").
					Label("IPFS 资源地址").AutoFocus().AlignSelf(ui.Stretch)
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
				a.openIPFSResErr = "打开本地网关失败：" + err.Error()
				return
			}
		}
		a.openIPFSResOpen = false
		a.setStatus("已在本地网关打开：" + targetURL)
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
			ui.Text(c, "IPFS 节点状态").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
			if toolbarIconBtn(c, "ipfs-close", lucide.X(), "关闭").Clicked() {
				m.open = false
			}
		})

		stateText := "未运行"
		if m.status.Disabled {
			stateText = "已停用"
		} else if m.status.Running && m.status.Online {
			stateText = "在线运行中"
		} else if m.status.Running {
			stateText = "本地运行中"
		}

		ui.Column(c).Padding(10).Gap(5).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, fmt.Sprintf("状态：%s  ·  已连接节点：%d", stateText, m.status.Peers)).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
			if m.status.Version != "" {
				ui.Text(c, "Kubo 版本："+m.status.Version).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
			}
			if m.status.Gateway != "" {
				ui.Text(c, "本地网关："+m.status.Gateway).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
			}
			ui.Text(c, fmt.Sprintf("仓库大小：%s  ·  ↓ %.1f / ↑ %.1f B/s", formatBytes(m.repoSize), m.status.RateIn, m.status.RateOut)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
		})

		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(11.5).TextColor(pal.danger)
		} else if m.msg != "" {
			ui.Text(c, m.msg).Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
		}

		ui.Row(c).Gap(6).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "ipfs-gc", "清理未固定数据", pal) {
				if err := a.backend.IPFSGarbageCollect(); err != nil {
					m.err = err.Error()
				} else {
					m.msg = "已完成垃圾回收"
					a.refreshIPFSModal()
				}
			}
			if macSecondaryButton(c, "ipfs-refresh", "刷新", pal) {
				a.refreshIPFSModal()
			}
			isRunning := !m.status.Disabled && m.status.Running
			toggleLabel := "启动节点"
			if isRunning {
				toggleLabel = "停用节点"
			}
			if macPrimaryButtonSmall(c, "ipfs-toggle", toggleLabel, pal) {
				if err := a.backend.SetIPFSEnabled(!isRunning); err != nil {
					m.err = err.Error()
				} else {
					m.msg = "节点状态已更新"
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
		Title:      "离线存储管理",
		Width:      440,
		HideFooter: true,
	}, pal, func() {
		ui.Column(c).Padding(12).Gap(6).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, fmt.Sprintf("已管理文件：%d 个（%s）", m.usage.ManagedFiles, formatBytes(m.usage.TotalBytes))).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
			ui.Text(c, fmt.Sprintf("可回收空间：%d 个（%s）", m.usage.ReclaimableFiles, formatBytes(m.usage.ReclaimableBytes))).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		})
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		} else if m.msg != "" {
			ui.Text(c, m.msg).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		}
		ui.Row(c).Gap(8).Justify(ui.End).AlignSelf(ui.Stretch).Children(func() {
			if macSecondaryButton(c, "offline-clean", "清理可回收空间", pal) {
				usage, err := a.backend.CleanupOfflineMedia()
				if err != nil {
					m.err = err.Error()
				} else {
					m.usage = usage
					m.msg = fmt.Sprintf("已释放 %s", formatBytes(usage.ReclaimedBytes))
				}
			}
			if macPrimaryButtonSmall(c, "offline-close", "关闭", pal) {
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
		Title:      "已发布文件夹",
		Width:      520,
		CancelText: "关闭",
	}, pal, func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
			ui.TextInput(c.Key("folder-new-name"), &m.newName).Placeholder("文件夹名称（可选）").Label("文件夹名称").Grow(1)
			ui.Checkbox(c.Key("folder-auto"), &m.autoPublish, "自动发布")
			if macPrimaryButtonSmall(c, "folder-add", "+ 添加", pal) {
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
					ui.Text(c, "暂无已发布文件夹").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for _, f := range m.folders {
					fid := f.ID
					ui.Row(c.Key("folder-row-"+fid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Text(c, fallbackTitle(f.Name)).Font(macPlanetFont).FontWeight(600).FontSize(12).TextColor(pal.ink)
							ui.Text(c, f.Path).Font(macPlanetFont).FontSize(11).TextColor(pal.muted)
						})
						if macSecondaryButton(c, "folder-pub-"+fid, "发布", pal) {
							if _, err := a.backend.PublishFolder(fid); err != nil {
								m.err = err.Error()
							} else {
								a.openFoldersModal()
							}
						}
						if macSecondaryButton(c, "folder-rm-"+fid, "移除", pal) {
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
		Title:      "已移除的站点（回收站）",
		Width:      480,
		CancelText: "关闭",
	}, pal, func() {
		if m.err != "" {
			ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
		ui.Scroll(c).Height(220).AlignSelf(ui.Stretch).Children(func() {
			ui.Column(c).Gap(6).Children(func() {
				if len(m.sites) == 0 {
					ui.Text(c, "回收站为空").Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				}
				for _, s := range m.sites {
					sid := s.ID
					ui.Row(c.Key("rm-site-"+sid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, fallbackTitle(s.Name)).Font(macPlanetFont).FontSize(13).TextColor(pal.ink).Grow(1)
						if macPrimaryButtonSmall(c, "restore-site-"+sid, "恢复", pal) {
							if _, err := a.backend.RestoreSite(sid); err != nil {
								m.err = err.Error()
							} else {
								a.Reload()
								a.openRemovedSitesModal()
								a.setStatus("已恢复站点")
							}
						}
						if macSecondaryButton(c, "purge-site-"+sid, "永久删除", pal) {
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
