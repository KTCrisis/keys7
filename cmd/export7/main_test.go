package main

import (
	"strings"
	"testing"

	"keys7/internal/smf"
)

func TestTranscribe(t *testing.T) {
	// A C4 struck then released 500 ms later, with a pedal press in between.
	journal := strings.Join([]string{
		`{"t":"2026-06-13T12:00:00.000Z","kind":"key","key":"C major"}`, // ignored
		`{"t":"2026-06-13T12:00:00.000Z","kind":"note","midi":60,"v":90,"on":true}`,
		`{"t":"2026-06-13T12:00:00.100Z","kind":"pedal","on":true}`,
		`{"t":"2026-06-13T12:00:00.500Z","kind":"note","midi":60,"on":false}`,
		`garbage line that should be skipped`,
	}, "\n")

	msgs, n, err := transcribe(strings.NewReader(journal))
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("transcribed %d events, want 3 (note on, pedal, note off)", n)
	}
	want := []smf.Msg{
		{MS: 0, Status: smf.NoteOn, D1: 60, D2: 90},
		{MS: 100, Status: smf.ControlChange, D1: 64, D2: 127},
		{MS: 500, Status: smf.NoteOff, D1: 60, D2: 0},
	}
	for i, w := range want {
		got := msgs[i]
		if got != w {
			t.Errorf("msg %d = %+v, want %+v", i, got, w)
		}
	}
}
