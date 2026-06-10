// play7 plays a JSON note/chord sequence on a MIDI output device — the output
// twin of keys7 (which listens). Where keys7 is a resident TUI on MIDI in,
// play7 is a silent one-shot: read a sequence, play it, exit.
//
//	play7 --list
//	play7 --port "P-125" sequence.json
//	echo '{"steps":[{"notes":["C4","E4","G4"],"beats":2}]}' | play7 --out=mock
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"keys7/internal/midi"
	"keys7/internal/sequence"
)

func main() {
	list := flag.Bool("list", false, "list MIDI output devices and exit")
	port := flag.String("port", "", `output port name match, e.g. "P-125" (first device if empty)`)
	outKind := flag.String("out", "device", "MIDI output: device|mock (mock prints messages instead of playing)")
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
	seq, err := sequence.Parse(in)
	if err != nil {
		fail(err)
	}

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

func fail(err error) {
	fmt.Fprintln(os.Stderr, "play7:", err)
	os.Exit(1)
}
