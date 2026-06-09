// Package theory holds keys7's deterministic music theory: no MIDI, no AI, no
// I/O — just pitch math. Phase-1 tiers: identify the chord, then key/cadence.
package theory

import "sort"

var pcNames = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// PitchClassName renders a pitch class (0-11) as a note name (sharp spelling).
func PitchClassName(pc uint8) string { return pcNames[pc%12] }

// Chord is an identified chord: a root pitch class, a quality suffix, and the
// bass pitch class (which differs from the root for an inversion).
type Chord struct {
	Root     uint8  // pitch class 0-11
	Suffix   string // "", "m", "maj7", "9", "m7b5", ...
	Bass     uint8  // pitch class of the lowest sounding note
	Inverted bool   // bass != root
}

// String renders the chord symbol, e.g. "C", "Am7", "Cmaj9", or "C/E" for an
// inversion (slash chord: chord over its bass note).
func (c Chord) String() string {
	s := PitchClassName(c.Root) + c.Suffix
	if c.Inverted {
		s += "/" + PitchClassName(c.Bass)
	}
	return s
}

// Identify names the chord formed by a set of MIDI note numbers. It needs at
// least three distinct pitch classes; otherwise it returns ok=false.
//
// Roots are tried bass-first, so a chord in root position keeps its root and an
// inversion is reported as a slash chord. This also resolves enharmonic ties by
// the bass (C-E-G-A is C6 over C, Am7 over A). The fifth is optional — common
// piano voicings drop it — and upper tensions (9/11/13) are named when present.
// It assumes the root (or bass) is sounding; fully rootless voicings aren't
// resolved. Works on the notes passed (the held keys); sustain-pedal
// accumulation is a later segmentation refinement.
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
		ivs := map[uint8]bool{}
		for pc := range seen {
			ivs[(pc-root+12)%12] = true
		}
		if suffix, ok := analyze(ivs); ok {
			return Chord{Root: root, Suffix: suffix, Bass: bass, Inverted: root != bass}, true
		}
	}
	return Chord{}, false
}

// analyze decodes a set of intervals above a candidate root (which includes 0)
// into a chord-symbol suffix. It returns ok=false when there is no recognizable
// triad core OR when any sounding interval isn't a chord tone of the candidate
// — that rejection is what lets bass-first try the next root, so an inversion
// like E-G-C reads as C/E rather than a bogus rootless "Em". The fifth and
// upper tensions are optional; every interval present, though, must be explained.
func analyze(ivs map[uint8]bool) (string, bool) {
	h := func(i uint8) bool { return ivs[i] }
	maj3, min3 := h(4), h(3)
	p5, d5, a5 := h(7), h(6), h(8)
	maj7, dom7 := h(11), h(10)
	sixth := h(9) // the 6th / 13th interval (also the dim7's bb7)
	ninth := h(2)
	eleventh := h(5)

	allowed := map[uint8]bool{0: true}
	// add marks intervals as chord tones; it returns a bool only so it can sit
	// in a tuple assignment alongside the suffix it accompanies.
	add := func(is ...uint8) bool {
		for _, i := range is {
			allowed[i] = true
		}
		return true
	}

	// upperExtensions names the highest tension on a chord that has a seventh,
	// and marks the intervals it consumes as allowed.
	upperExtensions := func(base, majPrefix string) string {
		num := base
		switch {
		case sixth:
			num, _ = "13", add(9)
		case eleventh:
			num, _ = "11", add(5)
		case ninth:
			num, _ = "9", add(2)
		}
		// lower tensions still count as chord tones even when a higher one names it
		if sixth {
			add(2, 5, 9)
		} else if eleventh {
			add(2, 5)
		}
		if majPrefix != "" {
			if num == "7" {
				return "maj7"
			}
			return "maj" + num
		}
		return num
	}
	// alterations names altered tensions and marks them allowed.
	alterations := func() string {
		var a string
		if h(1) {
			a, _ = a+"b9", add(1)
		}
		if maj3 && h(3) { // a minor-third interval over a major third is a #9
			a, _ = a+"#9", add(3)
		}
		if p5 && h(6) {
			a, _ = a+"#11", add(6)
		}
		if p5 && h(8) {
			a, _ = a+"b13", add(8)
		}
		return a
	}
	fifthAlt := func() string {
		switch {
		case p5:
			add(7)
			return ""
		case d5:
			add(6)
			return "b5"
		case a5:
			add(8)
			return "#5"
		}
		return ""
	}

	var suffix string
	switch {
	case maj3 && a5 && !p5: // augmented family
		add(4, 8)
		switch {
		case maj7:
			suffix, _ = "augmaj7", add(11)
		case dom7:
			suffix, _ = "aug7", add(10)
		default:
			suffix = "aug"
		}

	case maj3: // major family
		add(4)
		switch {
		case maj7:
			add(11)
			suffix = upperExtensions("7", "maj") + fifthAlt() + alterations()
		case dom7:
			add(10)
			suffix = upperExtensions("7", "") + fifthAlt() + alterations()
		case sixth && ninth:
			suffix, _ = "6/9", add(7, 9, 2)
		case sixth:
			suffix, _ = "6", add(7, 9)
		case ninth:
			suffix, _ = "add9", add(7, 2)
		default:
			suffix, _ = "", add(7)
		}

	case min3 && d5 && !p5: // diminished family
		add(3, 6)
		switch {
		case dom7:
			suffix, _ = "m7b5", add(10) // half-diminished
		case sixth && !maj7:
			suffix, _ = "dim7", add(9) // bb7
		case maj7:
			suffix, _ = "dimMaj7", add(11)
		default:
			suffix = "dim"
		}

	case min3: // minor family
		add(3)
		switch {
		case maj7:
			suffix, _ = "mMaj7", add(7, 11)
		case dom7:
			add(10)
			suffix = "m" + upperExtensions("7", "") + fifthAlt()
		case sixth && ninth:
			suffix, _ = "m6/9", add(7, 9, 2)
		case sixth:
			suffix, _ = "m6", add(7, 9)
		case ninth:
			suffix, _ = "madd9", add(7, 2)
		default:
			suffix, _ = "m", add(7)
		}

	case ninth && p5: // suspended 2nd (needs the fifth to be a real sus)
		suffix, _ = "sus2", add(2, 7)
	case eleventh && p5: // suspended 4th
		suffix, _ = "sus4", add(5, 7)

	default:
		return "", false
	}

	// Reject if any sounding interval isn't a chord tone of this candidate.
	for iv := range ivs {
		if !allowed[iv] {
			return "", false
		}
	}
	return suffix, true
}
