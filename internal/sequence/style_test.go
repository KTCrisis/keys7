package sequence

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func parseSeq(t *testing.T, js string) Sequence {
	t.Helper()
	seq, err := Parse(strings.NewReader(js))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return seq
}

// A step's "pedal" field emits a sustain control change at the step onset,
// ordered before the chord's note-ons so the pedal catches them.
func TestPedalField(t *testing.T) {
	seq := parseSeq(t, `{"tempo":60,"steps":[{"notes":["C4"],"beats":1,"pedal":"down"},{"beats":1,"pedal":"up"}]}`)

	var ctrls []Event
	for _, e := range seq.Events {
		if e.Ctrl {
			ctrls = append(ctrls, e)
		}
	}
	if len(ctrls) != 2 {
		t.Fatalf("got %d control events, want 2", len(ctrls))
	}
	if ctrls[0].Note != 64 || ctrls[0].Vel != 127 || ctrls[0].At != 0 {
		t.Errorf("pedal down = %+v, want CC64=127 at 0", ctrls[0])
	}
	if ctrls[1].Note != 64 || ctrls[1].Vel != 0 || ctrls[1].At != time.Second {
		t.Errorf("pedal up = %+v, want CC64=0 at 1s", ctrls[1])
	}
	// at t=0 the pedal-down must precede the note-on
	if seq.Events[0].At != 0 || !seq.Events[0].Ctrl {
		t.Errorf("first event = %+v, want the pedal-down control", seq.Events[0])
	}
}

func TestPedalBadValue(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"steps":[{"notes":["C4"],"pedal":"sideways"}]}`)); err == nil {
		t.Error("expected an error for a bad pedal value")
	}
}

func countOns(evs []Event) int {
	n := 0
	for _, e := range evs {
		if !e.Ctrl && e.On {
			n++
		}
	}
	return n
}

func TestStyleStraightIsIdentity(t *testing.T) {
	seq := parseSeq(t, `{"tempo":90,"steps":[{"notes":["C4","E4","G4"],"beats":2},{"notes":["F4"],"beats":1}]}`)
	st, _ := StyleByName("straight")
	got := st.Apply(seq, rand.New(rand.NewSource(1)))
	if !reflect.DeepEqual(got, seq) {
		t.Error("straight style changed the sequence")
	}
}

func TestStyleAmbient(t *testing.T) {
	seq := parseSeq(t, `{"tempo":90,"steps":[{"notes":["C4","E4","G4"],"beats":2},{"notes":["F4","A4","C5"],"beats":2}]}`)
	st, _ := StyleByName("ambient")

	out := st.Apply(seq, rand.New(rand.NewSource(42)))

	if got, want := countOns(out.Events), countOns(seq.Events); got != want {
		t.Errorf("note-on count = %d, want %d (style must not drop notes)", got, want)
	}
	var downs int
	for _, e := range out.Events {
		if e.Ctrl && e.Note == 64 && e.Vel == 127 {
			downs++
		}
		if !e.Ctrl && e.On && (e.Vel < 1 || e.Vel > 127) {
			t.Errorf("velocity out of range: %d", e.Vel)
		}
	}
	if downs < 2 {
		t.Errorf("auto-pedal: %d chord sustains, want >= 2 (one per chord)", downs)
	}

	// Determinism: same seed, same take.
	again := st.Apply(seq, rand.New(rand.NewSource(42)))
	if !reflect.DeepEqual(out, again) {
		t.Error("same seed produced a different take")
	}
}

// Explicit pedal events disable auto-pedal: playing wins over style.
func TestStyleExplicitPedalWins(t *testing.T) {
	seq := parseSeq(t, `{"tempo":90,"steps":[{"notes":["C4","E4","G4"],"beats":2,"pedal":"down"},{"notes":["F4"],"beats":2}]}`)
	st, _ := StyleByName("ambient")
	out := st.Apply(seq, rand.New(rand.NewSource(1)))

	var ctrls int
	for _, e := range out.Events {
		if e.Ctrl {
			ctrls++
		}
	}
	if ctrls != 1 {
		t.Errorf("control events = %d, want 1 (the explicit pedal, no auto-pedal added)", ctrls)
	}
}
