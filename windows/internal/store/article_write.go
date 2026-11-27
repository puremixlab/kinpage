package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Article struct {
	ID                     string            `json:"id"`
	SiteID                 string            `json:"siteId"`
	Title                  string            `json:"title"`
	Content                string            `json:"content"`
	Created                time.Time         `json:"created"`
	Modified               time.Time         `json:"modified"`
	Attachments            []string          `json:"attachments,omitempty"`
	ArticleNumber          *int              `json:"articleNumber,omitempty"`
	ArticleReference       string            `json:"articleReference,omitempty"`
	Slug                   string            `json:"slug,omitempty"`
	ArticleType            int               `json:"articleType,omitempty"`
	ExternalLink           string            `json:"externalLink,omitempty"`
	IsIncludedInNavigation bool              `json:"isIncludedInNavigation,omitempty"`
	NavigationWeight       int               `json:"navigationWeight"`
	Tags                   map[string]string `json:"tags"`
	AudioFilename          string            `json:"audioFilename,omitempty"`
	VideoFilename          string            `json:"videoFilename,omitempty"`
	AudioDuration          *int              `json:"audioDuration,omitempty"`
	AudioByteLength        *int64            `json:"audioByteLength,omitempty"`
	HeroImage              string            `json:"heroImage,omitempty"`
	HeroImageManual        bool              `json:"heroImageManual,omitempty"`
	Pinned                 *time.Time        `json:"pinned,omitempty"`
	Starred                bool              `json:"starred,omitempty"`
	StarType               int               `json:"starType,omitempty"`
}

// Draft is private local work and is never included in publication snapshots.
type Draft struct {
	ID                 string            `json:"id"`
	SiteID             string            `json:"siteId"`
	ArticleID          string            `json:"articleId,omitempty"`
	Title              string            `json:"title"`
	Content            string            `json:"content"`
	Tags               map[string]string `json:"tags"`
	Created            time.Time         `json:"created"`
	Modified           time.Time         `json:"modified"`
	BaseModified       *time.Time        `json:"baseModified,omitempty"`
	Attachments        []string          `json:"attachments,omitempty"`
	RemovedAttachments []string          `json:"removedAttachments,omitempty"`
	HeroImage          string            `json:"heroImage,omitempty"`
	HeroImageManual    bool              `json:"heroImageManual,omitempty"`
}

func copyTags(tags map[string]string) map[string]string {
	copy := make(map[string]string, len(tags))
	for k, v := range tags {
		copy[k] = v
	}
	return copy
}

func cloneArticle(article Article) Article {
	if article.Tags != nil {
		article.Tags = copyTags(article.Tags)
	}
	article.Attachments = append([]string(nil), article.Attachments...)
	if article.Pinned != nil {
		pinned := *article.Pinned
		article.Pinned = &pinned
	}
	return article
}

func cloneDraft(draft Draft) Draft {
	draft.Tags = copyTags(draft.Tags)
	draft.Attachments = append([]string(nil), draft.Attachments...)
	draft.RemovedAttachments = append([]string(nil), draft.RemovedAttachments...)
	return draft
}

func (s *Store) ListArticles(siteID string) ([]Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return nil, errors.New("站点不存在")
	}
	articles := []Article{}
	hasPinned := false
	for _, a := range s.data.Articles {
		if a.SiteID == siteID {
			if a.Pinned != nil {
				hasPinned = true
			}
			articles = append(articles, cloneArticle(a))
		}
	}
	if hasPinned {
		sort.SliceStable(articles, func(i, j int) bool {
			if (articles[i].Pinned != nil) != (articles[j].Pinned != nil) {
				return articles[i].Pinned != nil
			}
			if articles[i].Pinned != nil && articles[j].Pinned != nil && !articles[i].Pinned.Equal(*articles[j].Pinned) {
				return articles[i].Pinned.After(*articles[j].Pinned)
			}
			return false
		})
	}
	return articles, nil
}

// TagUsage reports one previously used tag label and how many articles use it.
type TagUsage struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

