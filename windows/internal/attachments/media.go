package attachments

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"

	"windows/internal/mediainfo"
	"windows/internal/store"
)

func ReadDraftPreviewImage(root string, draft store.Draft, name string) ([]byte, string, error) {
	f, err := OpenDraftAttachment(root, draft, name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	return readPreviewImage(f)
}

// ReadPreviewImage reads only a registered raster attachment from the current article.
func ReadPreviewImage(root string, site store.Site, article store.Article, name string) ([]byte, string, error) {
	f, err := OpenRegistered(root, site, article, name)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	return readPreviewImage(f)
}

func readPreviewImage(f *os.File) ([]byte, string, error) {
	const limit = 8 << 20
	info, err := f.Stat()
	if err != nil || info.Size() > limit {
		return nil, "", errors.New("图片无效或过大")
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || len(data) > limit {
		return nil, "", errors.New("图片读取失败或过大")
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
	default:
		return nil, "", errors.New("不支持预览此图片格式")
	}
	return data, mime, nil
}

// EnrichMedia derives optional audio metadata from the current registered file.
func EnrichMedia(root string, site store.Site, articles []store.Article) ([]store.Article, error) {
	out := append([]store.Article(nil), articles...)
	for i := range out {
		article := &out[i]
		if article.AudioFilename == "" {
			continue
		}
		f, err := OpenRegistered(root, site, *article, article.AudioFilename)
		if err != nil {
			return nil, err
		}
		info, err := f.Stat()
		path := f.Name()
		f.Close()
		if err != nil {
			return nil, err
		}
		size := info.Size()
		article.AudioByteLength = &size
		if article.AudioDuration == nil {
			article.AudioDuration = mediainfo.Duration(path)
		}
	}
	return out, nil
}

// GenerateVideoThumbnail creates a public poster if the system can extract one.
func GenerateVideoThumbnail(root string, site store.Site, article store.Article, destination string) error {
	if article.VideoFilename == "" {
		return nil
	}
	target := filepath.Join(destination, article.ID, "_videoThumbnail.png")
	if info, err := os.Lstat(target); err == nil {
		if !info.Mode().IsRegular() {
			return errors.New("视频缩略图无效")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	file, err := OpenRegistered(root, site, article, article.VideoFilename)
	if err != nil {
		return err
	}
	path := file.Name()
	file.Close()
	data, err := mediainfo.VideoThumbnail(path)
	if err != nil {
		return nil
	}
	return os.WriteFile(target, data, 0600)
}

// ThumbnailDataURL renders a small inline preview for the editor media tray.
// Decodable raster images are downscaled to a 160px PNG; WebP passes through
// untouched because the browser can scale it visually.
func ThumbnailDataURL(data []byte, mime string) (string, error) {
	if mime == "image/webp" {
		return "data:image/webp;base64," + base64.StdEncoding.EncodeToString(data), nil
	}
	var source image.Image
	switch mime {
	case "image/png":
		decoded, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return "", errors.New("图片无法解码")
		}
		source = decoded
	case "image/jpeg":
		decoded, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return "", errors.New("图片无法解码")
		}
		source = decoded
	case "image/gif":
		decoded, err := gif.Decode(bytes.NewReader(data))
		if err != nil {
			return "", errors.New("图片无法解码")
		}
		source = decoded
	default:
		return "", errors.New("不支持生成此图片预览")
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return "", errors.New("图片尺寸无效")
	}
	const limit = 160
	scale := 1.0
	if width > limit || height > limit {
		scale = math.Min(float64(limit)/float64(width), float64(limit)/float64(height))
	}
	targetW := int(float64(width)*scale + 0.5)
	targetH := int(float64(height)*scale + 0.5)
	if targetW < 1 {
		targetW = 1
	}
	if targetH < 1 {
		targetH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		startY := y * height / targetH
		endY := (y + 1) * height / targetH
		if endY <= startY {
			endY = startY + 1
		}
		for x := 0; x < targetW; x++ {
			startX := x * width / targetW
			endX := (x + 1) * width / targetW
			if endX <= startX {
				endX = startX + 1
			}
			var sumR, sumG, sumB, sumA, count float64
			for sy := startY; sy < endY; sy++ {
				for sx := startX; sx < endX; sx++ {
					r, g, b, a := source.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
					alpha := float64(a) / 0xffff
					sumR += float64(r) * alpha
					sumG += float64(g) * alpha
					sumB += float64(b) * alpha
					sumA += alpha
					count++
				}
			}
			if sumA <= 0 {
				dst.Set(x, y, color.NRGBA{0, 0, 0, 0})
				continue
			}
			dst.Set(x, y, color.NRGBA{
				R: uint8(sumR / sumA / 0x100),
				G: uint8(sumG / sumA / 0x100),
				B: uint8(sumB / sumA / 0x100),
				A: uint8(sumA / count / 0x10000 * 0xffff),
			})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes()), nil
}
