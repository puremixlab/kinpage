package attachments

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"windows/internal/store"
)

func TestGenerateGridThumbnailsFromHeroImage(t *testing.T) {
	root := t.TempDir()
	lib, err := store.Open(filepath.Join(root, "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, _ := lib.CreateSite("Gallery", "")
	art, _ := lib.SaveArticle(site.ID, "", "Photo Post", "![cover](cover.png)")

	imgPath := filepath.Join(root, "cover.png")
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{R: 10, G: 120, B: 220, A: 255})
		}
	}
	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	art, err = Add(root, lib, site.ID, art.ID, imgPath)
	if err != nil {
		t.Fatal(err)
	}
	art, err = lib.SetHeroImage(site.ID, art.ID, "cover.png")
	if err != nil {
		t.Fatal(err)
	}

	destSiteDir := filepath.Join(root, "public")
	siteObj, _, _ := findSite(lib, site.ID)
	if err := GenerateGridThumbnails(root, siteObj, art, destSiteDir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"_grid.jpg", "_grid.png"} {
		p := filepath.Join(destSiteDir, art.ID, name)
		st, err := os.Stat(p)
		if err != nil || st.Size() == 0 {
			t.Fatalf("expected generated %s, err=%v", name, err)
		}
	}
}
