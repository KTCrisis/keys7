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

func TestChordStringNotations(t *testing.T) {
	c := Chord{Root: 9, Suffix: "7", Bass: 9} // A7
	if got := c.StringIn(Letters); got != "A7" {
		t.Errorf("A7 letters = %q", got)
	}
	if got := c.StringIn(Solfege); got != "La7" {
		t.Errorf("A7 solfege = %q", got)
	}
}
