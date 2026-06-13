package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/session"
	"keys7/internal/theory"
)

// eventMsg carries a MIDI event into the Bubble Tea update loop.
type eventMsg midi.Event

// sourceClosedMsg signals the MIDI source channel has closed.
type sourceClosedMsg struct{}

// tickMsg drives a slow heartbeat so time-based UI (the MIDI liveness dot, the
// reply "new" flash) keeps refreshing even when no MIDI or reply arrives.
type tickMsg time.Time

func heartbeat() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

const maxRecent = 16

// Model is the phase-1 UI state: which source we're on, the currently held
// notes, pedal state, and a short ring buffer of recent events.
type Model struct {
	sourceKind string
	port       string
	events     <-chan midi.Event
	fwd        mesh.Forwarder
	sink       session.Sink // harmonic-event log for the AI layer
	lastLogKey string

	key         theory.Key
	splitMelody bool // peel isolated top notes as melody (vs fold into the chord)
	keySrc      KeySource
	conf        float64
	recentPCs   []uint8 // sliding window of recent note-on pitch classes
	held        map[uint8]bool
	sustained   map[uint8]bool // released while the pedal was down — still sounding
	pedal       bool
	recent      []midi.Event
	closed      bool
	cue         cueDetector
	cuedAt      time.Time // last completed cue, for the header badge
	replyPath   string    // assistant reply file (--reply); empty = no panel
	reply       string    // last content read from it
	texture     Texture   // declared playing texture (cycled with `t`)

	// ux state: terminal size (responsive layout), last-seen note (liveness
	// dot), and when the reply last changed (the "new" flash).
	width, height int
	lastNoteAt    time.Time
	replyAt       time.Time

	// derived chord state, recomputed when the held notes change
	core, melody []uint8
	chord        theory.Chord
	chordOK      bool
	prevChord    theory.Chord // the recognized chord before the current one
	prevOK       bool
	lastChord    theory.Chord // most recent recognized chord (persists through releases)
	lastOK       bool
}

// detectWindow is how many recent note-ons feed key detection.
const detectWindow = 32

// Texture is the player's declared playing mode — intent as a fact in the
// journal, a strong prior for the reader's melody/harmony segmentation
// (real-time inference can't tell an arpeggio from a line; the player can).
type Texture int

const (
	TextureFree     Texture = iota // default: the reader infers
	TextureBlock                   // chords are plaqués
	TextureArpeggio                // chords are broken/arpeggiated
)

func (t Texture) String() string {
	switch t {
	case TextureBlock:
		return "block"
	case TextureArpeggio:
		return "arpeggio"
	default:
		return "free"
	}
}

// Next cycles free → block → arpeggio.
func (t Texture) Next() Texture { return (t + 1) % 3 }

// KeySource is how the active key is chosen.
type KeySource int

const (
	KeyManual KeySource = iota // set by the user (←/→, m, r)
	KeyAuto                    // inferred from playing (Krumhansl-Schmuckler)
	KeyDrone                   // tonic pinned to the bass, mode from its third
)

