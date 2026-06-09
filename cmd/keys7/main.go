package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
	"keys7/internal/ui"
)

func main() {
	source := flag.String("source", "mock", "MIDI source: device|mock")
	port := flag.String("port", "", `device port name match, e.g. "P-125" (device source only)`)
	keyFlag := flag.String("key", "C", `key for cadence hints: "C", "Am", "F#m", or "auto" to infer it from playing`)
	notationFlag := flag.String("notation", "letters", `note spelling: "letters" (C D E) or "solfege" (Do Ré Mi)`)
	flag.Parse()

	if strings.EqualFold(*notationFlag, "solfege") || strings.EqualFold(*notationFlag, "fr") {
		theory.SetNotation(theory.Solfege)
	}

	var key theory.Key
	autoKey := strings.EqualFold(*keyFlag, "auto")
	if !autoKey {
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

	m := ui.New(*source, *port, key, autoKey, src.Events(), mesh.NopForwarder{})
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}
}
