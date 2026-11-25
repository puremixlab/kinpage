package nativeui

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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
	nextFile string
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
	return f.SaveDraftWithTags(siteID, draftID, articleID, title, content, "{}")
}

func (f *fakeBackend) SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (store.Draft, error) {
	var tags map[string]string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	now := time.Now()
	for i := range f.drafts {
		if f.drafts[i].ID == draftID && draftID != "" {
			f.drafts[i].Title = title
			f.drafts[i].Content = content
			f.drafts[i].Tags = tags
			f.drafts[i].Modified = now
			return f.drafts[i], nil
		}
	}
	f.nextID++
	d := store.Draft{
		ID: fmt.Sprintf("draft-%d", f.nextID), SiteID: siteID, ArticleID: articleID,
		Title: title, Content: content, Tags: tags, Created: now, Modified: now,
	}
	f.drafts = append(f.drafts, d)
	return d, nil
}

func (f *fakeBackend) PublishDraft(siteID, draftID, title, content string) (store.Article, error) {
	return f.PublishDraftWithTags(siteID, draftID, title, content, "{}")
}

func (f *fakeBackend) PublishDraftWithTags(siteID, draftID, title, content, tagsJSON string) (store.Article, error) {
	var tags map[string]string
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	now := time.Now()
	if draftID != "" {
		_ = f.DeleteDraft(siteID, draftID)
	}
	f.nextID++
	art := store.Article{
		ID: fmt.Sprintf("art-%d", f.nextID), SiteID: siteID,
		Title: title, Content: content, Tags: tags, Created: now, Modified: now,
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

func (f *fakeBackend) AddDraftAttachment(_ context.Context, siteID, draftID string) (store.Draft, error) {
	name := f.nextFile
	if name == "" {
		name = "cover.png"
	}
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].Attachments = append(f.drafts[i].Attachments, name)
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) RemoveDraftAttachment(siteID, draftID, name string) (store.Draft, error) {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].Attachments = slices.DeleteFunc(f.drafts[i].Attachments, func(s string) bool { return s == name })
			if f.drafts[i].HeroImage == name {
				f.drafts[i].HeroImage = ""
			}
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) SetDraftHeroImage(siteID, draftID, name string) (store.Draft, error) {
	for i := range f.drafts {
		if f.drafts[i].SiteID == siteID && f.drafts[i].ID == draftID {
			f.drafts[i].HeroImage = name
			return f.drafts[i], nil
		}
	}
	return store.Draft{}, fmt.Errorf("draft not found")
}

func (f *fakeBackend) SetArticlePinned(siteID, articleID string, pinned bool) (store.Article, error) {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			if pinned {
				now := time.Now()
				f.articles[i].Pinned = &now
			} else {
				f.articles[i].Pinned = nil
			}
			return f.articles[i], nil
		}
	}
	return store.Article{}, fmt.Errorf("article not found")
}

func (f *fakeBackend) UpdateArticleSettings(siteID, articleID, slug string, articleType int, externalLink string, inNavigation bool, weight int) (store.Article, error) {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			f.articles[i].Slug = slug
			f.articles[i].ArticleType = articleType
			f.articles[i].ExternalLink = externalLink
			f.articles[i].IsIncludedInNavigation = inNavigation
			f.articles[i].NavigationWeight = weight
			return f.articles[i], nil
		}
	}
	return store.Article{}, fmt.Errorf("article not found")
}

func (f *fakeBackend) DeleteArticle(siteID, articleID string) error {
	for i := range f.articles {
		if f.articles[i].SiteID == siteID && f.articles[i].ID == articleID {
			f.articles = append(f.articles[:i], f.articles[i+1:]...)
			return nil
		}
	}
	return nil
}

