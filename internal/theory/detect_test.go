package theory

import "testing"

// rep repeats each pitch class n times, to weight a tonic/dominant emphasis.
func rep(n int, pcs ...uint8) []uint8 {
	var out []uint8
	for i := 0; i < n; i++ {
		out = append(out, pcs...)
	}
	return out
}

func TestDetectKey(t *testing.T) {
	// C major: the scale plus emphasis on tonic/dominant (C, G, E).
	cMajor := append(rep(3, 0, 7, 4), 0, 2, 4, 5, 7, 9, 11)
	if k, conf, ok := DetectKey(cMajor); !ok || k.Tonic != 0 || k.Mode != Major {
		t.Errorf("C major: got %s (conf %.2f, ok %v)", k, conf, ok)
	}

	// A natural minor: same white keys but emphasis on A, E, C (A minor triad).
	aMinor := append(rep(3, 9, 4, 0), 9, 11, 0, 2, 4, 5, 7)
	if k, _, ok := DetectKey(aMinor); !ok || k.Tonic != 9 || k.Mode != NaturalMinor {
		t.Errorf("A minor: got %s (ok %v)", k, ok)
	}

	// G major: sharped 4th (F#) and tonic emphasis.
	gMajor := append(rep(3, 7, 2, 11), 7, 9, 11, 0, 2, 4, 6)
	if k, _, ok := DetectKey(gMajor); !ok || k.Tonic != 7 || k.Mode != Major {
		t.Errorf("G major: got %s (ok %v)", k, ok)
	}
}

func TestModeOverTonic(t *testing.T) {
	// over a C drone, an E (maj 3rd) reads major; an Eb (min 3rd) reads minor.
	if m := ModeOverTonic([]uint8{0, 4, 7, 4}, 0); m != Major {
		t.Errorf("C drone with E = %v, want Major", m)
	}
	if m := ModeOverTonic([]uint8{0, 3, 7, 3}, 0); m != NaturalMinor {
		t.Errorf("C drone with Eb = %v, want NaturalMinor", m)
	}
}

func TestDetectKeyTooFew(t *testing.T) {
	if _, _, ok := DetectKey([]uint8{0, 4}); ok {
		t.Error("expected ok=false for fewer than 3 notes")
	}
}
