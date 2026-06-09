package midi

import "fmt"

// octaveOffset maps MIDI note 60 (middle C) to an octave label. With -1, note
// 60 renders as "C4" (scientific pitch notation, the MIDI standard where
// A4 = 69 = 440 Hz). Renoise's OSC layer labels middle C differently
// (48 = "C-4"); change this single constant if we ever want keys7's display to
// match Renoise's reference instead.
const octaveOffset = -1

var noteNames = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// NoteName renders a MIDI note number (0-127) as a pitch label, e.g. "C4", "F#3".
func NoteName(n uint8) string {
	return fmt.Sprintf("%s%d", noteNames[n%12], int(n)/12+octaveOffset)
}
