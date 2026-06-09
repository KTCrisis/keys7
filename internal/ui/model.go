package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
)

// eventMsg carries a MIDI event into the Bubble Tea update loop.
type eventMsg midi.Event

// sourceClosedMsg signals the MIDI source channel has closed.
type sourceClosedMsg struct{}

const maxRecent = 16

// Model is the phase-1 UI state: which source we're on, the currently held
// notes, pedal state, and a short ring buffer of recent events.
type Model struct {
	sourceKind string
	port       string
	events     <-chan midi.Event
	fwd        mesh.Forwarder

	key         theory.Key
	splitMelody bool // peel isolated top notes as melody (vs fold into the chord)
	autoKey     bool // infer the key from what's played
	conf        float64
	recentPCs   []uint8 // sliding window of recent note-on pitch classes
	held        map[uint8]bool
	pedal       bool
	recent      []midi.Event
	closed      bool

	// derived chord state, recomputed when the held notes change
	core, melody    []uint8
	chord           theory.Chord
	chordOK         bool
	prevChord       theory.Chord // the recognized chord before the current one
	prevOK          bool
	lastChord       theory.Chord // most recent recognized chord (persists through releases)
	lastOK          bool
}

// detectWindow is how many recent note-ons feed key detection.
const detectWindow = 32

// New builds the model. `events` is the source's channel; `fwd` is the mesh
// seam (a NopForwarder for now); `key` is the starting key, and `autoKey`
// enables inferring it from what's played.
func New(sourceKind, port string, key theory.Key, autoKey bool, events <-chan midi.Event, fwd mesh.Forwarder) Model {
	return Model{
		sourceKind:  sourceKind,
		port:        port,
		key:         key,
		splitMelody: true,
		autoKey:     autoKey,
		events:      events,
		fwd:         fwd,
		held:        make(map[uint8]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return waitForEvent(m.events)
}

// waitForEvent blocks on the MIDI channel from inside a tea.Cmd. Bubble Tea
// runs Cmds on their own goroutines, which is how we turn an external channel
// into the message stream Update consumes. Each received event re-arms the
// command (see Update) so we keep listening for the next one.
func waitForEvent(ch <-chan midi.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return sourceClosedMsg{}
		}
		return eventMsg(ev)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "right":
			m.key.Tonic, m.autoKey = (m.key.Tonic+1)%12, false
		case "left":
			m.key.Tonic, m.autoKey = (m.key.Tonic+11)%12, false
		case "m":
			// cycle major → natural → harmonic → melodic minor (manual override)
			m.key.Mode, m.autoKey = m.key.Mode.Next(), false
		case "r":
			// jump to the relative key (same notes, move the tonic):
			// major → its relative natural minor, any minor → relative major.
			if m.key.Mode == theory.Major {
				m.key.Tonic, m.key.Mode = (m.key.Tonic+9)%12, theory.NaturalMinor
			} else {
				m.key.Tonic, m.key.Mode = (m.key.Tonic+3)%12, theory.Major
			}
			m.autoKey = false
		case "a":
			m.autoKey = !m.autoKey
			if m.autoKey {
				if k, conf, ok := theory.DetectKey(m.recentPCs); ok {
					m.key, m.conf = k, conf
				}
			}
		case "e":
			m.splitMelody = !m.splitMelody
			m.recompute()
		}
	case eventMsg:
		ev := midi.Event(msg)
		m.apply(ev)
		_ = m.fwd.Forward(ev) // phase-1 seam: NopForwarder, wired for the mesh later
		return m, waitForEvent(m.events)
	case sourceClosedMsg:
		m.closed = true
	}
	return m, nil
}

// apply folds one event into the view state.
func (m *Model) apply(ev midi.Event) {
	switch {
	case ev.Kind == midi.NoteOn && ev.Data2 > 0:
		m.held[ev.Data1] = true
		m.recentPCs = append(m.recentPCs, ev.Data1%12)
		if len(m.recentPCs) > detectWindow {
			m.recentPCs = m.recentPCs[len(m.recentPCs)-detectWindow:]
		}
		if m.autoKey {
			if k, conf, ok := theory.DetectKey(m.recentPCs); ok {
				m.key, m.conf = k, conf
			}
		}
	case ev.Kind == midi.NoteOff, ev.Kind == midi.NoteOn && ev.Data2 == 0:
		// A NoteOn with velocity 0 is the running-status convention for NoteOff.
		delete(m.held, ev.Data1)
	case ev.IsPedal():
		m.pedal = ev.PedalDown()
	}
	m.recent = append(m.recent, ev)
	if len(m.recent) > maxRecent {
		m.recent = m.recent[len(m.recent)-maxRecent:]
	}
	m.recompute()
}

// recompute derives the chord state from the held notes. It tracks the previous
// recognized chord (persisting across key releases) so the view can confirm
// when the chord just played fulfilled a suggestion from the one before it.
func (m *Model) recompute() {
	notes := sortedHeld(m.held)
	core, melody := notes, []uint8(nil)
	if m.splitMelody {
		core, melody = theory.SplitMelody(notes, theory.DefaultMelodyGap)
	}
	chord, ok := theory.Identify(core)
	if ok {
		if m.lastOK && chord.String() != m.lastChord.String() {
			m.prevChord, m.prevOK = m.lastChord, true
		}
		m.lastChord, m.lastOK = chord, true
	}
	m.core, m.melody, m.chord, m.chordOK = core, melody, chord, ok
}
