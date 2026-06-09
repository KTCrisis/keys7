package theory

import "sort"

// DefaultMelodyGap is the register gap (in semitones) above which an isolated
// top note is treated as melody rather than a chord tone. A perfect fourth:
// wider than the thirds inside a close voicing, so triads and close extensions
// are kept whole, but a melody note sitting well above the chord is peeled off.
const DefaultMelodyGap = 5

// SplitMelody separates isolated top melodic notes from the harmonic core by
// register gap. A note `gap` semitones or more above the rest is peeled as
// melody — but only while the remainder still forms a chord and keeps at least
// three notes. So a left-hand C with a right-hand D high above reads as "C"
// plus a D melody note, not "Cadd9"; a 9th nestled inside the voicing stays.
func SplitMelody(notes []uint8, gap uint8) (core, melody []uint8) {
	core = append([]uint8(nil), notes...)
	sort.Slice(core, func(i, j int) bool { return core[i] < core[j] })
	for len(core) >= 4 {
		n := len(core)
		if core[n-1]-core[n-2] < gap {
			break
		}
		if _, ok := Identify(core[:n-1]); !ok {
			break
		}
		melody = append([]uint8{core[n-1]}, melody...)
		core = core[:n-1]
	}
	return core, melody
}
