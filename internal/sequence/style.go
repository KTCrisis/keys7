package sequence

import (
	"math/rand"
	"sort"
	"time"
)

// A Style is a playing feel: a small bundle of humanisation, articulation and
// pedalling applied to a compiled sequence so play7 doesn't sound mechanical.
// straight is the identity (deterministic, byte-for-byte the parsed sequence);
// the others loosen timing, spread chords, bend durations and add sustain in a
// way that suits a register. The randomness is injected (a *rand.Rand) so a
// given seed reproduces a take exactly.
type Style struct {
	Name         string
	TimingJitter time.Duration // max absolute onset shift per chord
	ChordSpread  time.Duration // max roll across a chord's notes (low to high)
	Overlap      float64       // duration multiplier: >1 legato, <1 staccato
	VelJitter    int           // max absolute random velocity change
	VelBias      int           // constant velocity offset (softer / louder)
	AutoPedal    bool          // sustain per chord (skipped if the seq already pedals)
}

var styles = map[string]Style{
	"straight":   {Name: "straight", Overlap: 1},
	"ambient":    {Name: "ambient", TimingJitter: 12 * time.Millisecond, ChordSpread: 18 * time.Millisecond, Overlap: 1.18, VelJitter: 6, VelBias: -8, AutoPedal: true},
	"orchestral": {Name: "orchestral", TimingJitter: 18 * time.Millisecond, ChordSpread: 10 * time.Millisecond, Overlap: 1.10, VelJitter: 10, VelBias: 0, AutoPedal: true},
	"darksynth":  {Name: "darksynth", TimingJitter: 3 * time.Millisecond, ChordSpread: 0, Overlap: 0.85, VelJitter: 4, VelBias: 4, AutoPedal: false},
}

// StyleByName looks up a named style.
func StyleByName(name string) (Style, bool) { s, ok := styles[name]; return s, ok }

// StyleNames returns the available style names, sorted.
func StyleNames() []string {
	names := make([]string, 0, len(styles))
	for n := range styles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// chordEpsilon groups note-ons struck within this window into one chord.
const chordEpsilon = 10 * time.Millisecond

// Apply returns a humanised copy of seq in this style. straight returns the
// sequence unchanged. The transform pairs each note-on with its release, groups
// simultaneous onsets into chords, then per chord: shifts the onset (timing
// jitter), rolls the notes low-to-high (chord spread), scales the held duration
// (overlap) and perturbs velocity. With AutoPedal and no existing pedal events,
// it sustains each chord until the next.
func (st Style) Apply(seq Sequence, rng *rand.Rand) Sequence {
	if st.Name == "" || st.Name == "straight" {
		return seq
	}

	type note struct {
		on, off    time.Duration
		num, vel   uint8
	}
	var notes []note
	var ctrls []Event
	used := make([]bool, len(seq.Events))
	for i, e := range seq.Events {
		if e.Ctrl {
			ctrls = append(ctrls, e)
			continue
		}
		if !e.On || used[i] {
			continue
		}
		off := e.At
		for j := i + 1; j < len(seq.Events); j++ {
			ej := seq.Events[j]
			if !ej.Ctrl && !ej.On && ej.Note == e.Note && !used[j] {
				off, used[j] = ej.At, true
				break
			}
		}
		notes = append(notes, note{on: e.At, off: off, num: e.Note, vel: e.Vel})
	}
	sort.SliceStable(notes, func(i, j int) bool { return notes[i].on < notes[j].on })

	// Group successive notes whose onsets fall within chordEpsilon into chords.
	var groups [][]int
	for i := range notes {
		if len(groups) > 0 {
			g := groups[len(groups)-1]
			if notes[i].on-notes[g[0]].on <= chordEpsilon {
				groups[len(groups)-1] = append(g, i)
				continue
			}
		}
		groups = append(groups, []int{i})
	}

	var out []Event
	chordOnsets := make([]time.Duration, len(groups)) // humanised onset of each chord
	for gi, g := range groups {
		base := notes[g[0]].on
		shift := randDur(rng, st.TimingJitter)
		onset := base + shift
		if onset < 0 {
			onset = 0
		}
		chordOnsets[gi] = onset

		// roll low-to-high across the chord
		idx := append([]int(nil), g...)
		sort.SliceStable(idx, func(a, b int) bool { return notes[idx[a]].num < notes[idx[b]].num })
		for k, ni := range idx {
			roll := time.Duration(0)
			if st.ChordSpread > 0 && len(idx) > 1 {
				roll = time.Duration(int64(st.ChordSpread) * int64(k) / int64(len(idx)-1))
			}
			n := notes[ni]
			start := onset + roll
			dur := time.Duration(float64(n.off-n.on) * st.Overlap)
			if dur <= 0 {
				dur = n.off - n.on
			}
			out = append(out,
				Event{At: start, On: true, Note: n.num, Vel: clampVel(int(n.vel) + st.VelBias + randInt(rng, st.VelJitter))},
				Event{At: start + dur, On: false, Note: n.num},
			)
		}
	}

	// Auto-pedal: sustain each chord until the next onset, unless the sequence
	// already carries explicit pedal events (explicit playing wins over style).
	if st.AutoPedal && len(ctrls) == 0 {
		for gi, onset := range chordOnsets {
			down := onset
			if down < 0 {
				down = 0
			}
			// lift just before the next chord, so the change is clean
			if gi > 0 {
				prevUp := onset - time.Millisecond
				if prevUp < 0 {
					prevUp = 0
				}
				out = append(out, Event{At: prevUp, Ctrl: true, Note: sustainController, Vel: 0})
			}
			out = append(out, Event{At: down, Ctrl: true, Note: sustainController, Vel: 127})
		}
	}

	out = append(out, ctrls...) // carry explicit control events through unchanged
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].At != out[j].At {
			return out[i].At < out[j].At
		}
		return eventOrder(out[i]) < eventOrder(out[j])
	})
	return Sequence{Tempo: seq.Tempo, Channel: seq.Channel, Events: out}
}

func randDur(rng *rand.Rand, max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rng.Int63n(2*int64(max)+1)) - max
}

func randInt(rng *rand.Rand, max int) int {
	if max <= 0 {
		return 0
	}
	return rng.Intn(2*max+1) - max
}

func clampVel(v int) uint8 {
	switch {
	case v < 1:
		return 1
	case v > 127:
		return 127
	default:
		return uint8(v)
	}
}
