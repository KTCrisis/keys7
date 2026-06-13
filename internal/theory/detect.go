package theory

import "math"

// Krumhansl-Kessler tonal hierarchy profiles: the perceived stability of each
// scale degree in major and minor. Key-finding correlates a played pitch-class
// distribution against these (rotated to every tonic) and picks the best fit.
var (
	majorProfile = [12]float64{6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88}
	minorProfile = [12]float64{6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17}
)

// DetectKey infers the most likely key from a list of played pitch classes
// (Krumhansl-Schmuckler). It returns the best key, a confidence in [0,1] (the
// best profile correlation), and ok=false if there are too few notes to judge.
// Minor matches are reported as natural minor (the relative-minor reading);
// cycle to harmonic/melodic by ear if wanted.
func DetectKey(pcs []uint8) (Key, float64, bool) {
	if len(pcs) < 3 {
		return Key{}, 0, false
	}
	var hist [12]float64
	for _, pc := range pcs {
		hist[pc%12]++
	}

	best := Key{}
	bestR := -2.0
	for tonic := uint8(0); tonic < 12; tonic++ {
		for _, mc := range []struct {
			mode    Mode
			profile [12]float64
		}{{Major, majorProfile}, {NaturalMinor, minorProfile}} {
			var rotated [12]float64
			for pc := 0; pc < 12; pc++ {
				rotated[pc] = mc.profile[(pc-int(tonic)+12)%12]
			}
			if r := pearson(hist[:], rotated[:]); r > bestR {
				bestR, best = r, Key{Tonic: tonic, Mode: mc.mode}
			}
		}
	}
	if bestR < 0 {
		bestR = 0
	}
	return best, bestR, true
}

// ModeOverTonic names the mode for a FIXED tonic by reading the characteristic
// tones sounded above it — the right model for drone/pedal playing, where the
// bass is the centre and the colour over it (not a relative key) names the
// mode. The third splits major (ionian/lydian/mixolydian) from minor
// (dorian/phrygian/aeolian/locrian/harmonic/melodic); the distinguishing degree
// then picks within each family, falling back to ionian/aeolian when nothing
// distinctive is sounded.
func ModeOverTonic(pcs []uint8, tonic uint8) Mode {
	has := func(semi uint8) bool {
		want := (tonic + semi) % 12
		for _, pc := range pcs {
			if pc%12 == want {
				return true
			}
		}
		return false
	}

	if has(4) { // major third
		switch {
		case has(6) && !has(5): // raised 4th, no natural 4th
			return Lydian
		case has(10): // flat 7th
			return Mixolydian
		default:
			return Major // ionian
		}
	}
	// minor third (or no third sounded — defaults read as aeolian)
	switch {
	case has(1): // flat 2nd
		return Phrygian
	case has(6) && !has(7): // diminished 5th, no perfect 5th
		return Locrian
	case has(11): // raised 7th: harmonic (flat 6th) or melodic (natural 6th)
		if has(9) {
			return MelodicMinor
		}
		return HarmonicMinor
	case has(9): // natural 6th over a minor third
		return Dorian
	default:
		return NaturalMinor // aeolian
	}
}

func pearson(a, b []float64) float64 {
	n := float64(len(a))
	var sa, sb float64
	for i := range a {
		sa += a[i]
		sb += b[i]
	}
	ma, mb := sa/n, sb/n
	var num, da, db float64
	for i := range a {
		x, y := a[i]-ma, b[i]-mb
		num += x * y
		da += x * x
		db += y * y
	}
	if da == 0 || db == 0 {
		return 0
	}
	return num / math.Sqrt(da*db)
}
