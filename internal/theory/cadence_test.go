package theory

import (
	"strings"
	"testing"
)

// suggestionString renders suggestions as "Symbol:label" for easy assertions.
func suggestionString(ss []Suggestion) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = s.Chord.Chord.String() + ":" + s.Label
	}
	return strings.Join(parts, " | ")
}

func TestSuggest(t *testing.T) {
	cmaj := Key{Tonic: 0, Mode: Major}

	got := suggestionString(must(t, cmaj, Chord{Root: 7, Suffix: ""})) // on G (V)
	if !strings.Contains(got, "C:authentic cadence") || !strings.Contains(got, "Am:deceptive cadence") {
		t.Errorf("V suggestions = %q", got)
	}

	got = suggestionString(must(t, cmaj, Chord{Root: 5, Suffix: ""})) // on F (IV)
	if !strings.Contains(got, "G:to the dominant") || !strings.Contains(got, "C:plagal cadence") {
		t.Errorf("IV suggestions = %q", got)
	}

	got = suggestionString(must(t, cmaj, Chord{Root: 2, Suffix: "m"})) // on Dm (ii)
	if !strings.Contains(got, "G:to the dominant") || !strings.Contains(got, "C:ii–V–I") {
		t.Errorf("ii suggestions = %q", got)
	}

	got = suggestionString(must(t, cmaj, Chord{Root: 0, Suffix: ""})) // on C (I)
	if !strings.Contains(got, "F:") || !strings.Contains(got, "G:") || !strings.Contains(got, "Dm:") {
		t.Errorf("I suggestions = %q", got)
	}
}

func TestSuggestNonDiatonic(t *testing.T) {
	cmaj := Key{Tonic: 0, Mode: Major}
	if _, ok := Suggest(cmaj, Chord{Root: 1, Suffix: ""}); ok { // C# not in key
		t.Error("expected no suggestions for non-diatonic chord")
	}
}

func must(t *testing.T, k Key, c Chord) []Suggestion {
	t.Helper()
	ss, ok := Suggest(k, c)
	if !ok {
		t.Fatalf("Suggest(%v) returned ok=false", c)
	}
	return ss
}
