package store

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"windows/internal/feeds"
)

func TestSetSiteArchived(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := s.CreateSite("My Blog", "About")
	if err != nil {
		t.Fatal(err)
	}
	archived, err := s.SetSiteArchived(site.ID, true)
	if err != nil || !archived.Archived {
		t.Fatalf("expected archived site, got %+v, err=%v", archived, err)
	}
	unarchived, err := s.SetSiteArchived(site.ID, false)
	if err != nil || unarchived.Archived {
		t.Fatalf("expected unarchived site, got %+v, err=%v", unarchived, err)
	}
	if err := s.RemoveSite(site.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetSiteArchived(site.ID, true); err == nil {
		t.Fatal("expected error archiving removed site")
	}
}

func TestUpdateArticleDate(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := s.CreateSite("Blog", "")
	if err != nil {
		t.Fatal(err)
	}
	art, err := s.SaveArticle(site.ID, "", "Hello", "Body")
	if err != nil {
		t.Fatal(err)
	}
	customDate := time.Date(2024, 5, 20, 14, 30, 0, 0, time.UTC)
	updated, err := s.UpdateArticleDate(site.ID, art.ID, customDate)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Created.Equal(customDate) {
		t.Fatalf("expected created %v, got %v", customDate, updated.Created)
	}
	if _, err := s.UpdateArticleDate(site.ID, art.ID, time.Time{}); err == nil {
		t.Fatal("expected error for zero date")
	}
}

func TestArticleAndSubscriptionStarTypes(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := s.CreateSite("Blog", "")
	if err != nil {
		t.Fatal(err)
	}
	art, err := s.SaveArticle(site.ID, "", "Task Post", "Body")
	if err != nil {
		t.Fatal(err)
	}
	starred, err := s.SetArticleStar(site.ID, art.ID, true, StarTypeTodo)
	if err != nil || !starred.Starred || starred.StarType != StarTypeTodo {
		t.Fatalf("expected todo star, got %+v, err=%v", starred, err)
	}
	cleared, err := s.SetArticleStar(site.ID, art.ID, false, StarTypeHeart)
	if err != nil || cleared.Starred || cleared.StarType != StarTypeDefault {
		t.Fatalf("expected cleared star reset to default, got %+v, err=%v", cleared, err)
	}
	if _, err := s.SetArticleStar(site.ID, art.ID, true, 99); err == nil {
		t.Fatal("expected invalid starType error")
	}

	// Legacy JSON compatibility: starred:true without starType defaults to StarTypeDefault.
	var legacy Article
	if err := json.Unmarshal([]byte(`{"id":"a1","siteId":"s1","title":"Old","starred":true}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if !legacy.Starred || legacy.StarType != StarTypeDefault {
		t.Fatalf("legacy star compatibility failed: %+v", legacy)
	}

	// Subscription star types.
	subURL := "https://example.com/feed.xml"
	if _, err := s.SaveSubscription(subURL, feeds.Feed{
		Title: "Example",
		Items: []feeds.Item{{ID: "i1", Title: "Item 1"}},
	}); err != nil {
		t.Fatal(err)
	}
	subs, err := s.ListSubscriptions()
	if err != nil || len(subs) != 1 {
		t.Fatalf("expected 1 sub, got %d, err=%v", len(subs), err)
	}
	key := feeds.ItemKey(subs[0].Feed.Items[0])
	if err := s.SetSubscriptionStar(subURL, key, true, StarTypeSparkles); err != nil {
		t.Fatal(err)
	}
	subs, _ = s.ListSubscriptions()
	if !subs[0].Feed.Items[0].Starred || subs[0].Feed.Items[0].StarType != StarTypeSparkles {
		t.Fatalf("expected sparkles star on sub item, got %+v", subs[0].Feed.Items[0])
	}
}

func TestMoveArticleRecordAndSlugDeduplication(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	siteA, _ := s.CreateSite("Site A", "")
	siteB, _ := s.CreateSite("Site B", "")
	artA, _ := s.SaveArticle(siteA.ID, "", "About A", "Body A")
	artB, _ := s.SaveArticle(siteB.ID, "", "About B", "Body B")
	if _, err := s.UpdateArticleSettings(siteA.ID, artA.ID, "about", 0, "", false, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateArticleSettings(siteB.ID, artB.ID, "about", 0, "", false, 1); err != nil {
		t.Fatal(err)
	}

	moved, err := s.MoveArticleRecord(siteA.ID, artA.ID, siteB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.SiteID != siteB.ID {
		t.Fatalf("expected siteID %s, got %s", siteB.ID, moved.SiteID)
	}
	if moved.Slug != "about-2" {
		t.Fatalf("expected deduplicated slug about-2, got %q", moved.Slug)
	}
	artsA, _ := s.ListArticles(siteA.ID)
	artsB, _ := s.ListArticles(siteB.ID)
	if len(artsA) != 0 || len(artsB) != 2 {
		t.Fatalf("expected 0 in A and 2 in B, got %d and %d", len(artsA), len(artsB))
	}
}
