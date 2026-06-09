package theory

import (
	"strings"
	"testing"
)

func TestIntervalName(t *testing.T) {
	cases := map[uint8]string{0: "unison", 4: "M3", 7: "P5", 6: "tritone", 10: "m7"}
	for semi, want := range cases {
		if got := IntervalName(semi); got != want {
			t.Errorf("IntervalName(%d) = %q, want %q", semi, got, want)
		}
	}
}

func symbolList(cs []Chord) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

func TestDyadImplicationsInKey(t *testing.T) {
	cmaj := Key{Tonic: 0, Mode: Major}

	// C + E in C major: the chords built on I and vi that contain both.
	got := symbolList(DyadImplications(0, 4, &cmaj))
	for _, want := range []string{"C", "Cmaj7", "Am", "Am7"} {
		if !strings.Contains(got, want) {
			t.Errorf("C+E in C major = %q, missing %q", got, want)
		}
	}
	// Em (E G B) does not contain C, must be absent.
	if strings.Contains(got, "Em") {
		t.Errorf("C+E should not imply Em: %q", got)
	}

	// The tritone B+F in C major implies the dominant G7 (its guide tones).
	got = symbolList(DyadImplications(11, 5, &cmaj))
	if !strings.Contains(got, "G7") {
		t.Errorf("B+F in C major should imply G7: %q", got)
	}
}

func TestDyadImplicationsDegreesSorted(t *testing.T) {
	cmaj := Key{Tonic: 0, Mode: Major}
	cs := DyadImplications(0, 4, &cmaj)
	prev := 0
	for _, c := range cs {
		dc, _ := DegreeOf(cmaj, c)
		if dc.Degree < prev {
			t.Fatalf("not sorted by degree: %s", symbolList(cs))
		}
		prev = dc.Degree
	}
}
