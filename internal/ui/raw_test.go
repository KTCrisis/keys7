package ui

import (
	"testing"
	"time"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
)

// The raw capture layer: every attack, release and pedal move reaches the
// journal with its own millisecond timestamp — except cue-note taps, which are
// signalling, not music.
func TestRawCapture(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")
	t0 := time.Date(2026, 6, 10, 21, 0, 0, 0, time.UTC)

	m.apply(midi.Event{Kind: midi.NoteOn, Data1: 60, Data2: 72, Timestamp: t0})
	m.apply(midi.Event{Kind: midi.ControlChange, Data1: 64, Data2: 127, Timestamp: t0.Add(100 * time.Millisecond)})
	m.apply(midi.Event{Kind: midi.NoteOff, Data1: 60, Timestamp: t0.Add(200 * time.Millisecond)})
	m.apply(midi.Event{Kind: midi.ControlChange, Data1: 64, Data2: 0, Timestamp: t0.Add(300 * time.Millisecond)})
	m.apply(midi.Event{Kind: midi.NoteOn, Data1: 21, Data2: 70, Timestamp: t0.Add(time.Second)}) // A0 = signal key, excluded

	type raw struct {
		kind string
		midi uint8
		on   bool
	}
	var got []raw
	for _, e := range sink.events {
		if e.Kind == "note" || e.Kind == "pedal" {
			got = append(got, raw{e.Kind, e.Midi, *e.On})
		}
	}
	want := []raw{
		{"note", 60, true},
		{"pedal", 0, true},
		{"note", 60, false},
		{"pedal", 0, false},
	}
	if len(got) != len(want) {
		t.Fatalf("raw events = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("raw %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// Attack velocity and ms stamp on the note-on.
	if e := sink.events[0]; e.Vel != 72 || e.Time != "2026-06-10T21:00:00.000Z" || e.Note != "C4" {
		t.Errorf("note-on event = %+v", e)
	}
}

// Cycling `t` declares the texture in the journal and the header.
func TestTextureCycle(t *testing.T) {
	sink := &captureSink{}
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, sink, "")

	if m.texture.String() != "free" {
		t.Errorf("default texture = %q, want free", m.texture)
	}
	for _, want := range []string{"block", "arpeggio", "free"} {
		m.texture = m.texture.Next()
		if m.texture.String() != want {
			t.Errorf("texture after cycle = %q, want %q", m.texture, want)
		}
	}
}
