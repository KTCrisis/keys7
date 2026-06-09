package theory

import (
	"fmt"
	"strings"
)

// Mode is a key's quality. The three minor flavours differ by their 6th/7th
// degrees, which changes the diatonic chords: natural minor matches the
// relative-minor (white-key) palette, harmonic/melodic raise the 7th to give a
// major V and a vii° leading tone for cadences.
type Mode int

const (
	Major Mode = iota
	NaturalMinor
	HarmonicMinor
	MelodicMinor
)

func (m Mode) String() string {
	switch m {
	case NaturalMinor:
		return "natural minor"
	case HarmonicMinor:
		return "harmonic minor"
	case MelodicMinor:
		return "melodic minor"
	default:
		return "major"
	}
}

// IsMinor reports whether the mode is any minor flavour.
func (m Mode) IsMinor() bool { return m != Major }

// minorModes is the cycle order used by the UI's mode toggle.
var minorModes = []Mode{Major, NaturalMinor, HarmonicMinor, MelodicMinor}

// Next returns the next mode in the cycle major → natural → harmonic → melodic.
func (m Mode) Next() Mode {
	for i, mm := range minorModes {
		if mm == m {
			return minorModes[(i+1)%len(minorModes)]
		}
	}
	return Major
}

// Key is a tonic pitch class plus a mode.
type Key struct {
	Tonic uint8 // pitch class 0-11
	Mode  Mode
}

func (k Key) String() string { return PitchClassName(k.Tonic) + " " + k.Mode.String() }

var scaleSteps = map[Mode][7]uint8{
	Major:         {0, 2, 4, 5, 7, 9, 11},
	NaturalMinor:  {0, 2, 3, 5, 7, 8, 10},
	HarmonicMinor: {0, 2, 3, 5, 7, 8, 11}, // raised 7th
	MelodicMinor:  {0, 2, 3, 5, 7, 9, 11}, // raised 6th and 7th (ascending)
}

func (k Key) scalePCs() [7]uint8 {
	steps := scaleSteps[k.Mode]
	var pcs [7]uint8
	for i, s := range steps {
		pcs[i] = (k.Tonic + s) % 12
	}
	return pcs
}

// Function is a chord's harmonic role (Riemann-simplified).
type Function int

const (
	Tonic Function = iota
	Subdominant
	Dominant
)

func (f Function) String() string {
	switch f {
	case Subdominant:
		return "subdominant"
	case Dominant:
		return "dominant"
	default:
		return "tonic"
	}
}

var functions = map[Mode][7]Function{
	//                I/i    ii     iii    IV     V      vi     vii
	Major:         {Tonic, Subdominant, Tonic, Subdominant, Dominant, Tonic, Dominant},
	NaturalMinor:  {Tonic, Subdominant, Tonic, Subdominant, Dominant, Subdominant, Dominant},
	HarmonicMinor: {Tonic, Subdominant, Tonic, Subdominant, Dominant, Subdominant, Dominant},
	MelodicMinor:  {Tonic, Subdominant, Tonic, Subdominant, Dominant, Subdominant, Dominant},
}

// DegreeChord is one diatonic chord: its scale degree, roman numeral, the chord
// itself, and its harmonic function.
type DegreeChord struct {
	Degree   int // 1-7
	Roman    string
	Chord    Chord
	Function Function
}

var numerals = [7]string{"I", "II", "III", "IV", "V", "VI", "VII"}

// DiatonicTriads returns the seven diatonic triads of the key, in degree order.
func DiatonicTriads(k Key) []DegreeChord {
	pcs := k.scalePCs()
	out := make([]DegreeChord, 7)
	for i := 0; i < 7; i++ {
		root, third, fifth := pcs[i], pcs[(i+2)%7], pcs[(i+4)%7]
		suffix := triadQuality(root, third, fifth)
		out[i] = DegreeChord{
			Degree:   i + 1,
			Roman:    roman(i+1, suffix),
			Chord:    Chord{Root: root, Suffix: suffix, Bass: root},
			Function: functions[k.Mode][i],
		}
	}
	return out
}

func triadQuality(root, third, fifth uint8) string {
	t := (third - root + 12) % 12
	f := (fifth - root + 12) % 12
	switch {
	case t == 4 && f == 7:
		return ""
	case t == 3 && f == 7:
		return "m"
	case t == 3 && f == 6:
		return "dim"
	case t == 4 && f == 8:
		return "aug"
	}
	return ""
}

func roman(deg int, suffix string) string {
	n := numerals[deg-1]
	switch suffix {
	case "m":
		return strings.ToLower(n)
	case "dim":
		return strings.ToLower(n) + "°"
	case "aug":
		return n + "+"
	}
	return n
}

// DegreeOf finds which diatonic degree a chord occupies, matched by root and
// triad quality. Sevenths and sixths match their underlying triad (so G7 is the
// V of C); suspended chords have no third and don't match a degree.
func DegreeOf(k Key, c Chord) (DegreeChord, bool) {
	base := triadBase(c.Suffix)
	if base == "" {
		return DegreeChord{}, false
	}
	for _, dc := range DiatonicTriads(k) {
		if dc.Chord.Root == c.Root && triadBase(dc.Chord.Suffix) == base {
			return dc, true
		}
	}
	return DegreeChord{}, false
}

// triadBase reduces a chord suffix to its underlying triad quality, or "" if it
// has no plain triad (suspended chords). Prefix-based so extended chords map to
// their core: maj9→maj, m11→min, G13→maj, etc. Order matters (check "maj" and
// "m7b5" before the bare "m").
func triadBase(suffix string) string {
	switch {
	case strings.HasPrefix(suffix, "maj"):
		return "maj"
	case strings.HasPrefix(suffix, "m7b5"):
		return "dim"
	case strings.HasPrefix(suffix, "mMaj"):
		return "min"
	case strings.HasPrefix(suffix, "dim"):
		return "dim"
	case strings.HasPrefix(suffix, "aug"):
		return "aug"
	case strings.HasPrefix(suffix, "m"):
		return "min"
	case strings.HasPrefix(suffix, "sus"):
		return ""
	default:
		return "maj" // "", 6, 7, 9, 11, 13, add9, 6/9
	}
}

var letterPC = map[byte]uint8{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// ParseKey reads a key like "C", "Am", "F#m", "Bb", "A minor", "G major".
func ParseKey(s string) (Key, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Key{}, fmt.Errorf("empty key")
	}
	pc, ok := letterPC[s[0]&^0x20] // uppercase the letter
	if !ok {
		return Key{}, fmt.Errorf("bad key %q: expected a note A-G", s)
	}
	i := 1
	if i < len(s) {
		switch s[i] {
		case '#':
			pc, i = (pc+1)%12, i+1
		case 'b':
			pc, i = (pc+11)%12, i+1
		}
	}
	switch strings.ToLower(strings.TrimSpace(s[i:])) {
	case "", "maj", "major":
		return Key{Tonic: pc, Mode: Major}, nil
	case "m", "min", "minor":
		return Key{Tonic: pc, Mode: NaturalMinor}, nil
	default:
		return Key{}, fmt.Errorf("bad key %q: quality should be major/minor", s)
	}
}
