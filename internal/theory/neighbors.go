package theory

// NeighborKey is a closely related key and how it relates to the current one.
type NeighborKey struct {
	Key      Key
	Relation string // "dominant", "subdominant", "relative", "parallel"
}

// NeighborKeys returns the closely related keys: the dominant and subdominant
// (a fifth away, one accidental apart), the relative (same notes), and the
// parallel (same tonic, opposite quality) — the usual modulation targets.
func NeighborKeys(k Key) []NeighborKey {
	var rel, par Key
	if k.Mode == Major {
		rel = Key{Tonic: (k.Tonic + 9) % 12, Mode: NaturalMinor}
		par = Key{Tonic: k.Tonic, Mode: NaturalMinor}
	} else {
		rel = Key{Tonic: (k.Tonic + 3) % 12, Mode: Major}
		par = Key{Tonic: k.Tonic, Mode: Major}
	}
	return []NeighborKey{
		{Key{Tonic: (k.Tonic + 7) % 12, Mode: k.Mode}, "dominant"},
		{Key{Tonic: (k.Tonic + 5) % 12, Mode: k.Mode}, "subdominant"},
		{rel, "relative"},
		{par, "parallel"},
	}
}

// SecondaryDominant is a chromatic chord that tonicizes a diatonic degree.
type SecondaryDominant struct {
	Label  string // "V/ii", "V/V", ...
	Chord  Chord  // the secondary dominant (a dominant 7th)
	Target DegreeChord
}

// SecondaryDominants returns the common secondary dominants of a key — the V7 of
// ii, iii, IV, V and vi — passing chords that pull chromatically toward each
// diatonic degree. Diminished targets are skipped (not conventionally tonicized).
func SecondaryDominants(k Key) []SecondaryDominant {
	tri := DiatonicTriads(k)
	var out []SecondaryDominant
	for _, deg := range []int{2, 3, 4, 5, 6} {
		target := tri[deg-1]
		if triadBase(target.Chord.Suffix) == "dim" {
			continue
		}
		domRoot := (target.Chord.Root + 7) % 12
		out = append(out, SecondaryDominant{
			Label:  "V/" + target.Roman,
			Chord:  Chord{Root: domRoot, Suffix: "7", Bass: domRoot},
			Target: target,
		})
	}
	return out
}
