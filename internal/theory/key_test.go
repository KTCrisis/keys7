package theory

import "testing"

func romansOf(dcs []DegreeChord) []string {
	out := make([]string, len(dcs))
	for i, dc := range dcs {
		out[i] = dc.Roman
	}
	return out
}

func symbolsOf(dcs []DegreeChord) []string {
	out := make([]string, len(dcs))
	for i, dc := range dcs {
		out[i] = dc.Chord.String()
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiatonicMajor(t *testing.T) {
	tri := DiatonicTriads(Key{Tonic: 0, Mode: Major}) // C major
	if r := romansOf(tri); !eq(r, []string{"I", "ii", "iii", "IV", "V", "vi", "vii°"}) {
		t.Errorf("romans = %v", r)
	}
	if s := symbolsOf(tri); !eq(s, []string{"C", "Dm", "Em", "F", "G", "Am", "Bdim"}) {
		t.Errorf("symbols = %v", s)
	}
}

func TestDiatonicMinorHarmonic(t *testing.T) {
	tri := DiatonicTriads(Key{Tonic: 9, Mode: Minor}) // A minor
	if r := romansOf(tri); !eq(r, []string{"i", "ii°", "III+", "iv", "V", "VI", "vii°"}) {
		t.Errorf("romans = %v", r)
	}
	if s := symbolsOf(tri); !eq(s, []string{"Am", "Bdim", "Caug", "Dm", "E", "F", "G#dim"}) {
		t.Errorf("symbols = %v", s)
	}
}

func TestDegreeOf(t *testing.T) {
	cmaj := Key{Tonic: 0, Mode: Major}
	cases := []struct {
		chord Chord
		deg   int
		ok    bool
	}{
		{Chord{Root: 7, Suffix: "7"}, 5, true},      // G7 -> V
		{Chord{Root: 0, Suffix: "maj7"}, 1, true},   // Cmaj7 -> I
		{Chord{Root: 9, Suffix: "m"}, 6, true},      // Am -> vi
		{Chord{Root: 11, Suffix: "dim"}, 7, true},   // Bdim -> vii°
		{Chord{Root: 0, Suffix: "sus4"}, 0, false},  // sus has no degree
		{Chord{Root: 1, Suffix: ""}, 0, false},      // C# not in C major
	}
	for _, tc := range cases {
		dc, ok := DegreeOf(cmaj, tc.chord)
		if ok != tc.ok || (ok && dc.Degree != tc.deg) {
			t.Errorf("DegreeOf(%v) = (%d,%v), want (%d,%v)", tc.chord, dc.Degree, ok, tc.deg, tc.ok)
		}
	}
}

func TestParseKey(t *testing.T) {
	cases := []struct {
		in    string
		tonic uint8
		mode  Mode
	}{
		{"C", 0, Major},
		{"Am", 9, Minor},
		{"F#m", 6, Minor},
		{"Bb", 10, Major},
		{"g minor", 7, Minor},
		{"D major", 2, Major},
	}
	for _, tc := range cases {
		k, err := ParseKey(tc.in)
		if err != nil || k.Tonic != tc.tonic || k.Mode != tc.mode {
			t.Errorf("ParseKey(%q) = (%d,%v,%v), want (%d,%v)", tc.in, k.Tonic, k.Mode, err, tc.tonic, tc.mode)
		}
	}
	if _, err := ParseKey("H"); err == nil {
		t.Error("ParseKey(H) should fail")
	}
}
