package smf

import (
	"fmt"
	"io"
	"sort"
)

// Read parses a Standard MIDI File (type 0 or 1) into absolute-timed messages —
// the inverse of Write. It follows tempo meta events so the millisecond times
// reflect real playback, merges all tracks, and keeps only note and
// control-change channel events (enough to replay a piece). The returned bpm is
// the file's initial tempo. SMPTE time division is not supported.
func Read(r io.Reader) (msgs []Msg, bpm float64, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	if len(data) < 14 || string(data[0:4]) != "MThd" {
		return nil, 0, fmt.Errorf("not a Standard MIDI File (no MThd header)")
	}
	division := int(data[12])<<8 | int(data[13])
	if division&0x8000 != 0 {
		return nil, 0, fmt.Errorf("SMPTE time division is not supported")
	}
	ppq := division
	if ppq == 0 {
		return nil, 0, fmt.Errorf("bad PPQ division 0")
	}

	type chEvent struct {
		tick           uint32
		status, d1, d2 byte
	}
	type tempoEvent struct {
		tick uint32
		us   uint32 // microseconds per quarter note
	}
	var chs []chEvent
	var tempos []tempoEvent

	pos := 8 + int(beU32(data[4:8])) // skip header chunk
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		length := int(beU32(data[pos+4 : pos+8]))
		pos += 8
		if pos+length > len(data) {
			break
		}
		body := data[pos : pos+length]
		pos += length
		if id != "MTrk" {
			continue
		}
		// walk the track, accumulating absolute ticks
		var abs uint32
		var status byte
		i := 0
		for i < len(body) {
			delta, n := readVarLen(body[i:])
			i += n
			abs += delta
			if i >= len(body) {
				break
			}
			b := body[i]
			if b&0x80 != 0 {
				status = b
				i++
			} // else running status: reuse the previous status byte
			switch {
			case status == 0xFF: // meta event
				metaType := body[i]
				i++
				mlen, mn := readVarLen(body[i:])
				i += mn
				if metaType == 0x51 && mlen == 3 { // set tempo
					us := uint32(body[i])<<16 | uint32(body[i+1])<<8 | uint32(body[i+2])
					tempos = append(tempos, tempoEvent{tick: abs, us: us})
				}
				i += int(mlen)
			case status == 0xF0 || status == 0xF7: // sysex: skip
				slen, sn := readVarLen(body[i:])
				i += sn + int(slen)
			default: // channel voice message
				hi := status & 0xF0
				switch hi {
				case 0x80, 0x90, 0xA0, 0xB0, 0xE0: // two data bytes
					chs = append(chs, chEvent{tick: abs, status: status, d1: body[i], d2: body[i+1]})
					i += 2
				case 0xC0, 0xD0: // one data byte
					chs = append(chs, chEvent{tick: abs, status: status, d1: body[i]})
					i++
				default:
					i++ // unknown: best-effort skip
				}
			}
		}
	}

	// Build a tempo map and a tick→ms integrator.
	sort.SliceStable(tempos, func(i, j int) bool { return tempos[i].tick < tempos[j].tick })
	tickToMs := func(tick uint32) float64 {
		ms := 0.0
		var lastTick uint32
		us := uint32(500000) // default 120 BPM until the first tempo event
		for _, te := range tempos {
			if te.tick >= tick {
				break
			}
			ms += float64(te.tick-lastTick) * float64(us) / float64(ppq) / 1000.0
			lastTick, us = te.tick, te.us
		}
		ms += float64(tick-lastTick) * float64(us) / float64(ppq) / 1000.0
		return ms
	}

	for _, e := range chs {
		hi := e.status & 0xF0
		if hi != 0x80 && hi != 0x90 && hi != 0xB0 { // keep notes + control changes
			continue
		}
		msgs = append(msgs, Msg{MS: tickToMs(e.tick), Status: hi, D1: e.d1, D2: e.d2})
	}
	sort.SliceStable(msgs, func(i, j int) bool { return msgs[i].MS < msgs[j].MS })

	bpm = 120
	if len(tempos) > 0 {
		bpm = 60_000_000 / float64(tempos[0].us)
	}
	return msgs, bpm, nil
}

func beU32(b []byte) uint32 {
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// readVarLen decodes a MIDI variable-length quantity, returning the value and
// the number of bytes consumed.
func readVarLen(b []byte) (uint32, int) {
	var v uint32
	for i, c := range b {
		v = v<<7 | uint32(c&0x7F)
		if c&0x80 == 0 {
			return v, i + 1
		}
	}
	return v, len(b)
}
