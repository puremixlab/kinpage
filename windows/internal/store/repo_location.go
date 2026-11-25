package store

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const repoLocationFile = "repo-location.json"

type repoLocationPointer struct {
	DataDir string `json:"dataDir"`
}

// ResolveDataDir checks whether defaultDir contains a repo-location.json pointer
// to a custom repository directory. If present and valid, it returns that path;
// otherwise it returns defaultDir.
func ResolveDataDir(defaultDir string) string {
	raw, err := os.ReadFile(filepath.Join(defaultDir, repoLocationFile))
	if err != nil {
		return defaultDir
	}
	var ptr repoLocationPointer
	if json.Unmarshal(raw, &ptr) != nil || strings.TrimSpace(ptr.DataDir) == "" {
		return defaultDir
	}
	clean := filepath.Clean(ptr.DataDir)
	if info, err := os.Stat(clean); err == nil && info.IsDir() {
		return clean
	}
	return defaultDir
}

// MigrateDataDir copies managed Planet library data from currentDir to an
// existing empty targetDir and records the pointer in defaultDir/repo-location.json.
func MigrateDataDir(defaultDir, currentDir, targetDir string) error {
	curClean := filepath.Clean(strings.TrimSpace(currentDir))
	dstClean := filepath.Clean(strings.TrimSpace(targetDir))
	if dstClean == "" || curClean == "" {
		return errors.New("数据目录路径无效")
	}
	if strings.EqualFold(curClean, dstClean) {
		return errors.New("目标目录与当前数据目录相同")
	}
	rel, err := filepath.Rel(curClean, dstClean)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return errors.New("目标目录不能位于当前数据目录内部")
	}
	info, err := os.Stat(dstClean)
	if err != nil || !info.IsDir() {
		return errors.New("目标目录不存在")
	}
	entries, err := os.ReadDir(dstClean)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return errors.New("请选择一个空目录作为新的数据仓库位置")
	}

	managedItems := []string{
		"library.json",
		"nativeui-layout.json",
		"sites",
		"articles",
		"drafts",
		"imports",
		"services",
		"templates",
		"offline",
		"ipfs",
	}
	for _, name := range managedItems {
		src := filepath.Join(curClean, name)
		st, err := os.Stat(src)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		dst := filepath.Join(dstClean, name)
		if st.IsDir() {
			if err := copyDirRecursive(src, dst); err != nil {
				return err
			}
		} else {
			if err := copyFileRegular(src, dst); err != nil {
				return err
			}
		}
	}

	if err := os.MkdirAll(defaultDir, 0700); err != nil {
		return err
	}
	ptrBytes, err := json.MarshalIndent(repoLocationPointer{DataDir: dstClean}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(defaultDir, repoLocationFile), ptrBytes, 0600)
}

func copyDirRecursive(src, dst string) error {
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		sp := filepath.Join(src, e.Name())
		dp := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDirRecursive(sp, dp); err != nil {
				return err
			}
		} else if e.Type().IsRegular() {
			if err := copyFileRegular(sp, dp); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFileRegular(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}
