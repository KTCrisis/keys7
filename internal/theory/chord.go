// Package theory holds keys7's deterministic music theory: no MIDI, no AI, no
// I/O — just pitch math. Phase-1 tier: identify the chord from a set of notes.
package theory

import "sort"

var pcNames = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// PitchClassName renders a pitch class (0-11) as a note name (sharp spelling).
func PitchClassName(pc uint8) string { return pcNames[pc%12] }

// Chord is an identified chord: a root pitch class, a quality suffix, and the
// bass pitch class (which differs from the root for an inversion).
type Chord struct {
	Root     uint8  // pitch class 0-11
	Suffix   string // "", "m", "maj7", ...
	Bass     uint8  // pitch class of the lowest sounding note
	Inverted bool   // bass != root
}

// String renders the chord symbol, e.g. "C", "Am7", "Cmaj7", or "C/E" for an
// inversion (slash chord: chord over its bass note).
func (c Chord) String() string {
	s := PitchClassName(c.Root) + c.Suffix
	if c.Inverted {
		s += "/" + PitchClassName(c.Bass)
	}
	return s
}

// templates maps an interval-set (relative to the root, as a 12-bit mask) to a
// chord-symbol suffix. Order of declaration is irrelevant — matching is exact.
var templates = map[uint16]string{
	bits(0, 4, 7):      "",      // major triad
	bits(0, 3, 7):      "m",     // minor triad
	bits(0, 3, 6):      "dim",   // diminished triad
	bits(0, 4, 8):      "aug",   // augmented triad
	bits(0, 2, 7):      "sus2",  // suspended 2nd
	bits(0, 5, 7):      "sus4",  // suspended 4th
	bits(0, 4, 7, 10):  "7",     // dominant 7th
	bits(0, 4, 7, 11):  "maj7",  // major 7th
	bits(0, 3, 7, 10):  "m7",    // minor 7th
	bits(0, 3, 6, 10):  "m7b5",  // half-diminished
	bits(0, 3, 6, 9):   "dim7",  // fully diminished 7th
	bits(0, 4, 7, 9):   "6",     // major 6th
	bits(0, 3, 7, 9):   "m6",    // minor 6th
}

func bits(intervals ...uint8) uint16 {
	var b uint16
	for _, i := range intervals {
		b |= 1 << i
	}
	return b
}

// Identify names the chord formed by a set of MIDI note numbers. It needs at
// least three distinct pitch classes; otherwise it returns ok=false (a single
// note or a dyad isn't a chord at this tier).
//
// Roots are tried bass-first, so a chord in root position keeps its root and an
// inversion is reported as a slash chord. This also resolves enharmonic ties by
// the bass: C-E-G-A reads as C6 over C but as Am7 over A. Symmetric chords
// (dim7, aug) likewise default to the bass as root.
//
// Note: this works on the notes you pass — currently the physically held keys.
// Sustain-pedal accumulation (notes still sounding after release) is a later
// refinement of the segmentation layer, not handled here.
func Identify(notes []uint8) (Chord, bool) {
	if len(notes) < 3 {
		return Chord{}, false
	}

	sorted := append([]uint8(nil), notes...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	bass := sorted[0] % 12

	seen := map[uint8]bool{}
	for _, n := range sorted {
		seen[n%12] = true
	}
	if len(seen) < 3 {
		return Chord{}, false
	}

	// Candidate roots: the bass first, then the remaining pitch classes ascending.
	roots := []uint8{bass}
	for pc := uint8(0); pc < 12; pc++ {
		if pc != bass && seen[pc] {
			roots = append(roots, pc)
		}
	}

	for _, root := range roots {
		var mask uint16
		for pc := range seen {
			mask |= 1 << ((pc - root + 12) % 12)
		}
		if suffix, ok := templates[mask]; ok {
			return Chord{Root: root, Suffix: suffix, Bass: bass, Inverted: root != bass}, true
		}
	}
	return Chord{}, false
}
