package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"windows/internal/publisher"
	"windows/internal/store"
)

const (
	siteSettingsTabBasic      = 0
	siteSettingsTabProfile    = 1
	siteSettingsTabCustomCode = 2
	siteSettingsTabPodcast    = 3
)

type siteSettingsState struct {
	open           bool
	tab            int
	siteID         string
	name           string
	about          string
	slug           string
	author         string
	doNotIndex     bool
	domain         string
	twitter        string
	github         string
	telegram       string
	mastodon       string
	discord        string
	roundAvatar    bool
	headEnabled    bool
	head           string
	bodyStartOn    bool
	bodyStart      string
	bodyEndOn      bool
	bodyEnd        string
	publishIPNS    bool
	prewarmPost    bool
	podcastLang    string
	podcastExp     bool
	podcastCats    string
	err            string
	removeConfirm  bool
	removeSiteName string
}

type publishModalState struct {
	open     bool
	siteName string
	result   publisher.Result
	err      string
}

func (a *App) openSiteSettings(site store.Site) {
	a.siteSettings = siteSettingsState{
		open:        true,
		tab:         siteSettingsTabBasic,
		siteID:      site.ID,
		name:        site.Name,
		about:       site.About,
		slug:        site.Slug,
		author:      site.AuthorName,
		doNotIndex:  site.DoNotIndex,
		domain:      site.Domain,
		twitter:     site.TwitterUsername,
		github:      site.GithubUsername,
		telegram:    site.TelegramUsername,
		mastodon:    site.MastodonUsername,
		discord:     site.DiscordLink,
		roundAvatar: site.SaveRoundAvatar,
		headEnabled: site.CustomCodeHeadEnabled,
		head:        site.CustomCodeHead,
		bodyStartOn: site.CustomCodeBodyStartEnabled,
		bodyStart:   site.CustomCodeBodyStart,
		bodyEndOn:   site.CustomCodeBodyEndEnabled,
		bodyEnd:     site.CustomCodeBodyEnd,
		publishIPNS: site.ShouldPublishIPNS(),
		prewarmPost: site.ShouldPrewarmNewPost(),
		podcastLang: site.PodcastLanguage,
		podcastExp:  site.PodcastExplicit,
		podcastCats: strings.Join(site.PodcastCategories, ", "),
	}
}

