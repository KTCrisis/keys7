package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"keys7/internal/midi"
	"keys7/internal/theory"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	chordStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("123"))
	noteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("keys7"))
	b.WriteString(dimStyle.Render("  ·  MIDI capture (phase 1)"))
	b.WriteString("\n\n")

	status := okStyle.Render("listening")
	if m.closed {
		status = warnStyle.Render("source closed")
	}
	b.WriteString(labelStyle.Render("source ") + m.sourceKind + "   ")
	if m.port != "" {
		b.WriteString(labelStyle.Render("port ") + m.port + "   ")
	}
	b.WriteString(labelStyle.Render("status ") + status + "\n\n")

	notes := sortedHeld(m.held)

	b.WriteString(labelStyle.Render("held  "))
	if len(notes) == 0 {
		b.WriteString(dimStyle.Render("—"))
	} else {
		b.WriteString(noteStyle.Render(strings.Join(noteNames(notes), " ")))
	}
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("chord "))
	if c, ok := theory.Identify(notes); ok {
		b.WriteString(chordStyle.Render(c.String()))
	} else {
		b.WriteString(dimStyle.Render("—"))
	}
	b.WriteString("\n")

	pedal := dimStyle.Render("off")
	if m.pedal {
		pedal = okStyle.Render("DOWN")
	}
	b.WriteString(labelStyle.Render("pedal ") + pedal + "\n\n")

	b.WriteString(labelStyle.Render("recent") + "\n")
	for i := len(m.recent) - 1; i >= 0; i-- {
		b.WriteString("  " + formatEvent(m.recent[i]) + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("q quit"))
	return b.String()
}

func sortedHeld(held map[uint8]bool) []uint8 {
	notes := make([]uint8, 0, len(held))
	for n := range held {
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i] < notes[j] })
	return notes
}

func noteNames(notes []uint8) []string {
	names := make([]string, len(notes))
	for i, n := range notes {
		names[i] = midi.NoteName(n)
	}
	return names
}

func formatEvent(ev midi.Event) string {
	switch {
	case ev.Kind == midi.NoteOn && ev.Data2 > 0:
		return fmt.Sprintf("on   %-4s vel %d", midi.NoteName(ev.Data1), ev.Data2)
	case ev.Kind == midi.NoteOff, ev.Kind == midi.NoteOn && ev.Data2 == 0:
		return fmt.Sprintf("off  %-4s", midi.NoteName(ev.Data1))
	case ev.IsPedal():
		if ev.PedalDown() {
			return "pedal down"
		}
		return "pedal up"
	case ev.Kind == midi.ControlChange:
		return fmt.Sprintf("cc   %d = %d", ev.Data1, ev.Data2)
	}
	return "?"
}
