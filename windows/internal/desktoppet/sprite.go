package desktoppet

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"sync"

	xdraw "golang.org/x/image/draw"
)

//go:embed assets/amo.png
var amoPNG []byte

// All feature coordinates refer to the approved, unmodified Amo artwork.
var spriteCrop = image.Rect(221, 188, 1203, 977)
var eyeBands = [...]struct{ left, width int }{{491, 164}, {768, 161}}
var decodeSprite = sync.OnceValues(func() (image.Image, error) {
	return png.Decode(bytes.NewReader(amoPNG))
})

type sprite struct {
	base, frame     *image.RGBA
	ivory, charcoal color.RGBA
}

func newSprite(size image.Point) (*sprite, error) {
	art, err := decodeSprite()
	if err != nil {
		return nil, err
	}
	bounds := image.Rectangle{Max: size}
	s := &sprite{base: image.NewRGBA(bounds), frame: image.NewRGBA(bounds)}
	xdraw.NearestNeighbor.Scale(s.base, bounds, art, spriteCrop, draw.Src, nil)
	// Drop faint edge pixels so transparent parts of the window pass clicks through.
	for i := 0; i < len(s.base.Pix); i += 4 {
		if s.base.Pix[i+3] < 128 {
			clear(s.base.Pix[i : i+4])
		}
	}
	ivory, charcoal := s.point(512, 501), s.point(600, 470)
	s.ivory = s.base.RGBAAt(ivory.X, ivory.Y)
	s.ivory.A = 255
	s.charcoal = s.base.RGBAAt(charcoal.X, charcoal.Y)
	s.charcoal.A = 255
	return s, nil
}

func (s *sprite) point(x, y int) image.Point {
	return image.Pt(
		((x-spriteCrop.Min.X)*s.base.Bounds().Dx()+spriteCrop.Dx()/2)/spriteCrop.Dx(),
		((y-spriteCrop.Min.Y)*s.base.Bounds().Dy()+spriteCrop.Dy()/2)/spriteCrop.Dy(),
	)
}

func (s *sprite) area(x, y, width, height int) image.Rectangle {
	return image.Rectangle{Min: s.point(x, y), Max: s.point(x+width, y+height)}
}

func (s *sprite) fill(x, y, width, height int, c color.RGBA) {
	draw.Draw(s.frame, s.area(x, y, width, height), image.NewUniform(c), image.Point{}, draw.Src)
}

func (s *sprite) coverEye(left, width, height int) {
	draw.Draw(s.frame, s.area(left-2, 487, width+4, height), s.base, s.point(left-2, 433), draw.Src)
}

func (s *sprite) paint(p pose) *image.RGBA {
	copy(s.frame.Pix, s.base.Pix)
	for _, eye := range eyeBands {
		if p.gaze != gazeRest {
			s.coverEye(eye.left, eye.width, 55)
			s.fill(eye.left, 490, eye.width, 32, s.ivory)
			s.fill(eye.left+21, 522, eye.width-42, 17, s.ivory)
			pupil := eye.left + (eye.width-28)/2 + (int(p.gaze)-int(gazeCenter))*40
			s.fill(pupil, 490, 28, 39, s.charcoal)
		}
		switch p.lid {
		case eyesHalfClosed:
			s.coverEye(eye.left, eye.width, 35)
		case eyesClosed:
			s.coverEye(eye.left, eye.width, 55)
		}
	}
	switch p.paw {
	case pawLift:
		draw.Draw(s.frame, s.area(829, 638, 106, 30), s.base, s.point(829, 706), draw.Src)
	case pawReach:
		s.fill(830, 633, 104, 36, s.charcoal)
		s.fill(854, 669, 56, 29, s.charcoal)
	}
	return s.frame
}
