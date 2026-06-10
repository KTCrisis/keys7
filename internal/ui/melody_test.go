package ui

import (
	"testing"
	"time"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
)

// A melody note struck over a held chord must reach the sink as a "melody"
// event with the onset's own (millisecond) timestamp; the chord tones must not.
func TestModelLogsMelodyOnsets(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")

	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	press := func(note uint8, at time.Time) {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: note, Data2: 70, Timestamp: at})
	}

	// Left hand holds C major; the right hand line sits a register above.
	press(48, t0) // C3
	press(52, t0) // E3
	press(55, t0) // G3
	press(76, t0.Add(time.Second))                        // E5 — melody
	m.apply(midi.Event{Kind: midi.NoteOff, Data1: 76, Timestamp: t0.Add(time.Second + 400*time.Millisecond)})
	press(74, t0.Add(time.Second+500*time.Millisecond))   // D5 — melody

	var got []string
	for _, e := range sink.events {
		if e.Kind == "melody" {
			got = append(got, e.Note)
			if e.Midi == 0 || e.Vel != 70 {
				t.Errorf("melody event missing midi/vel: %+v", e)
			}
		}
	}
	if len(got) != 2 || got[0] != "E5" || got[1] != "D5" {
		t.Fatalf("melody notes logged = %v, want [E5 D5]", got)
	}

	// Millisecond-precision stamp, from the event's own clock.
	want := "2026-06-10T20:00:01.500Z"
	last := sink.events[len(sink.events)-1]
	for _, e := range sink.events {
		if e.Kind == "melody" && e.Note == "D5" {
			last = e
		}
	}
	if last.Time != want {
		t.Errorf("melody stamp = %q, want %q", last.Time, want)
	}
}

// With the split off (key `e`), nothing is classified as melody.
func TestNoMelodyLogWhenSplitOff(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	m.splitMelody = false

	t0 := time.Date(2026, 6, 10, 20, 0, 0, 0, time.UTC)
	for _, n := range []uint8{48, 52, 55, 76} {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: n, Data2: 70, Timestamp: t0})
	}
	for _, e := range sink.events {
		if e.Kind == "melody" {
			t.Fatalf("melody event logged with split off: %+v", e)
		}
	}
}
