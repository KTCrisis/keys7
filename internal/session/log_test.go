package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJSONLSink(t *testing.T) {
	var buf bytes.Buffer
	s := NewJSONLSink(&buf)
	s.Emit(HarmonicEvent{Time: "t1", Kind: "key", Key: "C major", Conf: 0.9})
	s.Emit(HarmonicEvent{Time: "t2", Kind: "chord", Chord: "G7", Roman: "V", Degree: 5})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	var e HarmonicEvent
	if err := json.Unmarshal([]byte(lines[1]), &e); err != nil {
		t.Fatalf("line 2 not valid JSON: %v", err)
	}
	if e.Chord != "G7" || e.Roman != "V" || e.Degree != 5 {
		t.Errorf("decoded %+v", e)
	}
	if n, err := s.Dropped(); n != 0 || err != nil {
		t.Errorf("clean sink reports dropped=%d err=%v", n, err)
	}
}

type failWriter struct{ err error }

func (w failWriter) Write([]byte) (int, error) { return 0, w.err }

func TestJSONLSinkCountsDroppedWrites(t *testing.T) {
	werr := errors.New("disk gone")
	s := NewJSONLSink(failWriter{err: werr})
	s.Emit(HarmonicEvent{Time: "t1", Kind: "note", Note: "A4"})
	s.Emit(HarmonicEvent{Time: "t2", Kind: "pedal"})

	n, err := s.Dropped()
	if n != 2 {
		t.Errorf("expected 2 dropped events, got %d", n)
	}
	if !errors.Is(err, werr) {
		t.Errorf("expected last error %v, got %v", werr, err)
	}
}
