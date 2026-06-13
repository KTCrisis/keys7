package ui

import "time"

// A cue is a signalling gesture played on the lowest piano keys — below any
// harmony register — so it never collides with musical material. Each cue is a
// double-tap of one dedicated key; the four lowest keys of an 88-key board form
// a "signal bar" that hands the assistant a specific intent without leaving the
// bench. keys7 only detects and journals the gesture; what the assistant does
// with replay / transpose / harmonise is the session protocol's business.

// Cue is a recognised signal-bar gesture.
type Cue int

const (
	CueNone      Cue = iota
	CueTurn          // A0  — your turn (answer now)
	CueReplay        // A#0 — play my last phrase back
	CueTranspose     // B0  — transpose / move it
	CueHarmonize     // C1  — harmonise / add voices
)

// cueKeys maps the signal-bar MIDI notes to their cue (A0, A#0, B0, C1).
var cueKeys = map[uint8]Cue{
	21: CueTurn,
	22: CueReplay,
	23: CueTranspose,
	24: CueHarmonize,
}

func (c Cue) String() string {
	switch c {
	case CueTurn:
		return "turn"
	case CueReplay:
		return "replay"
	case CueTranspose:
		return "transpose"
	case CueHarmonize:
		return "harmonise"
	default:
		return ""
	}
}

// isCueKey reports whether a note belongs to the signal bar — excluded from
// harmonic analysis, since it is a gesture, not a note.
func isCueKey(note uint8) bool { _, ok := cueKeys[note]; return ok }

// cueWindow is how close the two taps must be to count as a cue.
const cueWindow = 2 * time.Second

// cueDetector recognises a double-tap on one signal-bar key. Purely time-based,
// so it is testable without a UI or a clock.
type cueDetector struct {
	lastNote uint8
	last     time.Time
}

// Tap records a hit on signal-bar note `note` at t and returns the cue it
// completes, or CueNone. A completing tap consumes the pair; a different signal
// key, or a gap longer than the window, restarts rather than chaining.
func (c *cueDetector) Tap(note uint8, t time.Time) Cue {
	cue, ok := cueKeys[note]
	if !ok {
		return CueNone
	}
	if c.lastNote == note && !c.last.IsZero() && t.Sub(c.last) <= cueWindow {
		c.last, c.lastNote = time.Time{}, 0
		return cue
	}
	c.lastNote, c.last = note, t
	return CueNone
}
