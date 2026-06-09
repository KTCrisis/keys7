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
)

// HarmonicEvent is one thing keys7 heard: a chord, or a key change.
type HarmonicEvent struct {
	Time    string  `json:"t"`
	Kind    string  `json:"kind"`              // "chord" | "key"
	Chord   string  `json:"chord,omitempty"`   // letters, e.g. "Cmaj7"
	Solfege string  `json:"solfege,omitempty"` // e.g. "Domaj7"
	Key     string  `json:"key,omitempty"`     // active key when heard
	Roman   string  `json:"roman,omitempty"`   // diatonic degree, if any
	Degree  int     `json:"deg,omitempty"`
	Conf    float64 `json:"conf,omitempty"` // key-detection confidence
	Note    string  `json:"note,omitempty"` // e.g. "secondary dominant V/ii", "non-diatonic"
}

// Sink consumes harmonic events. Implementations must be safe for use from the
// model's single goroutine; JSONLSink guards writes anyway.
type Sink interface {
	Emit(HarmonicEvent)
}

// NopSink discards events (no --log given).
type NopSink struct{}

func (NopSink) Emit(HarmonicEvent) {}

// JSONLSink appends one JSON object per line to a writer.
type JSONLSink struct {
	mu sync.Mutex
	w  io.Writer
}

func NewJSONLSink(w io.Writer) *JSONLSink { return &JSONLSink{w: w} }

func (s *JSONLSink) Emit(e HarmonicEvent) {
	b, err := json.Marshal(e)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.w.Write(append(b, '\n'))
}
