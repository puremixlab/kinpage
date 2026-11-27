package store

import (
	"errors"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AddAttachment records an already-copied attachment for a Windows article.
func (s *Store) AddAttachment(siteID, articleID, name string) (Article, error) {
	return s.AddAttachmentWithMetadata(siteID, articleID, name, nil, nil)
}

func mediaType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".m4a", ".aac", ".wav", ".ogg", ".opus", ".flac":
		return "audio"
	case ".mp4", ".mov", ".m4v", ".webm", ".avi", ".mpeg", ".mpg":
		return "video"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "image"
	default:
		return ""
	}
}

var markdownImagePattern = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// DeriveHeroImage selects the first Markdown image attachment or the first image attachment.
func DeriveHeroImage(content string, attachments []string) string {
	if len(attachments) == 0 {
		return ""
	}
	for _, match := range markdownImagePattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		rawTarget := strings.TrimPrefix(strings.Trim(match[1], "<>"), "./")
		decoded, err := url.PathUnescape(rawTarget)
		if err != nil {
			decoded = rawTarget
		}
		for _, att := range attachments {
			if mediaType(att) == "image" && (strings.EqualFold(att, decoded) || strings.EqualFold(att, rawTarget)) {
				return att
			}
		}
	}
	for _, att := range attachments {
		if mediaType(att) == "image" {
			return att
		}
	}
	return ""
}

func (s *Store) AddAttachmentWithMetadata(siteID, articleID, name string, duration *int, size *int64) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var selected *Site
	for i := range s.data.Sites {
		if s.data.Sites[i].ID == siteID && !s.data.Sites[i].Removed {
			selected = &s.data.Sites[i]
			break
		}
	}
	if selected == nil {
		return Article{}, errors.New("站点不存在")
	}
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return Article{}, errors.New("附件名称无效")
	}
	for _, entry := range s.data.ArticleCleanup {
		if entry.SiteID == siteID && entry.ArticleID == articleID && strings.EqualFold(entry.Name, name) {
			return Article{}, errors.New("原附件等待清理，请稍后重试")
		}
	}
	next := s.data
	next.Articles = append([]Article{}, s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
		for _, existing := range article.Attachments {
			if strings.EqualFold(existing, name) {
				return Article{}, errors.New("附件名称已存在")
			}
		}
		article.Attachments = append(append([]string{}, article.Attachments...), name)
		switch mediaType(name) {
		case "audio":
			article.AudioFilename, article.AudioDuration, article.AudioByteLength = name, duration, size
		case "video":
			article.VideoFilename = name
		case "image":
			if article.HeroImage == "" && !article.HeroImageManual {
				article.HeroImage = DeriveHeroImage(article.Content, article.Attachments)
			}
		}
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, siteID, false)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

// RemoveAttachment removes a saved attachment reference from a Windows article.
func (s *Store) RemoveAttachment(siteID, articleID, name string) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var selected *Site
	for i := range s.data.Sites {
		if s.data.Sites[i].ID == siteID && !s.data.Sites[i].Removed {
			selected = &s.data.Sites[i]
			break
		}
	}
	if selected == nil {
		return Article{}, errors.New("站点不存在")
	}
	next := s.data
	next.Articles = append([]Article{}, s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
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
			return Article{}, errors.New("附件不存在")
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
		if article.HeroImage == name {
			// A manually chosen cover stays explicitly cleared, matching the
			// macOS writer; an auto-derived cover falls back to derivation.
			if article.HeroImageManual {
				article.HeroImage = ""
			} else {
				article.HeroImage = DeriveHeroImage(article.Content, remaining)
			}
		}
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, siteID, false)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

// SetHeroImage records an explicit cover choice for a saved article; an empty
// name clears the cover and keeps it from being re-derived automatically.
func (s *Store) SetHeroImage(siteID, articleID, name string) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Article{}, errors.New("站点不存在")
	}
	if name != "" && mediaType(name) != "image" {
		return Article{}, errors.New("封面必须是图片附件")
	}
	next := s.data
	next.Articles = append([]Article{}, s.data.Articles...)
	for i, article := range next.Articles {
		if article.SiteID != siteID || article.ID != articleID {
			continue
		}
		if name != "" {
			registered := false
			for _, item := range article.Attachments {
				if item == name {
					registered = true
					break
				}
			}
			if !registered {
				return Article{}, errors.New("附件未登记")
			}
		}
		article.HeroImage, article.HeroImageManual = name, true
		article.Modified = time.Now().UTC()
		next.Articles[i] = article
		markContentChange(&next, siteID, false)
		if err := s.write(next); err != nil {
			return Article{}, err
		}
		return cloneArticle(article), nil
	}
	return Article{}, errors.New("文章不存在")
}

