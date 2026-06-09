package theory

import "fmt"

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

func SetNotation(n Notation) { active = n }
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
