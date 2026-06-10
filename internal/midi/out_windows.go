//go:build windows

package midi

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Pure-Go MIDI output for Windows via WinMM, the mirror of device_windows.go.
// Simpler than the input side: we only push short messages, nothing calls back
// into Go, so there is no callback bridge and no instance registry.

var (
	procMidiOutGetNumDevs  = winmm.NewProc("midiOutGetNumDevs")
	procMidiOutGetDevCapsW = winmm.NewProc("midiOutGetDevCapsW")
	procMidiOutOpen        = winmm.NewProc("midiOutOpen")
	procMidiOutShortMsg    = winmm.NewProc("midiOutShortMsg")
	procMidiOutClose       = winmm.NewProc("midiOutClose")
)

var errNoOutputDevices = errors.New("no MIDI output devices found")

// midiOutCapsW mirrors the Win32 MIDIOUTCAPSW struct from midiOutGetDevCapsW.
type midiOutCapsW struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	PName         [maxPNameLen]uint16
	Technology    uint16
	Voices        uint16
	Notes         uint16
	ChannelMask   uint16
	Support       uint32
}

type deviceOut struct {
	handle uintptr
	once   sync.Once
}

func newDeviceOut(portMatch string) (MidiOut, error) {
	idx, err := findOutDevice(portMatch)
	if err != nil {
		return nil, err
	}
	var handle uintptr
	if r, _, _ := procMidiOutOpen.Call(
		uintptr(unsafe.Pointer(&handle)),
		uintptr(idx),
		0, 0, 0, // no callback: output never calls back into Go
	); r != 0 {
		return nil, fmt.Errorf("midiOutOpen(device %d) failed: mmresult=%d", idx, r)
	}
	return &deviceOut{handle: handle}, nil
}

// send packs a 3-byte short message the way midiOutShortMsg expects it:
// status in the low byte, then data1, then data2 (same layout as MIM_DATA).
func (o *deviceOut) send(status, d1, d2 byte) error {
	msg := uintptr(status) | uintptr(d1)<<8 | uintptr(d2)<<16
	if r, _, _ := procMidiOutShortMsg.Call(o.handle, msg); r != 0 {
		return fmt.Errorf("midiOutShortMsg failed: mmresult=%d", r)
	}
	return nil
}

func (o *deviceOut) NoteOn(ch, note, vel uint8) error {
	return o.send(0x90|ch&0x0F, note&0x7F, vel&0x7F)
}

func (o *deviceOut) NoteOff(ch, note uint8) error {
	return o.send(0x80|ch&0x0F, note&0x7F, 0)
}

func (o *deviceOut) Control(ch, controller, val uint8) error {
	return o.send(0xB0|ch&0x0F, controller&0x7F, val&0x7F)
}

func (o *deviceOut) Close() error {
	o.once.Do(func() {
		if o.handle != 0 {
			procMidiOutClose.Call(o.handle)
		}
	})
	return nil
}

func findOutDevice(portMatch string) (uint32, error) {
	n, _, _ := procMidiOutGetNumDevs.Call()
	if n == 0 {
		return 0, errNoOutputDevices
	}
	if portMatch == "" {
		return 0, nil // first available output
	}
	want := strings.ToLower(portMatch)
	for i := uintptr(0); i < n; i++ {
		var caps midiOutCapsW
		if r, _, _ := procMidiOutGetDevCapsW.Call(i, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps)); r != 0 {
			continue
		}
		name := windows.UTF16ToString(caps.PName[:])
		if strings.Contains(strings.ToLower(name), want) {
			return uint32(i), nil
		}
	}
	return 0, fmt.Errorf("no MIDI output device matching %q", portMatch)
}

// ListOutputs names every MIDI output device, for play7 --list.
func ListOutputs() ([]string, error) {
	n, _, _ := procMidiOutGetNumDevs.Call()
	names := make([]string, 0, n)
	for i := uintptr(0); i < n; i++ {
		var caps midiOutCapsW
		if r, _, _ := procMidiOutGetDevCapsW.Call(i, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps)); r != 0 {
			continue
		}
		names = append(names, windows.UTF16ToString(caps.PName[:]))
	}
	return names, nil
}