func (a *App) viewSiteSettingsModal(c *ui.Context) {
	pal := a.pal
	st := &a.siteSettings
	ui.Modal(c, &st.open, func() {
		// Wide 820x580 two-column macOS Preferences layout matching mac-planet.css:1040-1050.
		ui.Row(c).Width(820).Height(560).Background(pal.canvas).Radius(12).Clip().Children(func() {
			// Left 200px sidebar navigation.
			ui.Column(c).Width(200).Shrink(0).AlignSelf(ui.Stretch).Background(pal.sidebar).Padding(14).Gap(4).Children(func() {
				ui.Text(c, "站点设置").Font(macPlanetFont).FontSize(15).FontWeight(600).TextColor(pal.ink).PaddingX(8).PaddingY(6)
				tabs := []struct {
					idx   int
					label string
				}{
					{siteSettingsTabBasic, "基本"},
					{siteSettingsTabProfile, "资料与头像"},
					{siteSettingsTabCustomCode, "自定义代码"},
					{siteSettingsTabPodcast, "播客"},
				}
				for _, tab := range tabs {
					tIdx, tLbl := tab.idx, tab.label
					if sidebarNavRow(c, fmt.Sprintf("ss-nav-%d", tIdx), tLbl, st.tab == tIdx, 0, nil, pal) {
						st.tab = tIdx
						st.err = ""
					}
				}
				ui.Spacer(c)
				if ui.Button(c.Key("ss-remove-site"), "").Label("移入回收站").AlignSelf(ui.Stretch).
					Background(ui.Color{}).Border(0, ui.Color{}).Radius(6).PaddingX(8).PaddingY(6).
					Children(func() {
						ui.Text(c, "移入回收站").Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
					}).Clicked() {
					st.removeSiteName = st.name
					st.removeConfirm = true
				}
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.line)

			// Right form pane + #f7f7f9 footer bar.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(22).Gap(14).AlignSelf(ui.Stretch).Children(func() {
						switch st.tab {
						case siteSettingsTabBasic:
							ui.Text(c, "基本设置").Font(macPlanetFont).FontSize(16).FontWeight(600).TextColor(pal.ink)
							ui.Form(c, func() {
								ui.Field(c, "站点名称", func() {
									ui.TextInput(c.Key("ss-name"), &st.name).Label("站点名称").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "简介", func() {
									ui.TextArea(c.Key("ss-about"), &st.about).Lines(2, 4).Label("站点简介").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "路径别名 (Slug)", func() {
									ui.TextInput(c.Key("ss-slug"), &st.slug).Label("站点路径别名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "作者名", func() {
									ui.TextInput(c.Key("ss-author"), &st.author).Label("作者名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "搜索引擎", func() {
									ui.Checkbox(c.Key("ss-noindex"), &st.doNotIndex, "禁止搜索引擎收录 (noindex)")
								})
							})

						case siteSettingsTabProfile:
							ui.Text(c, "公开资料与头像").Font(macPlanetFont).FontSize(16).FontWeight(600).TextColor(pal.ink)
							ui.Form(c, func() {
								ui.Field(c, "自定义域名", func() {
									ui.TextInput(c.Key("ss-domain"), &st.domain).Placeholder("example.com").Label("自定义域名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "Twitter / X", func() {
									ui.TextInput(c.Key("ss-twitter"), &st.twitter).Label("Twitter 用户名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "GitHub", func() {
									ui.TextInput(c.Key("ss-github"), &st.github).Label("GitHub 用户名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "Telegram", func() {
									ui.TextInput(c.Key("ss-telegram"), &st.telegram).Label("Telegram 用户名").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "Mastodon", func() {
									ui.TextInput(c.Key("ss-mastodon"), &st.mastodon).Label("Mastodon").AlignSelf(ui.Stretch)
								})
								ui.Field(c, "Discord", func() {
									ui.TextInput(c.Key("ss-discord"), &st.discord).Label("Discord 链接").AlignSelf(ui.Stretch)
								})
							})
							hairlineH(c, pal)
							ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
								ui.Text(c, "站点头像").Font(macPlanetFont).FontSize(12).FontWeight(600).TextColor(pal.muted)
								if macSecondaryButton(c, "ss-avatar-pick", "选择本地图片", pal) {
									if _, err := a.backend.SelectSiteAvatar(context.Background(), st.siteID); err != nil {
										st.err = err.Error()
									} else {
										st.err = ""
										a.Reload()
										a.setStatus("已更新站点头像")
									}
								}
								if macSecondaryButton(c, "ss-avatar-rm", "移除头像", pal) {
									if _, err := a.backend.RemoveSiteAvatar(st.siteID); err != nil {
										st.err = err.Error()
									} else {
										st.err = ""
										a.Reload()
										a.setStatus("已移除站点头像")
									}
								}
								prevRound := st.roundAvatar
								ui.Checkbox(c.Key("ss-avatar-round"), &st.roundAvatar, "圆形裁切")
								if st.roundAvatar != prevRound {
									if _, err := a.backend.SetSaveRoundAvatar(st.siteID, st.roundAvatar); err != nil {
										st.err = err.Error()
									} else {
										a.Reload()
									}
								}
							})
							if presets := a.backend.PresetAvatars(); len(presets) > 0 {
								ui.Row(c).Gap(6).Wrap().Children(func() {
									for i, p := range presets {
										if i >= 8 {
											break
										}
										key, title := p.Key, p.Title
										if title == "" {
											title = key
										}
										if macSecondaryButton(c, "ss-preset-"+key, title, pal) {
											if _, err := a.backend.ApplyPresetAvatar(st.siteID, key); err != nil {
												st.err = err.Error()
											} else {
												st.err = ""
												a.Reload()
												a.setStatus("已应用预设头像：" + title)
											}
										}
									}
								})
							}

						case siteSettingsTabCustomCode:
							ui.Text(c, "自定义代码注入").Font(macPlanetFont).FontSize(16).FontWeight(600).TextColor(pal.ink)
							ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
								ui.Checkbox(c.Key("ss-head-on"), &st.headEnabled, "启用 <head> 自定义代码")
								if st.headEnabled {
									ui.TextArea(c.Key("ss-head"), &st.head).Lines(2, 4).Label("Head 代码").AlignSelf(ui.Stretch)
								}
								ui.Checkbox(c.Key("ss-bs-on"), &st.bodyStartOn, "启用 <body> 开头自定义代码")
								if st.bodyStartOn {
									ui.TextArea(c.Key("ss-bs"), &st.bodyStart).Lines(2, 4).Label("Body 开头代码").AlignSelf(ui.Stretch)
								}
								ui.Checkbox(c.Key("ss-be-on"), &st.bodyEndOn, "启用 <body> 结尾自定义代码")
								if st.bodyEndOn {
									ui.TextArea(c.Key("ss-be"), &st.bodyEnd).Lines(2, 4).Label("Body 结尾代码").AlignSelf(ui.Stretch)
								}
								ui.Row(c).Gap(16).Children(func() {
									ui.Checkbox(c.Key("ss-ipns"), &st.publishIPNS, "发布到 IPNS")
									ui.Checkbox(c.Key("ss-prewarm"), &st.prewarmPost, "新文章发布后预热网关")
								})
							})

						case siteSettingsTabPodcast:
							ui.Text(c, "播客设置").Font(macPlanetFont).FontSize(16).FontWeight(600).TextColor(pal.ink)
							ui.Form(c, func() {
								ui.Field(c, "播客语言", func() {
									ui.TextInput(c.Key("ss-pod-lang"), &st.podcastLang).Placeholder("zh-cn / en").Label("播客语言").Width(180)
								})
								ui.Field(c, "内容分级", func() {
									ui.Checkbox(c.Key("ss-pod-exp"), &st.podcastExp, "包含成人/敏感内容 (Explicit)")
								})
								ui.Field(c, "播客分类", func() {
									ui.TextInput(c.Key("ss-pod-cats"), &st.podcastCats).Placeholder("Technology, News（逗号分隔）").Label("播客分类").AlignSelf(ui.Stretch)
								})
							})
						}

						if st.err != "" {
							ui.Text(c, st.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
						}
					})
				})

				// Mac Sheet bottom action bar (#f7f7f9 with 0.5px top border).
				hairlineH(c, pal)
				ui.Row(c).PaddingX(22).PaddingY(12).Gap(8).Justify(ui.End).Background(pal.footer).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					if macSecondaryButton(c, "ss-cancel", "取消", pal) {
						st.open = false
					}
					if macPrimaryButton(c, "ss-save", "保存站点设置", pal) {
						a.saveSiteSettings()
					}
				})
			})
		})
	})

	switch ui.AlertDialog(c, &st.removeConfirm,
		"将站点移入回收站？",
		fmt.Sprintf("站点“%s”会被移入回收站，可随时在回收站中恢复。", fallbackTitle(st.removeSiteName)),
		"取消", "移入回收站") {
	case 1:
		if err := a.backend.RemoveSite(st.siteID); err != nil {
			st.err = err.Error()
		} else {
			st.open = false
			if a.selectedSiteID == st.siteID {
				a.selectedSiteID = ""
				a.mode = viewOverview
			}
			a.Reload()
			a.setStatus("站点已移入回收站")
		}
	}
}

