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
	t := c.Theme()
	st := &a.siteSettings
	ui.Modal(c, &st.open, func() {
		ui.Column(c).Width(560).Gap(12).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "站点设置").Bold().FontSize(16)
				ui.Segmented(c.Key("site-set-tab"), &st.tab, "基本", "资料与头像", "自定义代码", "播客").Label("站点设置分类")
			})

			switch st.tab {
			case siteSettingsTabBasic:
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
				ui.Box(c).Height(1).AlignSelf(ui.Stretch).Background(t.Border)
				ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
					ui.Text(c, "站点头像").FontSize(12).Bold().TextColor(t.TextMuted)
					if ui.Button(c.Key("ss-avatar-pick"), "选择本地图片").Clicked() {
						if _, err := a.backend.SelectSiteAvatar(context.Background(), st.siteID); err != nil {
							st.err = err.Error()
						} else {
							st.err = ""
							a.Reload()
							a.setStatus("已更新站点头像")
						}
					}
					if ui.Button(c.Key("ss-avatar-rm"), "移除头像").Clicked() {
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
							if ui.Button(c.Key("ss-preset-"+key), title).Clicked() {
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
				ui.Text(c, st.err).FontSize(12).TextColor(t.Danger)
			}

			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).AlignSelf(ui.Stretch).Children(func() {
				if ui.Button(c.Key("ss-remove-site"), "移入回收站").Clicked() {
					st.removeSiteName = st.name
					st.removeConfirm = true
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c.Key("ss-cancel"), "取消").Clicked() {
						st.open = false
					}
					if ui.PrimaryButton(c.Key("ss-save"), "保存站点设置").Clicked() {
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
