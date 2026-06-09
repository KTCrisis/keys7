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
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	panelStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
	panelTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("245"))
	labelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	noteStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	chordStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("123"))
	chordBigStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("159"))
	highlightStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	okStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	dimStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m Model) View() string {
	allNotes := sortedHeld(m.held)

	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		panelStyle.Width(32).MarginRight(2).Render(m.playingPanel(allNotes)),
		panelStyle.Width(32).Render(m.keyPanel()),
	)
	harmony := panelStyle.Width(68).Render(m.harmonyPanel())

	footer := dimStyle.Render("q quit · a auto · m mode · r relative · e split · n notation · ←/→ tonic")

	return lipgloss.JoinVertical(lipgloss.Left,
		m.header(), "",
		top, "",
		harmony, "",
		m.recentLine(), "",
		footer,
	)
}

func (m Model) header() string {
	status := okStyle.Render("listening")
	if m.closed {
		status = warnStyle.Render("source closed")
	}
	src := m.sourceKind
	if m.port != "" {
		src += " " + m.port
	}
	return titleStyle.Render("keys7") + dimStyle.Render("  ·  live harmony") +
		"      " + labelStyle.Render(src) + dimStyle.Render(" · ") + status
}

func (m Model) playingPanel(allNotes []uint8) string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("playing") + "\n")

	pcs := distinctPCs(m.core)
	chordStr := dimStyle.Render("—")
	switch {
	case m.chordOK:
		other := theory.ActiveNotation().Other()
		chordStr = chordBigStyle.Render(m.chord.String()) + dimStyle.Render("  "+m.chord.StringIn(other))
		if label, ok := m.fulfilledSuggestion(); ok {
			chordStr += highlightStyle.Render("  ✓ " + label)
		}
	case len(pcs) == 2:
		chordStr = noteStyle.Render(theory.IntervalName((pcs[1]-pcs[0]+12)%12))
		if impl := theory.DyadImplications(pcs[0], pcs[1], &m.key); len(impl) > 0 {
			chordStr += dimStyle.Render(" ⇒ ") + chordStyle.Render(symbolsJoin(impl, " "))
		}
	}
	b.WriteString(labelStyle.Render("chord  ") + chordStr + "\n")
	b.WriteString(labelStyle.Render("held   ") + valueOrDash(noteNames(allNotes)) + "\n")
	b.WriteString(labelStyle.Render("melody ") + valueOrDash(noteNames(m.melody)) + "\n")

	pedal := dimStyle.Render("off")
	if m.pedal {
		pedal = okStyle.Render("down")
	}
	b.WriteString(labelStyle.Render("pedal  ") + pedal)
	return b.String()
}

func (m Model) keyPanel() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("key") + "\n")
	other := theory.ActiveNotation().Other()
	b.WriteString(chordBigStyle.Render(m.key.String()) +
		dimStyle.Render("  "+theory.PitchClassNameIn(m.key.Tonic, other)) + "\n")
	if m.autoKey {
		b.WriteString(okStyle.Render(fmt.Sprintf("detected · %.0f%%", m.conf*100)) + "\n")
	} else {
		b.WriteString(dimStyle.Render("manual") + "\n")
	}
	b.WriteString(onOff("auto", m.autoKey) + "  " + onOff("split", m.splitMelody))
	return b.String()
}

