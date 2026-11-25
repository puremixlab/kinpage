package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
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
	macPreferencesWindow(c, &st.open, 940, 680, pal, func() {
		// Responsive two-column macOS Preferences layout matching mac-planet.css:1040-1260.
		ui.Row(c).Fill().Background(pal.canvas).Children(func() {
			// Left 216px sidebar navigation + site identity footer.
			ui.Column(c).Width(216).Shrink(0).AlignSelf(ui.Stretch).Background(pal.sidebar).Padding(18, 10, 14, 10).Gap(12).Children(func() {
				ui.Column(c).Gap(2).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, "站点设置").Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.muted).Padding(0, 10, 6, 10)
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
				})

				ui.Spacer(c)

				// Bottom site identity card + remove site action.
				ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
					hairlineH(c, pal)
					if site := a.findSiteByID(st.siteID); site != nil {
						ui.Row(c).Padding(6, 8, 2, 8).Gap(10).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
							a.siteAvatar(c, *site, 0, 28)
							ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, fallbackTitle(site.Name)).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.ink).MaxLines(1).Ellipsis("…")
								tpl := site.TemplateName
								if tpl == "" {
									tpl = "默认模板"
								}
								ui.Text(c, tpl).Font(macPlanetFont).FontSize(11).TextColor(pal.muted).MaxLines(1).Ellipsis("…")
							})
						})
					}
					if interactiveRow(c, "ss-remove-site", "移入回收站", ui.Transparent, pal.hover, pal.active).
						MinHeight(30).Radius(6).PaddingX(10).PaddingY(5).
						Children(func() {
							ui.Text(c, "移入回收站").Font(macPlanetFont).FontSize(12.5).TextColor(pal.danger)
						}).Clicked() {
						st.removeSiteName = st.name
						st.removeConfirm = true
					}
				})
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.line)

			// Right pane: fixed header + scrollable inset-grouped body + fixed #f7f7f9 footer bar.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.canvas).Children(func() {
				pageTitle, pageDesc := "基本设置", "配置站点名称、简介、路径别名与搜索引擎收录偏好。"
				switch st.tab {
				case siteSettingsTabProfile:
					pageTitle, pageDesc = "公开资料与头像", "设置站点头像、自定义域名与公开社交账号链接。"
				case siteSettingsTabCustomCode:
					pageTitle, pageDesc = "自定义代码与发布", "向生成的静态页面注入自定义 HTML，并控制 IPNS 与网关预热行为。"
				case siteSettingsTabPodcast:
					pageTitle, pageDesc = "播客设置", "配置 RSS 播客频道语言、分类与内容分级标记。"
				}

				ui.Row(c).Padding(20, 20, 12, 28).AlignItems(ui.Start).Justify(ui.SpaceBetween).Gap(12).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, pageTitle).Font(macPlanetFont).FontSize(18).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.ink)
						ui.Text(c, pageDesc).Font(macPlanetFont).FontSize(12.5).TextColor(pal.muted)
					})
					if toolbarIconBtn(c, "ss-close-x", lucide.X(), "关闭").Clicked() {
						st.open = false
					}
				})

				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(4, 28, 24, 28).Gap(16).AlignSelf(ui.Stretch).Children(func() {
						switch st.tab {
						case siteSettingsTabBasic:
							macInsetCard(c, pal, func() {
								macField(c, "站点名称", "", pal, func() {
									ui.TextInput(c.Key("ss-name"), &st.name).Label("站点名称").AlignSelf(ui.Stretch)
								})
								macField(c, "简介", "会显示在站点首页与 RSS 描述中", pal, func() {
									ui.TextArea(c.Key("ss-about"), &st.about).Lines(3, 5).Label("站点简介").AlignSelf(ui.Stretch)
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "路径别名 (Slug)", "", pal, func() {
											ui.TextInput(c.Key("ss-slug"), &st.slug).Placeholder("my-planet").Label("站点路径别名").AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "作者名", "", pal, func() {
											ui.TextInput(c.Key("ss-author"), &st.author).Placeholder("显示在文章署名中").Label("作者名").AlignSelf(ui.Stretch)
										})
									})
								})
							})
							macInsetCard(c, pal, func() {
								ui.Checkbox(c.Key("ss-noindex"), &st.doNotIndex, "禁止搜索引擎收录 (noindex)")
								ui.Text(c, "开启后会在页面头部输出 noindex 元标签，提示搜索引擎不要索引此站点。").Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
							})

						case siteSettingsTabProfile:
							macInsetCard(c, pal, func() {
								ui.Row(c).Gap(14).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
									if site := a.findSiteByID(st.siteID); site != nil {
										a.siteAvatar(c, *site, 0, 52)
									}
									ui.Column(c).Gap(6).Grow(1).MinWidth(0).Children(func() {
										ui.Text(c, "站点头像").Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.ink)
										ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
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
									})
								})
								if presets := a.backend.PresetAvatars(); len(presets) > 0 {
									hairlineH(c, pal)
									ui.Column(c).Gap(6).AlignSelf(ui.Stretch).Children(func() {
										ui.Text(c, "快速选择内置头像").Font(macPlanetFont).FontSize(11.5).TextColor(pal.muted)
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
									})
								}
							})

							macInsetCard(c, pal, func() {
								macField(c, "自定义域名", "", pal, func() {
									ui.TextInput(c.Key("ss-domain"), &st.domain).Placeholder("example.com").Label("自定义域名").AlignSelf(ui.Stretch)
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Twitter / X", "", pal, func() {
											ui.TextInput(c.Key("ss-twitter"), &st.twitter).Label("Twitter 用户名").AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "GitHub", "", pal, func() {
											ui.TextInput(c.Key("ss-github"), &st.github).Label("GitHub 用户名").AlignSelf(ui.Stretch)
										})
									})
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Telegram", "", pal, func() {
											ui.TextInput(c.Key("ss-telegram"), &st.telegram).Label("Telegram 用户名").AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Mastodon", "", pal, func() {
											ui.TextInput(c.Key("ss-mastodon"), &st.mastodon).Label("Mastodon").AlignSelf(ui.Stretch)
										})
									})
								})
								macField(c, "Discord", "", pal, func() {
									ui.TextInput(c.Key("ss-discord"), &st.discord).Label("Discord 链接").AlignSelf(ui.Stretch)
								})
							})

						case siteSettingsTabCustomCode:
							macInsetCard(c, pal, func() {
								ui.Checkbox(c.Key("ss-head-on"), &st.headEnabled, "启用 <head> 自定义代码")
								if st.headEnabled {
									ui.TextArea(c.Key("ss-head"), &st.head).Lines(3, 5).Label("Head 代码").AlignSelf(ui.Stretch)
								}
								ui.Checkbox(c.Key("ss-bs-on"), &st.bodyStartOn, "启用 <body> 开头自定义代码")
								if st.bodyStartOn {
									ui.TextArea(c.Key("ss-bs"), &st.bodyStart).Lines(3, 5).Label("Body 开头代码").AlignSelf(ui.Stretch)
								}
								ui.Checkbox(c.Key("ss-be-on"), &st.bodyEndOn, "启用 <body> 结尾自定义代码")
								if st.bodyEndOn {
									ui.TextArea(c.Key("ss-be"), &st.bodyEnd).Lines(3, 5).Label("Body 结尾代码").AlignSelf(ui.Stretch)
								}
							})
							macInsetCard(c, pal, func() {
								ui.Row(c).Gap(20).Wrap().Children(func() {
									ui.Checkbox(c.Key("ss-ipns"), &st.publishIPNS, "发布到 IPNS")
									ui.Checkbox(c.Key("ss-prewarm"), &st.prewarmPost, "新文章发布后预热网关")
								})
							})

						case siteSettingsTabPodcast:
							macInsetCard(c, pal, func() {
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Width(180).Shrink(0).Children(func() {
										macField(c, "播客语言", "", pal, func() {
											ui.TextInput(c.Key("ss-pod-lang"), &st.podcastLang).Placeholder("zh-cn / en").Label("播客语言").AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "播客分类", "逗号分隔", pal, func() {
											ui.TextInput(c.Key("ss-pod-cats"), &st.podcastCats).Placeholder("Technology, News").Label("播客分类").AlignSelf(ui.Stretch)
										})
									})
								})
								ui.Checkbox(c.Key("ss-pod-exp"), &st.podcastExp, "包含成人/敏感内容 (Explicit)")
							})
						}

						if st.err != "" {
							ui.Text(c, st.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
						}
					})
				})

				// Mac Sheet bottom action bar (#f7f7f9 with 0.5px top border).
				hairlineH(c, pal)
				ui.Row(c).Padding(12, 28, 12, 28).Gap(8).Justify(ui.End).Background(pal.footer).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
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

	if macConfirmSheet(c, &st.removeConfirm,
		"将站点移入回收站？",
		fmt.Sprintf("站点“%s”会被移入回收站，可随时在回收站中恢复。", fallbackTitle(st.removeSiteName)),
		"取消", "移入回收站", true, pal) {
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
	pal := a.pal
	pm := &a.publishModal
	if macSheet(c, &pm.open, macSheetProps{
		Title:       "发布详情 · " + pm.siteName,
		Width:       500,
		PrimaryText: "完成",
	}, pal, func() {
		if pm.err != "" {
			ui.Text(c, "发布失败："+pm.err).Font(macPlanetFont).FontSize(13).TextColor(pal.danger)
		} else {
			ui.Column(c).Padding(12).Gap(6).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "CID: "+pm.result.CID).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
				if pm.result.IPNS != "" {
					ui.Text(c, "IPNS: "+pm.result.IPNS).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
				}
				ui.Text(c, fmt.Sprintf("版本修订号: %d", pm.result.Revision)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				if pm.result.Pending {
					ui.Text(c, "IPNS 仍在后台同步中").Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
				if pm.result.PinnableWarning != "" {
					ui.Text(c, "Pinnable 提示："+pm.result.PinnableWarning).Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
				if pm.result.TelegramWarning != "" {
					ui.Text(c, "Telegram 提示："+pm.result.TelegramWarning).Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
			})
		}
	}) {
		pm.open = false
	}
}
