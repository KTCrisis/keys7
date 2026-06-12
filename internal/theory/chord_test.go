package theory

import "testing"

func TestIdentify(t *testing.T) {
	cases := []struct {
		name  string
		notes []uint8
		want  string
	}{
		{"C major root", []uint8{60, 64, 67}, "C"},
		{"C major 1st inversion", []uint8{64, 67, 72}, "C/E"},
		{"C major 2nd inversion", []uint8{67, 72, 76}, "C/G"},
		{"A minor", []uint8{57, 60, 64}, "Am"},
		{"B diminished", []uint8{59, 62, 65}, "Bdim"},
		{"C augmented", []uint8{60, 64, 68}, "Caug"},
		{"C sus4", []uint8{60, 65, 67}, "Csus4"},
		{"D sus2", []uint8{62, 64, 69}, "Dsus2"},
		{"G dominant 7", []uint8{55, 59, 62, 65}, "G7"},
		{"C major 7", []uint8{60, 64, 67, 71}, "Cmaj7"},
		{"D minor 7", []uint8{62, 65, 69, 72}, "Dm7"},
		{"B half-diminished", []uint8{59, 62, 65, 69}, "Bm7b5"},
		{"C6 over C", []uint8{60, 64, 67, 69}, "C6"},
		{"Am7 over A", []uint8{57, 60, 64, 67}, "Am7"},
		{"voicing across octaves", []uint8{48, 64, 67, 84}, "C"}, // C2 E4 G4 C6
		// fifth-less voicings (common on piano)
		{"C7 no fifth", []uint8{60, 64, 70}, "C7"},       // C E Bb
		{"Cmaj7 no fifth", []uint8{60, 64, 71}, "Cmaj7"}, // C E B
		{"Dm7 no fifth", []uint8{62, 65, 72}, "Dm7"},     // D F C
		// extended chords
		{"C9", []uint8{60, 64, 67, 70, 74}, "C9"},       // C E G Bb D
		{"Cmaj9", []uint8{60, 64, 67, 71, 74}, "Cmaj9"}, // C E G B D
		{"Cm9", []uint8{60, 63, 67, 70, 74}, "Cm9"},     // C Eb G Bb D
		{"C13", []uint8{60, 64, 70, 74, 81}, "C13"},     // C E Bb D A (no 5)
		{"Cadd9", []uint8{60, 64, 67, 74}, "Cadd9"},     // C E G D, no 7th
		{"C6/9", []uint8{60, 64, 67, 69, 74}, "C6/9"},   // C E G A D
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, ok := Identify(tc.notes)
			if !ok {
				t.Fatalf("Identify(%v): not recognized, want %q", tc.notes, tc.want)
			}
			if got := c.String(); got != tc.want {
				t.Errorf("Identify(%v) = %q, want %q", tc.notes, got, tc.want)
			}
		})
	}
}

func TestIdentifyRejectsNonChords(t *testing.T) {
	for _, notes := range [][]uint8{nil, {60}, {60, 67}, {60, 61, 62}} {
		if _, ok := Identify(notes); ok {
			t.Errorf("Identify(%v): expected not recognized", notes)
		}
	}
}
