package nativeui

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
)

type settingsSearchItem struct {
	tab        int
	targetID   string
	groupKey   string
	tabKey     string
	titleKey   string
	summaryKey string
	keywords   []string
	opensModal string
}

func globalSettingsSearchItems() []settingsSearchItem {
	return []settingsSearchItem{
		{
			tab: toolsTabGeneral, targetID: "global-language",
			groupKey: "settings.groupGeneral", tabKey: "settings.tabGeneral",
			titleKey: "settings.languageTitle", summaryKey: "settings.languageDesc",
			keywords: []string{"language", "locale", "lang", "语言", "中文", "英文", "english"},
		},
		{
			tab: toolsTabGeneral, targetID: "global-sleep",
			groupKey: "settings.groupGeneral", tabKey: "settings.tabGeneral",
			titleKey: "menu.preventSleep", summaryKey: "settings.search.sleepSummary",
			keywords: []string{"sleep", "awake", "idle", "休眠", "防休眠", "唤醒", "系统"},
		},
		{
			tab: toolsTabDataRepo, targetID: "global-data-repo",
			groupKey: "settings.groupData", tabKey: "settings.tabDataRepo",
			titleKey: "settings.dataDirTitle", summaryKey: "settings.dataDirDesc",
			keywords: []string{"data", "repo", "directory", "path", "migrate", "refresh", "数据仓库", "目录", "路径", "迁移", "刷新"},
		},
		{
			tab: toolsTabFolders, targetID: "global-folders",
			groupKey: "settings.groupData", tabKey: "menu.publishedFolders",
			titleKey: "menu.publishedFolders", summaryKey: "settings.folders.emptyDesc",
			keywords: []string{"folder", "publish", "ipfs", "文件夹", "发布文件夹", "本地目录"},
		},
		{
			tab: toolsTabOffline, targetID: "global-offline",
			groupKey: "settings.groupData", tabKey: "menu.offlineStorage",
			titleKey: "menu.offlineStorage", summaryKey: "settings.search.offlineSummary",
			keywords: []string{"offline", "cache", "media", "storage", "离线", "缓存", "媒体", "存储"},
		},
		{
			tab: toolsTabTrash, targetID: "global-trash",
			groupKey: "settings.groupData", tabKey: "menu.trash",
			titleKey: "menu.trash", summaryKey: "settings.search.trashSummary",
			keywords: []string{"trash", "removed", "restore", "delete", "回收站", "已删除", "恢复"},
		},
		{
			tab: toolsTabAPI, targetID: "global-api-port",
			groupKey: "settings.groupTools", tabKey: "menu.localAPI",
			titleKey: "settings.search.apiPortTitle", summaryKey: "settings.api.emptyDesc",
			keywords:   []string{"api", "rest", "port", "http", "auth", "passcode", "端口", "本地", "接口", "认证", "监听"},
			opensModal: "api",
		},
		{
			tab: toolsTabKeys, targetID: "global-keys",
			groupKey: "settings.groupTools", tabKey: "menu.keyManager",
			titleKey: "menu.keyManager", summaryKey: "tools.keys.empty",
			keywords: []string{"key", "ipns", "import", "export", "密钥", "导入", "导出"},
		},
		{
			tab: toolsTabLogs, targetID: "global-logs",
			groupKey: "settings.groupTools", tabKey: "menu.appLogs",
			titleKey: "menu.appLogs", summaryKey: "settings.search.logsSummary",
			keywords: []string{"log", "logs", "debug", "error", "日志", "诊断", "报错"},
		},
		{
			tab: toolsTabAI, targetID: "global-ai-service",
			groupKey: "settings.groupExtensions", tabKey: "nav.aiAssistant",
			titleKey: "nav.aiAssistant", summaryKey: "settings.ai.emptyDesc",
			keywords:   []string{"ai", "llm", "model", "openai", "ollama", "token", "模型", "助手", "智能", "服务地址"},
			opensModal: "ai",
		},
		{
			tab: toolsTabTip, targetID: "global-tip",
			groupKey: "settings.groupExtensions", tabKey: "settings.tabTip",
			titleKey: "settings.tabTip", summaryKey: "settings.tip.desc",
			keywords: []string{"tip", "donate", "eth", "打赏", "创作者", "赞助"},
		},
	}
}