func (a *App) saveSiteSettings() {
	st := &a.siteSettings
	var err error
	switch st.tab {
	case siteSettingsTabBasic:
		if strings.TrimSpace(st.name) == "" {
			st.err = "站点名称不能为空"
			return
		}
		_, err = a.backend.UpdateSiteDetails(st.siteID, strings.TrimSpace(st.name), strings.TrimSpace(st.about), st.doNotIndex, strings.TrimSpace(st.slug), strings.TrimSpace(st.author))
	case siteSettingsTabProfile:
		raw, _ := json.Marshal(store.SiteProfile{
			Domain:           strings.TrimSpace(st.domain),
			TwitterUsername:  strings.TrimSpace(st.twitter),
			GithubUsername:   strings.TrimSpace(st.github),
			TelegramUsername: strings.TrimSpace(st.telegram),
			MastodonUsername: strings.TrimSpace(st.mastodon),
			DiscordLink:      strings.TrimSpace(st.discord),
		})
		_, err = a.backend.UpdateSiteProfile(st.siteID, string(raw))
	case siteSettingsTabCustomCode:
		pub, pre := st.publishIPNS, st.prewarmPost
		raw, _ := json.Marshal(store.SiteCustomCode{
			HeadEnabled:      st.headEnabled,
			Head:             st.head,
			BodyStartEnabled: st.bodyStartOn,
			BodyStart:        st.bodyStart,
			BodyEndEnabled:   st.bodyEndOn,
			BodyEnd:          st.bodyEnd,
			PublishAsIPNS:    &pub,
			PrewarmNewPost:   &pre,
		})
		_, err = a.backend.UpdateSiteCustomCode(st.siteID, string(raw))
	case siteSettingsTabPodcast:
		var cats []string
		for _, part := range strings.Split(st.podcastCats, ",") {
			if p := strings.TrimSpace(part); p != "" {
				cats = append(cats, p)
			}
		}
		raw, _ := json.Marshal(cats)
		_, err = a.backend.UpdatePodcastSettings(st.siteID, strings.TrimSpace(st.podcastLang), st.podcastExp, string(raw))
	}
	if err != nil {
		st.err = err.Error()
		return
	}
	st.err = ""
	st.open = false
	a.Reload()
	a.setStatus("站点设置已保存")
}

