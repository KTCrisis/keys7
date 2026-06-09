package theory

import "testing"

func TestNeighborKeys(t *testing.T) {
	got := map[string]string{}
	for _, nk := range NeighborKeys(Key{Tonic: 0, Mode: Major}) { // C major
		got[nk.Relation] = nk.Key.String()
	}
	want := map[string]string{
		"dominant":    "G major",
		"subdominant": "F major",
		"relative":    "A natural minor",
		"parallel":    "C natural minor",
	}
	for rel, w := range want {
		if got[rel] != w {
			t.Errorf("%s = %q, want %q", rel, got[rel], w)
		}
	}
}

func TestSecondaryDominants(t *testing.T) {
	got := map[string]string{}
	for _, sd := range SecondaryDominants(Key{Tonic: 0, Mode: Major}) { // C major
		got[sd.Label] = sd.Chord.String()
	}
	want := map[string]string{
		"V/ii":  "A7", // -> Dm
		"V/iii": "B7", // -> Em
		"V/IV":  "C7", // -> F
		"V/V":   "D7", // -> G
		"V/vi":  "E7", // -> Am
	}
	for label, w := range want {
		if got[label] != w {
			t.Errorf("%s = %q, want %q", label, got[label], w)
		}
	}
}