func siteSettingsSearchItems() []settingsSearchItem {
	return []settingsSearchItem{
		{
			tab: siteSettingsTabBasic, targetID: "site-avatar",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navBasic",
			titleKey: "siteSettings.avatarTitle", summaryKey: "siteSettings.avatarPublishNote",
			keywords: []string{"avatar", "icon", "photo", "头像", "图标"},
		},
		{
			tab: siteSettingsTabBasic, targetID: "site-basic",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navBasic",
			titleKey: "siteSettings.navBasic", summaryKey: "siteSettings.search.basicSummary",
			keywords: []string{"name", "about", "author", "slug", "seo", "index", "名称", "简介", "作者", "别名", "收录", "基本资料"},
		},
		{
			tab: siteSettingsTabBasic, targetID: "site-danger",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navBasic",
			titleKey: "siteSettings.moveToTrash", summaryKey: "siteSettings.dangerDesc",
			keywords: []string{"trash", "delete", "remove", "danger", "回收站", "删除", "移除"},
		},
		{
			tab: siteSettingsTabProfile, targetID: "site-social",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navSocial",
			titleKey: "siteSettings.titleSocial", summaryKey: "siteSettings.search.socialSummary",
			keywords: []string{"domain", "twitter", "github", "telegram", "mastodon", "discord", "域名", "社交"},
		},
		{
			tab: siteSettingsTabTemplates, targetID: "site-templates",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navTemplates",
			titleKey: "siteSettings.navTemplates", summaryKey: "siteSettings.search.templatesSummary",
			keywords: []string{"template", "theme", "preview", "模板", "主题", "预览"},
		},
		{
			tab: siteSettingsTabCustomCode, targetID: "site-custom-code",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navCustomCode",
			titleKey: "siteSettings.navCustomCode", summaryKey: "siteSettings.search.customCodeSummary",
			keywords: []string{"code", "html", "head", "body", "script", "css", "自定义代码", "脚本"},
		},
		{
			tab: siteSettingsTabPodcast, targetID: "site-podcast",
			groupKey: "siteSettings.groupSite", tabKey: "siteSettings.navPodcast",
			titleKey: "siteSettings.navPodcast", summaryKey: "siteSettings.search.podcastSummary",
			keywords: []string{"podcast", "audio", "rss", "explicit", "播客", "音频"},
		},
		{
			tab: siteSettingsTabPublish, targetID: "site-publish-behavior",
			groupKey: "siteSettings.groupPublish", tabKey: "siteSettings.navPublishBehavior",
			titleKey: "siteSettings.navPublishBehavior", summaryKey: "siteSettings.publishBehaviorDesc",
			keywords: []string{"ipns", "prewarm", "gateway", "publish", "发布", "发布行为", "预热", "网关"},
		},
		{
			tab: siteSettingsTabDeploy, targetID: "site-deploy",
			groupKey: "siteSettings.groupPublish", tabKey: "siteSettings.navDeploy",
			titleKey: "siteSettings.navDeploy", summaryKey: "siteSettings.search.deploySummary",
			keywords: []string{"deploy", "cloudflare", "pages", "ssh", "rsync", "部署", "服务器"},
		},
		{
			tab: siteSettingsTabServices, targetID: "site-services",
			groupKey: "siteSettings.groupPublish", tabKey: "siteSettings.navServices",
			titleKey: "siteSettings.titleServices", summaryKey: "siteSettings.search.servicesSummary",
			keywords: []string{"plausible", "filebase", "pinnable", "dweb", "pin", "analytics", "固定", "统计"},
		},
		{
			tab: siteSettingsTabX, targetID: "site-x",
			groupKey: "siteSettings.groupSync", tabKey: "siteSettings.navX",
			titleKey: "siteSettings.navX", summaryKey: "int.x.emptyDesc",
			keywords: []string{"x", "twitter", "sync", "同步", "推特"},
		},
		{
			tab: siteSettingsTabTelegram, targetID: "site-telegram",
			groupKey: "siteSettings.groupSync", tabKey: "siteSettings.navTelegram",
			titleKey: "siteSettings.titleTelegram", summaryKey: "int.tg.emptyDesc",
			keywords: []string{"telegram", "bot", "channel", "tg", "电报", "同步"},
		},
		{
			tab: siteSettingsTabAggregate, targetID: "site-aggregate",
			groupKey: "siteSettings.groupSync", tabKey: "siteSettings.navAggregate",
			titleKey: "siteSettings.titleAggregate", summaryKey: "int.agg.emptyDesc",
			keywords: []string{"aggregate", "rss", "feed", "聚合", "订阅源"},
		},
		{
			tab: siteSettingsTabWeb3, targetID: "site-web3",
			groupKey: "siteSettings.groupSync", tabKey: "siteSettings.navWeb3",
			titleKey: "siteSettings.titleWeb3", summaryKey: "int.web3.emptyDesc",
			keywords: []string{"web3", "eth", "donate", "juicebox", "farcaster", "捐赠", "钱包"},
		},
	}
}

func (a *App) matchSettingsSearch(query string, items []settingsSearchItem) []settingsSearchItem {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []settingsSearchItem
	for _, it := range items {
		haystacks := []string{
			a.t(it.titleKey),
			a.t(it.summaryKey),
			a.t(it.groupKey),
			a.t(it.tabKey),
		}
		matched := false
		for _, h := range haystacks {
			if strings.Contains(strings.ToLower(h), q) {
				matched = true
				break
			}
		}
		if !matched {
			for _, kw := range it.keywords {
				if strings.Contains(strings.ToLower(kw), q) {
					matched = true
					break
				}
			}
		}
		if matched {
			out = append(out, it)
		}
	}
	return out
}

