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
	held        map[uint8]bool
	pedal       bool
	recent      []midi.Event
	closed      bool
}

// New builds the model. `events` is the source's channel; `fwd` is the mesh
// seam (a NopForwarder for now); `key` is the fixed key for cadence hints.
func New(sourceKind, port string, key theory.Key, events <-chan midi.Event, fwd mesh.Forwarder) Model {
	return Model{
		sourceKind: sourceKind,
		port:       port,
		key:         key,
		splitMelody: true,
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
			m.key.Tonic = (m.key.Tonic + 1) % 12
		case "left":
			m.key.Tonic = (m.key.Tonic + 11) % 12
		case "m":
			if m.key.Mode == theory.Major {
				m.key.Mode = theory.Minor
			} else {
				m.key.Mode = theory.Major
			}
		case "e":
			m.splitMelody = !m.splitMelody
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
}