func (f *fakeBackend) QuickPost(siteID, title, content, _, _ string) (store.Article, error) {
	now := time.Now()
	f.nextID++
	art := store.Article{
		ID: fmt.Sprintf("qp-%d", f.nextID), SiteID: siteID,
		Title: title, Content: content, Created: now, Modified: now,
	}
	f.articles = append(f.articles, art)
	return art, nil
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

	if err := tt.Click("我的星球"); err != nil {
		t.Fatalf("failed to click site in sidebar: %v", err)
	}
	if app.mode != viewSite || app.selectedSiteID != "s1" {
		t.Fatalf("expected site s1 selected, got mode=%v site=%q", app.mode, app.selectedSiteID)
	}

	if err := tt.Click("✎ 未完成草稿"); err != nil {
		t.Fatalf("failed to click draft: %v", err)
	}
	if !app.editing || app.activeDraftID != "d1" {
		t.Fatalf("expected editing draft d1, got editing=%v draft=%q", app.editing, app.activeDraftID)
	}

	// Open metadata panel, add tag and attachment, set hero image, insert markdown.
	if err := tt.Click("元数据"); err != nil {
		t.Fatalf("failed to toggle metadata panel: %v", err)
	}
	app.newTagInput = "Go Native"
	tt.Frame()
	if err := tt.Click("+ 标签"); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}
	if app.editorTags["go-native"] != "Go Native" {
		t.Fatalf("expected tag go-native, got %+v", app.editorTags)
	}

	backend.nextFile = "hero.png"
	if err := tt.Click("+ 添加附件"); err != nil {
		t.Fatalf("failed to add attachment: %v", err)
	}
	if len(app.editorAttachments) != 1 || app.editorAttachments[0] != "hero.png" {
		t.Fatalf("expected hero.png attachment, got %+v", app.editorAttachments)
	}
	if err := tt.Click("设为封面"); err != nil {
		t.Fatalf("failed to set hero image: %v", err)
	}
	if app.editorHeroImage != "hero.png" {
		t.Fatalf("expected hero image hero.png, got %q", app.editorHeroImage)
	}
	if err := tt.Click("插入"); err != nil {
		t.Fatalf("failed to insert attachment markdown: %v", err)
	}
	if !strings.Contains(app.editorBuffer.String(), "![hero.png](hero.png)") {
		t.Fatalf("expected inserted image markdown, got %q", app.editorBuffer.String())
	}

	// Modify title and save draft with tags.
	app.editorTitle = "更新后的标题"
	tt.Frame()
	if err := tt.Click("保存草稿"); err != nil {
		t.Fatalf("failed to click save draft: %v", err)
	}
	if backend.drafts[0].Title != "更新后的标题" || backend.drafts[0].Tags["go-native"] != "Go Native" {
		t.Fatalf("expected saved draft title and tags, got %+v", backend.drafts[0])
	}

	if err := tt.Click("预览"); err != nil {
		t.Fatalf("failed to switch to preview mode: %v", err)
	}
	if app.editorPaneMode != editorPaneView {
		t.Fatalf("expected preview mode, got %d", app.editorPaneMode)
	}

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

func TestNativeUIUnsavedGuardAndArticleActions(t *testing.T) {
	now := time.Now()
	backend := &fakeBackend{
		sites: []store.Site{{ID: "s1", Name: "主站", Created: now}},
		articles: []store.Article{
			{ID: "a1", SiteID: "s1", Title: "第一篇", Content: "内容一", Created: now, Modified: now},
			{ID: "a2", SiteID: "s1", Title: "第二篇", Content: "内容二", Created: now, Modified: now},
		},
	}

	app := New(backend)
	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	// Quick Post from sidebar.
	if err := tt.Click("⚡ 快捷发文"); err != nil {
		t.Fatalf("failed to open quick post: %v", err)
	}
	app.quickPostTitle = "快讯"
	app.quickPostContent = "刚刚发布"
	tt.Frame()
	if err := tt.Click("立即发布"); err != nil {
		t.Fatalf("failed to submit quick post: %v", err)
	}
	if len(backend.articles) != 3 {
		t.Fatalf("expected 3 articles after quick post, got %d", len(backend.articles))
	}

	// Pin current article and verify it sorts first.
	if err := tt.Click("置顶"); err != nil {
		t.Fatalf("failed to pin article: %v", err)
	}
	if app.articles[0].Title != "快讯" || app.articles[0].Pinned == nil {
		t.Fatalf("expected pinned article first, got %+v", app.articles[0])
	}

	// Open article settings and save slug.
	if err := tt.Click("文章设置"); err != nil {
		t.Fatalf("failed to open article settings: %v", err)
	}
	app.articleSettingsSlug = "quick-news"
	app.articleSettingsInNav = true
	app.articleSettingsWeight = "10"
	tt.Frame()
	if err := tt.Click("保存设置"); err != nil {
		t.Fatalf("failed to save article settings: %v", err)
	}
	if app.currentArticle().Slug != "quick-news" || !app.currentArticle().IsIncludedInNavigation || app.currentArticle().NavigationWeight != 10 {
		t.Fatalf("expected updated article settings, got %+v", app.currentArticle())
	}

	// Enter editor, dirty it, click 总览 -> should open unsaved guard.
	if err := tt.Click("编辑文章"); err != nil {
		t.Fatalf("failed to enter editor: %v", err)
	}
	app.editorTitle = "未保存的修改"
	tt.Frame()
	if err := tt.Click("总览"); err != nil {
		t.Fatalf("failed to click overview: %v", err)
	}
	if !app.unsavedDialogOpen || app.mode != viewSite {
		t.Fatalf("expected unsaved dialog to block navigation, open=%v mode=%v", app.unsavedDialogOpen, app.mode)
	}
	if err := tt.Click("放弃更改"); err != nil {
		t.Fatalf("failed to click discard in unsaved dialog: %v", err)
	}
	if app.mode != viewOverview || app.editing {
		t.Fatalf("expected overview after discard, mode=%v editing=%v", app.mode, app.editing)
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
