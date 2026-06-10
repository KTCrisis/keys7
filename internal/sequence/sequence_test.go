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

func TestParseDefaultsSingleVoice(t *testing.T) {
	seq := parse(t, `{"steps":[{"notes":["C4","E4","G4"]}]}`)
	if seq.Tempo != 90 {
		t.Errorf("default tempo = %v, want 90", seq.Tempo)
	}
	if seq.Channel != 0 {
		t.Errorf("default channel = %d, want 0 (wire for 1)", seq.Channel)
	}
	if len(seq.Events) != 6 { // 3 ons + 3 offs
		t.Fatalf("got %d events, want 6: %+v", len(seq.Events), seq.Events)
	}
	for _, ev := range seq.Events[:3] {
		if !ev.On || ev.At != 0 || ev.Vel != 80 {
			t.Errorf("expected on@0 vel 80, got %+v", ev)
		}
	}
	// 1 beat at 90 BPM = 666.67ms
	wantOff := beatsToTime(1, 90)
	for _, ev := range seq.Events[3:] {
		if ev.On || ev.At != wantOff {
			t.Errorf("expected off@%v, got %+v", wantOff, ev)
		}
	}
}

func TestParseVelocityResolution(t *testing.T) {
	seq := parse(t, `{"velocity":100,"voices":[
		{"steps":[{"notes":["C4"]},{"notes":["D4"],"velocity":40}]},
		{"velocity":60,"steps":[{"notes":["E4"]}]}
	]}`)
	got := map[uint8]uint8{}
	for _, ev := range seq.Events {
		if ev.On {
			got[ev.Note] = ev.Vel
		}
	}
	// step > voice > sequence
	if got[60] != 100 || got[62] != 40 || got[64] != 60 {
		t.Errorf("velocities C4/D4/E4 = %d/%d/%d, want 100/40/60", got[60], got[62], got[64])
	}
}

func TestParseRejects(t *testing.T) {
	bad := map[string]string{
		"no content":       `{}`,
		"empty steps":      `{"steps":[]}`,
		"steps and voices": `{"steps":[{"notes":["C4"]}],"voices":[{"steps":[{"notes":["E4"]}]}]}`,
		"empty voice":      `{"voices":[{"steps":[]}]}`,
		"bad note":         `{"steps":[{"notes":["H4"]}]}`,
		"bad channel":      `{"channel":17,"steps":[{"notes":["C4"]}]}`,
		"bad velocity":     `{"velocity":200,"steps":[{"notes":["C4"]}]}`,
		"bad beats":        `{"steps":[{"notes":["C4"],"beats":-1}]}`,
		"unknown field":    `{"bpm":90,"steps":[{"notes":["C4"]}]}`,
		"not json":         `tempo: 90`,
	}
	for name, src := range bad {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("%s: Parse accepted %q", name, src)
		}
	}
}

func TestPlaySchedule(t *testing.T) {
	// One voice: chord 1 beat, rest 2 beats, note 1 beat — at 60 BPM.
	seq := parse(t, `{"tempo":60,"channel":2,
		"steps":[{"notes":["C4","E4"],"beats":1},{"beats":2},{"notes":["G4"],"beats":1,"velocity":50}]}`)

	out := midi.NewMockOut(nil)
	var slept []time.Duration
	if err := Play(out, seq, func(d time.Duration) { slept = append(slept, d) }); err != nil {
		t.Fatalf("Play error: %v", err)
	}

	// Sleeps: 0→1s (hold chord), 1s→3s (offs then silence), 3s→4s (hold G4).
	want := []time.Duration{time.Second, 2 * time.Second, time.Second}
	if len(slept) != len(want) {
		t.Fatalf("slept %v, want %v", slept, want)
	}
	for i := range want {
		if slept[i] != want[i] {
			t.Errorf("sleep %d = %v, want %v", i, slept[i], want[i])
		}
	}

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

// The point of voices: a melody moves, louder, while the other voice holds a
// chord — independent clocks, one event stream.
func TestVoicesOverlap(t *testing.T) {
	seq := parse(t, `{"tempo":60,"voices":[
		{"velocity":90,"steps":[{"notes":["E5"],"beats":1},{"notes":["D5"],"beats":1}]},
		{"velocity":60,"steps":[{"notes":["C3","E3","G3"],"beats":2}]}
	]}`)

	out := midi.NewMockOut(nil)
	var slept []time.Duration
	if err := Play(out, seq, func(d time.Duration) { slept = append(slept, d) }); err != nil {
		t.Fatalf("Play error: %v", err)
	}

	want := []midi.OutMsg{
		// t=0: chord (60) and first melody note (90) together
		{Status: "on", Channel: 0, Data1: 76, Data2: 90},
		{Status: "on", Channel: 0, Data1: 48, Data2: 60},
		{Status: "on", Channel: 0, Data1: 52, Data2: 60},
		{Status: "on", Channel: 0, Data1: 55, Data2: 60},
		// t=1s: melody moves while the chord holds — off before on
		{Status: "off", Channel: 0, Data1: 76},
		{Status: "on", Channel: 0, Data1: 74, Data2: 90},
		// t=2s: everything releases
		{Status: "off", Channel: 0, Data1: 74},
		{Status: "off", Channel: 0, Data1: 48},
		{Status: "off", Channel: 0, Data1: 52},
		{Status: "off", Channel: 0, Data1: 55},
	}
	if len(out.Msgs) != len(want) {
		t.Fatalf("got %d messages, want %d: %+v", len(out.Msgs), len(want), out.Msgs)
	}
	for i := range want {
		// Same-instant ordering between different notes isn't fully pinned by
		// the sort (stable on input order); compare as sets per timestamp via
		// direct equality here since input order is deterministic.
		if out.Msgs[i] != want[i] {
			t.Errorf("msg %d = %+v, want %+v", i, out.Msgs[i], want[i])
		}
	}
	if len(slept) != 2 || slept[0] != time.Second || slept[1] != time.Second {
		t.Errorf("sleeps = %v, want [1s 1s]", slept)
	}
}
