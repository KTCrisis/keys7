package ui

import "time"

// The cue is the "your turn" gesture for the AI bridge: two taps of the cue
// note close together. It lets the player hand the turn to the assistant
// without leaving the piano — the assistant's loop reads the journal and only
// answers when it sees a cue event.

// cueNote is A0, the lowest piano key — comfortably below any harmony register,
// so double-tapping it is unambiguous as a gesture.
const cueNote = 21

// cueWindow is how close the two taps must be to count as a cue.
const cueWindow = 2 * time.Second

// cueDetector recognizes the double-tap. Purely time-based, so it is testable
// without a UI or a clock.
type cueDetector struct {
	last time.Time
}

// Tap records a cue-note hit at t and reports whether it completes a cue. A
// completing tap consumes the pair: a third tap starts over rather than
// chaining cues.
func (c *cueDetector) Tap(t time.Time) bool {
	if !c.last.IsZero() && t.Sub(c.last) <= cueWindow {
		c.last = time.Time{}
		return true
	}
	c.last = t
	return false
}
