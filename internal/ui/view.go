package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

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
	full, half := m.layout()

	top := lipgloss.JoinHorizontal(
		lipgloss.Top,
		panelStyle.Width(half).MarginRight(2).Render(m.playingPanel(m.sounding())),
		panelStyle.Width(half).Render(m.keyPanel()),
	)
	harmony := panelStyle.Width(full).Render(m.harmonyPanel())

	parts := []string{m.header(), "", top, "", harmony, ""}
	if m.replyPath != "" {
		parts = append(parts, panelStyle.Width(full).Render(m.replyPanel()), "")
	}
	parts = append(parts, m.recentLine(), "", m.footer())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// layout derives panel content widths from the terminal size, falling back to
// the historical fixed layout before the first WindowSizeMsg arrives. full is
// the wide panels (harmony, reply); half is each of the two top panels. The
// arithmetic keeps the two top panels aligned under the harmony panel:
// (half+4)*2 + 2-gap == full+4, i.e. half = (full-6)/2.
func (m Model) layout() (full, half int) {
	w := m.width
	if w <= 0 {
		w = 74 // pre-resize default: reproduces the old 68/31 layout
	}
	full = w - 6 // borders + padding + a small right margin
	if full < 60 {
		full = 60
	}
	if full > 120 {
		full = 120 // don't sprawl on ultra-wide terminals
	}
	half = (full - 6) / 2
	return full, half
}

// header is the logo monogram (a boxed 7, in the flux7 line-art spirit) beside
// the title and a live status line.
func (m Model) header() string {
	logo := lipgloss.JoinVertical(lipgloss.Left,
		dimStyle.Render("╭───╮"),
		dimStyle.Render("│  ")+highlightStyle.Render("7")+dimStyle.Render("│"),
		dimStyle.Render("╰───╯"),
	)
	right := lipgloss.JoinVertical(lipgloss.Left,
		"",
		titleStyle.Render("keys7")+"   "+m.statusLine(),
		dimStyle.Render("live harmony"),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, logo, "  ", right)
}

// statusLine reports the source, listening state, MIDI liveness, declared
// texture and the last cue — everything that changes during a session.
func (m Model) statusLine() string {
	status := okStyle.Render("listening")
	if m.closed {
		status = warnStyle.Render("source closed")
	}
	src := m.sourceKind
	if m.port != "" {
		src += " " + m.port
	}
	parts := []string{labelStyle.Render(src), status, m.midiDot()}
	if m.texture != TextureFree {
		parts = append(parts, warnStyle.Render(m.texture.String()))
	}
	if !m.cuedAt.IsZero() {
		parts = append(parts, highlightStyle.Render("cue "+m.lastCue.String()+" ✓ "+m.cuedAt.Format("15:04:05")))
	}
	return strings.Join(parts, dimStyle.Render(" · "))
}

// midiDot answers "is the piano being heard?" at a glance: a lit dot the instant
// a note lands, then a quiet seconds-since counter (refreshed by the heartbeat).
func (m Model) midiDot() string {
	if m.lastNoteAt.IsZero() {
		return dimStyle.Render("○ idle")
	}
	if time.Since(m.lastNoteAt) < 300*time.Millisecond {
		return okStyle.Render("● live")
	}
	return dimStyle.Render(fmt.Sprintf("○ %.0fs", time.Since(m.lastNoteAt).Seconds()))
}

