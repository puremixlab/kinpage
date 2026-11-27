// Package store persists the local sites and articles created in Planet Windows.
package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type data struct {
	Version          int                        `json:"version"`
	IPFSDisabled     bool                       `json:"ipfsDisabled,omitempty"`
	PreventSleep     bool                       `json:"preventSleep,omitempty"`
	Sites            []Site                     `json:"sites"`
	Articles         []Article                  `json:"articles"`
	Drafts           []Draft                    `json:"drafts,omitempty"`
	DraftCleanup     []DraftCleanup             `json:"draftCleanup,omitempty"`
	DraftStaging     []DraftCleanup             `json:"draftStaging,omitempty"`
	ArticleCleanup   []ArticleCleanup           `json:"articleCleanup,omitempty"`
	Subscriptions    []Subscription             `json:"subscriptions,omitempty"`
	PublishedFolders []PublishedFolder          `json:"publishedFolders,omitempty"`
	BackupMetadata   map[string]json.RawMessage `json:"backupMetadata,omitempty"`
}

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

type Store struct {
	mu          sync.Mutex
	path        string
	data        data
	lastModTime time.Time
	lastSize    int64
}

func (s *Store) IPFSDisabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.IPFSDisabled
}

func (s *Store) SetIPFSDisabled(disabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.IPFSDisabled == disabled {
		return nil
	}
	next := s.data
	next.IPFSDisabled = disabled
	return s.write(next)
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dir, "library.json"), data: data{Version: 4, Sites: []Site{}, Articles: []Article{}}}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if info, statErr := os.Stat(s.path); statErr == nil {
		s.lastModTime = info.ModTime()
		s.lastSize = info.Size()
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("read library: %w", err)
	}
	migrating := s.data.Version >= 1 && s.data.Version <= 3
	if migrating {
		// Capture the exact pre-migration bytes once. Version 4 deliberately
		// prevents historical Windows binaries from dropping newly added fields.
		digest := sha256.Sum256(b)
		snapshot := filepath.Join(dir, fmt.Sprintf("library-before-v4-%x.json", digest))
		if err := snapshotLibrary(snapshot, b); err != nil {
			return nil, fmt.Errorf("snapshot original library: %w", err)
		}
	}
	if s.data.Version == 1 {
		type oldArticle struct {
			ID            string   `json:"id"`
			Attachments   []string `json:"attachments"`
			ArticleNumber *int     `json:"articleNumber"`
		}
		for siteID, raw := range s.data.BackupMetadata {
			var original struct {
				Articles []oldArticle `json:"articles"`
			}
			if err := json.Unmarshal(raw, &original); err != nil {
				return nil, fmt.Errorf("read imported metadata: %w", err)
			}
			for i := range s.data.Articles {
				for _, old := range original.Articles {
					if s.data.Articles[i].SiteID == siteID && strings.EqualFold(s.data.Articles[i].ID, old.ID) {
						s.data.Articles[i].Attachments = append([]string(nil), old.Attachments...)
						s.data.Articles[i].ArticleNumber = old.ArticleNumber
					}
				}
			}
		}
		s.data.Version = 2
	}
	if s.data.Version == 2 {
		for i := range s.data.Articles {
			a := &s.data.Articles[i]
			var original struct {
				Articles []struct {
					ID    string `json:"id"`
					Audio string `json:"audioFilename"`
					Video string `json:"videoFilename"`
				} `json:"articles"`
			}
			raw, imported := s.data.BackupMetadata[a.SiteID]
			if imported {
				if err := json.Unmarshal(raw, &original); err != nil {
					return nil, fmt.Errorf("read imported metadata: %w", err)
				}
				for _, old := range original.Articles {
					if strings.EqualFold(old.ID, a.ID) {
						for _, name := range a.Attachments {
							if name == old.Audio {
								a.AudioFilename = name
							}
							if name == old.Video {
								a.VideoFilename = name
							}
						}
					}
				}
			} else {
				for _, name := range a.Attachments {
					switch mediaType(name) {
					case "audio":
						a.AudioFilename = name
					case "video":
						a.VideoFilename = name
					}
				}
			}
		}
		s.data.Version = 3
	}
	if s.data.Version == 3 {
		s.data.Version = 4
	}
	if s.data.Version != 4 {
		return nil, fmt.Errorf("unsupported library version: %d", s.data.Version)
	}
	// Older Windows libraries may already contain imported articles without the
	// numbering the Mac app assigns while importing a backup.
	next := s.data
	next.Sites = append([]Site(nil), s.data.Sites...)
	next.Articles = append([]Article(nil), s.data.Articles...)
	next.Drafts = append([]Draft(nil), s.data.Drafts...)
	changed := migrating
	for i := range next.Sites {
		if !next.Sites[i].Imported {
			continue
		}
		updated, err := normalizeImportedNumbers(&next.Sites[i], next.Articles, next.BackupMetadata[next.Sites[i].ID])
		if err != nil {
			return nil, fmt.Errorf("upgrade imported article numbers: %w", err)
		}
		if updated && next.Sites[i].LastPublishedCID != "" {
			if next.Sites[i].ContentRevision == ^uint64(0) {
				return nil, errors.New("站点修订版本已达到上限")
			}
			next.Sites[i].ContentRevision++
		}
		changed = changed || updated
		restored, unsupported, err := restoreImportedTags(next.Sites[i].ID, next.Articles, next.BackupMetadata[next.Sites[i].ID])
		if err != nil {
			return nil, fmt.Errorf("upgrade imported tags: %w", err)
		}
		changed = changed || restored
		if next.Sites[i].TagsUnsupported != unsupported {
			next.Sites[i].TagsUnsupported = unsupported
			changed = true
		}
	}
	for i := range next.Articles {
		if next.Articles[i].Tags == nil {
			unsupported := false
			for _, site := range next.Sites {
				if site.ID == next.Articles[i].SiteID && site.TagsUnsupported {
					unsupported = true
					break
				}
			}
			if unsupported {
				continue
			}
			next.Articles[i].Tags = map[string]string{}
			changed = true
		}
	}
	for i := range next.Drafts {
		if next.Drafts[i].Tags != nil {
			continue
		}
		next.Drafts[i].Tags = map[string]string{}
		for _, article := range next.Articles {
			if article.SiteID == next.Drafts[i].SiteID && article.ID == next.Drafts[i].ArticleID {
				next.Drafts[i].Tags = copyTags(article.Tags)
				break
			}
		}
		changed = true
	}
	if changed {
		if err := s.write(next); err != nil {
			return nil, fmt.Errorf("upgrade imported article numbers: %w", err)
		}
	}
	return s, nil
}

// A fully synced private temporary file is linked into place without replacing
// an existing snapshot. Interrupted writes never occupy the final digest name.
func snapshotLibrary(destination string, b []byte) error {
	if info, e := os.Lstat(destination); e == nil && !info.Mode().IsRegular() {
		return errors.New("original library snapshot is not a regular file")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	if old, err := os.ReadFile(destination); err == nil {
		if string(old) != string(b) {
			return errors.New("original library snapshot conflicts")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(destination), ".library-snapshot-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		return err
	}
	if err = os.Link(tmp, destination); errors.Is(err, os.ErrExist) {
		old, e := os.ReadFile(destination)
		if e == nil && string(old) == string(b) {
			return nil
		}
	}
	return err
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	s := hex.EncodeToString(b)
	return strings.ToUpper(s[0:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:32]), nil
}

// write commits the supplied snapshot before it becomes visible in memory.
func (s *Store) write(next data) error {
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".library-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmp, s.path); err != nil {
		return err
	}
	if info, statErr := os.Stat(s.path); statErr == nil {
		s.lastModTime = info.ModTime()
		s.lastSize = info.Size()
	}
	s.data = next
	return nil
}

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
