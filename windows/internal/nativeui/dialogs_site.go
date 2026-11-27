package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/field"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	"github.com/lobehub/lobe-mygo/pkg/theme"
	"windows/internal/publisher"
	"windows/internal/store"
)

const (
	siteSettingsTabBasic      = 0
	siteSettingsTabProfile    = 1
	siteSettingsTabTemplates  = 2
	siteSettingsTabCustomCode = 3
	siteSettingsTabPodcast    = 4
	siteSettingsTabDeploy     = 5
	siteSettingsTabServices   = 6
	siteSettingsTabX          = 7
	siteSettingsTabTelegram   = 8
	siteSettingsTabAggregate  = 9
	siteSettingsTabWeb3       = 10
	siteSettingsTabPublish    = 11
)

type siteSettingsState struct {
	open           bool
	tab            int
	siteID         string
	search         string
	returnSearch   string
	highlightID    string
	scroll         ui.ScrollState
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
	headExpanded   bool
	head           string
	bodyStartOn    bool
	bodyStartExp   bool
	bodyStart      string
	bodyEndOn      bool
	bodyEndExp     bool
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
	a.loadIntegrationsState(site)
}

func (a *App) viewSiteSettingsModal(c *ui.Context) {
	pal := a.pal
	st := &a.siteSettings
	m := &a.integrations
	macPreferencesWindow(c, &st.open, 960, 680, pal, func() {
		activeSite := a.findSiteByID(st.siteID)
		activeSiteName := a.fallbackTitle(st.name)
		if activeSite != nil {
			activeSiteName = a.fallbackTitle(activeSite.Name)
		}

		// Unified two-column macOS Sequoia Site Settings (12 tabs in 3 groups).
		ui.Row(c).Fill().Background(pal.prefsCanvas).Children(func() {
			// Left 232px cool-tinted sidebar navigation with site identity + search bar.
			ui.Column(c).Width(232).Shrink(0).AlignSelf(ui.Stretch).Background(pal.prefsSidebar).Padding(16, 10, 12, 10).Gap(10).Children(func() {
				// Window title + persistent current site identity pill.
				ui.Column(c).Gap(6).Padding(0, 8, 2, 8).AlignSelf(ui.Stretch).Children(func() {
					ui.Text(c, a.t("siteSettings.windowTitle")).Font(macPlanetFont).FontSize(15).FontWeight(600).TextColor(pal.prefsInk)
					ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(4, 8, 4, 6).Radius(6).
						Background(pal.hover).Border(0.5, pal.prefsCardBorder).AlignSelf(ui.Stretch).Children(func() {
						if activeSite != nil {
							a.siteAvatar(c, *activeSite, 0, 18)
						}
						ui.Text(c, activeSiteName).Font(macPlanetFont).FontSize(12).FontWeight(500).
							TextColor(pal.prefsInk).NoWrap().Ellipsis("…").Grow(1).MinWidth(0)
					})
				})

				prevSearch := st.search
				field.SearchBar(c.Key("site-settings-nav-search"), &st.search, field.SearchBarProps{
					Placeholder: a.t("settings.searchPlaceholder"),
					Variant:     field.InputFilled,
					Size:        theme.Small,
				})
				if st.search != prevSearch && strings.TrimSpace(st.search) != "" {
					st.returnSearch = ""
					st.highlightID = ""
				}
				if st.returnSearch != "" && strings.TrimSpace(st.search) == "" {
					if macSettingsReturnButton(c, "ss-return-search", a.t("settings.search.return"), pal) {
						st.search = st.returnSearch
						st.returnSearch = ""
						st.highlightID = ""
					}
				}

				type navSpec struct {
					idx   int
					label string
					icon  *ui.SVG
					tint  ui.Color
				}
				navGroup := func(heading string, items []navSpec) {
					ui.Column(c).Gap(2).AlignSelf(ui.Stretch).Children(func() {
						ui.Text(c, heading).Font(macPlanetFont).FontSize(11).FontWeight(600).TextColor(pal.prefsMuted).Padding(0, 10, 5, 10)
						for _, tab := range items {
							tIdx, tLbl := tab.idx, tab.label
							selected := strings.TrimSpace(st.search) == "" && st.tab == tIdx
							if prefsNavRow(c, fmt.Sprintf("ss-nav-%d", tIdx), tLbl, selected, tab.icon, tab.tint, pal) {
								st.search = ""
								st.returnSearch = ""
								st.highlightID = ""
								st.tab = tIdx
								st.scroll.Y = 0
								st.err = ""
								m.err = ""
								m.msg = ""
							}
						}
					})
				}

				ui.Scroll(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Gap(14).AlignSelf(ui.Stretch).Children(func() {
						navGroup(a.t("siteSettings.groupSite"), []navSpec{
							{siteSettingsTabBasic, a.t("siteSettings.navBasic"), lucide.SlidersHorizontal(), ui.Hex("#2563eb")},
							{siteSettingsTabProfile, a.t("siteSettings.navSocial"), lucide.Globe(), ui.Hex("#4f46e5")},
							{siteSettingsTabTemplates, a.t("siteSettings.navTemplates"), lucide.LayoutGrid(), ui.Hex("#0d9488")},
							{siteSettingsTabCustomCode, a.t("siteSettings.navCustomCode"), lucide.Code(), ui.Hex("#475569")},
							{siteSettingsTabPodcast, a.t("siteSettings.navPodcast"), lucide.Podcast(), ui.Hex("#7c3aed")},
						})
						navGroup(a.t("siteSettings.groupPublish"), []navSpec{
							{siteSettingsTabPublish, a.t("siteSettings.navPublishBehavior"), lucide.Radio(), ui.Hex("#0284c7")},
							{siteSettingsTabDeploy, a.t("siteSettings.navDeploy"), lucide.Upload(), ui.Hex("#0284c7")},
							{siteSettingsTabServices, a.t("siteSettings.navServices"), lucide.Activity(), ui.Hex("#d97706")},
						})
						navGroup(a.t("siteSettings.groupSync"), []navSpec{
							{siteSettingsTabX, a.t("siteSettings.navX"), lucide.Share2(), ui.Hex("#475569")},
							{siteSettingsTabTelegram, a.t("siteSettings.navTelegram"), lucide.Send(), ui.Hex("#0ea5e9")},
							{siteSettingsTabAggregate, a.t("siteSettings.navAggregate"), lucide.Rss(), ui.Hex("#ea580c")},
							{siteSettingsTabWeb3, a.t("siteSettings.navWeb3"), lucide.Wallet(), ui.Hex("#7c3aed")},
						})
					})
				})
			})
			ui.Box(c).Width(0.5).AlignSelf(ui.Stretch).Background(pal.prefsCardBorder)

			// Right pane: soft canvas + elevated white inset cards + cool-tinted footer bar.
			ui.Column(c).Grow(1).MinWidth(0).AlignSelf(ui.Stretch).Background(pal.prefsCanvas).Children(func() {
				tabTitles := map[int]string{
					siteSettingsTabBasic:      a.t("siteSettings.navBasic"),
					siteSettingsTabProfile:    a.t("siteSettings.titleSocial"),
					siteSettingsTabTemplates:  a.t("siteSettings.navTemplates"),
					siteSettingsTabCustomCode: a.t("siteSettings.navCustomCode"),
					siteSettingsTabPodcast:    a.t("siteSettings.navPodcast"),
					siteSettingsTabPublish:    a.t("siteSettings.navPublishBehavior"),
					siteSettingsTabDeploy:     a.t("siteSettings.navDeploy"),
					siteSettingsTabServices:   a.t("siteSettings.titleServices"),
					siteSettingsTabX:          a.t("siteSettings.navX"),
					siteSettingsTabTelegram:   a.t("siteSettings.titleTelegram"),
					siteSettingsTabAggregate:  a.t("siteSettings.titleAggregate"),
					siteSettingsTabWeb3:       a.t("siteSettings.titleWeb3"),
				}
				searching := strings.TrimSpace(st.search) != ""
				headerTitle := tabTitles[st.tab]
				if searching {
					headerTitle = a.t("settings.search.heading")
				}

				ui.Row(c).Padding(20, 20, 12, 28).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(10).MinWidth(0).Children(func() {
						ui.Text(c, headerTitle).Font(macPlanetFont).FontSize(18).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.prefsInk).NoWrap()
						ui.Box(c).Padding(2, 8, 2, 8).Radius(5).Background(pal.hover).Border(0.5, pal.prefsCardBorder).Children(func() {
							ui.Text(c, activeSiteName).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsSecondary).NoWrap().Ellipsis("…")
						})
					})
					if toolbarIconBtn(c, "ss-close-x", lucide.X(), a.t("common.close")).Clicked() {
						st.open = false
					}
				})

				ui.Scroll(c).TrackScroll(&st.scroll).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
					ui.Column(c).Padding(4, 28, 36, 28).Gap(16).AlignSelf(ui.Stretch).Children(func() {
						if searching {
							matches := a.matchSettingsSearch(st.search, siteSettingsSearchItems())
							suggestions := []string{
								a.t("siteSettings.search.sugPrewarm"),
								a.t("siteSettings.navDeploy"),
								a.t("siteSettings.navTemplates"),
								a.t("siteSettings.navSocial"),
							}
							a.viewSettingsSearchResults(c, "site", st.search, matches, suggestions, func(it settingsSearchItem) {
								st.returnSearch = strings.TrimSpace(st.search)
								st.search = ""
								st.tab = it.tab
								st.highlightID = it.targetID
								st.scroll.Y = 0
								st.err = ""
								m.err = ""
								m.msg = ""
							}, func(sug string) {
								st.search = sug
							})
							return
						}
						switch st.tab {
						case siteSettingsTabBasic:
							macInsetCard(c, pal, func() {
								ui.Row(c).Gap(14).AlignItems(ui.Center).AlignSelf(ui.Stretch).Children(func() {
									if site := a.findSiteByID(st.siteID); site != nil {
										a.siteAvatar(c, *site, 0, 52)
									}
									ui.Column(c).Gap(6).Grow(1).MinWidth(0).Children(func() {
										ui.Text(c, a.t("siteSettings.avatarTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
										ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
											if macSecondaryButton(c, "ss-avatar-pick", a.t("siteSettings.avatarPick"), pal) {
												if _, err := a.backend.SelectSiteAvatar(context.Background(), st.siteID); err != nil {
													st.err = err.Error()
												} else {
													st.err = ""
													a.Reload()
													a.setStatus(a.t("status.avatarUpdated"))
												}
											}
											if macSecondaryButton(c, "ss-avatar-rm", a.t("siteSettings.avatarRemove"), pal) {
												if _, err := a.backend.RemoveSiteAvatar(st.siteID); err != nil {
													st.err = err.Error()
												} else {
													st.err = ""
													a.Reload()
													a.setStatus(a.t("status.avatarRemoved"))
												}
											}
										})
										prevRound := st.roundAvatar
										ui.Checkbox(c.Key("ss-avatar-round"), &st.roundAvatar, a.t("siteSettings.avatarRound"))
										if st.roundAvatar != prevRound {
											if _, err := a.backend.SetSaveRoundAvatar(st.siteID, st.roundAvatar); err != nil {
												st.err = err.Error()
											} else {
												a.Reload()
											}
										}
									})
								})
								if presets := a.backend.PresetAvatars(); len(presets) > 0 {
									prefsHairlineH(c, pal)
									ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
										ui.Text(c, a.t("siteSettings.avatarPresets")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
										ui.Row(c).Gap(8).Wrap().AlignSelf(ui.Stretch).Center().Children(func() {
											for i, p := range presets {
												title := p.Title
												if title == "" {
													title = p.Key
												}
												btn := ui.ButtonBase(c.Key("ss-preset-"+p.Key)).Label(title).
													Size(40, 40).Radius(20).Border(0.5, pal.prefsCardBorder).
													Center().Clip().Cursor(ui.CursorPointer).Tooltip(title)
												if btn.Hovered() {
													btn.Border(0.5, pal.prefsInk)
												}
												btn.Children(func() {
													if bmp := a.bitmapForDataURL(p.PNG, siteAvatarBitmapMaxSide); bmp != nil {
														ui.Image(c, bmp).Size(38, 38)
													} else {
														avatarCircleIndexed(c, title, i, 38)
													}
												})
												if btn.Clicked() {
													if _, err := a.backend.ApplyPresetAvatar(st.siteID, p.Key); err != nil {
														st.err = err.Error()
													} else {
														st.err = ""
														a.Reload()
														a.setStatus(a.tf("status.presetAvatarApplied", title))
													}
												}
											}
										})
									})
								}
							})
							macInsetCard(c, pal, func() {
								macField(c, a.t("siteSettings.nameField"), "", pal, func() {
									ui.TextInput(c.Key("ss-name"), &st.name).Label(a.t("siteSettings.nameField")).AlignSelf(ui.Stretch)
								})
								macField(c, a.t("siteSettings.aboutField"), "", pal, func() {
									ui.TextArea(c.Key("ss-about"), &st.about).Lines(3, 5).Label(a.t("siteSettings.aboutLabel")).AlignSelf(ui.Stretch)
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, a.t("siteSettings.slugField"), "", pal, func() {
											ui.TextInput(c.Key("ss-slug"), &st.slug).Placeholder("my-planet").Label(a.t("siteSettings.slugLabel")).AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, a.t("siteSettings.authorField"), "", pal, func() {
											ui.TextInput(c.Key("ss-author"), &st.author).Placeholder(a.t("siteSettings.authorPlaceholder")).Label(a.t("siteSettings.authorField")).AlignSelf(ui.Stretch)
										})
									})
								})
							})
							macInsetCard(c, pal, func() {
								ui.Checkbox(c.Key("ss-noindex"), &st.doNotIndex, a.t("siteSettings.doNotIndex"))
							})
							// Danger Zone card at the bottom of Basic settings.
							ui.Column(c).Padding(14, 18, 14, 18).Gap(6).AlignSelf(ui.Stretch).
								Background(pal.prefsCard).Border(0.5, pal.danger.Alpha(0.32)).Radius(10).
								Shadow(0, 1, 3, 0, pal.prefsCardShadow).Children(func() {
								ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
										ui.Text(c, a.t("siteSettings.dangerTitle")).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.danger)
										ui.Text(c, a.t("siteSettings.dangerDesc")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted)
									})
									dangerBg := pal.danger.Alpha(0.10)
									if pal.dark {
										dangerBg = pal.danger.Alpha(0.16)
									}
									trashBtnLabel := a.t("siteSettings.moveToTrashBtn")
									rmBtn := ui.ButtonBase(c.Key("ss-remove-site")).Label(trashBtnLabel).Height(30).PaddingX(12).Radius(6).
										Background(dangerBg).Border(0.5, pal.danger.Alpha(0.35)).Center().Shrink(0).
										Cursor(ui.CursorPointer).Transition(rowHoverTransition)
									if rmBtn.Hovered() {
										rmBtn.Background(pal.danger.Alpha(0.20))
									}
									rmBtn.Children(func() {
										ui.Text(c, trashBtnLabel).Font(macPlanetFont).FontSize(12.5).FontWeight(500).TextColor(pal.danger).NoWrap()
									})
									if rmBtn.Clicked() {
										st.removeSiteName = st.name
										st.removeConfirm = true
									}
								})
							})

						case siteSettingsTabProfile:
							macInsetCard(c, pal, func() {
								macField(c, a.t("siteSettings.domainField"), "", pal, func() {
									ui.TextInput(c.Key("ss-domain"), &st.domain).Placeholder("example.com").Label(a.t("siteSettings.domainField")).AlignSelf(ui.Stretch)
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Twitter / X", "", pal, func() {
											ui.TextInput(c.Key("ss-twitter"), &st.twitter).Label(a.t("siteSettings.twitterLabel")).AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "GitHub", "", pal, func() {
											ui.TextInput(c.Key("ss-github"), &st.github).Label(a.t("siteSettings.githubLabel")).AlignSelf(ui.Stretch)
										})
									})
								})
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Telegram", "", pal, func() {
											ui.TextInput(c.Key("ss-telegram"), &st.telegram).Label(a.t("siteSettings.telegramLabel")).AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, "Mastodon", "", pal, func() {
											ui.TextInput(c.Key("ss-mastodon"), &st.mastodon).Label("Mastodon").AlignSelf(ui.Stretch)
										})
									})
								})
								macField(c, "Discord", "", pal, func() {
									ui.TextInput(c.Key("ss-discord"), &st.discord).Label(a.t("siteSettings.discordLabel")).AlignSelf(ui.Stretch)
								})
							})

						case siteSettingsTabCustomCode:
							macHighlightCard(c, pal, st.highlightID == "site-custom-code", func() {
								codeSection := func(key, checkLabel, areaLabel string, enabled, expanded *bool, content *string) {
									ui.Column(c).Gap(8).AlignSelf(ui.Stretch).Children(func() {
										ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(10).Children(func() {
											ui.Checkbox(c.Key(key+"-on"), enabled, checkLabel)
											expLabel := a.t("siteSettings.expandCode")
											if *expanded {
												expLabel = a.t("siteSettings.collapseCode")
											}
											if macSecondaryButton(c, key+"-exp", expLabel, pal) {
												*expanded = !*expanded
											}
										})
										if *expanded {
											ui.TextArea(c.Key(key), content).Lines(3, 5).Label(areaLabel).AlignSelf(ui.Stretch)
										}
									})
								}
								codeSection("ss-head", a.t("siteSettings.headCheck"), a.t("siteSettings.headLabel"), &st.headEnabled, &st.headExpanded, &st.head)
								prefsHairlineH(c, pal)
								codeSection("ss-bs", a.t("siteSettings.bodyStartCheck"), a.t("siteSettings.bodyStartLabel"), &st.bodyStartOn, &st.bodyStartExp, &st.bodyStart)
								prefsHairlineH(c, pal)
								codeSection("ss-be", a.t("siteSettings.bodyEndCheck"), a.t("siteSettings.bodyEndLabel"), &st.bodyEndOn, &st.bodyEndExp, &st.bodyEnd)
							})

						case siteSettingsTabPublish:
							macHighlightCard(c, pal, st.highlightID == "site-publish-behavior", func() {
								ui.Text(c, a.t("siteSettings.publishBehaviorDesc")).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
								ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
									ui.Checkbox(c.Key("ss-ipns"), &st.publishIPNS, a.t("siteSettings.publishIPNS"))
								})
								prefsHairlineH(c, pal)
								ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
									ui.Checkbox(c.Key("ss-prewarm"), &st.prewarmPost, a.t("siteSettings.prewarmPost"))
								})
							})

						case siteSettingsTabPodcast:
							macInsetCard(c, pal, func() {
								ui.Row(c).Gap(16).AlignSelf(ui.Stretch).Children(func() {
									ui.Box(c).Width(180).Shrink(0).Children(func() {
										macField(c, a.t("siteSettings.podcastLang"), "", pal, func() {
											ui.TextInput(c.Key("ss-pod-lang"), &st.podcastLang).Placeholder("zh-cn / en").Label(a.t("siteSettings.podcastLang")).AlignSelf(ui.Stretch)
										})
									})
									ui.Box(c).Grow(1).MinWidth(0).Children(func() {
										macField(c, a.t("siteSettings.podcastCats"), a.t("siteSettings.podcastCatsHint"), pal, func() {
											ui.TextInput(c.Key("ss-pod-cats"), &st.podcastCats).Placeholder("Technology, News").Label(a.t("siteSettings.podcastCats")).AlignSelf(ui.Stretch)
										})
									})
								})
								ui.Checkbox(c.Key("ss-pod-exp"), &st.podcastExp, a.t("siteSettings.podcastExplicit"))
							})
						case siteSettingsTabTemplates:
							a.viewIntTemplates(c)
						case siteSettingsTabDeploy:
							a.viewIntDeploy(c)
						case siteSettingsTabServices:
							a.viewIntServices(c)
						case siteSettingsTabX:
							a.viewIntX(c)
						case siteSettingsTabTelegram:
							a.viewIntTelegram(c)
						case siteSettingsTabAggregate:
							a.viewIntAggregate(c)
						case siteSettingsTabWeb3:
							a.viewIntWeb3(c)
						}

						if st.err != "" {
							ui.Text(c, st.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
						} else if m.err != "" {
							ui.Text(c, m.err).Font(macPlanetFont).FontSize(12).TextColor(pal.danger)
						} else if m.msg != "" {
							ui.Text(c, m.msg).Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
						}
					})
				})

				// Mac Sheet bottom action bar with 0.5px top border.
				isCoreTab := !searching && (st.tab == siteSettingsTabBasic || st.tab == siteSettingsTabProfile ||
					st.tab == siteSettingsTabCustomCode || st.tab == siteSettingsTabPodcast || st.tab == siteSettingsTabPublish)
				prefsHairlineH(c, pal)
				ui.Row(c).Padding(12, 28, 12, 28).Gap(8).AlignItems(ui.Center).Justify(ui.SpaceBetween).Background(pal.prefsFooter).Shrink(0).AlignSelf(ui.Stretch).Children(func() {
					if isCoreTab {
						ui.Text(c, a.t("siteSettings.republishScopeNote")).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsMuted).NoWrap()
					} else {
						ui.Box(c)
					}
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if isCoreTab {
							if macSecondaryButton(c, "ss-cancel", a.t("common.close"), pal) {
								st.open = false
							}
							if macPrimaryButton(c, "ss-save", a.t("siteSettings.saveBtn"), pal) {
								a.saveSiteSettings()
							}
						} else {
							if macSecondaryButton(c, "ss-close", a.t("common.close"), pal) {
								st.open = false
							}
						}
					})
				})
			})
		})
	})

	if macConfirmSheet(c, &st.removeConfirm,
		a.t("siteSettings.trashConfirmTitle"),
		a.tf("siteSettings.trashConfirmMsg", a.fallbackTitle(st.removeSiteName)),
		a.t("common.cancel"), a.t("siteSettings.moveToTrashBtn"), true, pal) {
		if err := a.backend.RemoveSite(st.siteID); err != nil {
			st.err = err.Error()
		} else {
			st.open = false
			if a.selectedSiteID == st.siteID {
				a.selectedSiteID = ""
				a.mode = viewOverview
			}
			a.Reload()
			a.setStatus(a.t("status.siteMovedToTrash"))
		}
	}

	a.viewIntegrationsSecondarySheets(c)
}