// SiteAvailableTags aggregates tag labels across the site's articles, skipping
// reserved keys, mirroring the macOS writer's Previously Used Tags list.
func (s *Store) SiteAvailableTags(siteID string) ([]TagUsage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, site := range s.data.Sites {
		if site.ID == siteID && !site.Removed {
			counts := map[string]int{}
			for _, article := range s.data.Articles {
				if article.SiteID != siteID {
					continue
				}
				for key, label := range article.Tags {
					if isReservedTagKey(key) {
						continue
					}
					counts[label]++
				}
			}
			out := make([]TagUsage, 0, len(counts))
			for label, count := range counts {
				out = append(out, TagUsage{Label: label, Count: count})
			}
			// Alphabetical order matches the macOS writer's Previously Used Tags.
			sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
			return out, nil
		}
	}
	return nil, errors.New("站点不存在")
}

func isReservedTagKey(key string) bool {
	switch strings.ToLower(key) {
	case "index", "tags", "archive", "archives":
		return true
	}
	return false
}

func (s *Store) SaveArticle(siteID, articleID, title, content string) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveArticleLocked(siteID, articleID, title, content, "", nil, nil)
}

// DraftPreparation stages files before the atomic store write and returns a
// rollback function to remove any files created if that write fails.
type DraftPreparation func(Article, []string) (Article, func(), error)

func (s *Store) saveArticleLocked(siteID, articleID, title, content, draftID string, prepare DraftPreparation, tags map[string]string) (Article, error) {
	if strings.TrimSpace(title) == "" {
		return Article{}, errors.New("请输入文章标题")
	}
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	var draftFiles, removedFiles []string
	var draftHero string
	var draftHeroManual bool
	next := s.data
	next.Articles = append([]Article{}, s.data.Articles...)
	if draftID != "" {
		next.Drafts = make([]Draft, 0, len(s.data.Drafts))
		for _, draft := range s.data.Drafts {
			if draft.ID != draftID || draft.SiteID != siteID {
				next.Drafts = append(next.Drafts, draft)
			} else {
				draftFiles = append([]string(nil), draft.Attachments...)
				removedFiles = append([]string(nil), draft.RemovedAttachments...)
				draftHero, draftHeroManual = draft.HeroImage, draft.HeroImageManual
				queueDraftCleanup(&next, draft, draft.Attachments)
			}
		}
	}
	if len(draftFiles) > 0 && prepare == nil {
		return Article{}, errors.New("草稿附件需要先复制到文章")
	}
	stage := func(article Article) (Article, func(), error) {
		if len(draftFiles) == 0 {
			return article, func() {}, nil
		}
		return prepare(article, draftFiles)
	}
	now := time.Now().UTC()
	if articleID != "" {
		for i, article := range next.Articles {
			if article.ID == articleID && article.SiteID == siteID {
				article.Title, article.Content, article.Modified = title, content, now
				if tags != nil {
					article.Tags = copyTags(tags)
				}
				for _, name := range removedFiles {
					found := false
					remaining := make([]string, 0, len(article.Attachments))
					for _, item := range article.Attachments {
						if item == name {
							found = true
						} else {
							remaining = append(remaining, item)
						}
					}
					if !found {
						return Article{}, errors.New("原文章附件已改变，请检查草稿")
					}
					article.Attachments = remaining
					if article.AudioFilename == name {
						article.AudioFilename, article.AudioDuration, article.AudioByteLength = "", nil, nil
						for _, item := range remaining {
							if mediaType(item) == "audio" {
								article.AudioFilename = item
							}
						}
					}
					if article.VideoFilename == name {
						article.VideoFilename = ""
						for _, item := range remaining {
							if mediaType(item) == "video" {
								article.VideoFilename = item
							}
						}
					}
					next.ArticleCleanup = append(append([]ArticleCleanup(nil), next.ArticleCleanup...), ArticleCleanup{SiteID: siteID, ArticleID: article.ID, Name: name})
				}
				article, rollback, err := stage(article)
				if err != nil {
					return Article{}, err
				}
				if draftID != "" && draftHeroManual {
					article.HeroImage, article.HeroImageManual = draftHero, true
				} else if !article.HeroImageManual {
					article.HeroImage = DeriveHeroImage(article.Content, article.Attachments)
					article.HeroImageManual = false
				}
				next.Articles[i] = article
				markContentChange(&next, siteID, true)
				if err := s.write(next); err != nil {
					rollback()
					return Article{}, err
				}
				return cloneArticle(article), nil
			}
		}
		return Article{}, errors.New("文章不存在")
	}
	id, err := newID()
	if err != nil {
		return Article{}, err
	}
	var original struct {
		Next     int    `json:"nextArticleNumber"`
		Slug     string `json:"slug"`
		Articles []struct {
			Number *int `json:"articleNumber"`
		} `json:"articles"`
	}
	if raw := s.data.BackupMetadata[siteID]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &original); err != nil {
			return Article{}, fmt.Errorf("站点备份元数据无效：%w", err)
		}
	}
	number := 1
	for _, site := range s.data.Sites {
		if site.ID == siteID && site.NextArticleNumber > number {
			number = site.NextArticleNumber
		}
	}
	if original.Next > number {
		number = original.Next
	}
	for _, old := range original.Articles {
		if old.Number != nil && *old.Number >= number {
			if *old.Number == int(^uint(0)>>1) {
				return Article{}, errors.New("文章编号已达到上限")
			}
			number = *old.Number + 1
		}
	}
	for _, existing := range s.data.Articles {
		if existing.SiteID == siteID && existing.ArticleNumber != nil && *existing.ArticleNumber >= number {
			if *existing.ArticleNumber == int(^uint(0)>>1) {
				return Article{}, errors.New("文章编号已达到上限")
			}
			number = *existing.ArticleNumber + 1
		}
	}
	if number == int(^uint(0)>>1) {
		return Article{}, errors.New("文章编号已达到上限")
	}
	prefix := strings.ToUpper(strings.TrimSpace(original.Slug))
	for _, site := range s.data.Sites {
		if site.ID == siteID && !site.Imported {
			prefix = strings.ToUpper(site.Slug)
			break
		}
	}
	if prefix == "" {
		prefix = strings.ToUpper(siteID)
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
	}
	article := Article{ID: id, SiteID: siteID, Title: title, Content: content, Created: now, Modified: now, Tags: copyTags(tags), ArticleNumber: &number, ArticleReference: fmt.Sprintf("%s-%d", prefix, number), NavigationWeight: 1}
	article, rollback, err := stage(article)
	if err != nil {
		return Article{}, err
	}
	if draftHeroManual {
		article.HeroImage, article.HeroImageManual = draftHero, true
	} else {
		article.HeroImage, article.HeroImageManual = DeriveHeroImage(article.Content, article.Attachments), false
	}
	next.Articles = append(next.Articles, article)
	markContentChange(&next, siteID, true)
	for i := range next.Sites {
		if next.Sites[i].ID == siteID {
			next.Sites[i].NextArticleNumber = number + 1
			break
		}
	}
	if err := s.write(next); err != nil {
		rollback()
		return Article{}, err
	}
	return cloneArticle(article), nil
}

