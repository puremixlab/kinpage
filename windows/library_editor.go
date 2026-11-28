package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"windows/internal/attachments"
	"windows/internal/preview"
	"windows/internal/store"
)

func (l *Library) Articles(siteID string) ([]store.Article, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.ListArticles(siteID)
}

func (l *Library) Drafts(siteID string) ([]store.Draft, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.ListDrafts(siteID)
}

func (l *Library) SaveDraft(siteID, draftID, articleID, title, content string) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	return s.SaveDraft(siteID, draftID, articleID, title, content)
}

func (l *Library) SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (store.Draft, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Draft{}, err
	}
	return s.SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON)
}

func (l *Library) DeleteDraft(siteID, draftID string) error {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	if err := s.DeleteDraft(siteID, draftID); err != nil {
		return err
	}
	attachments.CleanupDraftFiles(root, s)
	return nil
}

func (l *Library) PublishDraft(siteID, draftID, title, content string) (store.Article, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.PublishDraft(root, s, siteID, draftID, title, content)
}

func (l *Library) PublishDraftWithTags(siteID, draftID, title, content, tagsJSON string) (store.Article, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.PublishDraftWithTags(root, s, siteID, draftID, title, content, tagsJSON)
}

// SiteAvailableTags lists the labels already used across the site's articles
// with per-label counts, mirroring the macOS writer's Previously Used Tags.
func (l *Library) SiteAvailableTags(siteID string) ([]store.TagUsage, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.SiteAvailableTags(siteID)
}

// PreviewDraft supports both private new files and the original article's files.
func (l *Library) PreviewDraft(siteID, draftID, content string) (string, error) {
	l.publicMu.Lock()
	defer l.publicMu.Unlock()
	resolve, err := l.draftPreviewImages(siteID, draftID)
	if err != nil {
		return "", err
	}
	return preview.MarkdownWithImages(content, resolve)
}

func (l *Library) draftPreviewImages(siteID, draftID string) (func(string) ([]byte, string, error), error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	site, articles, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return nil, err
	}
	drafts, err := s.ListDrafts(siteID)
	if err != nil {
		return nil, err
	}
	var draft *store.Draft
	for i := range drafts {
		if drafts[i].ID == draftID {
			draft = &drafts[i]
			break
		}
	}
	if draft == nil {
		return nil, errors.New("草稿不存在")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil, err
	}
	return func(name string) ([]byte, string, error) {
		for _, item := range draft.Attachments {
			if item == name {
				return attachments.ReadDraftPreviewImage(root, *draft, name)
			}
		}
		for _, article := range articles {
			if article.ID == draft.ArticleID {
				for _, removed := range draft.RemovedAttachments {
					if removed == name {
						return nil, "", errors.New("草稿中已移除该附件")
					}
				}
				return attachments.ReadPreviewImage(root, site, article, name)
			}
		}
		return nil, "", errors.New("草稿附件不存在")
	}, nil
}

func (l *Library) SaveArticle(siteID, articleID, title, content string) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.SaveArticle(siteID, articleID, title, content)
}

func (l *Library) UpdateArticleSettings(siteID, articleID, slug string, articleType int, externalLink string, inNavigation bool, weight int) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.UpdateArticleSettings(siteID, articleID, slug, articleType, externalLink, inNavigation, weight)
}

func (l *Library) ToggleArticleTodo(siteID, articleID string, lineIndex int) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.ToggleArticleTodo(siteID, articleID, lineIndex)
}

func (l *Library) SetArticlePinned(siteID, articleID string, pinned bool) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.SetArticlePinned(siteID, articleID, pinned)
}

func (l *Library) UpdateArticleDate(siteID, articleID string, created time.Time) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.UpdateArticleDate(siteID, articleID, created)
}

func (l *Library) SetArticleStar(siteID, articleID string, starred bool, starType int) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.SetArticleStar(siteID, articleID, starred, starType)
}

func (l *Library) MoveArticle(fromSiteID, articleID, toSiteID string) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return store.Article{}, err
	}
	return attachments.MoveArticle(root, s, fromSiteID, articleID, toSiteID)
}

func (l *Library) SaveArticleWithTags(siteID, articleID, title, content, tagsJSON string) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	return s.SaveArticleWithTags(siteID, articleID, title, content, tagsJSON)
}

func (l *Library) Preview(content string) (string, error) {
	return preview.MarkdownWithImages(content, func(string) ([]byte, string, error) { return nil, "", os.ErrNotExist })
}

// PreviewArticle renders the draft with images registered to its saved article.
func (l *Library) PreviewArticle(siteID, articleID, content string) (string, error) {
	resolve, err := l.articlePreviewImages(siteID, articleID)
	if err != nil {
		return "", err
	}
	return preview.MarkdownWithImages(content, resolve)
}

func (l *Library) articlePreviewImages(siteID, articleID string) (func(string) ([]byte, string, error), error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	site, articles, _, err := s.PublicationSnapshot(siteID)
	if err != nil {
		return nil, err
	}
	var article *store.Article
	for i := range articles {
		if articles[i].ID == articleID {
			article = &articles[i]
			break
		}
	}
	if article == nil {
		return nil, errors.New("文章不存在")
	}
	root, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil, err
	}
	return func(name string) ([]byte, string, error) {
		return attachments.ReadPreviewImage(root, site, *article, name)
	}, nil
}

func (l *Library) ReadMarkdownAttachment(siteID, articleID, draftID, name string) ([]byte, string, error) {
	var resolve func(string) ([]byte, string, error)
	var err error
	if draftID != "" {
		resolve, err = l.draftPreviewImages(siteID, draftID)
	} else {
		resolve, err = l.articlePreviewImages(siteID, articleID)
	}
	if err != nil {
		return nil, "", err
	}
	return resolve(name)
}

func (l *Library) DeleteArticle(siteID, articleID string) error {
	s, err := l.open()
	if err != nil {
		return err
	}
	return s.DeleteArticle(siteID, articleID)
}

func (l *Library) SearchArticles(query string) ([]store.SearchResult, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.SearchArticles(query)
}

func (l *Library) RelatedArticles(siteID, articleID, subscriptionURL, itemKey string) ([]store.SearchResult, error) {
	s, err := l.open()
	if err != nil {
		return nil, err
	}
	return s.RelatedArticles(siteID, articleID, subscriptionURL, itemKey, 6)
}

// Quick Post (Phase 4.3)
func (l *Library) QuickPost(siteID, title, content, attachmentName, attachmentBase64 string) (store.Article, error) {
	s, err := l.open()
	if err != nil {
		return store.Article{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "快速动态 " + time.Now().Format("2006-01-02 15:04")
	}
	article, err := s.SaveArticle(siteID, "", title, content)
	if err != nil {
		return store.Article{}, err
	}
	if strings.TrimSpace(attachmentBase64) != "" {
		root, err := mygo.App.Path(mygo.PathUserData)
		if err == nil {
			if updated, e := attachments.AddBytes(root, s, siteID, article.ID, attachmentName, attachmentBase64); e == nil {
				article = updated
				if len(article.Attachments) > 0 {
					last := article.Attachments[len(article.Attachments)-1]
					nextContent := content
					if nextContent != "" {
						nextContent += "\n\n"
					}
					nextContent += fmt.Sprintf("![%s](%s)", last, last)
					if saved, e2 := s.SaveArticle(siteID, article.ID, article.Title, nextContent); e2 == nil {
						article = saved
					}
				}
			}
		}
	}
	return article, nil
}
