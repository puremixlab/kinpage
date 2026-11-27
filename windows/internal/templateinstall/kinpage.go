package templateinstall

import (
	"bytes"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"windows/internal/templatecatalog"
)

// Keep shipped versions embedded so sites using earlier installs can still rebuild.
const kinpageV1 = "builtins/kinpage/v1"

func kinpageSource(id string) string {
	for _, item := range templatecatalog.Kinpage() {
		if item.ID == id {
			return kinpageV1 + "/" + id
		}
	}
	return ""
}

func verifyKinpage(root string, metadata []byte) error {
	metadata = normalizeTemplateText(metadata)
	for _, item := range templatecatalog.Kinpage() {
		source := kinpageV1 + "/" + item.ID
		expected, err := bundled.ReadFile(source + "/template.json")
		if err != nil {
			return err
		}
		if bytes.Equal(metadata, normalizeTemplateText(expected)) {
			return verifyKinpageFiles(root, source)
		}
	}
	return errors.New("模板信息与已验证版本不同")
}

func verifyKinpageFiles(root, source string) error {
	for _, name := range []string{"templates", "assets"} {
		files := map[string]string{}
		for _, dir := range []string{kinpageV1 + "/common/" + name, source + "/" + name} {
			err := fs.WalkDir(bundled, dir, func(file string, entry fs.DirEntry, err error) error {
				if err != nil || entry.IsDir() {
					return err
				}
				data, err := bundled.ReadFile(file)
				if err != nil {
					return err
				}
				files[strings.TrimPrefix(file, dir+"/")] = templateTextDigest(data)
				return nil
			})
			if err != nil {
				return err
			}
		}
		if err := verifyFiles(filepath.Join(root, name), files); err != nil {
			return err
		}
	}
	return nil
}
