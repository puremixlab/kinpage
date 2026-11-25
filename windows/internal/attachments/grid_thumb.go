package attachments

import (
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"windows/internal/store"
)

const gridThumbMaxSize = 512

// GenerateGridThumbnails creates center-cropped square _grid.jpg and _grid.png
// derivative thumbnails for Grid/Sepia gallery templates from the article's
// HeroImage (or generated _videoThumbnail.png). These files are written only
// into the rendered site output directory and are never added to attachments.
func GenerateGridThumbnails(root string, site store.Site, article store.Article, destSiteDir string) error {
	var src image.Image
	if article.HeroImage != "" && validName(article.HeroImage) {
		f, err := OpenRegistered(root, site, article, article.HeroImage)
		if err == nil {
			img, _, decErr := image.Decode(f)
			_ = f.Close()
			if decErr == nil {
				src = img
			}
		}
	}
	articleOutDir := filepath.Join(destSiteDir, article.ID)
	if src == nil {
		vtPath := filepath.Join(articleOutDir, "_videoThumbnail.png")
		if f, err := os.Open(vtPath); err == nil {
			img, _, decErr := image.Decode(f)
			_ = f.Close()
			if decErr == nil {
				src = img
			}
		}
	}
	if src == nil {
		return nil
	}

	square := centerCropAndScaleSquare(src, gridThumbMaxSize)
	if err := os.MkdirAll(articleOutDir, 0700); err != nil {
		return err
	}

	jpgPath := filepath.Join(articleOutDir, "_grid.jpg")
	jf, err := os.OpenFile(jpgPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(jf, square, &jpeg.Options{Quality: 85}); err != nil {
		_ = jf.Close()
		return err
	}
	if err := jf.Close(); err != nil {
		return err
	}

	pngPath := filepath.Join(articleOutDir, "_grid.png")
	pf, err := os.OpenFile(pngPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if err := png.Encode(pf, square); err != nil {
		_ = pf.Close()
		return err
	}
	return pf.Close()
}

func centerCropAndScaleSquare(src image.Image, maxSize int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	side := w
	if h < side {
		side = h
	}
	if side <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	x0 := b.Min.X + (w-side)/2
	y0 := b.Min.Y + (h-side)/2

	outSize := side
	if outSize > maxSize {
		outSize = maxSize
	}
	dst := image.NewRGBA(image.Rect(0, 0, outSize, outSize))
	if outSize == side {
		draw.Draw(dst, dst.Bounds(), src, image.Pt(x0, y0), draw.Src)
		return dst
	}
	// Area/nearest-neighbor downscale without external imaging dependencies.
	for dy := 0; dy < outSize; dy++ {
		sy := y0 + (dy*side)/outSize
		for dx := 0; dx < outSize; dx++ {
			sx := x0 + (dx*side)/outSize
			dst.Set(dx, dy, src.At(sx, sy))
		}
	}
	return dst
}
