// play7 plays a JSON note/chord sequence on a MIDI output device — the output
// twin of keys7 (which listens). Where keys7 is a resident TUI on MIDI in,
// play7 is a silent one-shot: read a sequence, play it, exit.
//
//	play7 --list
//	play7 --port "P-125" sequence.json
//	echo '{"steps":[{"notes":["C4","E4","G4"],"beats":2}]}' | play7 --out=mock
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"keys7/internal/midi"
	"keys7/internal/sequence"
	"keys7/internal/smf"
)

func main() {
	list := flag.Bool("list", false, "list MIDI output devices and exit")
	port := flag.String("port", "", `output port name match, e.g. "P-125" (first device if empty)`)
	outKind := flag.String("out", "device", "MIDI output: device|mock (mock prints messages instead of playing)")
	style := flag.String("style", "straight", "playing feel: "+strings.Join(sequence.StyleNames(), "|"))
	seed := flag.Int64("seed", 0, "random seed for --style humanisation (0 = time-based)")
	export := flag.String("export", "", "write the sequence to a .mid instead of playing it (straight timing, for notation)")
	flag.Parse()

	if *list {
		names, err := midi.ListOutputs()
		if err != nil {
			fail(err)
		}
		for i, name := range names {
			fmt.Printf("%d: %s\n", i, name)
		}
		return
	}

	var in io.Reader = os.Stdin
	if flag.NArg() > 0 {
		f, err := os.Open(flag.Arg(0))
		if err != nil {
			fail(err)
		}
		defer f.Close()
		in = f
	}
	// Accept either a JSON sequence or a Standard MIDI File, told apart by the
	// "MThd" magic — so play7 can replay a .mid (e.g. one edited in MuseScore),
	// running it through the same style + playback path as a JSON sequence.
	raw, err := io.ReadAll(in)
	if err != nil {
		fail(err)
	}
	var seq sequence.Sequence
	if len(raw) >= 4 && string(raw[:4]) == "MThd" {
		msgs, bpm, rerr := smf.Read(bytes.NewReader(raw))
		if rerr != nil {
			fail(rerr)
		}
		seq = midiToSequence(msgs, bpm)
	} else {
		seq, err = sequence.Parse(bytes.NewReader(raw))
		if err != nil {
			fail(err)
		}
	}

	// --export writes the straight sequence to a .mid (notation, bridge to
	// MuseScore) and exits — no style humanisation, so the timing quantises
	// cleanly into a readable score.
	if *export != "" {
		if err := exportMIDI(*export, seq); err != nil {
			fail(err)
		}
		fmt.Fprintf(os.Stderr, "play7: wrote %s\n", *export)
		return
	}

	st, ok := sequence.StyleByName(*style)
	if !ok {
		fail(fmt.Errorf("unknown style %q (have: %s)", *style, strings.Join(sequence.StyleNames(), ", ")))
	}
	sd := *seed
	if sd == 0 {
		sd = time.Now().UnixNano()
	}
	seq = st.Apply(seq, rand.New(rand.NewSource(sd)))

	out, err := midi.NewOut(*outKind, *port, os.Stdout)
	if err != nil {
		fail(err)
	}
	defer out.Close()

	// A real piano keeps ringing if we die between note-on and note-off, so an
	// interrupt silences the channel before exiting.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		out.Control(seq.Channel, midi.AllNotesOff, 0)
		out.Close()
		os.Exit(1)
	}()

	if err := sequence.Play(out, seq, time.Sleep); err != nil {
		out.Control(seq.Channel, midi.AllNotesOff, 0)
		fail(err)
	}
}

// midiToSequence turns parsed SMF messages into a playable sequence, so a .mid
// flows through the same style and playback path as a JSON sequence.
func midiToSequence(msgs []smf.Msg, bpm float64) sequence.Sequence {
	evs := make([]sequence.Event, 0, len(msgs))
	for _, m := range msgs {
		at := time.Duration(m.MS * float64(time.Millisecond))
		switch m.Status {
		case smf.ControlChange:
			evs = append(evs, sequence.Event{At: at, Ctrl: true, Note: m.D1, Vel: m.D2})
		case smf.NoteOn:
			if m.D2 > 0 {
				evs = append(evs, sequence.Event{At: at, On: true, Note: m.D1, Vel: m.D2})
			} else {
				evs = append(evs, sequence.Event{At: at, On: false, Note: m.D1})
			}
		case smf.NoteOff:
			evs = append(evs, sequence.Event{At: at, On: false, Note: m.D1})
		}
	}
	return sequence.Sequence{Tempo: bpm, Channel: 0, Events: evs}
}

// exportMIDI writes a compiled sequence to a Standard MIDI File, carrying the
// sequence tempo so a notation editor quantises it correctly.
func exportMIDI(path string, seq sequence.Sequence) error {
	msgs := make([]smf.Msg, 0, len(seq.Events))
	for _, e := range seq.Events {
		ms := float64(e.At.Microseconds()) / 1000.0
		switch {
		case e.Ctrl:
			msgs = append(msgs, smf.Msg{MS: ms, Status: smf.ControlChange, D1: e.Note, D2: e.Vel})
		case e.On:
			msgs = append(msgs, smf.Msg{MS: ms, Status: smf.NoteOn, D1: e.Note, D2: e.Vel})
		default:
			msgs = append(msgs, smf.Msg{MS: ms, Status: smf.NoteOff, D1: e.Note, D2: 0})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return smf.Write(f, msgs, seq.Tempo)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "play7:", err)
	os.Exit(1)
}