// SetDraftHeroImage records an explicit cover choice on a draft; an empty name
// clears it. Existing article attachments stay eligible unless the draft marks
// them removed.
func (s *Store) SetDraftHeroImage(siteID, draftID, name string) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Draft{}, errors.New("站点不存在")
	}
	next := s.data
	next.Drafts = append([]Draft(nil), s.data.Drafts...)
	for i, draft := range next.Drafts {
		if draft.ID != draftID || draft.SiteID != siteID {
			continue
		}
		if name != "" {
			registered := false
			for _, item := range draft.Attachments {
				if item == name {
					registered = true
					break
				}
			}
			if !registered {
				for _, article := range s.data.Articles {
					if article.ID == draft.ArticleID && article.SiteID == siteID {
						removed := false
						for _, item := range draft.RemovedAttachments {
							if item == name {
								removed = true
								break
							}
						}
						if !removed {
							for _, item := range article.Attachments {
								if item == name {
									registered = true
									break
								}
							}
						}
						break
					}
				}
			}
			if !registered || mediaType(name) != "image" {
				return Draft{}, errors.New("封面必须是本文已登记的图片附件")
			}
		}
		draft.HeroImage, draft.HeroImageManual = name, true
		draft.Modified = time.Now().UTC()
		next.Drafts[i] = draft
		if err := s.write(next); err != nil {
			return Draft{}, err
		}
		return cloneDraft(draft), nil
	}
	return Draft{}, errors.New("草稿不存在")
}

// ChangeDraftAttachment changes only the private draft metadata.
func (s *Store) ChangeDraftAttachment(siteID, draftID, name string, add bool) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Draft{}, errors.New("站点不存在")
	}
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return Draft{}, errors.New("附件名称无效")
	}
	next := s.data
	next.Drafts = append([]Draft(nil), s.data.Drafts...)
	for i, draft := range next.Drafts {
		if draft.ID != draftID || draft.SiteID != siteID {
			continue
		}
		if add {
			for _, pending := range s.data.DraftCleanup {
				if pending.SiteID == siteID && pending.DraftID == draftID && strings.EqualFold(pending.Name, name) {
					return Draft{}, errors.New("草稿附件等待清理，请稍后重试")
				}
			}
			for _, item := range draft.Attachments {
				if strings.EqualFold(item, name) {
					return Draft{}, errors.New("附件名称已存在")
				}
			}
			for _, article := range s.data.Articles {
				if article.ID == draft.ArticleID && article.SiteID == siteID {
					for _, item := range article.Attachments {
						if strings.EqualFold(item, name) {
							return Draft{}, errors.New("文章已有同名附件")
						}
					}
				}
			}
			draft.Attachments = append(append([]string(nil), draft.Attachments...), name)
			next.DraftStaging = make([]DraftCleanup, 0, len(s.data.DraftStaging))
			for _, entry := range s.data.DraftStaging {
				if entry.SiteID != siteID || entry.DraftID != draftID || entry.Name != name {
					next.DraftStaging = append(next.DraftStaging, entry)
				}
			}
		} else {
			files := make([]string, 0, len(draft.Attachments))
			found := false
			for _, item := range draft.Attachments {
				if item == name {
					found = true
				} else {
					files = append(files, item)
				}
			}
			if !found {
				return Draft{}, errors.New("草稿附件不存在")
			}
			queueDraftCleanup(&next, draft, []string{name})
			draft.Attachments = files
			if draft.HeroImage == name {
				draft.HeroImage, draft.HeroImageManual = "", false
			}
		}
		draft.Modified = time.Now().UTC()
		next.Drafts[i] = draft
		if err := s.write(next); err != nil {
			return Draft{}, err
		}
		return cloneDraft(draft), nil
	}
	return Draft{}, errors.New("草稿不存在")
}

// SetDraftExistingAttachmentRemoved changes only the draft until publication.
func (s *Store) SetDraftExistingAttachmentRemoved(siteID, draftID, name string, removed bool) (Draft, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return Draft{}, errors.New("站点不存在")
	}
	next := s.data
	next.Drafts = append([]Draft(nil), s.data.Drafts...)
	for i, draft := range next.Drafts {
		if draft.SiteID != siteID || draft.ID != draftID {
			continue
		}
		valid := false
		for _, article := range s.data.Articles {
			if article.SiteID == siteID && article.ID == draft.ArticleID {
				for _, item := range article.Attachments {
					if item == name {
						valid = true
					}
				}
			}
		}
		if !valid {
			return Draft{}, errors.New("原文章附件不存在")
		}
		found := false
		remaining := make([]string, 0, len(draft.RemovedAttachments))
		for _, item := range draft.RemovedAttachments {
			if item == name {
				found = true
			} else {
				remaining = append(remaining, item)
			}
		}
		if removed && !found {
			remaining = append(remaining, name)
		}
		if !removed && !found {
			return Draft{}, errors.New("该附件未标记为删除")
		}
		draft.RemovedAttachments = remaining
		if removed && draft.HeroImage == name {
			draft.HeroImage, draft.HeroImageManual = "", false
		}
		draft.Modified = time.Now().UTC()
		next.Drafts[i] = draft
		if err := s.write(next); err != nil {
			return Draft{}, err
		}
		return cloneDraft(draft), nil
	}
	return Draft{}, errors.New("草稿不存在")
}
