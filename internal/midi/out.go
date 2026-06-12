package midi

import (
	"fmt"
	"io"
)

// MidiOut is anything that can receive note messages — the output mirror of
// MidiSource. The device implementation drives a real instrument (play7 → the
// piano); the mock records messages for tests and can echo them to a writer.
type MidiOut interface {
	NoteOn(channel, note, velocity uint8) error
	NoteOff(channel, note uint8) error
	Control(channel, controller, value uint8) error
	Close() error
}

// AllNotesOff is the channel-mode controller (CC123) that silences every
// sounding note — sent before exiting so a real piano doesn't keep ringing.
const AllNotesOff = 123

// NewOut builds an output by kind: "device" (real port) or "mock" (echo is
// where mock messages are printed; nil keeps it silent).
func NewOut(kind, portMatch string, echo io.Writer) (MidiOut, error) {
	switch kind {
	case "", "device":
		return newDeviceOut(portMatch)
	case "mock":
		return NewMockOut(echo), nil
	default:
		return nil, fmt.Errorf("unknown output %q (use device|mock)", kind)
	}
}

// OutMsg is one recorded mock output message.
type OutMsg struct {
	Status                string // "on", "off", "cc"
	Channel, Data1, Data2 uint8
}

// MockOut records every message and optionally echoes it, so sequences can be
// tested (and auditioned on WSL) without hardware.
type MockOut struct {
	Msgs []OutMsg
	echo io.Writer
}

func NewMockOut(echo io.Writer) *MockOut { return &MockOut{echo: echo} }

func (m *MockOut) record(status string, ch, d1, d2 uint8) error {
	m.Msgs = append(m.Msgs, OutMsg{Status: status, Channel: ch, Data1: d1, Data2: d2})
	if m.echo != nil {
		fmt.Fprintf(m.echo, "%-3s ch=%d %d %d\n", status, ch+1, d1, d2)
	}
	return nil
}

func (m *MockOut) NoteOn(ch, note, vel uint8) error { return m.record("on", ch, note, vel) }
func (m *MockOut) NoteOff(ch, note uint8) error     { return m.record("off", ch, note, 0) }
func (m *MockOut) Control(ch, controller, val uint8) error {
	return m.record("cc", ch, controller, val)
}
func (m *MockOut) Close() error { return nil }
