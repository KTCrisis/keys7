// Package sequence parses and plays note/chord sequences for play7. Like
// theory, it is a pure layer: parsing knows nothing about MIDI transport, and
// playback only sees the MidiOut interface.
//
// A sequence is one or more voices, each a list of steps advancing on its own
// clock — a melody can move, louder, over a chord the other voice holds. The
// voices compile into a single sorted stream of timed note events.
package sequence

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"keys7/internal/theory"
)

// Defaults applied when the JSON omits a field.
const (
	defaultTempo    = 90.0
	defaultChannel  = 1 // human-facing 1-16
	defaultVelocity = 80
	defaultBeats    = 1.0
)

// Sequence is a compiled, validated sequence: timed events ready to play.
type Sequence struct {
	Tempo   float64
	Channel uint8 // wire channel (0-15)
	Events  []Event
}

// Event is one scheduled MIDI action. Events are sorted by At; at the same
// instant, note-offs come first, then control changes (so a pedal-down catches
// the chord), then note-ons — a re-struck note retriggers cleanly.
type Event struct {
	At   time.Duration
	On   bool
	Note uint8 // note number, or the controller number for a control event
	Vel  uint8 // note velocity, or the controller value for a control event
	Ctrl bool  // a control change (e.g. sustain pedal), not a note
}

// fileSeq / fileStep / fileVoice mirror the JSON document:
//
//	{
//	  "tempo": 65, "channel": 1, "velocity": 80,
//	  "voices": [
//	    {"velocity": 90, "steps": [{"notes": ["D5"], "beats": 1}, {"notes": ["F5"], "beats": 2}]},
//	    {"velocity": 58, "steps": [{"notes": ["Bb2", "D3", "F3", "A3"], "beats": 3}]}
//	  ]
//	}
//
// Voices start together and each advances by its own beats — that is what lets
// a line move while a chord holds. A top-level "steps" array is shorthand for
// a single voice. Notes use scientific pitch (C4 = middle C); a step with no
// notes is a rest. Velocity resolves step > voice > sequence.
type fileSeq struct {
	Tempo    float64     `json:"tempo"`
	Channel  uint8       `json:"channel"`
	Velocity uint8       `json:"velocity"`
	Steps    []fileStep  `json:"steps"`
	Voices   []fileVoice `json:"voices"`
}

type fileVoice struct {
	Velocity uint8      `json:"velocity"`
	Steps    []fileStep `json:"steps"`
}

type fileStep struct {
	Notes    []string `json:"notes"`
	Beats    float64  `json:"beats"`
	Velocity uint8    `json:"velocity"`
	Pedal    string   `json:"pedal,omitempty"` // "down" | "up": a sustain change at the step's onset
}

// sustainController is CC64, the damper (sustain) pedal.
const sustainController = 64

// Parse reads, validates and compiles a JSON sequence into timed events.
func Parse(r io.Reader) (Sequence, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var f fileSeq
	if err := dec.Decode(&f); err != nil {
		return Sequence{}, fmt.Errorf("bad sequence: %w", err)
	}

	if f.Tempo == 0 {
		f.Tempo = defaultTempo
	}
	if f.Tempo < 0 {
		return Sequence{}, fmt.Errorf("bad tempo %v: must be positive", f.Tempo)
	}
	if f.Channel == 0 {
		f.Channel = defaultChannel
	}
	if f.Channel > 16 {
		return Sequence{}, fmt.Errorf("bad channel %d: must be 1-16", f.Channel)
	}
	if f.Velocity == 0 {
		f.Velocity = defaultVelocity
	}
	if f.Velocity > 127 {
		return Sequence{}, fmt.Errorf("bad velocity %d: must be 1-127", f.Velocity)
	}

	if len(f.Steps) > 0 && len(f.Voices) > 0 {
		return Sequence{}, fmt.Errorf("use either steps or voices, not both")
	}
	voices := f.Voices
	if len(f.Steps) > 0 {
		voices = []fileVoice{{Steps: f.Steps}}
	}
	if len(voices) == 0 {
		return Sequence{}, fmt.Errorf("empty sequence: no steps or voices")
	}

	seq := Sequence{Tempo: f.Tempo, Channel: f.Channel - 1}
	for vi, v := range voices {
		if v.Velocity == 0 {
			v.Velocity = f.Velocity
		}
		if v.Velocity > 127 {
			return Sequence{}, fmt.Errorf("voice %d: bad velocity %d: must be 1-127", vi+1, v.Velocity)
		}
		if len(v.Steps) == 0 {
			return Sequence{}, fmt.Errorf("voice %d: no steps", vi+1)
		}
		cursor := 0.0 // beats since the start; all voices share t=0
		for si, fs := range v.Steps {
			beats, vel := fs.Beats, fs.Velocity
			if beats == 0 {
				beats = defaultBeats
			}
			if beats < 0 {
				return Sequence{}, fmt.Errorf("voice %d step %d: bad beats %v: must be positive", vi+1, si+1, fs.Beats)
			}
			if vel == 0 {
				vel = v.Velocity
			}
			if vel > 127 {
				return Sequence{}, fmt.Errorf("voice %d step %d: bad velocity %d: must be 1-127", vi+1, si+1, fs.Velocity)
			}
			on, off := beatsToTime(cursor, f.Tempo), beatsToTime(cursor+beats, f.Tempo)
			if fs.Pedal != "" {
				val, err := pedalValue(fs.Pedal)
				if err != nil {
					return Sequence{}, fmt.Errorf("voice %d step %d: %w", vi+1, si+1, err)
				}
				seq.Events = append(seq.Events, Event{At: on, Ctrl: true, Note: sustainController, Vel: val})
			}
			for _, name := range fs.Notes {
				n, err := theory.ParseNote(name)
				if err != nil {
					return Sequence{}, fmt.Errorf("voice %d step %d: %w", vi+1, si+1, err)
				}
				seq.Events = append(seq.Events,
					Event{At: on, On: true, Note: n, Vel: vel},
					Event{At: off, On: false, Note: n},
				)
			}
			cursor += beats
		}
	}

	sort.SliceStable(seq.Events, func(i, j int) bool {
		a, b := seq.Events[i], seq.Events[j]
		if a.At != b.At {
			return a.At < b.At
		}
		return eventOrder(a) < eventOrder(b) // same instant: offs, then control, then ons
	})
	return seq, nil
}

// eventOrder ranks simultaneous events: note-offs (0) before control changes (1)
// before note-ons (2) — so a pedal-down catches the chord it precedes, and a
// re-struck note isn't killed by its own off.
func eventOrder(e Event) int {
	switch {
	case e.Ctrl:
		return 1
	case !e.On:
		return 0
	default:
		return 2
	}
}

// pedalValue maps a step's "pedal" field to a CC64 value.
func pedalValue(s string) (uint8, error) {
	switch s {
	case "down":
		return 127, nil
	case "up":
		return 0, nil
	default:
		return 0, fmt.Errorf("bad pedal %q: use \"down\" or \"up\"", s)
	}
}

// beatsToTime converts a beat position to wall time at a tempo. Positions stay
// in beats until this conversion so long sequences don't accumulate drift.
func beatsToTime(beats, tempo float64) time.Duration {
	return time.Duration(beats * 60 / tempo * float64(time.Second))
}
