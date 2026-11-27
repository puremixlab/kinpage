package store

import (
	"errors"
	"regexp"
	"slices"
	"strings"
)

var (
	ErrMobileSiteUnavailable = errors.New("mobile publication requires an active IPNS site")
	ErrMobileArticleConflict = errors.New("mobile article ID already belongs to different content")
	ErrMobileArticleID       = errors.New("mobile article ID must be a UUID v4")
	mobileArticleID          = regexp.MustCompile(`^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$`)
)

// SaveMobileArticle allows a phone to retry after a dropped connection using the
// same article ID. A retry never overwrites desktop edits.
func (s *Store) SaveMobileArticle(siteID, id, title, content string) (Article, error) {
	id = strings.ToUpper(id)
	if !mobileArticleID.MatchString(id) {
		return Article{}, ErrMobileArticleID
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	active := slices.ContainsFunc(s.data.Sites, func(site Site) bool {
		return site.ID == siteID && !site.Removed && !site.Archived && site.ShouldPublishIPNS()
	})
	if !active {
		return Article{}, ErrMobileSiteUnavailable
	}
	for _, article := range s.data.Articles {
		if !strings.EqualFold(article.ID, id) {
			continue
		}
		if article.SiteID != siteID || article.Title != title || article.Content != content {
			return Article{}, ErrMobileArticleConflict
		}
		return cloneArticle(article), nil
	}
	return s.saveArticleLocked(siteID, "", title, content, "", id, nil, nil)
}