// New builds the model. `events` is the source's channel; `fwd` is the mesh
// seam (a NopForwarder for now); `key` is the starting key, and `autoKey`
// enables inferring it from what's played. `replyPath` is the assistant reply
// file to poll ("" disables the panel).
func New(sourceKind, port string, key theory.Key, keySrc KeySource, events <-chan midi.Event, fwd mesh.Forwarder, sink session.Sink, replyPath string) Model {
	return Model{
		sourceKind:  sourceKind,
		port:        port,
		key:         key,
		splitMelody: true,
		keySrc:      keySrc,
		events:      events,
		fwd:         fwd,
		sink:        sink,
		replyPath:   replyPath,
		held:        make(map[uint8]bool),
		sustained:   make(map[uint8]bool),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{waitForEvent(m.events), heartbeat()}
	if m.replyPath != "" {
		cmds = append(cmds, replyTick(m.replyPath))
	}
	return tea.Batch(cmds...)
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
			m.key.Tonic, m.keySrc = (m.key.Tonic+1)%12, KeyManual
		case "left":
			m.key.Tonic, m.keySrc = (m.key.Tonic+11)%12, KeyManual
		case "m":
			// cycle major → natural → harmonic → melodic minor (manual override)
			m.key.Mode, m.keySrc = m.key.Mode.Next(), KeyManual
		case "r":
			// jump to the relative key (same notes, move the tonic):
			// major → its relative natural minor, any minor → relative major.
			if m.key.Mode == theory.Major {
				m.key.Tonic, m.key.Mode = (m.key.Tonic+9)%12, theory.NaturalMinor
			} else {
				m.key.Tonic, m.key.Mode = (m.key.Tonic+3)%12, theory.Major
			}
			m.keySrc = KeyManual
		case "a":
			if m.keySrc == KeyAuto {
				m.keySrc = KeyManual
			} else {
				m.keySrc = KeyAuto
			}
			m.trackKey()
		case "d":
			if m.keySrc == KeyDrone {
				m.keySrc = KeyManual
			} else {
				m.keySrc = KeyDrone
			}
			m.trackKey()
		case "e":
			m.splitMelody = !m.splitMelody
			m.recompute()
		case "n":
			theory.ToggleNotation() // display-only; chord state is unchanged
		case "t":
			m.texture = m.texture.Next()
			if m.sink != nil {
				m.sink.Emit(session.HarmonicEvent{
					Time: session.Stamp(time.Now()),
					Kind: "texture",
					Mode: m.texture.String(),
				})
			}
		case "x":
			m.reset() // forget everything played, keep settings
		}
		m.logKeyIfChanged()
	case eventMsg:
		ev := midi.Event(msg)
		if ev.Kind == midi.NoteOn && ev.Data2 > 0 {
			m.lastNoteAt = time.Now() // reception time, for the liveness dot
		}
		m.apply(ev)
		_ = m.fwd.Forward(ev) // phase-1 seam: NopForwarder, wired for the mesh later
		return m, waitForEvent(m.events)
	case replyMsg:
		s := string(msg)
		if s != "" && s != m.reply {
			m.replyAt = time.Now() // a genuinely new reply — flash it
		}
		m.reply = s
		return m, replyTick(m.replyPath)
	case tickMsg:
		return m, heartbeat() // re-arm; the redraw refreshes time-based UI
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case sourceClosedMsg:
		m.closed = true
	}
	return m, nil
}

// apply folds one event into the view state.
func (m *Model) apply(ev midi.Event) {
	m.logRaw(ev)
	switch {
	case ev.Kind == midi.NoteOn && ev.Data2 > 0:
		if ev.Data1 == cueNote && m.cue.Tap(ev.Timestamp) {
			m.cuedAt = ev.Timestamp
			if m.sink != nil {
				m.sink.Emit(session.HarmonicEvent{Time: session.Stamp(time.Now()), Kind: "cue"})
			}
		}
		m.held[ev.Data1] = true
		delete(m.sustained, ev.Data1) // re-struck: physically held again
		m.recentPCs = append(m.recentPCs, ev.Data1%12)
		if len(m.recentPCs) > detectWindow {
			m.recentPCs = m.recentPCs[len(m.recentPCs)-detectWindow:]
		}
		m.trackKey()
	case ev.Kind == midi.NoteOff, ev.Kind == midi.NoteOn && ev.Data2 == 0:
		// A NoteOn with velocity 0 is the running-status convention for NoteOff.
		delete(m.held, ev.Data1)
		if m.pedal {
			// The damper is up: the string keeps sounding. Keep the note in the
			// harmonic picture until the pedal releases it — this is what lets a
			// pedaled chord stay under a melody line played hands-off.
			m.sustained[ev.Data1] = true
		}
	case ev.IsPedal():
		wasDown := m.pedal
		m.pedal = ev.PedalDown()
		if wasDown && !m.pedal {
			m.sustained = make(map[uint8]bool) // dampers fall: only held keys sound
		}
	}
	m.recent = append(m.recent, ev)
	if len(m.recent) > maxRecent {
		m.recent = m.recent[len(m.recent)-maxRecent:]
	}
	// Low-melody detection is temporal, so it compares against the state from
	// *before* this event: a note landing well below an already-sounding chord
	// is a moving line; a bass struck together with its chord is the chord's.
	wasChord, prevBottom := m.chordOK, uint8(0)
	if len(m.core) > 0 {
		prevBottom = m.core[0]
	}
	m.recompute()
	m.logMelodyOnset(ev, wasChord, prevBottom)
	m.logKeyIfChanged()
}