func (s *Store) ListDrafts(siteID string) ([]Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return nil, errors.New("站点不存在")
	}
	out := []Draft{}
	for _, draft := range s.data.Drafts {
		if draft.SiteID == siteID {
			out = append(out, cloneDraft(draft))
		}
	}
	return out, nil
}

func (s *Store) SaveDraft(siteID, draftID, articleID, title, content string) (Draft, error) {
	return s.saveDraft(siteID, draftID, articleID, title, content, nil)
}

func (s *Store) SaveDraftWithTags(siteID, draftID, articleID, title, content, tagsJSON string) (Draft, error) {
	tags, err := parseEditedTags(tagsJSON)
	if err != nil {
		return Draft{}, err
	}
	return s.saveDraft(siteID, draftID, articleID, title, content, tags)
}

func (s *Store) saveDraft(siteID, draftID, articleID, title, content string, tags map[string]string) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Draft{}, errors.New("站点不存在")
	}
	if tags != nil {
		if err := s.validateTagsLocked(siteID, articleID, tags); err != nil {
			return Draft{}, err
		}
	}
	next := s.data
	next.Drafts = append([]Draft(nil), s.data.Drafts...)
	now := time.Now().UTC()
	if draftID != "" {
		for i, draft := range next.Drafts {
			if draft.ID != draftID || draft.SiteID != siteID {
				continue
			}
			if draft.ArticleID != articleID {
				return Draft{}, errors.New("草稿关联的文章已改变")
			}
			draft.Title, draft.Content, draft.Modified = title, content, now
			if tags != nil {
				draft.Tags = copyTags(tags)
			}
			next.Drafts[i] = draft
			if err := s.write(next); err != nil {
				return Draft{}, err
			}
			return cloneDraft(draft), nil
		}
		return Draft{}, errors.New("草稿不存在")
	}
	var base *time.Time
	inheritedTags := map[string]string{}
	inheritedHero, inheritedHeroManual := "", false
	if articleID != "" {
		for _, article := range s.data.Articles {
			if article.SiteID == siteID && article.ID == articleID {
				value := article.Modified
				base = &value
				inheritedTags = copyTags(article.Tags)
				inheritedHero, inheritedHeroManual = article.HeroImage, article.HeroImageManual
				break
			}
		}
		if base == nil {
			return Draft{}, errors.New("原文章不存在")
		}
	}
	id, err := newID()
	if err != nil {
		return Draft{}, err
	}
	if tags != nil {
		inheritedTags = copyTags(tags)
	}
	draft := Draft{ID: id, SiteID: siteID, ArticleID: articleID, Title: title, Content: content, Tags: inheritedTags, Created: now, Modified: now, BaseModified: base, HeroImage: inheritedHero, HeroImageManual: inheritedHeroManual}
	next.Drafts = append(next.Drafts, draft)
	if err := s.write(next); err != nil {
		return Draft{}, err
	}
	return cloneDraft(draft), nil
}