func (a *App) viewSettingsSearchResults(
	c *ui.Context,
	scopeKey string,
	query string,
	matches []settingsSearchItem,
	suggestions []string,
	onSelect func(settingsSearchItem),
	onPickSuggestion func(string),
) {
	pal := a.pal
	ui.Column(c.Key(scopeKey + "-search-results")).Gap(10).AlignSelf(ui.Stretch).Children(func() {
		ui.Text(c, a.tf("settings.search.resultCount", len(matches), strings.TrimSpace(query))).
			Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)

		if len(matches) == 0 {
			macInsetCard(c, pal, func() {
				ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingY(18).Gap(10).Children(func() {
					ui.Icon(c, lucide.Search()).Size(20, 20).TextColor(pal.prefsMuted)
					ui.Text(c, a.t("settings.search.emptyTitle")).
						Font(macPlanetFont).FontSize(13.5).FontWeight(600).TextColor(pal.prefsInk)
					ui.Text(c, a.t("settings.search.emptyHint")).
						Font(macPlanetFont).FontSize(12).TextColor(pal.prefsMuted)
					if len(suggestions) > 0 {
						ui.Row(c).Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
							for i, sug := range suggestions {
								s := sug
								if macSecondaryButton(c, scopeKey+"-sug-"+s+string(rune('0'+i)), s, pal) {
									onPickSuggestion(s)
								}
							}
						})
					}
				})
			})
			return
		}

		macInsetCard(c, pal, func() {
			for i, item := range matches {
				it := item
				if i > 0 {
					prefsHairlineH(c, pal)
				}
				title := a.t(it.titleKey)
				crumb := a.t(it.groupKey) + " · " + a.t(it.tabKey)
				row := ui.ButtonBase(c.Key(scopeKey+"-sr-"+it.targetID)).Label(title+" ("+crumb+")").
					AlignSelf(ui.Stretch).Padding(8, 10, 8, 10).Radius(8).
					Cursor(ui.CursorPointer).Transition(rowHoverTransition)
				if row.Hovered() {
					row.Background(pal.hover)
				}
				row.Children(func() {
					ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(12).Children(func() {
						ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
								ui.Text(c, title).Font(macPlanetFont).FontSize(13).FontWeight(600).TextColor(pal.prefsInk)
								ui.Box(c).Padding(1, 6, 1, 6).Radius(4).Background(pal.hover).Children(func() {
									ui.Text(c, crumb).Font(macPlanetFont).FontSize(10.5).TextColor(pal.prefsMuted).NoWrap()
								})
							})
							if summary := a.t(it.summaryKey); summary != "" {
								ui.Text(c, summary).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsSecondary).MaxLines(2).Ellipsis("…")
							}
						})
						ui.Icon(c, lucide.ChevronRight()).Size(14, 14).TextColor(pal.prefsMuted).Shrink(0)
					})
				})
				if row.Clicked() {
					onSelect(it)
				}
			}
		})
	})
}

func macSettingsReturnButton(c *ui.Context, key, label string, pal macPalette) bool {
	btn := ui.ButtonBase(c.Key(key)).Label(label).
		AlignSelf(ui.Stretch).Padding(5, 8, 5, 8).Radius(6).
		Background(pal.hover).Border(0.5, pal.prefsCardBorder).
		Cursor(ui.CursorPointer).Transition(rowHoverTransition)
	if btn.Hovered() {
		btn.Background(pal.active)
	}
	btn.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Icon(c, lucide.ArrowLeft()).Size(12.5, 12.5).TextColor(pal.prefsInk)
			ui.Text(c, label).Font(macPlanetFont).FontSize(12).FontWeight(500).TextColor(pal.prefsInk).NoWrap()
		})
	})
	return btn.Clicked()
}

func macHighlightCard(c *ui.Context, pal macPalette, highlighted bool, fn func()) {
	card := ui.Column(c).Gap(12).Padding(16, 18, 16, 18).Radius(10).
		Background(pal.prefsCard).
		Shadow(0, 1, 3, 0, pal.prefsCardShadow).
		AlignSelf(ui.Stretch)
	if highlighted {
		card.Border(0.5, pal.accent).Shadow(0, 0, 0, 2, pal.accent.Alpha(0.18))
	} else {
		card.Border(0.5, pal.prefsCardBorder)
	}
	card.Children(fn)
}

func settingsPrerequisiteNote(c *ui.Context, pal macPalette, message string) {
	ui.Row(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).Gap(8).
		Padding(8, 12, 8, 12).Radius(8).
		Background(pal.hover).Border(0.5, pal.prefsCardBorder).Children(func() {
		ui.Icon(c, lucide.Info()).Size(13.5, 13.5).TextColor(pal.prefsMuted).Shrink(0)
		ui.Text(c, message).Font(macPlanetFont).FontSize(11.5).TextColor(pal.prefsSecondary).Grow(1).MinWidth(0)
	})
}
