package ui

import (
	"testing"
	"time"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
)

const ccSustain = 64

// The live scenario that motivated pedal segmentation: chord pedaled hands-off,
// melody played on top. The chord must stay in the harmonic picture and the
// line must be classified (and journaled) as melody.
func TestPedalSustainsChordUnderMelody(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)

	ev := func(kind midi.Kind, d1, d2 uint8, at time.Time) {
		m.apply(midi.Event{Kind: kind, Data1: d1, Data2: d2, Timestamp: at})
	}

	// Pedal down, play A minor, lift the fingers.
	ev(midi.ControlChange, ccSustain, 127, t0)
	for _, n := range []uint8{45, 52, 57, 60} { // A2 E3 A3 C4
		ev(midi.NoteOn, n, 70, t0)
	}
	for _, n := range []uint8{45, 52, 57, 60} {
		ev(midi.NoteOff, n, 0, t0.Add(time.Second))
	}

	if !m.chordOK || m.chord.StringIn(theory.Letters) != "Am" {
		t.Fatalf("pedaled chord lost after releasing keys: chordOK=%v chord=%q", m.chordOK, m.chord.StringIn(theory.Letters))
	}

	// Melody over the pedaled chord must journal.
	ev(midi.NoteOn, 76, 65, t0.Add(2*time.Second)) // E5
	var melody []string
	for _, e := range sink.events {
		if e.Kind == "melody" {
			melody = append(melody, e.Note)
		}
	}
	if len(melody) != 1 || melody[0] != "E5" {
		t.Fatalf("melody over pedaled chord = %v, want [E5]", melody)
	}

	// Pedal up: the dampers fall, only physically held keys remain.
	ev(midi.ControlChange, ccSustain, 0, t0.Add(3*time.Second))
	if got := m.sounding(); len(got) != 1 || got[0] != 76 {
		t.Fatalf("after pedal up, sounding = %v, want [76]", got)
	}
}

// Re-striking a sustained note must not leave a stale copy: releasing it again
// with the pedal up takes it out of the picture.
func TestRestrikeClearsSustained(t *testing.T) {
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, nil, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)
	ev := func(kind midi.Kind, d1, d2 uint8, at time.Time) {
		m.apply(midi.Event{Kind: kind, Data1: d1, Data2: d2, Timestamp: at})
	}

	ev(midi.ControlChange, ccSustain, 127, t0)
	ev(midi.NoteOn, 60, 70, t0)
	ev(midi.NoteOff, 60, 0, t0) // sustained by the pedal
	ev(midi.ControlChange, ccSustain, 0, t0.Add(time.Second))
	ev(midi.NoteOn, 60, 70, t0.Add(2*time.Second)) // re-struck, pedal up
	ev(midi.NoteOff, 60, 0, t0.Add(3*time.Second)) // plain release

	if got := m.sounding(); len(got) != 0 {
		t.Fatalf("sounding = %v, want empty", got)
	}
}
