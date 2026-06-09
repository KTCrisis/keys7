package theory

import "sort"

var intervalNames = [12]string{
	"unison", "m2", "M2", "m3", "M3", "P4", "tritone", "P5", "m6", "M6", "m7", "M7",
}

// IntervalName names an interval by its size in semitones (interval class).
func IntervalName(semitones uint8) string { return intervalNames[semitones%12] }

// chordShapes are the qualities enumerated when finding chords that contain a
// dyad. Triads and common sevenths/sixths — enough for harmonic exploration.
var chordShapes = []struct {
	suffix string
	ivs    []uint8
}{
	{"", []uint8{0, 4, 7}},
	{"m", []uint8{0, 3, 7}},
	{"dim", []uint8{0, 3, 6}},
	{"aug", []uint8{0, 4, 8}},
	{"7", []uint8{0, 4, 7, 10}},
	{"maj7", []uint8{0, 4, 7, 11}},
	{"m7", []uint8{0, 3, 7, 10}},
	{"m7b5", []uint8{0, 3, 6, 10}},
	{"dim7", []uint8{0, 3, 6, 9}},
	{"6", []uint8{0, 4, 7, 9}},
	{"m6", []uint8{0, 3, 7, 9}},
}

// DyadImplications lists chords that contain both pitch classes — the harmonies
// the two notes could imply, to be completed by a third (often a melody note).
// With a key, only diatonic chords are returned, sorted by scale degree: the
// relevant completions for the key you're working in. Without a key, every
// match is returned.
func DyadImplications(a, b uint8, key *Key) []Chord {
	a, b = a%12, b%12
	if a == b {
		return nil
	}
	var out []Chord
	for root := uint8(0); root < 12; root++ {
		for _, sh := range chordShapes {
			pcs := map[uint8]bool{}
			for _, iv := range sh.ivs {
				pcs[(root+iv)%12] = true
			}
			if pcs[a] && pcs[b] {
				out = append(out, Chord{Root: root, Suffix: sh.suffix, Bass: root})
			}
		}
	}
	if key == nil {
		return out
	}
	var dia []Chord
	for _, c := range out {
		if _, ok := DegreeOf(*key, c); ok {
			dia = append(dia, c)
		}
	}
	sort.SliceStable(dia, func(i, j int) bool {
		di, _ := DegreeOf(*key, dia[i])
		dj, _ := DegreeOf(*key, dia[j])
		return di.Degree < dj.Degree
	})
	return dia
}
