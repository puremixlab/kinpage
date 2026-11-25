package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobemd "github.com/lobehub/lobe-mygo/pkg/markdown"
)

func (a *App) viewDetailPane(c *ui.Context) {
	ui.Column(c).Fill().Background(a.pal.canvas).Children(func() {
		a.viewDetailToolbar(c, true)
		if a.statusText != "" {
			a.viewStatusBar(c)
		}
		ui.Column(c).Grow(1).MinHeight(0).AlignSelf(ui.Stretch).Children(func() {
			if a.editing {
				a.viewEditor(c)
			} else {
				a.viewArticleReader(c)
			}
		})
	})
}

func (a *App) viewArticleReader(c *ui.Context) {
	pal := a.pal
	art := a.currentArticle()
	if art == nil {
		ui.Column(c).Fill().Background(pal.canvas).Center().Children(func() {
			display.Empty(c, display.EmptyProps{
				Icon:        lucide.BookOpen(),
				Title:       "选择一篇文章开始阅读",
				Description: "或创建一篇新文章，草稿会先安全保存在本地",
				Action: func() {
					if site := a.currentSite(); site != nil {
						if macPrimaryButton(c, "empty-compose", "写新文章", pal) {
							a.startNewDraft(site.ID)
						}
					}
				},
			})
		})
		return
	}

	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		ui.Scroll(c).TrackScroll(&a.readerScroll).Fill().Background(pal.canvas).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
				ui.Column(c).AlignSelf(ui.Stretch).MaxWidth(740).Gap(12).Children(func() {
					ui.Text(c, fallbackTitle(art.Title)).Font(macPlanetFont).FontSize(28).FontWeight(700).LetterSpacing(-0.02).TextColor(pal.ink)
					var meta []string
					if site := a.currentSite(); site != nil {
						meta = append(meta, fallbackTitle(site.Name))
					}
					if !art.Modified.IsZero() {
						meta = append(meta, formatSlashDate(art.Modified))
					}
					if art.Pinned != nil {
						meta = append(meta, "已置顶")
					}
					if art.Slug != "" {
						meta = append(meta, "/"+art.Slug)
					}
					if len(art.Tags) > 0 {
						meta = append(meta, fmt.Sprintf("%d 标签", len(art.Tags)))
					}
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Gap(8).Wrap().AlignSelf(ui.Stretch).Children(func() {
						if len(meta) > 0 {
							ui.Text(c, strings.Join(meta, "   ·   ")).Font(macPlanetFont).FontSize(12).TextColor(pal.muted)
						}
						ui.Row(c).Gap(4).AlignItems(ui.Center).Shrink(0).Children(func() {
							pinLabel := "置顶"
							if art.Pinned != nil {
								pinLabel = "取消置顶"
							}
							speakLabel := "朗读"
							if a.speaker != nil && a.speaker.Speaking() {
								speakLabel = "停止朗读"
							}
							if ghostIconButton(c, "reader-speak", speakLabel, speakLabel, pal) {
								a.toggleSpeak(art.Title, art.Content)
							}
							if ghostIconButton(c, "reader-related", "相关文章", "相关文章", pal) {
								a.openRelatedArticles(art.SiteID, art.ID, "", "", art.Title)
							}
							if ghostIconButton(c, "reader-pin", pinLabel, pinLabel, pal) {
								a.toggleArticlePinned(*art)
							}
							if ghostIconButton(c, "reader-settings", "文章设置", "文章设置", pal) {
								a.openArticleSettings(*art)
							}
							if ghostIconButton(c, "reader-delete", "删除", "删除文章", pal) {
								a.deleteArticleID = art.ID
								a.deleteArticleTitle = art.Title
								a.deleteArticleOpen = true
							}
						})
					})
					hairlineH(c, pal)
					lobemd.Markdown(c, art.Content, lobemd.MarkdownProps{
						FontSize:              15,
						LineHeight:            1.8,
						HeaderMultiple:        0.65,
						MarginMultiple:        1.2,
						FullFeaturedCodeBlock: true,
					})
				})
			})
		})
	})
}
