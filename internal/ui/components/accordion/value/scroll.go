package value

import (
	"time"

	"github.com/charmbracelet/x/ansi"
)

const (
	scrollStepInterval = 300
	scrollIdleInterval = 2500
)

// ScrollState represents the current scroll animation state.
type ScrollState struct {
	Offset     int
	Dir        int
	Gen        int
	InstanceID int64
}

// HandleTick advances the scroll state by one tick. Returns the new state,
// the next tick interval, and whether to continue scrolling.
func (s ScrollState) HandleTick(
	msg ScrollState,
	rawValue string,
	width int,
) (ScrollState, time.Duration, bool) {
	if msg.InstanceID != s.InstanceID || msg.Gen != s.Gen || width <= 0 {
		return s, 0, false
	}

	maxOff := ansi.StringWidth(rawValue) - width
	if maxOff <= 0 {
		return s, 0, false
	}

	s.Offset += s.Dir

	switch {
	case s.Offset >= maxOff:
		s.Offset = maxOff
		s.Dir = -1

		return s, scrollIdleInterval * time.Millisecond, true
	case s.Offset <= 0:
		s.Offset = 0
		s.Dir = 1

		return s, scrollIdleInterval * time.Millisecond, true
	default:
		return s, scrollStepInterval * time.Millisecond, true
	}
}
