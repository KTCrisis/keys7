package midi

import (
	"testing"
	"time"
)

// TestMockSourceEmits verifies the synthetic source produces a coherent stream:
// at least one Note-On for middle C and one sustain-pedal event. This exercises
// the whole capture seam (NewSource -> channel -> Event) without a MIDI device,
// which is the point of the mock source on machines like WSL.
func TestMockSourceEmits(t *testing.T) {
	src, err := NewSource("mock", "")
	if err != nil {
		t.Fatalf("NewSource: %v", err)
	}
	t.Cleanup(func() { _ = src.Close() })

	var sawMiddleC, sawPedal bool
	deadline := time.After(5 * time.Second)

	for !(sawMiddleC && sawPedal) {
		select {
		case ev, ok := <-src.Events():
			if !ok {
				t.Fatal("event channel closed before expected events arrived")
			}
			if ev.Kind == NoteOn && ev.Data1 == 60 && ev.Data2 > 0 {
				sawMiddleC = true
			}
			if ev.IsPedal() {
				sawPedal = true
			}
		case <-deadline:
			t.Fatalf("timed out (middleC=%v pedal=%v)", sawMiddleC, sawPedal)
		}
	}
}

func TestNoteName(t *testing.T) {
	cases := map[uint8]string{60: "C4", 69: "A4", 61: "C#4", 48: "C3"}
	for n, want := range cases {
		if got := NoteName(n); got != want {
			t.Errorf("NoteName(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestUnknownSource(t *testing.T) {
	if _, err := NewSource("bogus", ""); err == nil {
		t.Fatal("expected error for unknown source kind")
	}
}
