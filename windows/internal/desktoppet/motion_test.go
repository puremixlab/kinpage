package desktoppet

import (
	"image"
	"testing"
	"time"
)

func TestAmoMotion(t *testing.T) {
	start := time.Unix(100, 0)
	m := motion{started: start}
	check := func(elapsed time.Duration, target gaze, want pose) {
		t.Helper()
		if got := m.frame(start.Add(elapsed), target); got != want {
			t.Fatalf("at %v, frame = %+v, want %+v", elapsed, got, want)
		}
	}
	check(0, gazeRest, pose{})
	check(2300*time.Millisecond, gazeRest, pose{lid: eyesHalfClosed})
	check(2400*time.Millisecond, gazeRest, pose{lid: eyesClosed})
	check(2600*time.Millisecond, gazeRest, pose{})
	check(3000*time.Millisecond, gazeLeft, pose{})
	check(3150*time.Millisecond, gazeLeft, pose{gaze: gazeLeft})
	check(3200*time.Millisecond, gazeRest, pose{gaze: gazeLeft})
	check(3410*time.Millisecond, gazeRest, pose{})

	m.tapped = start.Add(4 * time.Second)
	check(4*time.Second, gazeRest, pose{gaze: gazeCenter, paw: pawLift})
	check(4150*time.Millisecond, gazeRest, pose{gaze: gazeCenter, paw: pawReach})
	check(4320*time.Millisecond, gazeRest, pose{gaze: gazeCenter})
	check(4580*time.Millisecond, gazeRest, pose{})

	m.reduced = true
	m.tapped = start.Add(6300 * time.Millisecond)
	check(6400*time.Millisecond, gazeRight, pose{gaze: gazeCenter})
	check(6900*time.Millisecond, gazeRight, pose{gaze: gazeRight})
}

func TestAmoPlacementAndGazeOnSecondaryDisplay(t *testing.T) {
	work := image.Rect(-1920, 0, 0, 1040)
	size := image.Pt(152, 122)
	position := clampPosition(image.Pt(-80, 1010), size, work)
	if want := image.Pt(-152, 918); position != want {
		t.Fatalf("clamped position = %v, want %v", position, want)
	}
	bounds := image.Rectangle{Min: position, Max: position.Add(size)}
	for _, test := range []struct {
		cursor image.Point
		want   gaze
	}{
		{image.Pt(-210, 960), gazeRest},
		{image.Pt(-165, 960), gazeLeft},
		{image.Pt(-76, 960), gazeCenter},
		{image.Pt(-10, 960), gazeRight},
		{image.Pt(-76, 800), gazeRest},
	} {
		if got := gazeToward(test.cursor, bounds); got != test.want {
			t.Errorf("gaze toward %v = %v, want %v", test.cursor, got, test.want)
		}
	}
}
