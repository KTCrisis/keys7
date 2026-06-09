//go:build !midi_device

package midi

import "errors"

// errNoDeviceSupport is returned when keys7 is built without the device driver.
// The default build is pure Go (no CGO) and ships only the mock source; the
// real RtMidi-backed source lives in device_rtmidi.go behind the midi_device
// build tag.
var errNoDeviceSupport = errors.New(
	"keys7 built without MIDI device support; rebuild with `-tags midi_device` (requires CGO + RtMidi)")

func newDeviceSource(string) (MidiSource, error) { return nil, errNoDeviceSupport }
