package midi

import "time"

// mockSource emits a small synthetic loop so the wiring and UI can be exercised
// on a machine with no MIDI hardware (e.g. WSL during development). It is the
// default source. A later iteration can swap the synthetic loop for .mid file
// replay — gomidi/smf is pure Go, so that stays CGO-free.
type mockSource struct {
	ch   chan Event
	done chan struct{}
}

func newMockSource() *mockSource {
	s := &mockSource{
		ch:   make(chan Event, 64),
		done: make(chan struct{}),
	}
	go s.loop()
	return s
}

func (s *mockSource) Events() <-chan Event { return s.ch }

func (s *mockSource) Close() error {
	close(s.done)
	return nil
}

// step is one scripted action: wait `after`, then emit `ev`.
type step struct {
	after time.Duration
	ev    Event
}

func (s *mockSource) loop() {
	// C major: a held triad, a short pedalled scale run, then a breath.
	script := []step{
		{700 * time.Millisecond, noteOn(60, 90)}, // breath, then C E G
		{0, noteOn(64, 88)},
		{0, noteOn(67, 92)},
		{900 * time.Millisecond, noteOff(60)},
		{0, noteOff(64)},
		{0, noteOff(67)},
		{300 * time.Millisecond, cc(sustainController, 127)}, // pedal down
		{250 * time.Millisecond, noteOn(62, 80)},
		{250 * time.Millisecond, noteOn(64, 80)},
		{250 * time.Millisecond, noteOn(65, 80)},
		{250 * time.Millisecond, noteOn(67, 80)},
		{600 * time.Millisecond, cc(sustainController, 0)}, // pedal up
		{0, noteOff(62)},
		{0, noteOff(64)},
		{0, noteOff(65)},
		{0, noteOff(67)},
	}
	for {
		for _, st := range script {
			select {
			case <-s.done:
				close(s.ch)
				return
			case <-time.After(st.after):
			}
			ev := st.ev
			ev.Timestamp = time.Now()
			select {
			case <-s.done:
				close(s.ch)
				return
			case s.ch <- ev:
			}
		}
	}
}

func noteOn(note, vel uint8) Event { return Event{Kind: NoteOn, Data1: note, Data2: vel} }
func noteOff(note uint8) Event     { return Event{Kind: NoteOff, Data1: note} }
func cc(ctrl, val uint8) Event     { return Event{Kind: ControlChange, Data1: ctrl, Data2: val} }
