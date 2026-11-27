package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Multi-type star constants aligned with macOS Planet StarType.
const (
	StarTypeDefault  = 0 // 星标
	StarTypeTodo     = 1 // 待办
	StarTypeDone     = 2 // 已完成
	StarTypeSparkles = 3 // 灵感
	StarTypeHeart    = 4 // 喜爱
	StarTypePlan     = 5 // 计划
)

func ValidStarType(t int) bool {
	return t >= StarTypeDefault && t <= StarTypePlan
}

// SetSiteArchived archives or unarchives a local site without deleting its
// data, static files, or IPNS key.
func (s *Store) SetSiteArchived(siteID string, archived bool) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	for i := range next.Sites {
		site := &next.Sites[i]
		if site.ID != siteID || site.Removed || site.DeletionPending {
			continue
		}
		if site.Archived == archived {
			return cloneSite(*site), nil
		}
		site.Archived = archived
		if err := s.write(next); err != nil {
			return Site{}, err
		}
		return cloneSite(*site), nil
	}
	return Site{}, errors.New("站点不存在")
}

// UpdateArticleDate updates a local article's creation date and queues a rebuild.
func (s *Store) UpdateArticleDate(siteID, articleID string, created time.Time) (Article, error) {
	if created.IsZero() {
		return Article{}, errors.New("发布日期无效")
	}
	created = created.UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	next := s.data
	next.Articles = append([]Article(nil), s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
		if article.Created.Equal(created) {
			return cloneArticle(article), nil
		}
		article.Created = created
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, siteID, true)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

// SetArticleStar sets or clears a local article's star and star type.
func (s *Store) SetArticleStar(siteID, articleID string, starred bool, starType int) (Article, error) {
	if !ValidStarType(starType) {
		return Article{}, errors.New("星标类型无效")
	}
	if !starred {
		starType = StarTypeDefault
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	next := s.data
	next.Articles = append([]Article(nil), s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
		if article.Starred == starred && article.StarType == starType {
			return cloneArticle(article), nil
		}
		article.Starred = starred
		article.StarType = starType
		next.Articles[i] = article
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

func uniqueSlugForSite(articles []Article, siteID, excludeID, desired string) string {
	slug := strings.TrimSpace(strings.ToLower(desired))
	if slug == "" || !ValidArticleSlug(slug) {
		return ""
	}
	conflicts := func(candidate string) bool {
		for _, a := range articles {
			if a.SiteID != siteID || a.ID == excludeID {
				continue
			}
			if strings.EqualFold(candidate, a.ID) || (a.Slug != "" && strings.EqualFold(candidate, a.Slug)) {
				return true
			}
		}
		return false
	}
	if !conflicts(slug) {
		return slug
	}
	for n := 2; n < 1000; n++ {
		suffix := fmt.Sprintf("-%d", n)
		base := slug
		if len(base)+len(suffix) > 120 {
			base = base[:120-len(suffix)]
		}
		candidate := base + suffix
		if ValidArticleSlug(candidate) && !conflicts(candidate) {
			return candidate
		}
	}
	return ""
}

// MoveArticleRecord moves an article metadata record from one site to another.
// Callers moving attachments should use attachments.MoveArticle.
func (s *Store) MoveArticleRecord(fromSiteID, articleID, toSiteID string) (Article, error) {
	if fromSiteID == "" || toSiteID == "" || articleID == "" {
		return Article{}, errors.New("移动参数无效")
	}
	if fromSiteID == toSiteID {
		return Article{}, errors.New("目标站点不能与当前站点相同")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(fromSiteID) || !s.hasSite(toSiteID) {
		return Article{}, errors.New("站点不存在")
	}
	next := s.data
	next.Articles = append([]Article(nil), s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != fromSiteID || article.ID != articleID {
			continue
		}
		article.SiteID = toSiteID
		article.ArticleNumber = nil
		article.ArticleReference = ""
		if article.Slug != "" {
			article.Slug = uniqueSlugForSite(next.Articles, toSiteID, article.ID, article.Slug)
		}
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, fromSiteID, false)
		markContentChange(&next, toSiteID, true)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

// ImportArticleRecord inserts an imported article record into siteID with a
// fresh UUID and deduplicated slug. Callers importing packages with attachments
// should use attachments.ImportArticleZip.
func (s *Store) ImportArticleRecord(siteID string, imported Article) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	now := time.Now().UTC()
	created := imported.Created.UTC()
	if created.IsZero() {
		created = now
	}
	modified := imported.Modified.UTC()
	if modified.IsZero() {
		modified = created
	}
	id, err := newID()
	if err != nil {
		return Article{}, err
	}
	starType := imported.StarType
	if !imported.Starred || !ValidStarType(starType) {
		starType = StarTypeDefault
	}
	articleType := imported.ArticleType
	if articleType != 0 && articleType != 1 {
		articleType = 0
	}
	extLink := strings.TrimSpace(imported.ExternalLink)
	if !ValidArticleExternalLink(extLink) {
		extLink = ""
	}
	var tags map[string]string
	if len(imported.Tags) > 0 {
		tags = make(map[string]string, len(imported.Tags))
		for k, v := range imported.Tags {
			if tk := strings.TrimSpace(k); tk != "" {
				tv := strings.TrimSpace(v)
				if tv == "" {
					tv = tk
				}
				tags[tk] = tv
			}
		}
	}
	next := s.data
	next.Articles = append([]Article(nil), s.data.Articles...)
	art := Article{
		ID:                     id,
		SiteID:                 siteID,
		Title:                  strings.TrimSpace(imported.Title),
		Content:                imported.Content,
		Created:                created,
		Modified:               modified,
		Attachments:            append([]string(nil), imported.Attachments...),
		Slug:                   uniqueSlugForSite(next.Articles, siteID, id, imported.Slug),
		ArticleType:            articleType,
		ExternalLink:           extLink,
		IsIncludedInNavigation: imported.IsIncludedInNavigation,
		NavigationWeight:       imported.NavigationWeight,
		Tags:                   tags,
		AudioFilename:          imported.AudioFilename,
		VideoFilename:          imported.VideoFilename,
		HeroImage:              imported.HeroImage,
		HeroImageManual:        imported.HeroImageManual,
		Pinned:                 imported.Pinned,
		Starred:                imported.Starred,
		StarType:               starType,
	}
	next.Articles = append(next.Articles, art)
	markContentChange(&next, siteID, true)
	if err := s.write(next); err != nil {
		return Article{}, err
	}
	return cloneArticle(art), nil
}
