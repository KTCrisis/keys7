//go:build midi_device

package midi

import (
	"errors"
	"fmt"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
)

// IMPORTANT — register an RtMidi driver for the Windows build. The gomidi port
// API below (GetInPorts/ListenTo) needs a backend driver registered via a blank
// import, normally:
//
//	import _ "gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
//
// That import is deliberately omitted from committed code: rtmididrv's only
// published pseudo-version has a broken nested go.mod (imported/rtmidi) that
// fails `go mod tidy` on Linux. Add it on the Windows host where CGO + a C
// toolchain are present, pinning a version that resolves. Without a driver,
// GetInPorts/ListenTo have no backend and the device source finds no ports.
//
// deviceSource reads a real MIDI input port via RtMidi (WinMM on Windows, ALSA
// on Linux). Compiled only with `-tags midi_device`, which also enables CGO.
//
// NOTE: this path has NOT been validated against live hardware in this repo —
// run it on the Windows host with the P-125 connected and adjust the gomidi
// calls if the installed API differs. The mock source is the verified path.
type deviceSource struct {
	ch   chan Event
	stop func()
}

func newDeviceSource(portMatch string) (MidiSource, error) {
	in, err := pickInPort(portMatch)
	if err != nil {
		return nil, err
	}

	s := &deviceSource{ch: make(chan Event, 64)}

	// ListenTo runs the callback on the driver's own goroutine. We never block
	// it: if the UI falls behind we drop the event rather than stall the MIDI
	// thread (a stalled callback would back up the whole input port).
	stop, err := midi.ListenTo(in, func(msg midi.Message, _ int32) {
		var ch, d1, d2 uint8
		var ev Event
		switch {
		case msg.GetNoteStart(&ch, &d1, &d2):
			ev = Event{Kind: NoteOn, Channel: ch, Data1: d1, Data2: d2}
		case msg.GetNoteEnd(&ch, &d1):
			ev = Event{Kind: NoteOff, Channel: ch, Data1: d1}
		case msg.GetControlChange(&ch, &d1, &d2):
			ev = Event{Kind: ControlChange, Channel: ch, Data1: d1, Data2: d2}
		default:
			return
		}
		ev.Timestamp = time.Now()
		select {
		case s.ch <- ev:
		default:
		}
	})
	if err != nil {
		return nil, fmt.Errorf("listen on MIDI input: %w", err)
	}
	s.stop = stop
	return s, nil
}

func pickInPort(portMatch string) (drivers.In, error) {
	if portMatch != "" {
		in, err := midi.FindInPort(portMatch)
		if err != nil {
			return nil, fmt.Errorf("no MIDI input matching %q: %w", portMatch, err)
		}
		return in, nil
	}
	ports := midi.GetInPorts()
	if len(ports) == 0 {
		return nil, errors.New("no MIDI input ports found")
	}
	return ports[0], nil
}

func (s *deviceSource) Events() <-chan Event { return s.ch }

func (s *deviceSource) Close() error {
	if s.stop != nil {
		s.stop()
	}
	close(s.ch)
	midi.CloseDriver()
	return nil
}
