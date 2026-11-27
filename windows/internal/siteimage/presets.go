package siteimage

import (
	"embed"
	"encoding/base64"
	"errors"
	"slices"
	"strings"
)

type PresetAvatar struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	Title    string `json:"title"`
	PNG      string `json:"png"`
}

//go:embed presets/*.png
var presetArtwork embed.FS

var presetAvatars = loadPresetAvatars()

func loadPresetAvatars() []PresetAvatar {
	specs := [...]struct{ key, title string }{
		{"ink", "Ink Bottle"},
		{"fox", "Fox"},
		{"orbit", "Ringed Planet"},
		{"fern", "Fern"},
		{"dawn", "Sunrise"},
		{"moth", "Luna Moth"},
		{"whale", "Blue Whale"},
		{"cloud", "Thundercloud"},
		{"lantern", "Paper Lantern"},
		{"tea", "Tea Cup"},
		{"boat", "Paper Boat"},
		{"peach", "Peach"},
	}
	gallery := make([]PresetAvatar, 0, len(specs))
	for _, spec := range specs {
		png, err := presetArtwork.ReadFile("presets/" + spec.key + ".png")
		if err != nil {
			panic(err)
		}
		gallery = append(gallery, PresetAvatar{
			Key:      spec.key,
			Category: "Illustration",
			Title:    spec.title,
			PNG:      base64.StdEncoding.EncodeToString(png),
		})
	}
	return gallery
}

// PresetAvatars returns the built-in gallery in display order.
func PresetAvatars() []PresetAvatar {
	return slices.Clone(presetAvatars)
}

func PresetAvatarPNG(key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	for _, item := range presetAvatars {
		if strings.EqualFold(item.Key, trimmed) {
			return item.PNG, nil
		}
	}
	return "", errors.New("未找到该内置头像")
}
