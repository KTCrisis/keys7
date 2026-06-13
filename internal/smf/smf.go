// Package smf writes a Standard MIDI File from timed channel messages. Pure Go,
// no third-party MIDI dependency — the same no-CGO line as keys7's WinMM
// backends. It exists so a keys7 session journal can be turned into an editable
// .mid (the bridge back to Renoise / MuseScore).
package smf

import (
	"bufio"
	"encoding/binary"
	"io"
	"sort"
)

// PPQ is the file's time division: ticks per quarter note.
const PPQ = 480

// Channel-message status bytes (channel 0; keys7 exports a single channel).
const (
	NoteOff       = 0x80
	NoteOn        = 0x90
	ControlChange = 0xB0
)

// Msg is one channel message at an absolute time in milliseconds from the start
// of the session. D2 is the velocity (notes) or value (control changes).
type Msg struct {
	MS     float64
	Status byte
	D1, D2 byte
}

// Write emits a type-0 SMF (one track) to w: a tempo meta, then the messages in
// time order (note-offs before note-ons at the same instant, so a re-struck
// note retriggers cleanly), then end-of-track. bpm only sets the tempo meta and
// the ms→tick conversion; absolute timing is preserved either way because real
// milliseconds are encoded as ticks at that tempo.
func Write(w io.Writer, msgs []Msg, bpm float64) error {
	if bpm <= 0 {
		bpm = 120
	}
	usPerQN := uint32(60_000_000 / bpm)
	ticksPerMS := float64(PPQ) * bpm / 60_000

	ordered := append([]Msg(nil), msgs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].MS != ordered[j].MS {
			return ordered[i].MS < ordered[j].MS
		}
		return isOff(ordered[i].Status) && !isOff(ordered[j].Status)
	})

	var track []byte
	// tempo meta at tick 0: FF 51 03 tt tt tt
	track = append(track, 0x00, 0xFF, 0x51, 0x03,
		byte(usPerQN>>16), byte(usPerQN>>8), byte(usPerQN))

	prevTick := uint32(0)
	for _, m := range ordered {
		tick := uint32(m.MS*ticksPerMS + 0.5)
		if tick < prevTick {
			tick = prevTick // never go backwards
		}
		track = appendVarLen(track, tick-prevTick)
		track = append(track, m.Status, m.D1, m.D2)
		prevTick = tick
	}
	// end of track: delta 0, FF 2F 00
	track = append(track, 0x00, 0xFF, 0x2F, 0x00)

	bw := bufio.NewWriter(w)
	// MThd: format 0, 1 track, division = PPQ
	bw.WriteString("MThd")
	writeU32(bw, 6)
	writeU16(bw, 0)
	writeU16(bw, 1)
	writeU16(bw, PPQ)
	// MTrk
	bw.WriteString("MTrk")
	writeU32(bw, uint32(len(track)))
	bw.Write(track)
	return bw.Flush()
}

func isOff(status byte) bool { return status&0xF0 == NoteOff }

// appendVarLen appends a MIDI variable-length quantity (7 bits per byte, high
// bit set on all but the last).
func appendVarLen(b []byte, v uint32) []byte {
	buf := []byte{byte(v & 0x7F)}
	for v >>= 7; v > 0; v >>= 7 {
		buf = append(buf, byte(v&0x7F|0x80))
	}
	for i := len(buf) - 1; i >= 0; i-- {
		b = append(b, buf[i])
	}
	return b
}

func writeU16(w io.Writer, v uint16) { binary.Write(w, binary.BigEndian, v) }
func writeU32(w io.Writer, v uint32) { binary.Write(w, binary.BigEndian, v) }
