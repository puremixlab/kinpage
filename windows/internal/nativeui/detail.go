package nativeui

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/lobehub/lobe-mygo/pkg/display"
	"github.com/lobehub/lobe-mygo/pkg/lucide"
	lobemd "github.com/lobehub/lobe-mygo/pkg/markdown"
	"windows/internal/store"
)

func (a *App) detailColumnWidth(c *ui.Context, maxW float32) float32 {
	winW, _ := c.Size()
	if winW <= 0 {
		return maxW
	}
	used := float32(0)
	if !a.sidebarCollapsed {
		sw := a.sidebarWidth
		if sw < 220 {
			sw = 228
		}
		used += sw
	}
	if !a.postsCollapsed {
		pw := a.postsPanelWidth
		if pw < 240 {
			pw = 280
		}
		used += pw
	}
	avail := winW - used - 72 // 36px left + 36px right
	if avail < 360 {
		avail = 360
	}
	if avail > maxW {
		return maxW
	}
	return avail
}

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
				Title:       a.t("reader.emptyTitle"),
				Description: a.t("reader.emptyDesc"),
				Action: func() {
					if site := a.currentSite(); site != nil {
						if macPrimaryButton(c, "empty-compose", a.t("reader.writeNewPost"), pal) {
							a.startNewDraft(site.ID)
						}
					}
				},
			})
		})
		return
	}

	colW := a.detailColumnWidth(c, 740)
	ui.Column(c).Fill().Background(pal.canvas).Children(func() {
		ui.Scroll(c).TrackScroll(&a.readerScroll).Fill().Background(pal.canvas).Children(func() {
			ui.Column(c).AlignSelf(ui.Stretch).AlignItems(ui.Center).PaddingX(36).PaddingY(28).Children(func() {
				ui.Column(c).Width(colW).Gap(12).Children(func() {
					ui.Text(c, a.fallbackTitle(art.Title)).Font(macPlanetFont).FontSize(28).FontWeight(600).LetterSpacing(-0.015).TextColor(pal.ink)
					var meta []string
					if site := a.currentSite(); site != nil {
						meta = append(meta, a.fallbackTitle(site.Name))
					}
					if !art.Modified.IsZero() {
						meta = append(meta, a.formatReaderDate(art.Modified))
					}
					if art.Pinned != nil {
						meta = append(meta, a.t("reader.pinnedBadge"))
					}
					if art.Slug != "" {
						meta = append(meta, "/"+art.Slug)
					}
					if len(art.Tags) > 0 {
						meta = append(meta, a.tf("reader.tagCount", len(art.Tags)))
					}
					if len(meta) > 0 {
						ui.Text(c, strings.Join(meta, "   ·   ")).Font(macPlanetFont).FontSize(12.5).TextColor(pal.muted)
					}
					hairlineH(c, pal)
					if todos := store.ParseMarkdownTodos(art.Content); len(todos) > 0 {
						doneCount := 0
						for _, td := range todos {
							if td.Checked {
								doneCount++
							}
						}
						macInsetCard(c, pal, func() {
							ui.Text(c, a.tf("reader.todosHeading", doneCount, len(todos))).
								Font(macPlanetFont).FontSize(12).FontWeight(600).TextColor(pal.muted)
							for _, td := range todos {
								item := td
								checked := item.Checked
								ui.Checkbox(c.Key(fmt.Sprintf("todo-%s-%d", art.ID, item.LineIndex)), &checked, item.Text)
								if checked != item.Checked && a.backend != nil {
									if _, err := a.backend.ToggleArticleTodo(art.SiteID, art.ID, item.LineIndex); err != nil {
										a.setError(a.tf("status.todoToggleErr", err.Error()))
									} else {
										a.Reload()
									}
								}
							}
						})
					}
					key := fmt.Sprintf("article-markdown-%s-%s-%d", art.SiteID, art.ID, art.Modified.UnixNano())
					lobemd.Markdown(c.Key(key), art.Content, lobemd.MarkdownProps{
						FontSize:              15,
						LineHeight:            1.8,
						HeaderMultiple:        0.65,
						MarginMultiple:        1.2,
						FullFeaturedCodeBlock: true,
						LoadImage:             a.markdownAttachmentLoader(art.SiteID, art.ID, ""),
					})
				})
			})
		})
	})
}