// logRaw journals the faithful capture layer: every note attack/release and
// pedal move, milliseconds and velocity included. This is what a reader
// segments with hindsight — arpeggio vs line is undecidable in real time but
// easy once releases and accumulation are visible. Cue-note taps stay out:
// they are signalling, not music.
func (m *Model) logRaw(ev midi.Event) {
	if m.sink == nil || (ev.Data1 == cueNote && ev.Kind != midi.ControlChange) {
		return
	}
	switch {
	case ev.Kind == midi.NoteOn && ev.Data2 > 0:
		m.sink.Emit(session.HarmonicEvent{
			Time: session.Stamp(ev.Timestamp), Kind: "note",
			Note: theory.NoteNameIn(ev.Data1, theory.Letters), Midi: ev.Data1,
			Vel: ev.Data2, On: session.Bool(true),
		})
	case ev.Kind == midi.NoteOff, ev.Kind == midi.NoteOn && ev.Data2 == 0:
		m.sink.Emit(session.HarmonicEvent{
			Time: session.Stamp(ev.Timestamp), Kind: "note",
			Note: theory.NoteNameIn(ev.Data1, theory.Letters), Midi: ev.Data1,
			On: session.Bool(false),
		})
	case ev.IsPedal():
		m.sink.Emit(session.HarmonicEvent{
			Time: session.Stamp(ev.Timestamp), Kind: "pedal",
			On: session.Bool(ev.PedalDown()),
		})
	}
}

// logMelodyOnset emits a melody event when the note just struck is melodic —
// either peeled off the top by the split ("high"), or struck well below a
// chord that was already sounding ("low"). The journal's melodic line, where
// chords alone are blind. Uses the event's own timestamp (millisecond
// precision) so the reader can reconstruct rhythm from inter-onset gaps.
//
// The low case deliberately does not touch chord recognition: a planted bass
// (slash chord, root under a right-hand triad) arrives with its chord, not
// under one already heard, so it stays harmonic. The price is one spurious
// melody event when a bass is added late under a held chord — context (a
// single low onset vs a moving line) disambiguates in the journal.
func (m *Model) logMelodyOnset(ev midi.Event, wasChord bool, prevBottom uint8) {
	if m.sink == nil || !m.splitMelody || ev.Kind != midi.NoteOn || ev.Data2 == 0 || ev.Data1 == cueNote {
		return
	}
	reg := ""
	for _, n := range m.melody {
		if n == ev.Data1 {
			reg = "high"
		}
	}
	if reg == "" && wasChord && ev.Data1+theory.DefaultMelodyGap <= prevBottom {
		reg = "low"
	}
	if reg == "" {
		return
	}
	m.sink.Emit(session.HarmonicEvent{
		Time: session.Stamp(ev.Timestamp),
		Kind: "melody",
		Note: theory.NoteNameIn(ev.Data1, theory.Letters),
		Midi: ev.Data1,
		Vel:  ev.Data2,
		Reg:  reg,
	})
}

// reset forgets everything played — held notes, recent events, the key-detection
// window and chord history — while keeping the user's settings (key, notation,
// melody-split, auto-key). A clean slate to start a new idea.
func (m *Model) reset() {
	m.held = make(map[uint8]bool)
	m.sustained = make(map[uint8]bool)
	m.recentPCs = nil
	m.recent = nil
	m.core, m.melody = nil, nil
	m.chord, m.chordOK = theory.Chord{}, false
	m.prevChord, m.prevOK = theory.Chord{}, false
	m.lastChord, m.lastOK = theory.Chord{}, false
	m.conf = 0
	m.pedal = false
	if m.sink != nil {
		m.sink.Emit(session.HarmonicEvent{Time: session.Stamp(time.Now()), Kind: "reset"})
	}
	m.lastLogKey = "" // re-log the key on the next note, to anchor the new segment
}

