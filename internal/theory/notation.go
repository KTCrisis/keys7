package theory

import (
	"fmt"
	"strconv"
	"strings"
)

// Notation selects how note names are spelled.
type Notation int

const (
	Letters Notation = iota // C C# D ... B
	Solfege                 // Do Do# Ré ... Si
)

var (
	letterNames  = [12]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	solfegeNames = [12]string{"Do", "Do#", "Ré", "Ré#", "Mi", "Fa", "Fa#", "Sol", "Sol#", "La", "La#", "Si"}
)

// active is the default display notation, process-global state set by the UI — a
// pure presentation choice, so chord/key String() methods can stay parameterless.
var active = Letters

func SetNotation(n Notation)   { active = n }
func ActiveNotation() Notation { return active }

func ToggleNotation() {
	if active == Letters {
		active = Solfege
	} else {
		active = Letters
	}
}

// Other returns the spelling that isn't n — used to show both at once.
func (n Notation) Other() Notation {
	if n == Letters {
		return Solfege
	}
	return Letters
}

// octaveOffset maps MIDI note 60 (middle C) to "C4" — scientific pitch notation
// (A4 = 69 = 440 Hz). Renoise labels middle C differently (48 = "C-4"); change
// this one constant to match a different reference.
const octaveOffset = -1

// PitchClassNameIn renders a pitch class (0-11) in a given notation.
func PitchClassNameIn(pc uint8, n Notation) string {
	if n == Solfege {
		return solfegeNames[pc%12]
	}
	return letterNames[pc%12]
}

// NoteNameIn renders a MIDI note number (0-127) with its octave, in a notation.
func NoteNameIn(note uint8, n Notation) string {
	return fmt.Sprintf("%s%d", PitchClassNameIn(note, n), int(note)/12+octaveOffset)
}

// PitchClassName / NoteName use the active notation.
func PitchClassName(pc uint8) string { return PitchClassNameIn(pc, active) }
func NoteName(note uint8) string     { return NoteNameIn(note, active) }

// ParseNote reads scientific pitch notation ("C4", "F#3", "Bb-1") into a MIDI
// note number — the inverse of NoteNameIn. Letters only: sequences are data,
// solfege stays a display concern.
func ParseNote(s string) (uint8, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty note")
	}
	pcU, ok := letterPC[s[0]&^0x20]
	if !ok {
		return 0, fmt.Errorf("bad note %q: expected a letter A-G", s)
	}
	// Signed, no %12 wrap: Cb4 must come out below C4 (= B3), not as B4.
	pc, i := int(pcU), 1
	if i < len(s) {
		switch s[i] {
		case '#':
			pc, i = pc+1, i+1
		case 'b':
			pc, i = pc-1, i+1
		}
	}
	oct, err := strconv.Atoi(s[i:])
	if err != nil {
		return 0, fmt.Errorf("bad note %q: expected an octave like C4", s)
	}
	n := (oct-octaveOffset)*12 + pc
	if n < 0 || n > 127 {
		return 0, fmt.Errorf("note %q is outside the MIDI range 0-127", s)
	}
	return uint8(n), nil
}
