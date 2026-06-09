package midi

import (
	"fmt"
	"time"
)

// Kind classifies the MIDI events keys7 cares about in phase 1.
type Kind uint8

const (
	NoteOn Kind = iota
	NoteOff
	ControlChange
)

// Event is a minimal, transport-agnostic MIDI event. Data1/Data2 are the raw
// MIDI data bytes: for notes, Data1 = note number and Data2 = velocity; for
// control changes, Data1 = controller number and Data2 = value.
type Event struct {
	Kind      Kind
	Channel   uint8
	Data1     uint8
	Data2     uint8
	Timestamp time.Time
}

const sustainController = 64

// IsPedal reports whether the event is a sustain-pedal control change (CC64).
func (e Event) IsPedal() bool { return e.Kind == ControlChange && e.Data1 == sustainController }

// PedalDown interprets a CC value as pedal state (>=64 is conventionally "down").
func (e Event) PedalDown() bool { return e.Data2 >= 64 }

// MidiSource is anything that produces a stream of MIDI events. The device and
// mock implementations are interchangeable; the UI never knows which it holds.
// This interface is the seam that carries the "Windows and Linux" requirement:
// real hardware on Windows, a synthetic stream anywhere (e.g. WSL).
type MidiSource interface {
	Events() <-chan Event
	Close() error
}

// NewSource builds a source by kind: "device" (real port) or "mock" (synthetic).
func NewSource(kind, portMatch string) (MidiSource, error) {
	switch kind {
	case "", "mock":
		return newMockSource(), nil
	case "device":
		return newDeviceSource(portMatch)
	default:
		return nil, fmt.Errorf("unknown source %q (use device|mock)", kind)
	}
}