func (m Model) harmonyPanel() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("harmony") + "\n")

	curRoot, playing := int(-1), ""
	if m.chordOK {
		curRoot, playing = int(m.chord.Root), m.chord.String()
	}

	// diatonic palette, current degree highlighted
	curDeg := 0
	if m.chordOK {
		if dc, ok := theory.DegreeOf(m.key, m.chord); ok {
			curDeg = dc.Degree
		}
	}
	cells := make([]string, 0, 7)
	for _, dc := range theory.DiatonicTriads(m.key) {
		if dc.Degree == curDeg {
			cells = append(cells, highlightStyle.Render(dc.Roman))
		} else {
			cells = append(cells, noteStyle.Render(dc.Roman))
		}
	}
	b.WriteString(labelStyle.Render("in key  ") + strings.Join(cells, "  ") + "\n")

	// next chords — highlight one if the player is currently on it
	b.WriteString(labelStyle.Render("next    "))
	switch {
	case !m.chordOK:
		b.WriteString(dimStyle.Render("— play a chord"))
	default:
		if ss, ok := theory.Suggest(m.key, m.chord); ok && len(ss) > 0 {
			parts := make([]string, len(ss))
			for i, s := range ss {
				parts[i] = tagChord(s.Chord.Chord.String(), s.Label, s.Chord.Chord.String() == playing)
			}
			b.WriteString(strings.Join(parts, dimStyle.Render("  ·  ")))
		} else {
			b.WriteString(dimStyle.Render("— out of key"))
		}
	}
	b.WriteString("\n")

	// neighbouring keys — highlight one whose tonic chord is being played
	nks := theory.NeighborKeys(m.key)
	near := make([]string, len(nks))
	for i, nk := range nks {
		sk := shortKey(nk.Key)
		style := chordStyle
		if int(nk.Key.Tonic) == curRoot {
			style = highlightStyle
		}
		near[i] = dimStyle.Render(nk.Relation[:3]+" ") + style.Render(sk)
	}
	b.WriteString(labelStyle.Render("near    ") + strings.Join(near, dimStyle.Render(" · ")) + "\n")

	// secondary dominants — highlight the one currently played
	sds := theory.SecondaryDominants(m.key)
	pass := make([]string, len(sds))
	for i, sd := range sds {
		pass[i] = tagChord(sd.Chord.String(), sd.Label, sd.Chord.String() == playing)
	}
	b.WriteString(labelStyle.Render("pass    ") + strings.Join(pass, dimStyle.Render(" · ")))
	return b.String()
}

// tagChord renders "symbol label", in pink when the player is on it.
func tagChord(symbol, label string, active bool) string {
	if active {
		return highlightStyle.Render(symbol) + highlightStyle.Render(" "+label)
	}
	return chordStyle.Render(symbol) + dimStyle.Render(" "+label)
}

// fulfilledSuggestion reports whether the current chord is one the previous
// chord suggested, returning the cadence label (e.g. "authentic cadence").
func (m Model) fulfilledSuggestion() (string, bool) {
	if !m.chordOK || !m.prevOK {
		return "", false
	}
	ss, ok := theory.Suggest(m.key, m.prevChord)
	if !ok {
		return "", false
	}
	cur := m.chord.String()
	for _, s := range ss {
		if s.Chord.Chord.String() == cur {
			return s.Label, true
		}
	}
	return "", false
}

func (m Model) recentLine() string {
	if len(m.recent) == 0 {
		return labelStyle.Render("recent  ") + dimStyle.Render("—")
	}
	const show = 8
	start := len(m.recent) - show
	if start < 0 {
		start = 0
	}
	parts := make([]string, 0, show)
	for i := len(m.recent) - 1; i >= start; i-- {
		parts = append(parts, formatEventShort(m.recent[i]))
	}
	return labelStyle.Render("recent  ") + dimStyle.Render(strings.Join(parts, "  "))
}

// --- helpers ---

func valueOrDash(names []string) string {
	if len(names) == 0 {
		return dimStyle.Render("—")
	}
	return noteStyle.Render(strings.Join(names, " "))
}

func shortKey(k theory.Key) string {
	s := theory.PitchClassName(k.Tonic)
	if k.Mode.IsMinor() {
		s += "m"
	}
	return s
}

func onOff(label string, on bool) string {
	if on {
		return okStyle.Render(label + " on")
	}
	return dimStyle.Render(label + " off")
}

func distinctPCs(notes []uint8) []uint8 {
	seen := map[uint8]bool{}
	var pcs []uint8
	for _, n := range notes {
		if pc := n % 12; !seen[pc] {
			seen[pc] = true
			pcs = append(pcs, pc)
		}
	}
	sort.Slice(pcs, func(i, j int) bool { return pcs[i] < pcs[j] })
	return pcs
}

func symbolsJoin(cs []theory.Chord, sep string) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = c.String()
	}
	return strings.Join(parts, sep)
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
		names[i] = theory.NoteName(n)
	}
	return names
}

func formatEventShort(ev midi.Event) string {
	switch {
	case ev.Kind == midi.NoteOn && ev.Data2 > 0:
		return fmt.Sprintf("%s↓%d", theory.NoteName(ev.Data1), ev.Data2)
	case ev.Kind == midi.NoteOff, ev.Kind == midi.NoteOn && ev.Data2 == 0:
		return theory.NoteName(ev.Data1) + "↑"
	case ev.IsPedal():
		if ev.PedalDown() {
			return "ped↓"
		}
		return "ped↑"
	case ev.Kind == midi.ControlChange:
		return fmt.Sprintf("cc%d=%d", ev.Data1, ev.Data2)
	}
	return "?"
}
