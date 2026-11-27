package siteimage

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJPEGConversionAndGeneratedICO(t *testing.T) {
	var in bytes.Buffer
	if err := jpeg.Encode(&in, image.NewRGBA(image.Rect(0, 0, 12, 8)), nil); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "image.jpg")
	if err := os.WriteFile(file, in.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := ReadAvatar(file)
	if err != nil {
		t.Fatal(err)
	}
	img, data, err := DecodePNG(value)
	if err != nil || img.Bounds().Dx() != 12 || img.Bounds().Dy() != 8 {
		t.Fatalf("conversion failed: %v", err)
	}
	if _, err = png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err = WriteFiles(dir, value); err != nil {
		t.Fatal(err)
	}
	ico, err := os.ReadFile(filepath.Join(dir, "favicon.ico"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ico) < 22 || binary.LittleEndian.Uint16(ico[2:4]) != 1 || binary.LittleEndian.Uint16(ico[4:6]) != 1 || binary.LittleEndian.Uint32(ico[18:22]) != 22 {
		t.Fatal("invalid ICO header")
	}
	icon, err := png.Decode(bytes.NewReader(ico[22:]))
	if err != nil || icon.Bounds().Dx() != 64 || icon.Bounds().Dy() != 64 {
		t.Fatalf("invalid icon: %v", err)
	}
}

func TestPresetAvatarsAndCircleCropPNG(t *testing.T) {
	presets := PresetAvatars()
	if len(presets) < 12 {
		t.Fatalf("expected at least 12 preset avatars, got %d", len(presets))
	}
	for _, preset := range presets {
		t.Run(preset.Key, func(t *testing.T) {
			stored, err := PresetAvatarPNG(" " + strings.ToUpper(preset.Key) + " ")
			if err != nil || stored == "" || stored != preset.PNG {
				t.Fatalf("preset lookup did not return the displayed artwork: %v", err)
			}
			cropped, err := CircleCropPNG(stored)
			if err != nil {
				t.Fatal(err)
			}
			img, _, err := DecodePNG(cropped)
			if err != nil {
				t.Fatal(err)
			}
			if bounds := img.Bounds(); bounds.Dx() != 384 || bounds.Dy() != 384 {
				t.Fatalf("expected a 384px square avatar, got %v", bounds)
			}
			_, _, _, alpha := img.At(0, 0).RGBA()
			if alpha != 0 {
				t.Fatalf("expected a transparent corner after circle crop, got alpha %d", alpha)
			}
		})
	}
}
