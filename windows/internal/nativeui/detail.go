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
						meta = append(meta, formatReaderDate(art.Modified))
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
					if len(meta) > 0 {
						ui.Text(c, strings.Join(meta, "   ·   ")).Font(macPlanetFont).FontSize(12.5).TextColor(pal.muted)
					}
					hairlineH(c, pal)
					lobemd.Markdown(c, art.Content, lobemd.MarkdownProps{
						FontSize:              15,
						LineHeight:            1.8,
						HeaderMultiple:        0.65,
						MarginMultiple:        1.2,
						FullFeaturedCodeBlock: true,
						LoadImage:             a.markdownImageLoader("", ""),
					})
				})
			})
		})
	})
}
