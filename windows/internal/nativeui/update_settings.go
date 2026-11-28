package nativeui

import (
	"context"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/updates"
)

type updateOperation struct {
	download bool
	silent   bool
	cancel   context.CancelFunc
	done     chan updateOutcome
}

type updateOutcome struct {
	check *updates.CheckInfo
	path  string
	err   error
}

type updateSettingsState struct {
	info       updates.SettingsInfo
	loaded     bool
	configOpen bool
	feedURL    string
	key        string
	auto       bool
	configErr  string
	check      *updates.CheckInfo
	downloaded bool
	opErr      string
	operation  *updateOperation
}

func (a *App) loadUpdateSettings() {
	m := &a.updateSettings
	if m.loaded || a.backend == nil {
		return
	}
	m.loaded = true
	info, err := a.backend.UpdateSettings()
	m.info = info
	if err != nil {
		m.opErr = err.Error()
	}
}

// CheckUpdatesOnStartup honors the user's saved trust configuration and opt-in.
func (a *App) CheckUpdatesOnStartup() {
	a.loadUpdateSettings()
	if a.updateSettings.info.Configured && a.updateSettings.info.Settings.AutoCheck {
		a.startUpdateOperation(false, true)
	}
}

func (a *App) startUpdateOperation(download, silent bool) {
	m := &a.updateSettings
	if a.backend == nil || m.operation != nil || !m.info.Configured || a.taskContext().Err() != nil {
		return
	}
	var selected updates.CheckInfo
	if download {
		if m.check == nil || !m.check.Available {
			return
		}
		selected = *m.check
	} else {
		m.check = nil
	}
	m.downloaded = false
	m.opErr = ""
	ctx, cancel := context.WithCancel(a.taskContext())
	task := &updateOperation{download: download, silent: silent, cancel: cancel, done: make(chan updateOutcome, 1)}
	m.operation = task
	backend, invalidate := a.backend, a.invalidateWindow
	go func() {
		defer cancel()
		var result updateOutcome
		if download {
			result.path, result.err = backend.DownloadUpdateContext(ctx, selected.Version, selected.SHA256)
		} else {
			value, err := backend.CheckUpdatesContext(ctx)
			result.check, result.err = &value, err
		}
		task.done <- result
		if invalidate != nil && ctx.Err() == nil {
			invalidate()
		}
	}()
}

func (a *App) finishUpdateOperation() {
	m := &a.updateSettings
	task := m.operation
	if task == nil {
		return
	}
	select {
	case result := <-task.done:
		m.operation = nil
		if result.err != nil {
			m.opErr = result.err.Error()
			if !task.silent {
				a.setError(a.tf("update.failed", m.opErr))
			}
			return
		}
		if task.download {
			m.downloaded = result.path != ""
			a.setStatus(a.t("update.downloaded"))
		} else {
			m.check = result.check
			if m.check.Available {
				a.setStatus(a.tf("update.available", m.check.Version))
			} else if !task.silent {
				a.setStatus(a.t("update.current"))
			}
		}
	default:
	}
}

func (a *App) openUpdateConfig() {
	m := &a.updateSettings
	m.feedURL, m.key, m.auto = m.info.Settings.FeedURL, m.info.Settings.PublicKey, m.info.Settings.AutoCheck
	m.configErr = ""
	m.configOpen = true
}

func (a *App) viewUpdateSettingsCard(c *ui.Context) {
	if a.backend == nil {
		return
	}
	a.loadUpdateSettings()
	m, pal := &a.updateSettings, a.pal
	macHighlightCard(c, pal, a.toolsModal.highlightID == "global-update", func() {
		ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
			ui.Text(c, a.t("update.title")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
			if m.info.CurrentVersion != "" {
				ui.Text(c, a.tf("update.version", m.info.CurrentVersion)).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
			}
			if m.operation != nil {
				progress := a.t("update.checking")
				if m.operation.download {
					progress = a.t("update.downloading")
				}
				ui.Text(c, progress).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
			} else if m.opErr != "" {
				ui.Text(c, m.opErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
			} else if m.downloaded {
				ui.Text(c, a.t("update.downloaded")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
			} else if m.check != nil {
				status := a.t("update.current")
				if m.check.Available {
					status = a.tf("update.available", m.check.Version)
				}
				ui.Text(c, status).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
			} else if !m.info.Configured {
				ui.Text(c, a.t("update.noSource")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
			}
			ui.Row(c).Gap(8).Wrap().Children(func() {
				if m.operation != nil {
					if macSecondaryButton(c, "cancel-update", a.t("common.cancel"), pal) {
						m.operation.cancel()
						a.backend.CancelUpdateOperation()
						m.operation = nil
						m.check = nil
					}
					return
				}
				if macSecondaryButton(c, "check-update", a.t("update.check"), pal) {
					if m.info.Configured {
						a.startUpdateOperation(false, false)
					} else {
						a.openUpdateConfig()
					}
				}
				if macSecondaryButton(c, "configure-update", a.t("update.configure"), pal) {
					a.openUpdateConfig()
				}
				if m.downloaded {
					if macSecondaryButton(c, "show-update", a.t("update.reveal"), pal) {
						if err := a.backend.RevealDownloadedUpdate(); err != nil {
							m.opErr = err.Error()
						}
					}
				} else if m.check != nil && m.check.Available &&
					macPrimaryButtonSmall(c, "download-update", a.t("update.download"), pal) {
					a.startUpdateOperation(true, false)
				}
			})
		})
	})
}

func (a *App) viewUpdateConfigSheet(c *ui.Context) {
	m, pal := &a.updateSettings, a.pal
	if macSheet(c, &m.configOpen, macSheetProps{
		Title:        a.t("update.configure"),
		Width:        500,
		PrimaryText:  a.t("update.save"),
		CancelText:   a.t("common.cancel"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		ui.Text(c, a.t("update.trustHint")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
		macField(c, a.t("update.feed"), "", pal, func() {
			ui.TextInput(c.Key("update-feed"), &m.feedURL).Label(a.t("update.feed")).AlignSelf(ui.Stretch)
		})
		macField(c, a.t("update.publicKey"), "", pal, func() {
			ui.TextInput(c.Key("update-key"), &m.key).Label(a.t("update.publicKey")).AlignSelf(ui.Stretch)
		})
		ui.Checkbox(c.Key("update-on-start"), &m.auto, a.t("update.autoCheck"))
		if m.configErr != "" {
			ui.Text(c, m.configErr).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
		}
	}) {
		info, err := a.backend.SaveUpdateSettings(strings.TrimSpace(m.feedURL), strings.TrimSpace(m.key), m.auto)
		if err != nil {
			m.configErr = err.Error()
			return
		}
		m.info, m.check, m.downloaded, m.opErr = info, nil, false, ""
		m.configOpen = false
	}
}
