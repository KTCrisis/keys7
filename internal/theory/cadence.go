package theory

// Suggestion is a recommended next chord and the cadence it forms.
type Suggestion struct {
	Chord DegreeChord
	Label string
}

// Suggest proposes next chords from the current one, by functional harmony:
// dominant resolves to tonic (authentic) or deceptively to vi/VI; subdominant
// moves to the dominant (and IV→I is plagal, ii→V the predominant step); tonic
// opens toward subdominant or dominant. Returns ok=false if the current chord
// isn't diatonic to the key (nothing reliable to say).
func Suggest(k Key, current Chord) ([]Suggestion, bool) {
	dc, ok := DegreeOf(k, current)
	if !ok {
		return nil, false
	}
	tri := DiatonicTriads(k)
	deg := func(d int) DegreeChord { return tri[d-1] }

	var out []Suggestion
	switch dc.Function {
	case Dominant:
		out = append(out, Suggestion{deg(1), "authentic cadence"})
		if dc.Degree == 5 {
			out = append(out, Suggestion{deg(6), "deceptive cadence"})
		}
	case Subdominant:
		out = append(out, Suggestion{deg(5), "to the dominant"})
		switch dc.Degree {
		case 4:
			out = append(out, Suggestion{deg(1), "plagal cadence"})
		case 2:
			out = append(out, Suggestion{deg(1), "ii–V–I"}) // the ii–V resolves on to I
		}
	case Tonic:
		out = append(out,
			Suggestion{deg(4), "open: subdominant"},
			Suggestion{deg(5), "open: dominant"},
			Suggestion{deg(2), "open: predominant"},
		)
	}
	return dedupe(out), true
}

func dedupe(in []Suggestion) []Suggestion {
	seen := map[int]bool{}
	out := in[:0]
	for _, s := range in {
		if seen[s.Chord.Degree] {
			continue
		}
		seen[s.Chord.Degree] = true
		out = append(out, s)
	}
	return out
}
