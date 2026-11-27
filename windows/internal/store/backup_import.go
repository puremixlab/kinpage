package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

func (s *Store) Import(site Site, articles []Article, raw json.RawMessage) error {
	if site.ID == "" || site.IPNS == "" || !json.Valid(raw) {
		return errors.New("备份元数据无效")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.data.Sites {
		if existing.ID == site.ID {
			return errors.New("站点已经存在或已移至回收站")
		}
	}
	for _, article := range articles {
		if article.SiteID != site.ID || article.ID == "" {
			return errors.New("备份文章无效")
		}
	}
	articles = append([]Article(nil), articles...)
	_, unsupported, err := restoreImportedTags(site.ID, articles, raw)
	if err != nil {
		return err
	}
	site.TagsUnsupported = unsupported
	for i := range articles {
		articles[i] = cloneArticle(articles[i])
	}
	if _, err := normalizeImportedNumbers(&site, articles, raw); err != nil {
		return err
	}
	next := s.data
	next.Sites = append(append([]Site{}, s.data.Sites...), site)
	next.Articles = append(append([]Article{}, s.data.Articles...), articles...)
	next.BackupMetadata = make(map[string]json.RawMessage, len(s.data.BackupMetadata)+1)
	for id, value := range s.data.BackupMetadata {
		next.BackupMetadata[id] = value
	}
	next.BackupMetadata[site.ID] = append(json.RawMessage(nil), raw...)
	return s.write(next)
}

func restoreImportedTags(siteID string, articles []Article, raw json.RawMessage) (bool, bool, error) {
	var original struct {
		Tags     json.RawMessage `json:"tags"`
		Articles []struct {
			ID   string          `json:"id"`
			Tags json.RawMessage `json:"tags"`
		} `json:"articles"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &original); err != nil {
			return false, false, err
		}
	}
	byID := map[string]map[string]string{}
	invalid := map[string]bool{}
	unsupported := false
	var rootTags map[string]string
	if len(original.Tags) > 0 && string(original.Tags) != "null" && json.Unmarshal(original.Tags, &rootTags) != nil {
		unsupported = true
	}
	for _, a := range original.Articles {
		id := strings.ToUpper(a.ID)
		var tags map[string]string
		if len(a.Tags) > 0 && string(a.Tags) != "null" && json.Unmarshal(a.Tags, &tags) != nil {
			invalid[id] = true
			unsupported = true
			continue
		}
		byID[id] = tags
	}
	changed := false
	for i := range articles {
		a := &articles[i]
		if a.SiteID == siteID && a.Tags == nil {
			if invalid[strings.ToUpper(a.ID)] {
				continue
			}
			a.Tags = copyTags(byID[strings.ToUpper(a.ID)])
			changed = true
		}
	}
	return changed, unsupported, nil
}

// normalizeImportedNumbers follows Planet's chronological allocation rule and
// preserves already assigned numbers across deletion, publication and export.
func normalizeImportedNumbers(site *Site, articles []Article, raw json.RawMessage) (bool, error) {
	var original struct {
		Next     int    `json:"nextArticleNumber"`
		Slug     string `json:"slug"`
		Articles []struct {
			Number *int `json:"articleNumber"`
		} `json:"articles"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &original); err != nil {
			return false, err
		}
	}
	indices := make([]int, 0, len(articles))
	for i := range articles {
		if articles[i].SiteID == site.ID {
			indices = append(indices, i)
		}
	}
	sort.Slice(indices, func(i, j int) bool {
		a, b := articles[indices[i]], articles[indices[j]]
		if !a.Created.Equal(b.Created) {
			return a.Created.Before(b.Created)
		}
		return strings.ToUpper(a.ID) < strings.ToUpper(b.ID)
	})
	used := map[int]bool{}
	number := 1
	if original.Next > number {
		number = original.Next
	}
	if site.NextArticleNumber > number {
		number = site.NextArticleNumber
	}
	maxInt := int(^uint(0) >> 1)
	for _, old := range original.Articles {
		if old.Number != nil && *old.Number >= number {
			if *old.Number == maxInt {
				return false, errors.New("文章编号已达到上限")
			}
			number = *old.Number + 1
		}
	}
	changed := false
	for _, i := range indices {
		a := &articles[i]
		if a.ArticleNumber == nil {
			continue
		}
		value := *a.ArticleNumber
		if value <= 0 || used[value] {
			a.ArticleNumber = nil
			changed = true
			continue
		}
		used[value] = true
		if value >= number {
			if value == maxInt {
				return false, errors.New("文章编号已达到上限")
			}
			number = value + 1
		}
	}
	prefix := strings.ToUpper(strings.TrimSpace(original.Slug))
	if prefix == "" {
		prefix = strings.ToUpper(site.ID)
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
	}
	for _, i := range indices {
		a := &articles[i]
		if a.ArticleNumber == nil {
			if number == maxInt {
				return false, errors.New("文章编号已达到上限")
			}
			assigned := number
			a.ArticleNumber = &assigned
			number++
			changed = true
		}
		reference := fmt.Sprintf("%s-%d", prefix, *a.ArticleNumber)
		if a.ArticleReference != reference {
			a.ArticleReference = reference
			changed = true
		}
	}
	if site.NextArticleNumber != number {
		site.NextArticleNumber = number
		changed = true
	}
	return changed, nil
}

func (s *Store) BackupMetadata(siteID string) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.hasSite(siteID) {
		return nil, errors.New("站点不存在")
	}
	return append(json.RawMessage(nil), s.data.BackupMetadata[siteID]...), nil
}
