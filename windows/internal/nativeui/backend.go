package nativeui

import (
	"context"

	"windows/internal/store"
)

// Backend abstracts the Planet Library operations needed by the native UI.
// Having this interface lets the native UI run directly against *Library in
// production and against an in-memory stub in headless ui.Tester tests.
type Backend interface {
	Dashboard() (store.DashboardSnapshot, error)
	WritingActivity() ([]store.DailyActivity, error)
	Sites() ([]store.Site, error)
	CreateSite(name, about string) (store.Site, error)
	Articles(siteID string) ([]store.Article, error)
	Drafts(siteID string) ([]store.Draft, error)
	SaveDraft(siteID, draftID, articleID, title, content string) (store.Draft, error)
	SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (store.Draft, error)
	PublishDraft(siteID, draftID, title, content string) (store.Article, error)
	PublishDraftWithTags(siteID, draftID, title, content, tagsJSON string) (store.Article, error)
	DeleteDraft(siteID, draftID string) error
	AddDraftAttachment(ctx context.Context, siteID, draftID string) (store.Draft, error)
	RemoveDraftAttachment(siteID, draftID, name string) (store.Draft, error)
	SetDraftHeroImage(siteID, draftID, name string) (store.Draft, error)
	SetArticlePinned(siteID, articleID string, pinned bool) (store.Article, error)
	UpdateArticleSettings(siteID, articleID, slug string, articleType int, externalLink string, inNavigation bool, weight int) (store.Article, error)
	DeleteArticle(siteID, articleID string) error
	QuickPost(siteID, title, content, attachmentName, attachmentBase64 string) (store.Article, error)
}
