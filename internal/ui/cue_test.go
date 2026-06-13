package ui

import (
	"testing"
	"time"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/session"
	"keys7/internal/theory"
)

func TestCueDetector(t *testing.T) {
	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	var c cueDetector

	if c.Tap(21, t0) != CueNone {
		t.Error("first tap completed a cue")
	}
	if c.Tap(21, t0.Add(500*time.Millisecond)) != CueTurn {
		t.Error("second tap of A0 within the window did not complete a turn cue")
	}
	// The pair was consumed: this tap starts a new one.
	if c.Tap(21, t0.Add(time.Second)) != CueNone {
		t.Error("tap after a completed cue chained into another cue")
	}
	if c.Tap(21, t0.Add(2*time.Second)) != CueTurn {
		t.Error("second tap of the new pair did not complete a cue")
	}
}

func TestCueDetectorGestures(t *testing.T) {
	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)

	// Each signal key double-taps to its own gesture.
	for note, want := range map[uint8]Cue{21: CueTurn, 22: CueReplay, 23: CueTranspose, 24: CueHarmonize} {
		var c cueDetector
		c.Tap(note, t0)
		if got := c.Tap(note, t0.Add(300*time.Millisecond)); got != want {
			t.Errorf("double-tap of %d = %v, want %v", note, got, want)
		}
	}

	// Two different signal keys do not complete a cue.
	var c cueDetector
	c.Tap(21, t0)
	if c.Tap(22, t0.Add(300*time.Millisecond)) != CueNone {
		t.Error("taps on different signal keys completed a cue")
	}
}

func TestCueDetectorWindowExpires(t *testing.T) {
	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	var c cueDetector

	c.Tap(21, t0)
	if c.Tap(21, t0.Add(cueWindow+time.Millisecond)) != CueNone {
		t.Error("taps further apart than the window completed a cue")
	}
	// The late tap re-armed the detector: one more inside the window cues.
	if c.Tap(21, t0.Add(cueWindow+500*time.Millisecond)) != CueTurn {
		t.Error("tap within the window of the re-armed detector did not cue")
	}
}

// captureSink records emitted events for assertions.
type captureSink struct{ events []session.HarmonicEvent }

func (s *captureSink) Emit(e session.HarmonicEvent) { s.events = append(s.events, e) }

// A double-tap of a signal key must travel through apply and reach the sink as a
// cue carrying the gesture, without polluting the harmonic state.
func TestModelEmitsCueEvent(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")

	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	tap := func(note uint8, at time.Time) {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: note, Data2: 64, Timestamp: at})
		m.apply(midi.Event{Kind: midi.NoteOff, Data1: note, Timestamp: at})
	}

	tap(22, t0) // A#0 = replay
	tap(22, t0.Add(time.Second))

	var cues []session.HarmonicEvent
	for _, e := range sink.events {
		if e.Kind == "cue" {
			cues = append(cues, e)
		}
	}
	if len(cues) != 1 {
		t.Fatalf("got %d cue events, want 1 (events: %+v)", len(cues), sink.events)
	}
	if cues[0].Cue != "replay" {
		t.Errorf("cue gesture = %q, want replay", cues[0].Cue)
	}
	if m.cuedAt.IsZero() || m.lastCue != CueReplay {
		t.Errorf("cuedAt/lastCue not set (lastCue=%v)", m.lastCue)
	}
	// The signal key must not have entered the harmonic picture.
	if len(m.held) != 0 {
		t.Errorf("signal key leaked into held notes: %v", m.held)
	}
}