// sounding merges physically held keys with pedal-sustained notes — the set the
// harmonic analysis should hear, as opposed to the keys under the fingers.
func (m *Model) sounding() []uint8 {
	all := make(map[uint8]bool, len(m.held)+len(m.sustained))
	for n := range m.held {
		all[n] = true
	}
	for n := range m.sustained {
		all[n] = true
	}
	return sortedHeld(all)
}

// recompute derives the chord state from the sounding notes. It tracks the
// previous recognized chord (persisting across key releases) so the view can
// confirm when the chord just played fulfilled a suggestion from the one before.
func (m *Model) recompute() {
	notes := m.sounding()
	core, melody := notes, []uint8(nil)
	if m.splitMelody {
		core, melody = theory.SplitMelody(notes, theory.DefaultMelodyGap)
	}
	chord, ok := theory.Identify(core)
	if ok {
		isNew := !m.lastOK || chord.StringIn(theory.Letters) != m.lastChord.StringIn(theory.Letters)
		if isNew {
			if m.lastOK {
				m.prevChord, m.prevOK = m.lastChord, true
			}
			m.logChord(chord)
		}
		m.lastChord, m.lastOK = chord, true
	}
	m.core, m.melody, m.chord, m.chordOK = core, melody, chord, ok
}

// trackKey updates the active key from playing, per the key source. Auto runs
// Krumhansl-Schmuckler over the recent window; drone pins the tonic to the bass
// (lowest held note) and reads the mode from its third — steady on pedal/modal
// playing, where auto flickers between relative keys. Manual does nothing.
func (m *Model) trackKey() {
	switch m.keySrc {
	case KeyAuto:
		if k, conf, ok := theory.DetectKey(m.recentPCs); ok {
			m.key, m.conf = k, conf
		}
	case KeyDrone:
		if bass, ok := m.bassPC(); ok {
			m.key = theory.Key{Tonic: bass, Mode: theory.ModeOverTonic(m.recentPCs, bass)}
		}
	}
}

// bassPC returns the pitch class of the lowest sounding note (held or
// pedal-sustained) — a pedaled drone keeps anchoring the key in drone mode.
func (m *Model) bassPC() (uint8, bool) {
	lowest, found := uint8(0), false
	for _, set := range []map[uint8]bool{m.held, m.sustained} {
		for n := range set {
			if !found || n < lowest {
				lowest, found = n, true
			}
		}
	}
	return lowest % 12, found
}

// logChord emits a harmonic event for a newly recognized chord, annotated with
// its role in the current key (degree, or secondary dominant / non-diatonic).
func (m *Model) logChord(chord theory.Chord) {
	if m.sink == nil {
		return
	}
	ev := session.HarmonicEvent{
		Time:    session.Stamp(time.Now()),
		Kind:    "chord",
		Chord:   chord.StringIn(theory.Letters),
		Solfege: chord.StringIn(theory.Solfege),
		Key:     m.key.StringIn(theory.Letters),
	}
	if dc, ok := theory.DegreeOf(m.key, chord); ok {
		ev.Roman, ev.Degree = dc.Roman, dc.Degree
	} else {
		ev.Note = "non-diatonic"
		for _, sd := range theory.SecondaryDominants(m.key) {
			if sd.Chord.StringIn(theory.Letters) == chord.StringIn(theory.Letters) {
				ev.Note = "secondary dominant " + sd.Label + " → " + sd.Target.Roman
				break
			}
		}
	}
	m.sink.Emit(ev)
}

// logKeyIfChanged emits a key event when the active key changed since last logged.
func (m *Model) logKeyIfChanged() {
	if m.sink == nil {
		return
	}
	k := m.key.StringIn(theory.Letters)
	if k == m.lastLogKey {
		return
	}
	m.lastLogKey = k
	m.sink.Emit(session.HarmonicEvent{
		Time: session.Stamp(time.Now()),
		Kind: "key",
		Key:  k,
		Conf: m.conf,
	})
}
