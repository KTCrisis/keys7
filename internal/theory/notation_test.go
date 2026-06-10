package theory

import "testing"

func TestNoteNameNotations(t *testing.T) {
	letters := map[uint8]string{60: "C4", 69: "A4", 61: "C#4", 48: "C3"}
	for n, want := range letters {
		if got := NoteNameIn(n, Letters); got != want {
			t.Errorf("NoteNameIn(%d, Letters) = %q, want %q", n, got, want)
		}
	}
	solfege := map[uint8]string{60: "Do4", 69: "La4", 67: "Sol4", 61: "Do#4"}
	for n, want := range solfege {
		if got := NoteNameIn(n, Solfege); got != want {
			t.Errorf("NoteNameIn(%d, Solfege) = %q, want %q", n, got, want)
		}
	}
}

func TestParseNote(t *testing.T) {
	good := map[string]uint8{
		"C4": 60, "A4": 69, "F#3": 54, "Bb2": 46, "c4": 60,
		"C-1": 0, "G9": 127, " E4 ": 64,
	}
	for s, want := range good {
		got, err := ParseNote(s)
		if err != nil {
			t.Errorf("ParseNote(%q) error: %v", s, err)
			continue
		}
		if got != want {
			t.Errorf("ParseNote(%q) = %d, want %d", s, got, want)
		}
	}
	for _, s := range []string{"", "H4", "C", "C#", "Cx4", "G#9", "Cb-1", "C 4"} {
		if n, err := ParseNote(s); err == nil {
			t.Errorf("ParseNote(%q) = %d, want error", s, n)
		}
	}
}

// ParseNote must invert NoteNameIn over the whole MIDI range.
func TestParseNoteRoundTrip(t *testing.T) {
	for n := 0; n <= 127; n++ {
		name := NoteNameIn(uint8(n), Letters)
		got, err := ParseNote(name)
		if err != nil {
			t.Fatalf("ParseNote(%q) error: %v", name, err)
		}
		if got != uint8(n) {
			t.Fatalf("ParseNote(NoteNameIn(%d)) = %d", n, got)
		}
	}
}

func TestChordStringNotations(t *testing.T) {
	c := Chord{Root: 9, Suffix: "7", Bass: 9} // A7
	if got := c.StringIn(Letters); got != "A7" {
		t.Errorf("A7 letters = %q", got)
	}
	if got := c.StringIn(Solfege); got != "La7" {
		t.Errorf("A7 solfege = %q", got)
	}
}
