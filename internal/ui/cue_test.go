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

	if c.Tap(t0) {
		t.Error("first tap completed a cue")
	}
	if !c.Tap(t0.Add(500 * time.Millisecond)) {
		t.Error("second tap within the window did not complete a cue")
	}
	// The pair was consumed: this tap starts a new one.
	if c.Tap(t0.Add(time.Second)) {
		t.Error("tap after a completed cue chained into another cue")
	}
	if !c.Tap(t0.Add(2 * time.Second)) {
		t.Error("second tap of the new pair did not complete a cue")
	}
}

func TestCueDetectorWindowExpires(t *testing.T) {
	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	var c cueDetector

	c.Tap(t0)
	if c.Tap(t0.Add(cueWindow + time.Millisecond)) {
		t.Error("taps further apart than the window completed a cue")
	}
	// The late tap re-armed the detector: one more inside the window cues.
	if !c.Tap(t0.Add(cueWindow + 500*time.Millisecond)) {
		t.Error("tap within the window of the re-armed detector did not cue")
	}
}

// captureSink records emitted events for assertions.
type captureSink struct{ events []session.HarmonicEvent }

func (s *captureSink) Emit(e session.HarmonicEvent) { s.events = append(s.events, e) }

// A double-tap of the cue note must travel through apply and reach the sink.
func TestModelEmitsCueEvent(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")

	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	tap := func(at time.Time) {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: cueNote, Data2: 64, Timestamp: at})
		m.apply(midi.Event{Kind: midi.NoteOff, Data1: cueNote, Timestamp: at})
	}

	tap(t0)
	tap(t0.Add(time.Second))

	var cues int
	for _, e := range sink.events {
		if e.Kind == "cue" {
			cues++
		}
	}
	if cues != 1 {
		t.Fatalf("got %d cue events, want 1 (events: %+v)", cues, sink.events)
	}
	if m.cuedAt.IsZero() {
		t.Error("cuedAt not set for the header badge")
	}
}