func (a *App) triggerPublishSite(site store.Site) {
	res, err := a.backend.PublishSite(site.ID)
	a.publishModal = publishModalState{
		open:     true,
		siteName: fallbackTitle(site.Name),
		result:   res,
	}
	if err != nil {
		a.publishModal.err = err.Error()
		return
	}
	a.Reload()
	a.setStatus("站点已发布：" + fallbackTitle(site.Name))
}

func (a *App) viewPublishModal(c *ui.Context) {
	t := c.Theme()
	pm := &a.publishModal
	ui.Modal(c, &pm.open, func() {
		ui.Column(c).Width(480).Gap(12).Children(func() {
			ui.Text(c, "发布详情 · "+pm.siteName).Bold().FontSize(16)
			if pm.err != "" {
				ui.Text(c, "发布失败："+pm.err).FontSize(13).TextColor(t.Danger)
			} else {
				ui.Column(c).Padding(12).Gap(6).Background(t.Surface).Border(1, t.Border).Radius(6).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "CID: "+pm.result.CID).FontSize(12)
					if pm.result.IPNS != "" {
						ui.Text(c, "IPNS: "+pm.result.IPNS).FontSize(12)
					}
					ui.Text(c, fmt.Sprintf("版本修订号: %d", pm.result.Revision)).FontSize(12).TextColor(t.TextMuted)
					if pm.result.Pending {
						ui.Text(c, "IPNS 仍在后台同步中").FontSize(12).TextColor(t.Warning)
					}
					if pm.result.PinnableWarning != "" {
						ui.Text(c, "Pinnable 提示："+pm.result.PinnableWarning).FontSize(12).TextColor(t.Warning)
					}
					if pm.result.TelegramWarning != "" {
						ui.Text(c, "Telegram 提示："+pm.result.TelegramWarning).FontSize(12).TextColor(t.Warning)
					}
				})
			}
			ui.Row(c).Justify(ui.End).Children(func() {
				if ui.PrimaryButton(c.Key("pub-close"), "完成").Clicked() {
					pm.open = false
				}
			})
		})
	})
}
