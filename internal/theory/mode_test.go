package theory

import "testing"

func TestModeOverTonicChurchModes(t *testing.T) {
	const C = 0 // pin the tonic to C; pitch classes are absolute
	cases := []struct {
		name string
		pcs  []uint8
		want Mode
	}{
		{"ionian", []uint8{0, 4, 7}, Major},
		{"lydian", []uint8{0, 4, 6, 7}, Lydian},          // #4, no natural 4
		{"mixolydian", []uint8{0, 4, 7, 10}, Mixolydian}, // major 3rd, b7
		{"dorian", []uint8{0, 3, 7, 9}, Dorian},          // minor 3rd, natural 6
		{"phrygian", []uint8{0, 1, 3, 7}, Phrygian},      // b2
		{"locrian", []uint8{0, 3, 6}, Locrian},           // minor 3rd, b5, no P5
		{"aeolian", []uint8{0, 3, 7, 8, 10}, NaturalMinor},
		{"harmonic minor", []uint8{0, 3, 7, 8, 11}, HarmonicMinor}, // b6 + raised 7
		{"melodic minor", []uint8{0, 3, 7, 9, 11}, MelodicMinor},   // natural 6 + raised 7
	}
	for _, c := range cases {
		if got := ModeOverTonic(c.pcs, C); got != c.want {
			t.Errorf("%s: ModeOverTonic = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestDegreeName(t *testing.T) {
	cases := []struct {
		key    Key
		degree int
		want   string
	}{
		{Key{Tonic: 0, Mode: Major}, 1, "tonique"},
		{Key{Tonic: 0, Mode: Major}, 3, "médiante"},
		{Key{Tonic: 0, Mode: Major}, 6, "sus-dominante"},
		{Key{Tonic: 0, Mode: Major}, 7, "sensible"},        // B: a semitone below C
		{Key{Tonic: 9, Mode: NaturalMinor}, 7, "sous-tonique"}, // G: a whole tone below A
		{Key{Tonic: 9, Mode: HarmonicMinor}, 7, "sensible"},    // G#: raised, a semitone below
		{Key{Tonic: 2, Mode: Dorian}, 7, "sous-tonique"},       // C: a whole tone below D
	}
	for _, c := range cases {
		if got := DegreeName(c.key, c.degree); got != c.want {
			t.Errorf("%v degree %d = %q, want %q", c.key, c.degree, got, c.want)
		}
	}
}

// TestChurchModeDiatonics sanity-checks that the new modes plug into the rest of
// theory: a D dorian scale yields seven triads with the expected qualities
// (i ii bIII IV v vi° bVII).
func TestChurchModeDiatonics(t *testing.T) {
	k := Key{Tonic: 2, Mode: Dorian} // D dorian
	tr := DiatonicTriads(k)
	if len(tr) != 7 {
		t.Fatalf("got %d triads, want 7", len(tr))
	}
	wantRoman := []string{"i", "ii", "III", "IV", "v", "vi°", "VII"}
	for i, w := range wantRoman {
		if tr[i].Roman != w {
			t.Errorf("degree %d roman = %q, want %q", i+1, tr[i].Roman, w)
		}
	}
	if !k.Mode.IsMinor() {
		t.Error("dorian should report IsMinor")
	}
}
