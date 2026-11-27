package desktoppet

import (
	"image"
	"time"
)

type gaze int8

const (
	gazeRest gaze = iota
	gazeLeft
	gazeCenter
	gazeRight
)

type eyelid uint8

const (
	eyesOpen eyelid = iota
	eyesHalfClosed
	eyesClosed
)

type paw uint8

const (
	pawRest paw = iota
	pawLift
	pawReach
)

type pose struct {
	gaze gaze
	lid  eyelid
	paw  paw
}

type motion struct {
	started        time.Time
	tapped         time.Time
	gaze           gaze
	requestedGaze  gaze
	requestChanged time.Time
	reduced        bool
}

func (m *motion) frame(now time.Time, target gaze) pose {
	if target != m.requestedGaze {
		m.requestedGaze = target
		m.requestChanged = now
	}
	delay := 135 * time.Millisecond
	if target == gazeRest {
		delay = 210 * time.Millisecond
	}
	if m.reduced || now.Sub(m.requestChanged) >= delay {
		m.gaze = target
	}
	frame := pose{gaze: m.gaze}
	if !m.reduced {
		frame.lid = blinkAt(now.Sub(m.started))
	}

	sinceTap := now.Sub(m.tapped)
	if !m.tapped.IsZero() && 0 <= sinceTap && sinceTap < 580*time.Millisecond {
		frame.gaze = gazeCenter
		frame.lid = eyesOpen
		if !m.reduced {
			switch {
			case sinceTap < 110*time.Millisecond:
				frame.paw = pawLift
			case sinceTap < 320*time.Millisecond:
				frame.paw = pawReach
			}
		}
	}
	return frame
}

func blinkAt(elapsed time.Duration) eyelid {
	const firstBlink = 2300 * time.Millisecond
	if elapsed < firstBlink {
		return eyesOpen
	}
	switch phase := (elapsed - firstBlink) % (3885 * time.Millisecond); {
	case phase < 70*time.Millisecond:
		return eyesHalfClosed
	case phase < 200*time.Millisecond:
		return eyesClosed
	case phase < 285*time.Millisecond:
		return eyesHalfClosed
	default:
		return eyesOpen
	}
}

func gazeToward(cursor image.Point, bounds image.Rectangle) gaze {
	if !cursor.In(bounds.Inset(-bounds.Dx() * 3 / 8)) {
		return gazeRest
	}
	offset := cursor.X - (bounds.Min.X + bounds.Dx()/2)
	deadZone := bounds.Dx() / 10
	switch {
	case offset < -deadZone:
		return gazeLeft
	case offset > deadZone:
		return gazeRight
	default:
		return gazeCenter
	}
}

func clampPosition(position, size image.Point, workArea image.Rectangle) image.Point {
	return image.Pt(
		max(workArea.Min.X, min(position.X, workArea.Max.X-size.X)),
		max(workArea.Min.Y, min(position.Y, workArea.Max.Y-size.Y)),
	)
}
