package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/session"
	"keys7/internal/theory"
	"keys7/internal/ui"
)

func main() {
	source := flag.String("source", "mock", "MIDI source: device|mock")
	port := flag.String("port", "", `device port name match, e.g. "P-125" (device source only)`)
	keyFlag := flag.String("key", "C", `key for cadence hints: "C", "Am", "F#m", "auto" (infer), or "drone" (pin to bass)`)
	notationFlag := flag.String("notation", "letters", `note spelling: "letters" (C D E) or "solfege" (Do Ré Mi)`)
	logFlag := flag.String("log", "", "append heard chords/keys as JSONL to this file (the AI bridge)")
	flag.Parse()

	if strings.EqualFold(*notationFlag, "solfege") || strings.EqualFold(*notationFlag, "fr") {
		theory.SetNotation(theory.Solfege)
	}

	var key theory.Key
	keySrc := ui.KeyManual
	switch {
	case strings.EqualFold(*keyFlag, "auto"):
		keySrc = ui.KeyAuto
	case strings.EqualFold(*keyFlag, "drone"):
		keySrc = ui.KeyDrone
	default:
		var err error
		if key, err = theory.ParseKey(*keyFlag); err != nil {
			fmt.Fprintln(os.Stderr, "keys7:", err)
			os.Exit(1)
		}
	}

	src, err := midi.NewSource(*source, *port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}
	defer src.Close()

	var sink session.Sink = session.NopSink{}
	if *logFlag != "" {
		f, err := os.OpenFile(*logFlag, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "keys7:", err)
			os.Exit(1)
		}
		defer f.Close()
		sink = session.NewJSONLSink(f)
	}

	m := ui.New(*source, *port, key, keySrc, src.Events(), mesh.NopForwarder{}, sink)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}
}
