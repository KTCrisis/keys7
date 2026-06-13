package smf

import (
	"bytes"
	"testing"
)

// TestWriteRoundTrip writes two notes and parses the bytes back, asserting the
// header, the tempo meta, and that timing survives as ticks (500 ms at 120 BPM
// = 480 ticks = one quarter note at PPQ 480).
func TestWriteRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	msgs := []Msg{
		{MS: 0, Status: NoteOn, D1: 60, D2: 100},
		{MS: 500, Status: NoteOff, D1: 60, D2: 0},
	}
	if err := Write(&buf, msgs, 120); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()

	if string(b[0:4]) != "MThd" {
		t.Fatalf("header = %q, want MThd", b[0:4])
	}
	// division (bytes 12-13) = PPQ
	if div := int(b[12])<<8 | int(b[13]); div != PPQ {
		t.Errorf("division = %d, want %d", div, PPQ)
	}
	// track chunk
	if string(b[14:18]) != "MTrk" {
		t.Fatalf("track header = %q, want MTrk", b[14:18])
	}

	// Parse the track events and check the two notes land at the right ticks.
	track := b[22:]
	events := parseTrack(t, track)
	want := []struct {
		tick   uint32
		status byte
		d1, d2 byte
	}{
		{0, NoteOn, 60, 100},
		{480, NoteOff, 60, 0},
	}
	if len(events) != len(want) {
		t.Fatalf("got %d note events, want %d", len(events), len(want))
	}
	for i, w := range want {
		e := events[i]
		if e.tick != w.tick || e.status != w.status || e.d1 != w.d1 || e.d2 != w.d2 {
			t.Errorf("event %d = {tick:%d %#x %d %d}, want {tick:%d %#x %d %d}",
				i, e.tick, e.status, e.d1, e.d2, w.tick, w.status, w.d1, w.d2)
		}
	}
}

type ev struct {
	tick   uint32
	status byte
	d1, d2 byte
}

// parseTrack walks a track chunk, returning only the channel-voice events with
// their absolute ticks. It skips the tempo meta and end-of-track.
func parseTrack(t *testing.T, track []byte) []ev {
	t.Helper()
	var out []ev
	var abs uint32
	i := 0
	for i < len(track) {
		delta, n := readVarLen(track[i:])
		i += n
		abs += delta
		if i >= len(track) {
			break
		}
		status := track[i]
		switch {
		case status == 0xFF: // meta: FF type len data
			i++
			metaLen := int(track[i+1])
			i += 2 + metaLen
		case status&0x80 != 0:
			out = append(out, ev{tick: abs, status: status, d1: track[i+1], d2: track[i+2]})
			i += 3
		default:
			t.Fatalf("unexpected byte %#x at %d", status, i)
		}
	}
	return out
}

// TestReadRoundTrip writes messages then reads them back, asserting the notes,
// control change, tempo and millisecond times survive the SMF round-trip.
func TestReadRoundTrip(t *testing.T) {
	in := []Msg{
		{MS: 0, Status: NoteOn, D1: 60, D2: 90},
		{MS: 0, Status: NoteOn, D1: 64, D2: 80},
		{MS: 500, Status: ControlChange, D1: 64, D2: 127},
		{MS: 1000, Status: NoteOff, D1: 60, D2: 0},
		{MS: 1000, Status: NoteOff, D1: 64, D2: 0},
	}
	var buf bytes.Buffer
	if err := Write(&buf, in, 120); err != nil {
		t.Fatal(err)
	}
	got, bpm, err := Read(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if bpm != 120 {
		t.Errorf("bpm = %v, want 120", bpm)
	}
	if len(got) != len(in) {
		t.Fatalf("got %d messages, want %d", len(got), len(in))
	}
	key := func(m Msg) [4]int { return [4]int{int(m.MS), int(m.Status), int(m.D1), int(m.D2)} }
	seen := map[[4]int]int{}
	for _, m := range got {
		seen[key(m)]++
	}
	for _, m := range in {
		if seen[key(m)] == 0 {
			t.Errorf("message %+v missing from round-trip", m)
		}
		seen[key(m)]--
	}
}
