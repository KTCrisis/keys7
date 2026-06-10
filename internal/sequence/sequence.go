// Package sequence parses and plays note/chord sequences for play7. Like
// theory, it is a pure layer: parsing knows nothing about MIDI transport, and
// playback only sees the MidiOut interface.
package sequence

import (
	"encoding/json"
	"fmt"
	"io"
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

// Sequence is a validated sequence ready to play. Channel is the wire channel
// (0-15); velocities are already resolved per step.
type Sequence struct {
	Tempo   float64
	Channel uint8
	Steps   []Step
}

// Step is one playable unit: a note (one entry), a chord (several), or a rest
// (none), held for Beats at the sequence tempo.
type Step struct {
	Notes    []uint8
	Beats    float64
	Velocity uint8
}

// Duration converts a step's beats to wall time at the sequence tempo.
func (s Sequence) Duration(st Step) time.Duration {
	return time.Duration(st.Beats * 60 / s.Tempo * float64(time.Second))
}

// fileSeq / fileStep mirror the JSON document:
//
//	{
//	  "tempo": 90, "channel": 1, "velocity": 80,
//	  "steps": [
//	    {"notes": ["A3", "C4", "E4"], "beats": 2},
//	    {"notes": ["G3"], "beats": 0.5, "velocity": 100},
//	    {"beats": 1}
//	  ]
//	}
//
// Notes use scientific pitch notation (C4 = middle C = MIDI 60); a step with
// no notes is a rest. Channel is 1-16 as on the instrument's panel.
type fileSeq struct {
	Tempo    float64    `json:"tempo"`
	Channel  uint8      `json:"channel"`
	Velocity uint8      `json:"velocity"`
	Steps    []fileStep `json:"steps"`
}

type fileStep struct {
	Notes    []string `json:"notes"`
	Beats    float64  `json:"beats"`
	Velocity uint8    `json:"velocity"`
}

// Parse reads, validates and resolves a JSON sequence.
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
	if len(f.Steps) == 0 {
		return Sequence{}, fmt.Errorf("empty sequence: no steps")
	}

	seq := Sequence{Tempo: f.Tempo, Channel: f.Channel - 1, Steps: make([]Step, 0, len(f.Steps))}
	for i, fs := range f.Steps {
		st := Step{Beats: fs.Beats, Velocity: fs.Velocity}
		if st.Beats == 0 {
			st.Beats = defaultBeats
		}
		if st.Beats < 0 {
			return Sequence{}, fmt.Errorf("step %d: bad beats %v: must be positive", i+1, fs.Beats)
		}
		if st.Velocity == 0 {
			st.Velocity = f.Velocity
		}
		if st.Velocity > 127 {
			return Sequence{}, fmt.Errorf("step %d: bad velocity %d: must be 1-127", i+1, fs.Velocity)
		}
		for _, name := range fs.Notes {
			n, err := theory.ParseNote(name)
			if err != nil {
				return Sequence{}, fmt.Errorf("step %d: %w", i+1, err)
			}
			st.Notes = append(st.Notes, n)
		}
		seq.Steps = append(seq.Steps, st)
	}
	return seq, nil
}
