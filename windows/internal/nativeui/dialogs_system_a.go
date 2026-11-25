package nativeui

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/ipfs"
	"windows/internal/offline"
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

func (a *App) viewIPFSModal(c *ui.Context) {
	t := c.Theme()
	m := &a.ipfsModal
	ui.Modal(c, &m.open, func() {
		ui.Column(c).Width(480).Gap(12).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "IPFS 节点状态").Bold().FontSize(16)
				toggleLabel := "停用节点"
				if m.status.Disabled {
					toggleLabel = "启用节点"
				}
				if ui.Button(c.Key("ipfs-toggle"), toggleLabel).Clicked() {
					if err := a.backend.SetIPFSEnabled(m.status.Disabled); err != nil {
						m.err = err.Error()
					} else {
						m.msg = "节点状态已更新"
						a.refreshIPFSModal()
					}
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

			ui.Column(c).Padding(12).Gap(6).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, fmt.Sprintf("状态：%s  ·  已连接节点：%d", stateText, m.status.Peers)).FontSize(12)
				if m.status.Version != "" {
					ui.Text(c, "Kubo 版本："+m.status.Version).FontSize(12).TextColor(t.TextMuted)
				}
				if m.status.Gateway != "" {
					ui.Text(c, "本地网关："+m.status.Gateway).FontSize(12).TextColor(t.TextMuted)
				}
				ui.Text(c, fmt.Sprintf("仓库大小：%s  ·  速率：↓ %.1f B/s  ↑ %.1f B/s", formatBytes(m.repoSize), m.status.RateIn, m.status.RateOut)).FontSize(12).TextColor(t.TextMuted)
			})

			if m.err != "" {
				ui.Text(c, m.err).FontSize(12).TextColor(t.Danger)
			} else if m.msg != "" {
				ui.Text(c, m.msg).FontSize(12).TextColor(t.TextMuted)
			}

			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("ipfs-gc"), "清理未固定数据").Clicked() {
					if err := a.backend.IPFSGarbageCollect(); err != nil {
						m.err = err.Error()
					} else {
						m.msg = "已完成垃圾回收"
						a.refreshIPFSModal()
					}
				}
				if ui.Button(c.Key("ipfs-refresh"), "刷新").Clicked() {
					a.refreshIPFSModal()
				}
				if ui.PrimaryButton(c.Key("ipfs-close"), "关闭").Clicked() {
					m.open = false
				}
			})
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
	t := c.Theme()
	m := &a.offlineModal
	ui.Modal(c, &m.open, func() {
		ui.Column(c).Width(420).Gap(12).Children(func() {
			ui.Text(c, "离线存储管理").Bold().FontSize(16)
			ui.Column(c).Padding(12).Gap(6).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, fmt.Sprintf("已管理文件：%d 个（%s）", m.usage.ManagedFiles, formatBytes(m.usage.TotalBytes))).FontSize(12)
				ui.Text(c, fmt.Sprintf("可回收空间：%d 个（%s）", m.usage.ReclaimableFiles, formatBytes(m.usage.ReclaimableBytes))).FontSize(12).TextColor(t.TextMuted)
			})
			if m.err != "" {
				ui.Text(c, m.err).FontSize(12).TextColor(t.Danger)
			} else if m.msg != "" {
				ui.Text(c, m.msg).FontSize(12).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("offline-clean"), "清理可回收空间").Clicked() {
					usage, err := a.backend.CleanupOfflineMedia()
					if err != nil {
						m.err = err.Error()
					} else {
						m.usage = usage
						m.msg = fmt.Sprintf("已释放 %s", formatBytes(usage.ReclaimedBytes))
					}
				}
				if ui.PrimaryButton(c.Key("offline-close"), "关闭").Clicked() {
					m.open = false
				}
			})
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
	t := c.Theme()
	m := &a.foldersModal
	ui.Modal(c, &m.open, func() {
		ui.Column(c).Width(500).Gap(12).Children(func() {
			ui.Text(c, "已发布文件夹").Bold().FontSize(16)
			ui.Row(c).Gap(8).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
				ui.TextInput(c.Key("folder-new-name"), &m.newName).Placeholder("文件夹名称（可选）").Label("文件夹名称").Grow(1)
				ui.Checkbox(c.Key("folder-auto"), &m.autoPublish, "自动发布")
				if ui.PrimaryButton(c.Key("folder-add"), "+ 添加").Clicked() {
					if _, err := a.backend.AddPublishedFolder(context.Background(), strings.TrimSpace(m.newName), m.autoPublish); err != nil {
						m.err = err.Error()
					} else {
						m.newName = ""
						a.openFoldersModal()
					}
				}
			})
			if m.err != "" {
				ui.Text(c, m.err).FontSize(12).TextColor(t.Danger)
			}
			ui.Scroll(c).Height(220).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					if len(m.folders) == 0 {
						ui.Text(c, "暂无已发布文件夹").FontSize(12).TextColor(t.TextMuted)
					}
					for _, f := range m.folders {
						fid := f.ID
						ui.Row(c.Key("folder-row-"+fid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
							ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, fallbackTitle(f.Name)).Bold().FontSize(12)
								ui.Text(c, f.Path).FontSize(11).TextColor(t.TextMuted)
							})
							if ui.Button(c.Key("folder-pub-"+fid), "发布").Clicked() {
								if _, err := a.backend.PublishFolder(fid); err != nil {
									m.err = err.Error()
								} else {
									a.openFoldersModal()
								}
							}
							if ui.Button(c.Key("folder-rm-"+fid), "移除").Clicked() {
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
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("folder-close"), "关闭").Clicked() {
					m.open = false
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
	t := c.Theme()
	m := &a.removedSites
	ui.Modal(c, &m.open, func() {
		ui.Column(c).Width(460).Gap(12).Children(func() {
			ui.Text(c, "已移除的站点（回收站）").Bold().FontSize(16)
			if m.err != "" {
				ui.Text(c, m.err).FontSize(12).TextColor(t.Danger)
			}
			ui.Scroll(c).Height(220).AlignSelf(ui.Stretch).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					if len(m.sites) == 0 {
						ui.Text(c, "回收站为空").FontSize(12).TextColor(t.TextMuted)
					}
					for _, s := range m.sites {
						sid := s.ID
						ui.Row(c.Key("rm-site-"+sid)).Padding(8).Gap(8).AlignItems(ui.Center).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
							ui.Text(c, fallbackTitle(s.Name)).FontSize(13).Grow(1)
							if ui.PrimaryButton(c.Key("restore-site-"+sid), "恢复").Clicked() {
								if _, err := a.backend.RestoreSite(sid); err != nil {
									m.err = err.Error()
								} else {
									a.Reload()
									a.openRemovedSitesModal()
									a.setStatus("已恢复站点")
								}
							}
							if ui.Button(c.Key("purge-site-"+sid), "永久删除").Clicked() {
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
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.Button(c.Key("rm-sites-close"), "关闭").Clicked() {
					m.open = false
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
