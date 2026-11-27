package store

import (
	"errors"
	"path/filepath"
	"strings"
)

type DraftCleanup struct {
	SiteID  string `json:"siteId"`
	DraftID string `json:"draftId"`
	Name    string `json:"name"`
}

type ArticleCleanup struct {
	SiteID    string `json:"siteId"`
	ArticleID string `json:"articleId"`
	Name      string `json:"name"`
}

func (s *Store) PendingArticleCleanup() []ArticleCleanup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]ArticleCleanup(nil), s.data.ArticleCleanup...)
}

func (s *Store) ArticleFileReferences() []ArticleCleanup {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ArticleCleanup
	for _, article := range s.data.Articles {
		for _, name := range article.Attachments {
			out = append(out, ArticleCleanup{SiteID: article.SiteID, ArticleID: article.ID, Name: name})
		}
	}
	return out
}

func (s *Store) CompleteArticleCleanup(entry ArticleCleanup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.ArticleCleanup = make([]ArticleCleanup, 0, len(s.data.ArticleCleanup))
	found := false
	for _, current := range s.data.ArticleCleanup {
		if current == entry {
			found = true
		} else {
			next.ArticleCleanup = append(next.ArticleCleanup, current)
		}
	}
	if !found {
		return nil
	}
	return s.write(next)
}

func queueDraftCleanup(next *data, draft Draft, names []string) {
	if len(names) == 0 {
		return
	}
	next.DraftCleanup = append([]DraftCleanup(nil), next.DraftCleanup...)
	for _, name := range names {
		next.DraftCleanup = append(next.DraftCleanup, DraftCleanup{SiteID: draft.SiteID, DraftID: draft.ID, Name: name})
	}
}

func (s *Store) PendingDraftCleanup() []DraftCleanup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]DraftCleanup(nil), s.data.DraftCleanup...)
}

func (s *Store) PendingDraftStaging() []DraftCleanup {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]DraftCleanup(nil), s.data.DraftStaging...)
}

func (s *Store) BeginDraftAttachment(siteID, draftID, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) || name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return errors.New("草稿附件无效")
	}
	for _, entry := range append(append([]DraftCleanup(nil), s.data.DraftCleanup...), s.data.DraftStaging...) {
		if entry.SiteID == siteID && entry.DraftID == draftID && strings.EqualFold(entry.Name, name) {
			return errors.New("草稿附件等待清理，请稍后重试")
		}
	}
	for _, draft := range s.data.Drafts {
		if draft.SiteID != siteID || draft.ID != draftID {
			continue
		}
		for _, existing := range draft.Attachments {
			if strings.EqualFold(existing, name) {
				return errors.New("附件名称已存在")
			}
		}
		for _, article := range s.data.Articles {
			if article.SiteID == siteID && article.ID == draft.ArticleID {
				for _, existing := range article.Attachments {
					if strings.EqualFold(existing, name) {
						return errors.New("文章已有同名附件")
					}
				}
			}
		}
		next := s.data
		next.DraftStaging = append(append([]DraftCleanup(nil), s.data.DraftStaging...), DraftCleanup{SiteID: siteID, DraftID: draftID, Name: name})
		return s.write(next)
	}
	return errors.New("草稿不存在")
}

func (s *Store) CompleteDraftStaging(entry DraftCleanup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.DraftStaging = make([]DraftCleanup, 0, len(s.data.DraftStaging))
	found := false
	for _, current := range s.data.DraftStaging {
		if current == entry {
			found = true
		} else {
			next.DraftStaging = append(next.DraftStaging, current)
		}
	}
	if !found {
		return nil
	}
	return s.write(next)
}

// DraftFileReferences includes removed sites, whose private files must survive.
func (s *Store) DraftFileReferences() []DraftCleanup {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []DraftCleanup
	for _, draft := range s.data.Drafts {
		for _, name := range draft.Attachments {
			out = append(out, DraftCleanup{SiteID: draft.SiteID, DraftID: draft.ID, Name: name})
		}
	}
	return out
}

func (s *Store) CompleteDraftCleanup(entry DraftCleanup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data
	next.DraftCleanup = make([]DraftCleanup, 0, len(s.data.DraftCleanup))
	found := false
	for _, current := range s.data.DraftCleanup {
		if current == entry {
			found = true
		} else {
			next.DraftCleanup = append(next.DraftCleanup, current)
		}
	}
	if !found {
		return nil
	}
	return s.write(next)
}
