package sequence

import (
	"time"

	"keys7/internal/midi"
)

// Play sends the compiled events to out, blocking until done. sleep is
// injected so tests can assert the schedule without waiting through it (pass
// time.Sleep for real playback).
func Play(out midi.MidiOut, seq Sequence, sleep func(time.Duration)) error {
	var now time.Duration
	for _, ev := range seq.Events {
		if ev.At > now {
			sleep(ev.At - now)
			now = ev.At
		}
		var err error
		switch {
		case ev.Ctrl:
			err = out.Control(seq.Channel, ev.Note, ev.Vel)
		case ev.On:
			err = out.NoteOn(seq.Channel, ev.Note, ev.Vel)
		default:
			err = out.NoteOff(seq.Channel, ev.Note)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
