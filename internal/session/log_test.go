package session

import (
	"bytes"
	"encoding/json"
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
}
