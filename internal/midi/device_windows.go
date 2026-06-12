//go:build windows

package midi

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Pure-Go MIDI input for Windows via WinMM (winmm.dll). No CGO: the driver is
// reached through syscalls and a callback created with windows.NewCallback —
// the bridge that lets WinMM's C runtime call back into Go. This is what lets
// keys7.exe be cross-compiled from WSL with `GOOS=windows CGO_ENABLED=0`, with
// no mingw, no RtMidi, and no third-party MIDI module.

var (
	winmm                 = windows.NewLazyDLL("winmm.dll")
	procMidiInGetNumDevs  = winmm.NewProc("midiInGetNumDevs")
	procMidiInGetDevCapsW = winmm.NewProc("midiInGetDevCapsW")
	procMidiInOpen        = winmm.NewProc("midiInOpen")
	procMidiInStart       = winmm.NewProc("midiInStart")
	procMidiInStop        = winmm.NewProc("midiInStop")
	procMidiInClose       = winmm.NewProc("midiInClose")
)

const (
	callbackFunction = 0x00030000 // CALLBACK_FUNCTION
	mimData          = 0x3C3      // MIM_DATA: a complete short MIDI message
	maxPNameLen      = 32
)

var errNoInputDevices = errors.New("no MIDI input devices found")

// midiInCapsW mirrors the Win32 MIDIINCAPSW struct from midiInGetDevCapsW.
type midiInCapsW struct {
	Mid           uint16
	Pid           uint16
	DriverVersion uint32
	PName         [maxPNameLen]uint16
	Support       uint32
}

type deviceSource struct {
	ch       chan Event
	handle   uintptr
	instance uintptr // registry key; removed on Close and on open failure
	once     sync.Once
}

// The WinMM callback runs on an OS thread with no Go closure context, so we
// reach the right source through a small registry keyed by an instance id
// passed to midiInOpen, rather than capturing the channel in a closure.
var (
	regMu    sync.Mutex
	regNext  uintptr
	registry = map[uintptr]*deviceSource{}
)

func newDeviceSource(portMatch string) (MidiSource, error) {
	idx, err := findDevice(portMatch)
	if err != nil {
		return nil, err
	}
	s := &deviceSource{ch: make(chan Event, 64)}

	regMu.Lock()
	regNext++
	s.instance = regNext
	registry[s.instance] = s
	regMu.Unlock()

	var handle uintptr
	cb := windows.NewCallback(midiInProc)
	if r, _, _ := procMidiInOpen.Call(
		uintptr(unsafe.Pointer(&handle)),
		uintptr(idx),
		cb,
		s.instance,
		callbackFunction,
	); r != 0 {
		unregister(s.instance)
		return nil, fmt.Errorf("midiInOpen(device %d) failed: mmresult=%d", idx, r)
	}
	s.handle = handle

	if r, _, _ := procMidiInStart.Call(handle); r != 0 {
		procMidiInClose.Call(handle)
		unregister(s.instance)
		return nil, fmt.Errorf("midiInStart failed: mmresult=%d", r)
	}
	return s, nil
}

func unregister(instance uintptr) {
	regMu.Lock()
	delete(registry, instance)
	regMu.Unlock()
}

// midiInProc is the WinMM input callback. For MIM_DATA, dwParam1 packs the short
// message: status in the low byte, then data1, then data2.
func midiInProc(_, wMsg, dwInstance, dwParam1, _ uintptr) uintptr {
	if wMsg != mimData {
		return 0
	}
	regMu.Lock()
	s := registry[dwInstance]
	regMu.Unlock()
	if s == nil {
		return 0
	}
	ev, ok := decodeShort(byte(dwParam1), byte(dwParam1>>8), byte(dwParam1>>16))
	if !ok {
		return 0
	}
	ev.Timestamp = time.Now()
	select {
	case s.ch <- ev: // never block the MIDI callback; drop if the UI is behind
	default:
	}
	return 0
}

// decodeShort turns a 3-byte short MIDI message into our Event model.
func decodeShort(status, d1, d2 byte) (Event, bool) {
	ch := status & 0x0F
	switch status & 0xF0 {
	case 0x90: // note on (velocity 0 is the running-status note off)
		return Event{Kind: NoteOn, Channel: ch, Data1: d1, Data2: d2}, true
	case 0x80: // note off
		return Event{Kind: NoteOff, Channel: ch, Data1: d1, Data2: d2}, true
	case 0xB0: // control change (includes sustain pedal CC64)
		return Event{Kind: ControlChange, Channel: ch, Data1: d1, Data2: d2}, true
	}
	return Event{}, false
}

func findDevice(portMatch string) (uint32, error) {
	n, _, _ := procMidiInGetNumDevs.Call()
	if n == 0 {
		return 0, errNoInputDevices
	}
	if portMatch == "" {
		return 0, nil // first available input
	}
	want := strings.ToLower(portMatch)
	for i := uintptr(0); i < n; i++ {
		var caps midiInCapsW
		if r, _, _ := procMidiInGetDevCapsW.Call(i, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps)); r != 0 {
			continue
		}
		name := windows.UTF16ToString(caps.PName[:])
		if strings.Contains(strings.ToLower(name), want) {
			return uint32(i), nil
		}
	}
	return 0, fmt.Errorf("no MIDI input device matching %q", portMatch)
}

func (s *deviceSource) Events() <-chan Event { return s.ch }

func (s *deviceSource) Close() error {
	s.once.Do(func() {
		if s.handle != 0 {
			procMidiInStop.Call(s.handle)
			procMidiInClose.Call(s.handle)
		}
		// Unregister before closing the channel: a callback that fires after
		// midiInStop must not find s and send on a closed channel (panic).
		unregister(s.instance)
		close(s.ch)
	})
	return nil
}
