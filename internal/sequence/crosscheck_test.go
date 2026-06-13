package sequence_test

// Cross-check between play7 and keys7: a chord written as a play7 sequence,
// run through play7's real parser (sequence.Parse), should be recognised by
// keys7's identifier (theory.Identify) as the same chord. This pins the two
// tools to a shared understanding — the JSON note encoding play7 emits and the
// chord names keys7 reads back. Note names use sharp spelling (keys7's default
// letters notation).

import (
	"sort"
	"strings"
	"testing"

	"keys7/internal/sequence"
	"keys7/internal/theory"
)

func TestCrossCheckPlay7Keys7(t *testing.T) {
	cases := []struct {
		want  []string // accepted recognitions (>1 when the pitch-class set is ambiguous)
		notes []string
	}{
		// triads
		{[]string{"C"}, []string{"C4", "E4", "G4"}},
		{[]string{"Am"}, []string{"A3", "C4", "E4"}},
		{[]string{"Bdim"}, []string{"B3", "D4", "F4"}},
		{[]string{"C#aug"}, []string{"C#4", "F4", "A4"}},
		// sevenths
		{[]string{"Cmaj7"}, []string{"C4", "E4", "G4", "B4"}},
		{[]string{"G7"}, []string{"G3", "B3", "D4", "F4"}},
		{[]string{"Am7"}, []string{"A3", "C4", "E4", "G4"}},
		{[]string{"Bm7b5"}, []string{"B3", "D4", "F4", "A4"}},
		{[]string{"Bdim7"}, []string{"B3", "D4", "F4", "G#4"}},
		{[]string{"AmMaj7"}, []string{"A3", "C4", "E4", "G#4"}},
		// sixths / adds / extensions
		{[]string{"C6"}, []string{"C4", "E4", "G4", "A4"}},
		{[]string{"Am6"}, []string{"A3", "C4", "E4", "F#4"}},
		{[]string{"C6/9"}, []string{"C4", "E4", "G4", "A4", "D5"}},
		{[]string{"Cadd9"}, []string{"C4", "E4", "G4", "D5"}},
		{[]string{"G9"}, []string{"G3", "B3", "D4", "F4", "A4"}},
		{[]string{"Cmaj9"}, []string{"C4", "E4", "G4", "B4", "D5"}},
		// suspended
		{[]string{"Csus2"}, []string{"C4", "D4", "G4"}},
		{[]string{"Csus4"}, []string{"C4", "F4", "G4"}},
		// inversions (slash chords)
		{[]string{"C/E"}, []string{"E3", "G3", "C4"}},
		{[]string{"C/G"}, []string{"G3", "C4", "E4"}},
		// Genuine enharmonic ambiguity: G-A-C-E is the same pitch-class set as
		// both C6 (C-E-G-A) and Am7 (A-C-E-G) over a G bass. Without a tonal
		// centre, either reading is correct; keys7 resolves bass-first then by
		// ascending pitch class, landing on C6/G. Accept both.
		{[]string{"C6/G", "Am7/G"}, []string{"G3", "A3", "C4", "E4"}},
	}

	for _, c := range cases {
		notes := notesAtZero(t, c.notes)
		got, ok := theory.Identify(notes)
		if !ok {
			t.Errorf("%v: Identify failed, want %v", c.notes, c.want)
			continue
		}
		name := got.String()
		if !contains(c.want, name) {
			t.Errorf("%v: got %q, want one of %v", c.notes, name, c.want)
		}
	}
}

// notesAtZero runs the names through play7's JSON pipeline and returns the MIDI
// notes sounding at t=0 — what keys7 would hear if play7 played this chord.
func notesAtZero(t *testing.T, names []string) []uint8 {
	t.Helper()
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = `"` + n + `"`
	}
	js := `{"steps":[{"notes":[` + strings.Join(q, ",") + `],"beats":1}]}`
	seq, err := sequence.Parse(strings.NewReader(js))
	if err != nil {
		t.Fatalf("parse %v: %v", names, err)
	}
	var on []uint8
	for _, e := range seq.Events {
		if e.At == 0 && e.On {
			on = append(on, e.Note)
		}
	}
	sort.Slice(on, func(i, j int) bool { return on[i] < on[j] })
	return on
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