func (s *Store) DeleteDraft(siteID, draftID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return errors.New("站点不存在")
	}
	next := s.data
	next.Drafts = make([]Draft, 0, len(s.data.Drafts))
	found := false
	for _, draft := range s.data.Drafts {
		if draft.ID == draftID && draft.SiteID == siteID {
			found = true
			queueDraftCleanup(&next, draft, draft.Attachments)
			continue
		}
		next.Drafts = append(next.Drafts, draft)
	}
	if !found {
		return errors.New("草稿不存在")
	}
	return s.write(next)
}

func (s *Store) PublishDraft(siteID, draftID, title, content string) (Article, error) {
	return s.PublishDraftPrepared(siteID, draftID, title, content, nil)
}

func (s *Store) PublishDraftPrepared(siteID, draftID, title, content string, prepare DraftPreparation) (Article, error) {
	return s.publishDraftPrepared(siteID, draftID, title, content, prepare, nil)
}

func (s *Store) PublishDraftPreparedWithTags(siteID, draftID, title, content, tagsJSON string, prepare DraftPreparation) (Article, error) {
	tags, err := parseEditedTags(tagsJSON)
	if err != nil {
		return Article{}, err
	}
	return s.publishDraftPrepared(siteID, draftID, title, content, prepare, tags)
}

func (s *Store) publishDraftPrepared(siteID, draftID, title, content string, prepare DraftPreparation, tags map[string]string) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, draft := range s.data.Drafts {
		if draft.ID != draftID || draft.SiteID != siteID {
			continue
		}
		if draft.ArticleID != "" {
			found := false
			for _, article := range s.data.Articles {
				if article.SiteID == siteID && article.ID == draft.ArticleID {
					found = true
					if draft.BaseModified == nil || !article.Modified.Equal(*draft.BaseModified) {
						return Article{}, errors.New("原文章已更改，请先检查草稿以免覆盖新内容")
					}
					break
				}
			}
			if !found {
				return Article{}, errors.New("原文章已删除，无法发布此草稿")
			}
		}
		for _, site := range s.data.Sites {
			if site.ID == siteID && site.TagsUnsupported && tags == nil {
				// Preserve the published article's existing tags; unsupported raw
				// metadata and any private draft tags must not be promoted.
				return s.saveArticleLocked(siteID, draft.ArticleID, title, content, draft.ID, prepare, nil)
			}
		}
		if tags == nil {
			tags = draft.Tags
		}
		if err := s.validateTagsLocked(siteID, draft.ArticleID, tags); err != nil {
			return Article{}, err
		}
		return s.saveArticleLocked(siteID, draft.ArticleID, title, content, draft.ID, prepare, tags)
	}
	return Article{}, errors.New("草稿不存在")
}

func (s *Store) DeleteArticle(siteID, articleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return errors.New("站点不存在")
	}
	next := s.data
	next.Articles = make([]Article, 0, len(s.data.Articles))
	found := false
	for _, article := range s.data.Articles {
		if article.SiteID == siteID && article.ID == articleID {
			found = true
			continue
		}
		next.Articles = append(next.Articles, article)
	}
	if !found {
		return errors.New("文章不存在")
	}
	markContentChange(&next, siteID, false)
	return s.write(next)
}
