// Package session records the harmonic events keys7 hears — chords and key
// changes — as a JSONL stream. This is the bridge to the AI layer: an assistant
// reads the log to know what's being played and suggest over it. It's also the
// concrete realisation of the mesh Forwarder seam; an HTTP/SSE sink can replace
// the file sink later without touching the model.
package session

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

// HarmonicEvent is one thing keys7 heard. Two layers share the stream: the
// faithful capture ("note" on/off and "pedal" events — the raw material a
// reader segments with hindsight) and the live interpretation ("chord", "key",
// "melody" — real-time hints, imperfect on arpeggios by nature). "texture" is
// the player's declared intent (block chords / arpeggio / free), a strong
// prior for the reader's segmentation.
type HarmonicEvent struct {
	Time    string  `json:"t"`
	Kind    string  `json:"kind"`              // "note" | "pedal" | "chord" | "key" | "melody" | "texture" | "reset" | "cue"
	Chord   string  `json:"chord,omitempty"`   // letters, e.g. "Cmaj7"
	Solfege string  `json:"solfege,omitempty"` // e.g. "Domaj7"
	Key     string  `json:"key,omitempty"`     // active key when heard
	Roman   string  `json:"roman,omitempty"`   // diatonic degree, if any
	Degree  int     `json:"deg,omitempty"`
	Conf    float64 `json:"conf,omitempty"` // key-detection confidence
	Note    string  `json:"note,omitempty"` // chord annotation, or the note name ("A4")
	Midi    uint8   `json:"midi,omitempty"` // note number (note/melody events)
	Vel     uint8   `json:"v,omitempty"`    // onset velocity
	Reg     string  `json:"reg,omitempty"`  // melody register: "high" | "low"
	On      *bool   `json:"on,omitempty"`   // note: attack (true) or release; pedal: down or up
	Mode    string  `json:"mode,omitempty"` // texture: "free" | "block" | "arpeggio"
}

// Stamp formats an event time: RFC3339 with milliseconds, so melody rhythm can
// be reconstructed from inter-onset gaps (whole seconds are too coarse).
func Stamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z07:00") }

// Bool returns a pointer for the On field ("on":false must serialize, so the
// field is a pointer under omitempty).
func Bool(b bool) *bool { return &b }

// Sink consumes harmonic events. Implementations must be safe for use from the
// model's single goroutine; JSONLSink guards writes anyway.
type Sink interface {
	Emit(HarmonicEvent)
}

// NopSink discards events (no --log given).
type NopSink struct{}

func (NopSink) Emit(HarmonicEvent) {}

// JSONLSink appends one JSON object per line to a writer. Emit never blocks
// the model on an error; failures are counted instead, and the caller checks
// Dropped() at shutdown — a journal that lost events must not pretend it
// didn't (the AI layer reads it as the ground truth of the session).
type JSONLSink struct {
	mu      sync.Mutex
	w       io.Writer
	dropped int
	lastErr error
}

func NewJSONLSink(w io.Writer) *JSONLSink { return &JSONLSink{w: w} }

func (s *JSONLSink) Emit(e HarmonicEvent) {
	b, err := json.Marshal(e)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		_, err = s.w.Write(append(b, '\n'))
	}
	if err != nil {
		s.dropped++
		s.lastErr = err
	}
}

// Dropped reports how many events were lost and the last error seen.
func (s *JSONLSink) Dropped() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped, s.lastErr
}
