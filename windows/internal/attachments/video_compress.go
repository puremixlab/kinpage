package attachments

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"windows/internal/mediainfo"
	"windows/internal/store"
)

// CompressDraftVideo transcodes a draft's attached video using ffmpeg into a
// temporary file and atomically replaces the original attachment on success.
func CompressDraftVideo(root string, library *store.Store, siteID, draftID, name, preset string) (store.Draft, error) {
	if !validName(name) || attachmentKind(name) != "video" {
		return store.Draft{}, errors.New("只能压缩视频附件")
	}
	drafts, err := library.ListDrafts(siteID)
	if err != nil {
		return store.Draft{}, err
	}
	var found *store.Draft
	for i := range drafts {
		if drafts[i].ID == draftID {
			found = &drafts[i]
			break
		}
	}
	if found == nil {
		return store.Draft{}, errors.New("草稿不存在")
	}
	dir, err := managedDir(root, "drafts", siteID, draftID, false)
	if err != nil {
		return store.Draft{}, err
	}
	srcPath := filepath.Join(dir, name)
	if _, err := os.Stat(srcPath); err != nil {
		return store.Draft{}, errors.New("视频附件文件不存在")
	}
	tmpPath := filepath.Join(dir, ".compress-"+strings.TrimSuffix(name, filepath.Ext(name))+".mp4")
	defer os.Remove(tmpPath)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := mediainfo.CompressVideo(ctx, srcPath, tmpPath, preset); err != nil {
		return store.Draft{}, err
	}
	if err := os.Rename(tmpPath, srcPath); err != nil {
		return store.Draft{}, err
	}
	return *found, nil
}
