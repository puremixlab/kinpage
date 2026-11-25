package nativeui

import "windows/internal/store"

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
	PublishDraft(siteID, draftID, title, content string) (store.Article, error)
	DeleteDraft(siteID, draftID string) error
}