func (a *App) saveSiteSettings() {
	st := &a.siteSettings
	var err error
	switch st.tab {
	case siteSettingsTabBasic:
		if strings.TrimSpace(st.name) == "" {
			st.err = a.t("siteSettings.nameRequired")
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
		// Preserve persisted publish behavior flags when saving Custom Code.
		pub, pre := st.publishIPNS, st.prewarmPost
		if cur := a.findSiteByID(st.siteID); cur != nil {
			pub = cur.ShouldPublishIPNS()
			pre = cur.ShouldPrewarmNewPost()
		}
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
	case siteSettingsTabPublish:
		// Preserve persisted custom code fields when saving Publish Behavior.
		pub, pre := st.publishIPNS, st.prewarmPost
		code := store.SiteCustomCode{
			HeadEnabled:      st.headEnabled,
			Head:             st.head,
			BodyStartEnabled: st.bodyStartOn,
			BodyStart:        st.bodyStart,
			BodyEndEnabled:   st.bodyEndOn,
			BodyEnd:          st.bodyEnd,
			PublishAsIPNS:    &pub,
			PrewarmNewPost:   &pre,
		}
		if cur := a.findSiteByID(st.siteID); cur != nil {
			code.HeadEnabled = cur.CustomCodeHeadEnabled
			code.Head = cur.CustomCodeHead
			code.BodyStartEnabled = cur.CustomCodeBodyStartEnabled
			code.BodyStart = cur.CustomCodeBodyStart
			code.BodyEndEnabled = cur.CustomCodeBodyEndEnabled
			code.BodyEnd = cur.CustomCodeBodyEnd
		}
		raw, _ := json.Marshal(code)
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
	a.Reload()
	a.setStatus(a.t("status.siteSettingsSaved"))
}

func (a *App) triggerPublishSite(site store.Site) {
	res, err := a.backend.PublishSite(site.ID)
	a.publishModal = publishModalState{
		open:     true,
		siteName: a.fallbackTitle(site.Name),
		result:   res,
	}
	if err != nil {
		a.publishModal.err = err.Error()
		return
	}
	a.Reload()
	a.setStatus(a.tf("status.sitePublished", a.fallbackTitle(site.Name)))
}

func (a *App) viewPublishModal(c *ui.Context) {
	pal := a.pal
	pm := &a.publishModal
	if macSheet(c, &pm.open, macSheetProps{
		Title:        a.tf("publishModal.title", pm.siteName),
		Width:        500,
		PrimaryText:  a.t("common.done"),
		CloseTooltip: a.t("common.close"),
	}, pal, func() {
		if pm.err != "" {
			ui.Text(c, a.tf("publishModal.failed", pm.err)).Font(macPlanetFont).FontSize(13).TextColor(pal.danger)
		} else {
			ui.Column(c).Padding(12).Gap(6).Background(pal.subtle).Border(0.5, pal.line).Radius(6).AlignSelf(ui.Stretch).Children(func() {
				ui.Text(c, "CID: "+pm.result.CID).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
				if pm.result.IPNS != "" {
					ui.Text(c, "IPNS: "+pm.result.IPNS).Font(macPlanetFont).FontSize(12).TextColor(pal.ink)
				}
				ui.Text(c, a.tf("publishModal.revision", pm.result.Revision)).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
				if pm.result.Pending {
					ui.Text(c, a.t("publishModal.ipnsPending")).Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
				if pm.result.PinnableWarning != "" {
					ui.Text(c, a.tf("publishModal.pinnableWarn", pm.result.PinnableWarning)).Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
				if pm.result.TelegramWarning != "" {
					ui.Text(c, a.tf("publishModal.telegramWarn", pm.result.TelegramWarning)).Font(macPlanetFont).FontSize(12).TextColor(pal.amber)
				}
			})
		}
	}) {
		pm.open = false
	}
}
