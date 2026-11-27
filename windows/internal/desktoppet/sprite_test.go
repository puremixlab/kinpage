package desktoppet

import (
	"image"
	"testing"
)

func TestAmoSpriteKeepsItsTransparentSilhouette(t *testing.T) {
	for _, size := range []image.Point{image.Pt(152, 122), image.Pt(228, 183), image.Pt(304, 244)} {
		t.Run(size.String(), func(t *testing.T) {
			s, err := newSprite(size)
			if err != nil {
				t.Fatal(err)
			}
			rest := s.paint(pose{})
			if rest.RGBAAt(0, 0).A != 0 || rest.RGBAAt(size.X-1, size.Y-1).A != 0 {
				t.Fatal("the window corners must remain transparent")
			}
			check := func(frame pose, x, y int, light bool) {
				t.Helper()
				point := s.point(x, y)
				got := s.paint(frame).RGBAAt(point.X, point.Y)
				if got.A < 128 || (got.R > 150) != light {
					t.Fatalf("pose %+v at %v = %v, want opaque pixel (light=%v)", frame, point, got, light)
				}
				for i := 3; i < len(s.frame.Pix); i += 4 {
					if (s.frame.Pix[i] != 0) != (s.base.Pix[i] != 0) {
						t.Fatalf("pose %+v changed the sprite silhouette at pixel %d", frame, i/4)
					}
				}
			}
			check(pose{}, 530, 505, true)
			check(pose{lid: eyesClosed}, 530, 505, false)
			check(pose{gaze: gazeLeft}, 530, 505, false)
			check(pose{gaze: gazeRight}, 530, 505, true)
			check(pose{paw: pawReach}, 875, 683, false)
			check(pose{}, 875, 683, true)
		})
	}
}
