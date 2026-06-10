package ui

import (
	"strings"
	"testing"

	"keys7/internal/mesh"
	"keys7/internal/session"
	"keys7/internal/theory"
)

// A replyMsg must land in the model, show up in the view, and re-arm the poll.
func TestReplyPanel(t *testing.T) {
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, session.NopSink{}, "reply.txt")

	if !strings.Contains(m.View(), "waiting for a reply") {
		t.Error("empty reply: view missing the placeholder")
	}

	updated, cmd := m.Update(replyMsg("try the dorian IV here"))
	if cmd == nil {
		t.Error("replyMsg did not re-arm the poll tick")
	}
	if v := updated.View(); !strings.Contains(v, "try the dorian IV here") {
		t.Errorf("view missing the reply text:\n%s", v)
	}
}

// Without --reply, the panel stays out of the layout.
func TestNoReplyPanelWithoutFlag(t *testing.T) {
	m := New("mock", "", theory.Key{}, KeyManual, nil, mesh.NopForwarder{}, session.NopSink{}, "")
	if strings.Contains(m.View(), "assistant") {
		t.Error("reply panel rendered without --reply")
	}
}
