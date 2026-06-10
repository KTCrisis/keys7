package sequence

import (
	"strings"
	"testing"
	"time"

	"keys7/internal/midi"
)

func parse(t *testing.T, src string) Sequence {
	t.Helper()
	seq, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	return seq
}

func TestParseDefaults(t *testing.T) {
	seq := parse(t, `{"steps":[{"notes":["C4","E4","G4"]}]}`)
	if seq.Tempo != 90 {
		t.Errorf("default tempo = %v, want 90", seq.Tempo)
	}
	if seq.Channel != 0 {
		t.Errorf("default channel = %d, want 0 (wire for 1)", seq.Channel)
	}
	st := seq.Steps[0]
	if st.Beats != 1 || st.Velocity != 80 {
		t.Errorf("step defaults = %v beats, vel %d; want 1 beat, vel 80", st.Beats, st.Velocity)
	}
	if len(st.Notes) != 3 || st.Notes[0] != 60 || st.Notes[1] != 64 || st.Notes[2] != 67 {
		t.Errorf("notes = %v, want [60 64 67]", st.Notes)
	}
}

func TestParseOverrides(t *testing.T) {
	seq := parse(t, `{"tempo":120,"channel":3,"velocity":100,
		"steps":[{"notes":["A3"],"beats":0.5,"velocity":40},{"beats":2}]}`)
	if seq.Tempo != 120 || seq.Channel != 2 {
		t.Errorf("tempo/channel = %v/%d, want 120/2", seq.Tempo, seq.Channel)
	}
	if seq.Steps[0].Velocity != 40 {
		t.Errorf("step velocity = %d, want 40 (step overrides sequence)", seq.Steps[0].Velocity)
	}
	if len(seq.Steps[1].Notes) != 0 || seq.Steps[1].Velocity != 100 {
		t.Errorf("rest step = %+v, want no notes, inherited vel 100", seq.Steps[1])
	}
	if d := seq.Duration(seq.Steps[0]); d != 250*time.Millisecond {
		t.Errorf("0.5 beats at 120 BPM = %v, want 250ms", d)
	}
}

func TestParseRejects(t *testing.T) {
	bad := map[string]string{
		"no steps":      `{"steps":[]}`,
		"bad note":      `{"steps":[{"notes":["H4"]}]}`,
		"bad channel":   `{"channel":17,"steps":[{"notes":["C4"]}]}`,
		"bad velocity":  `{"velocity":200,"steps":[{"notes":["C4"]}]}`,
		"bad beats":     `{"steps":[{"notes":["C4"],"beats":-1}]}`,
		"unknown field": `{"bpm":90,"steps":[{"notes":["C4"]}]}`,
		"not json":      `tempo: 90`,
	}
	for name, src := range bad {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("%s: Parse accepted %q", name, src)
		}
	}
}

func TestPlaySchedule(t *testing.T) {
	seq := parse(t, `{"tempo":60,"channel":2,
		"steps":[{"notes":["C4","E4"],"beats":1},{"beats":2},{"notes":["G4"],"beats":1,"velocity":50}]}`)

	out := midi.NewMockOut(nil)
	var slept []time.Duration
	if err := Play(out, seq, func(d time.Duration) { slept = append(slept, d) }); err != nil {
		t.Fatalf("Play error: %v", err)
	}

	// One sleep per step, beats at 60 BPM = seconds.
	want := []time.Duration{time.Second, 2 * time.Second, time.Second}
	if len(slept) != len(want) {
		t.Fatalf("slept %d times, want %d", len(slept), len(want))
	}
	for i := range want {
		if slept[i] != want[i] {
			t.Errorf("sleep %d = %v, want %v", i, slept[i], want[i])
		}
	}

	// Legato order: ons, hold, offs; the rest sends nothing.
	wantMsgs := []midi.OutMsg{
		{Status: "on", Channel: 1, Data1: 60, Data2: 80},
		{Status: "on", Channel: 1, Data1: 64, Data2: 80},
		{Status: "off", Channel: 1, Data1: 60},
		{Status: "off", Channel: 1, Data1: 64},
		{Status: "on", Channel: 1, Data1: 67, Data2: 50},
		{Status: "off", Channel: 1, Data1: 67},
	}
	if len(out.Msgs) != len(wantMsgs) {
		t.Fatalf("got %d messages, want %d: %+v", len(out.Msgs), len(wantMsgs), out.Msgs)
	}
	for i := range wantMsgs {
		if out.Msgs[i] != wantMsgs[i] {
			t.Errorf("msg %d = %+v, want %+v", i, out.Msgs[i], wantMsgs[i])
		}
	}
}
