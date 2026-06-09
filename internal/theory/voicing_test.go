package theory

import (
	"reflect"
	"testing"
)

func TestSplitMelody(t *testing.T) {
	cases := []struct {
		name       string
		notes      []uint8
		wantCore   []uint8
		wantMelody []uint8
	}{
		{
			name:       "high melody note peeled off the triad",
			notes:      []uint8{60, 64, 67, 74}, // C E G + D an octave+ above
			wantCore:   []uint8{60, 64, 67},
			wantMelody: []uint8{74},
		},
		{
			name:       "close add9 kept whole",
			notes:      []uint8{60, 62, 64, 67}, // C D E G — D nestled inside
			wantCore:   []uint8{60, 62, 64, 67},
			wantMelody: nil,
		},
		{
			name:       "plain triad untouched",
			notes:      []uint8{60, 64, 67},
			wantCore:   []uint8{60, 64, 67},
			wantMelody: nil,
		},
		{
			name:       "two melody notes peeled, triad kept",
			notes:      []uint8{60, 64, 67, 74, 81}, // C E G + D + A high
			wantCore:   []uint8{60, 64, 67},
			wantMelody: []uint8{74, 81},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			core, melody := SplitMelody(tc.notes, DefaultMelodyGap)
			if !reflect.DeepEqual(core, tc.wantCore) || !reflect.DeepEqual(melody, tc.wantMelody) {
				t.Errorf("SplitMelody(%v) = core %v, melody %v; want core %v, melody %v",
					tc.notes, core, melody, tc.wantCore, tc.wantMelody)
			}
		})
	}
}
