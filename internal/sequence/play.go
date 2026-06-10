package sequence

import (
	"time"

	"keys7/internal/midi"
)

// Play sends the sequence to out, blocking until done. Steps are legato: all
// notes of a step sound for its full duration, then stop as the next begins.
// sleep is injected so tests can assert the schedule without waiting through
// it (pass time.Sleep for real playback).
func Play(out midi.MidiOut, seq Sequence, sleep func(time.Duration)) error {
	for _, st := range seq.Steps {
		for _, n := range st.Notes {
			if err := out.NoteOn(seq.Channel, n, st.Velocity); err != nil {
				return err
			}
		}
		sleep(seq.Duration(st))
		for _, n := range st.Notes {
			if err := out.NoteOff(seq.Channel, n); err != nil {
				return err
			}
		}
	}
	return nil
}
