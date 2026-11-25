package nativeui

import (
	"fmt"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"windows/internal/store"
)

type fakeBackend struct {
	sites    []store.Site
	articles []store.Article
	drafts   []store.Draft
	activity []store.DailyActivity
	nextID   int
}

func (f *fakeBackend) Dashboard() (store.DashboardSnapshot, error) {
	snap := store.DashboardSnapshot{
		SiteCount:    len(f.sites),
		ArticleCount: len(f.articles),
		DraftCount:   len(f.drafts),
	}
	for _, s := range f.sites {
		snap.Sites = append(snap.Sites, store.DashboardSite{ID: s.ID, Name: s.Name})
	}
	for _, d := range f.drafts {
		snap.RecentDrafts = append(snap.RecentDrafts, store.DashboardEntry{
			ID: d.ID, SiteID: d.SiteID, SiteName: f.siteName(d.SiteID), Title: d.Title, Modified: d.Modified,
		})
	}
	for _, a := range f.articles {
		snap.RecentArticles = append(snap.RecentArticles, store.DashboardEntry{
			ID: a.ID, SiteID: a.SiteID, SiteName: f.siteName(a.SiteID), Title: a.Title, Modified: a.Modified,
		})
	}
	return snap, nil
}

func (f *fakeBackend) siteName(id string) string {
	for _, s := range f.sites {
		if s.ID == id {
			return s.Name
		}
	}
	return ""
}

func (f *fakeBackend) WritingActivity() ([]store.DailyActivity, error) {
	return append([]store.DailyActivity(nil), f.activity...), nil
}

func (f *fakeBackend) Sites() ([]store.Site, error) {
	return append([]store.Site(nil), f.sites...), nil
}

func (f *fakeBackend) CreateSite(name, about string) (store.Site, error) {
	f.nextID++
	s := store.Site{ID: fmt.Sprintf("site-%d", f.nextID), Name: name, About: about, Created: time.Now()}
	f.sites = append(f.sites, s)
	return s, nil
}

func (f *fakeBackend) Articles(siteID string) ([]store.Article, error) {
	var out []store.Article
	for _, a := range f.articles {
		if a.SiteID == siteID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeBackend) Drafts(siteID string) ([]store.Draft, error) {
	var out []store.Draft
	for _, d := range f.drafts {
		if d.SiteID == siteID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f *fakeBackend) SaveDraft(siteID, draftID, articleID, title, content string) (store.Draft, error) {
	now := time.Now()
	for i := range f.drafts {
		if f.drafts[i].ID == draftID && draftID != "" {
			f.drafts[i].Title = title
			f.drafts[i].Content = content
			f.drafts[i].Modified = now
			return f.drafts[i], nil
		}
	}
	f.nextID++
	d := store.Draft{
		ID: fmt.Sprintf("draft-%d", f.nextID), SiteID: siteID, ArticleID: articleID,
		Title: title, Content: content, Created: now, Modified: now,
	}
	f.drafts = append(f.drafts, d)
	return d, nil
}

func (f *fakeBackend) PublishDraft(siteID, draftID, title, content string) (store.Article, error) {
	now := time.Now()
	if draftID != "" {
		_ = f.DeleteDraft(siteID, draftID)
	}
	f.nextID++
	art := store.Article{
		ID: fmt.Sprintf("art-%d", f.nextID), SiteID: siteID,
		Title: title, Content: content, Created: now, Modified: now,
	}
	f.articles = append(f.articles, art)
	return art, nil
}

func (f *fakeBackend) DeleteDraft(siteID, draftID string) error {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts = append(f.drafts[:i], f.drafts[i+1:]...)
			return nil
		}
	}
	return nil
}

func TestNativeUIDashboardAndEditorFlow(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "我的星球", Created: now}},
		articles: []store.Article{{
			ID: "a1", SiteID: "s1", Title: "欢迎文章", Content: "# 你好\n\n这是**原生渲染**的 Markdown。",
			Created: now, Modified: now,
		}},
		drafts: []store.Draft{{
			ID: "d1", SiteID: "s1", Title: "未完成草稿", Content: "## 草稿正文\n\n- 第一项\n- 第二项",
			Created: now, Modified: now,
		}},
		activity: []store.DailyActivity{{Date: now.Format("2006-01-02"), Articles: 1, Drafts: 1}},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// Click into site from sidebar.
	if err := tt.Click("我的星球"); err != nil {
		t.Fatalf("failed to click site in sidebar: %v", err)
	}
	if app.mode != viewSite || app.selectedSiteID != "s1" {
		t.Fatalf("expected site s1 selected, got mode=%v site=%q", app.mode, app.selectedSiteID)
	}

	// Open draft from posts panel.
	if err := tt.Click("✎ 未完成草稿"); err != nil {
		t.Fatalf("failed to click draft: %v", err)
	}
	if !app.editing || app.activeDraftID != "d1" {
		t.Fatalf("expected editing draft d1, got editing=%v draft=%q", app.editing, app.activeDraftID)
	}

	// Modify buffer and save draft.
	app.editorTitle = "更新后的标题"
	app.editorBuffer.Set("## 更新内容\n\n测试保存")
	tt.Frame()
	if err := tt.Click("保存草稿"); err != nil {
		t.Fatalf("failed to click save draft: %v", err)
	}
	if backend.drafts[0].Title != "更新后的标题" {
		t.Fatalf("expected saved title updated, got %q", backend.drafts[0].Title)
	}

	// Switch to preview mode and render without panicking.
	if err := tt.Click("预览"); err != nil {
		t.Fatalf("failed to switch to preview mode: %v", err)
	}
	if app.editorPaneMode != editorPaneView {
		t.Fatalf("expected preview mode, got %d", app.editorPaneMode)
	}

	// Publish draft and verify transition to reader.
	if err := tt.Click("发布"); err != nil {
		t.Fatalf("failed to publish draft: %v", err)
	}
	if app.editing {
		t.Fatal("expected editor to exit after publish")
	}
	if len(backend.articles) != 2 {
		t.Fatalf("expected 2 published articles, got %d", len(backend.articles))
	}
}

func TestNativeUIEmptyStateAndCreateSite(t *testing.T) {
	backend := &fakeBackend{}
	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	if err := tt.Click("创建第一个站点"); err != nil {
		t.Fatalf("failed to open create site modal: %v", err)
	}
	if !app.createSiteOpen {
		t.Fatal("expected createSiteOpen to be true")
	}

	app.newSiteName = "测试新站点"
	tt.Frame()
	if err := tt.Click("创建"); err != nil {
		t.Fatalf("failed to submit create site modal: %v", err)
	}
	if len(backend.sites) != 1 || backend.sites[0].Name != "测试新站点" {
		t.Fatalf("expected created site, got %+v", backend.sites)
	}
	if app.mode != viewSite || app.selectedSiteID != backend.sites[0].ID {
		t.Fatalf("expected switched to new site, got mode=%v site=%q", app.mode, app.selectedSiteID)
	}
}
