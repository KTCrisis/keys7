package theory

import "testing"

func TestScalePCs(t *testing.T) {
	cases := []struct {
		key  Key
		want [7]uint8
	}{
		{Key{Tonic: 0, Mode: Major}, [7]uint8{0, 2, 4, 5, 7, 9, 11}},        // C major
		{Key{Tonic: 9, Mode: NaturalMinor}, [7]uint8{9, 11, 0, 2, 4, 5, 7}}, // A natural minor
		{Key{Tonic: 9, Mode: HarmonicMinor}, [7]uint8{9, 11, 0, 2, 4, 5, 8}}, // A harmonic minor (raised 7th: G#)
	}
	for _, c := range cases {
		got := c.key.ScalePCs()
		if len(got) != 7 {
			t.Fatalf("%v: got %d notes, want 7", c.key, len(got))
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("%v: degree %d = %d, want %d", c.key, i+1, got[i], c.want[i])
			}
		}
	}
}

func TestInScale(t *testing.T) {
	c := Key{Tonic: 0, Mode: Major} // C major: no sharps/flats
	in := []uint8{60, 62, 64, 65, 67, 69, 71}                            // C D E F G A B (any octave)
	out := []uint8{61, 63, 66, 68, 70}                                   // C# D# F# G# A#
	for _, n := range in {
		if !c.InScale(n) {
			t.Errorf("InScale(%d) = false, want true (in C major)", n)
		}
	}
	for _, n := range out {
		if c.InScale(n) {
			t.Errorf("InScale(%d) = true, want false (accidental in C major)", n)
		}
	}
}
