package ui

import (
	"testing"
	"time"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
)

// A line walking below an already-sounding chord must journal as low melody.
func TestLowMelodyUnderHeldChord(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)
	on := func(n uint8, at time.Time) {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: n, Data2: 66, Timestamp: at})
	}
	off := func(n uint8, at time.Time) {
		m.apply(midi.Event{Kind: midi.NoteOff, Data1: n, Timestamp: at})
	}

	// Right hand holds C major (C4 E4 G4); left hand walks D2, E2, F2.
	for _, n := range []uint8{60, 64, 67} {
		on(n, t0)
	}
	on(38, t0.Add(time.Second)) // D2
	off(38, t0.Add(1500*time.Millisecond))
	on(40, t0.Add(2*time.Second)) // E2
	off(40, t0.Add(2500*time.Millisecond))
	on(41, t0.Add(3*time.Second)) // F2

	var got []string
	for _, e := range sink.events {
		if e.Kind == "melody" {
			if e.Reg != "low" {
				t.Errorf("melody %s reg = %q, want low", e.Note, e.Reg)
			}
			got = append(got, e.Note)
		}
	}
	if len(got) != 3 || got[0] != "D2" || got[1] != "E2" || got[2] != "F2" {
		t.Fatalf("low melody = %v, want [D2 E2 F2]", got)
	}
}

// A bass struck together with its chord is harmonic, not melodic: no melody
// event, and the chord keeps its bass (slash chords survive).
func TestBlockChordBassIsNotMelody(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)

	// G2 + C4 E4 G4 struck as one block (same instant): C/G.
	for _, n := range []uint8{43, 60, 64, 67} {
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: n, Data2: 70, Timestamp: t0})
	}
	for _, e := range sink.events {
		if e.Kind == "melody" {
			t.Fatalf("block-chord bass logged as melody: %+v", e)
		}
	}
	if !m.chordOK || m.chord.StringIn(theory.Letters) != "C/G" {
		t.Errorf("chord = %q (ok=%v), want C/G", m.chord.StringIn(theory.Letters), m.chordOK)
	}
}

// High melody events now carry the register annotation.
func TestHighMelodyRegAnnotation(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)
	for _, n := range []uint8{48, 52, 55} { // C3 E3 G3
		m.apply(midi.Event{Kind: midi.NoteOn, Data1: n, Data2: 70, Timestamp: t0})
	}
	m.apply(midi.Event{Kind: midi.NoteOn, Data1: 76, Data2: 70, Timestamp: t0.Add(time.Second)}) // E5
	for _, e := range sink.events {
		if e.Kind == "melody" {
			if e.Reg != "high" {
				t.Errorf("high melody reg = %q, want high", e.Reg)
			}
			return
		}
	}
	t.Fatal("no melody event for the top line")
}
