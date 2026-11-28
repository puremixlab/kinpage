package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const repoLocationFile = "repo-location.json"

type repoLocationPointer struct {
	DataDir        string `json:"dataDir,omitempty"`
	PendingDataDir string `json:"pendingDataDir,omitempty"`
}

// ResolveDataDir checks whether defaultDir contains a repo-location.json pointer
// to a custom repository directory. If present and valid, it returns that path;
// otherwise it returns defaultDir.
func ResolveDataDir(defaultDir string) string {
	ptr, err := readRepositoryLocation(defaultDir)
	if err != nil {
		return defaultDir
	}
	return ptr.resolve(defaultDir)
}

func (ptr repoLocationPointer) resolve(defaultDir string) string {
	if ptr.DataDir != "" {
		clean := filepath.Clean(ptr.DataDir)
		if info, err := os.Stat(clean); err == nil && info.IsDir() {
			return clean
		}
	}
	return defaultDir
}

func readRepositoryLocation(defaultDir string) (repoLocationPointer, error) {
	var ptr repoLocationPointer
	raw, err := os.ReadFile(filepath.Join(defaultDir, repoLocationFile))
	if errors.Is(err, os.ErrNotExist) {
		return ptr, nil
	}
	if err != nil {
		return ptr, err
	}
	err = json.Unmarshal(raw, &ptr)
	return ptr, err
}

// ScheduleDataDirMigration records the selected directory for the next startup.
// The current repository stays active until the stopped application can copy it.
func ScheduleDataDirMigration(defaultDir, currentDir, targetDir string) error {
	current, target, err := migrationPaths(currentDir, targetDir)
	if err != nil {
		return err
	}
	return writeRepositoryLocation(defaultDir, repoLocationPointer{DataDir: current, PendingDataDir: target})
}

// ApplyScheduledDataDirMigration runs after the instance lock and before any
// repository consumer starts. A failed copy leaves the old root active.
func ApplyScheduledDataDirMigration(defaultDir string) (string, error) {
	ptr, err := readRepositoryLocation(defaultDir)
	if err != nil {
		return defaultDir, err
	}
	current := ptr.resolve(defaultDir)
	if ptr.PendingDataDir == "" {
		return current, nil
	}
	if !strings.EqualFold(current, filepath.Clean(ptr.DataDir)) {
		return "", errors.New("暂时无法访问原数据仓库，请连接原磁盘后重试")
	}
	if err := MigrateDataDir(defaultDir, current, ptr.PendingDataDir); err != nil {
		return current, err
	}
	return ResolveDataDir(defaultDir), nil
}

func migrationPaths(currentDir, targetDir string) (string, string, error) {
	if !filepath.IsAbs(currentDir) || !filepath.IsAbs(targetDir) {
		return "", "", errors.New("数据目录路径无效")
	}
	current, err := filepath.EvalSymlinks(currentDir)
	if err != nil {
		return "", "", err
	}
	target, err := filepath.EvalSymlinks(targetDir)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(current, target)
	if strings.EqualFold(current, target) || (err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", "", errors.New("目标目录不能是当前数据仓库或位于其内部")
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return "", "", err
	}
	if len(entries) != 0 {
		return "", "", errors.New("请选择一个空目录作为新的数据仓库位置")
	}
	return current, target, nil
}

// MigrateDataDir copies the managed data of a stopped repository. A failed
// copy leaves the original pointer in place.
func MigrateDataDir(defaultDir, currentDir, targetDir string) error {
	current, target, err := migrationPaths(currentDir, targetDir)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(target, ".kinpage-migration-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)

	managedItems := []string{
		"library.json",
		"nativeui-layout.json",
		"update-settings.json",
		"sites",
		"articles",
		"drafts",
		"attachments",
		"draft-attachments",
		"imports",
		"integrations",
		"services",
		"templates",
		"offline",
		"offline-media",
		"Public",
		"ipfs",
	}
	for _, name := range managedItems {
		src := filepath.Join(current, name)
		info, err := os.Lstat(src)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err == nil {
			err = copyMigrationEntry(src, filepath.Join(stage, name), info)
		}
		if err != nil {
			return fmt.Errorf("复制 %s 失败：%w", name, err)
		}
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(stage) {
		return errors.New("目标目录在迁移过程中发生变化")
	}
	items, err := os.ReadDir(stage)
	if err != nil {
		return err
	}
	var moved []string
	committed := false
	defer func() {
		if !committed {
			for _, name := range moved {
				_ = os.Rename(filepath.Join(target, name), filepath.Join(stage, name))
			}
		}
	}()
	for _, item := range items {
		name := item.Name()
		destination := filepath.Join(target, name)
		_, err := os.Lstat(destination)
		if !errors.Is(err, os.ErrNotExist) {
			return errors.New("目标目录在迁移过程中发生变化")
		}
		if err := os.Rename(filepath.Join(stage, name), destination); err != nil {
			return err
		}
		moved = append(moved, name)
	}
	if err := writeRepositoryLocation(defaultDir, repoLocationPointer{DataDir: target}); err != nil {
		return err
	}
	committed = true
	return nil
}

func writeRepositoryLocation(defaultDir string, ptr repoLocationPointer) error {
	if err := os.MkdirAll(defaultDir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(defaultDir, ".repo-location-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(ptr)
	if err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(defaultDir, repoLocationFile))
}

func copyMigrationEntry(src, dst string, st os.FileInfo) error {
	if st.IsDir() && st.Mode()&os.ModeSymlink == 0 {
		if err := os.Mkdir(dst, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err == nil {
				err = copyMigrationEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), info)
			}
			if err != nil {
				return fmt.Errorf("复制 %s：%w", entry.Name(), err)
			}
		}
		return nil
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("数据仓库项目不是普通文件或目录：%s", filepath.Base(src))
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	size, err := io.Copy(out, in)
	if err == nil && size != st.Size() {
		err = errors.New("数据仓库文件在复制期间发生变化")
	}
	if err == nil {
		err = out.Sync()
	}
	return errors.Join(err, out.Close())
}
