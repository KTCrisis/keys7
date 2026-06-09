package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"keys7/internal/mesh"
	"keys7/internal/midi"
	"keys7/internal/theory"
	"keys7/internal/ui"
)

func main() {
	source := flag.String("source", "mock", "MIDI source: device|mock")
	port := flag.String("port", "", `device port name match, e.g. "P-125" (device source only)`)
	keyFlag := flag.String("key", "C", `fixed key for cadence hints, e.g. "C", "Am", "F#m" (change live with ←/→ and m)`)
	flag.Parse()

	key, err := theory.ParseKey(*keyFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}

	src, err := midi.NewSource(*source, *port)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}
	defer src.Close()

	m := ui.New(*source, *port, key, src.Events(), mesh.NopForwarder{})
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "keys7:", err)
		os.Exit(1)
	}
}