// footer groups the key bindings by concern (scale / play / session) so the
// long flat strip reads as three short, scannable rows.
func (m Model) footer() string {
	hint := func(key, desc string) string {
		return highlightStyle.Render(key) + " " + dimStyle.Render(desc)
	}
	join := func(hs ...string) string { return strings.Join(hs, dimStyle.Render(" · ")) }
	rows := []string{
		labelStyle.Render("gamme ") + join(hint("←/→", "tonic"), hint("m", "mode"), hint("r", "relative")),
		labelStyle.Render("jeu   ") + join(hint("e", "split"), hint("t", "texture"), hint("n", "notation")),
		labelStyle.Render("sess  ") + join(hint("a", "auto"), hint("d", "drone"), hint("x", "reset"), hint("q", "quit")),
		// the signal bar: double-tap one of the four lowest keys to cue the assistant
		labelStyle.Render("cue   ") + join(
			hint(theory.NoteName(21), "turn"),
			hint(theory.NoteName(22), "replay"),
			hint(theory.NoteName(23), "transpose"),
			hint(theory.NoteName(24), "harmonise"),
		),
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// replyPanel shows the assistant's reply file content (the journal's return
// channel), with a freshness badge so a just-arrived answer is obvious.
func (m Model) replyPanel() string {
	title := panelTitleStyle.Render("assistant")
	if !m.replyAt.IsZero() {
		stamp := m.replyAt.Format("15:04:05")
		if time.Since(m.replyAt) < 5*time.Second {
			title += "  " + highlightStyle.Render("• new "+stamp)
		} else {
			title += dimStyle.Render("  · "+stamp)
		}
	}
	body := dimStyle.Render("waiting for a reply…")
	if m.reply != "" {
		full, _ := m.layout()
		bw := full - 4
		if bw < 20 {
			bw = 20
		}
		body = noteStyle.Width(bw).Render(m.reply)
	}
	return title + "\n" + body
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
		chordStr = noteStyle.Render(theory.IntervalName((pcs[1] - pcs[0] + 12) % 12))
		if impl := theory.DyadImplications(pcs[0], pcs[1], &m.key); len(impl) > 0 {
			chordStr += dimStyle.Render(" ⇒ ") + chordStyle.Render(symbolsJoin(impl, " "))
		}
	}
	b.WriteString(labelStyle.Render("chord  ") + chordStr + "\n")
	b.WriteString(labelStyle.Render("notes  ") + m.styledNotes(allNotes) + "\n")
	b.WriteString(labelStyle.Render("melody ") + valueOrDash(noteNames(m.melody)) + "\n")

	pedal := dimStyle.Render("off")
	if m.pedal {
		pedal = okStyle.Render("down")
	}
	b.WriteString(labelStyle.Render("pedal  ") + pedal)
	return b.String()
}

// styledNotes renders sounding notes, marking out-of-key ones (accidentals) in
// the warn colour — so a note outside the chosen scale stands out at once.
func (m Model) styledNotes(notes []uint8) string {
	if len(notes) == 0 {
		return dimStyle.Render("—")
	}
	parts := make([]string, len(notes))
	for i, n := range notes {
		name := theory.NoteName(n)
		if m.key.InScale(n) {
			parts[i] = noteStyle.Render(name)
		} else {
			parts[i] = warnStyle.Render(name)
		}
	}
	return strings.Join(parts, " ")
}

func (m Model) keyPanel() string {
	var b strings.Builder
	b.WriteString(panelTitleStyle.Render("key") + "\n")
	other := theory.ActiveNotation().Other()
	b.WriteString(chordBigStyle.Render(m.key.String()) +
		dimStyle.Render("  "+theory.PitchClassNameIn(m.key.Tonic, other)) + "\n")
	switch m.keySrc {
	case KeyAuto:
		b.WriteString(okStyle.Render(fmt.Sprintf("auto · %.0f%%", m.conf*100)) + "\n")
	case KeyDrone:
		b.WriteString(okStyle.Render("drone · bass-pinned") + "\n")
	default:
		b.WriteString(dimStyle.Render("manual") + "\n")
	}
	b.WriteString(onOff("auto", m.keySrc == KeyAuto) + " " + onOff("drone", m.keySrc == KeyDrone) + " " + onOff("split", m.splitMelody) + "\n")
	b.WriteString(labelStyle.Render("scale  ") + m.scaleLine())
	return b.String()
}

// scaleLine shows the seven notes of the chosen scale, lighting those currently
// sounding — the scale you pick with ←/→, m, r, made visible.
func (m Model) scaleLine() string {
	sounding := map[uint8]bool{}
	for _, n := range m.sounding() {
		sounding[n%12] = true
	}
	names := make([]string, 0, 7)
	for _, pc := range m.key.ScalePCs() {
		name := theory.PitchClassName(pc)
		if sounding[pc] {
			names = append(names, highlightStyle.Render(name))
		} else {
			names = append(names, noteStyle.Render(name))
		}
	}
	return strings.Join(names, " ")
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
