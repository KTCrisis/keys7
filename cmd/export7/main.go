// export7 turns a keys7 session journal (JSONL) into a Standard MIDI File — the
// bridge back to Renoise / MuseScore. The journal already holds every note
// attack/release and pedal move with millisecond timestamps and velocity, so
// the export is a faithful transcription of what was played, not a re-quantised
// approximation.
//
//	export7 session.jsonl            # -> session.mid
//	export7 -o take.mid session.jsonl
//	export7 - < session.jsonl > out.mid
//	export7 -bpm 72 session.jsonl    # tempo meta (timing is preserved either way)
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"keys7/internal/session"
	"keys7/internal/smf"
)

func main() {
	out := flag.String("o", "", "output .mid (default: input with .mid; stdout for stdin)")
	bpm := flag.Float64("bpm", 120, "tempo meta in BPM (absolute timing is preserved regardless)")
	flag.Parse()

	in := flag.Arg(0)
	if in == "" {
		fmt.Fprintln(os.Stderr, "usage: export7 [-o out.mid] [-bpm N] <journal.jsonl|->")
		os.Exit(2)
	}

	r := os.Stdin
	if in != "-" {
		f, err := os.Open(in)
		if err != nil {
			fmt.Fprintln(os.Stderr, "export7:", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	msgs, n, err := transcribe(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, "export7:", err)
		os.Exit(1)
	}
	if n == 0 {
		fmt.Fprintln(os.Stderr, "export7: no note/pedal events in the journal")
		os.Exit(1)
	}

	w := os.Stdout
	dest := *out
	if dest == "" && in != "-" {
		dest = strings.TrimSuffix(in, ".jsonl") + ".mid"
	}
	if dest != "" {
		f, err := os.Create(dest)
		if err != nil {
			fmt.Fprintln(os.Stderr, "export7:", err)
			os.Exit(1)
		}
		defer f.Close()
		w = f
	}

	if err := smf.Write(w, msgs, *bpm); err != nil {
		fmt.Fprintln(os.Stderr, "export7:", err)
		os.Exit(1)
	}
	if dest != "" {
		fmt.Fprintf(os.Stderr, "export7: %d events -> %s\n", n, dest)
	}
}

// transcribe reads the journal and returns the note/pedal messages with their
// times in milliseconds from the first such event.
func transcribe(r io.Reader) ([]smf.Msg, int, error) {
	var msgs []smf.Msg
	var t0 time.Time
	var have0 bool
	n := 0

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // tolerate long lines
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e session.HarmonicEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines rather than abort the take
		}
		if e.Kind != "note" && e.Kind != "pedal" {
			continue
		}
		ts, err := time.Parse(time.RFC3339Nano, e.Time)
		if err != nil {
			continue
		}
		if !have0 {
			t0, have0 = ts, true
		}
		ms := float64(ts.Sub(t0).Microseconds()) / 1000.0

		switch e.Kind {
		case "note":
			if e.On != nil && *e.On {
				msgs = append(msgs, smf.Msg{MS: ms, Status: smf.NoteOn, D1: e.Midi, D2: e.Vel})
			} else {
				msgs = append(msgs, smf.Msg{MS: ms, Status: smf.NoteOff, D1: e.Midi, D2: 0})
			}
		case "pedal":
			val := byte(0)
			if e.On != nil && *e.On {
				val = 127
			}
			msgs = append(msgs, smf.Msg{MS: ms, Status: smf.ControlChange, D1: 64, D2: val})
		}
		n++
	}
	return msgs, n, sc.Err()
}
