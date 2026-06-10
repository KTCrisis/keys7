package ui

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The reply panel is the journal's return channel: the assistant writes a small
// text file, keys7 polls it and shows the content. Polling (not fsnotify): the
// file is written from WSL and read across the 9P mount, where change
// notifications don't propagate — explicit reads always see fresh data.

// replyPollEvery is the poll cadence. Half a second reads as immediate for a
// few lines of text and costs nothing on a local file.
const replyPollEvery = 500 * time.Millisecond

// replyMsg carries the (possibly unchanged) reply file content into Update.
type replyMsg string

// replyTick re-arms the poll: read the file, deliver its content, repeat from
// Update. tea.Tick fires once after the delay — each replyMsg handled in Update
// schedules the next tick, the same re-arming pattern as waitForEvent.
func replyTick(path string) tea.Cmd {
	return tea.Tick(replyPollEvery, func(time.Time) tea.Msg {
		b, err := os.ReadFile(path)
		if err != nil {
			return replyMsg("") // absent file = no reply yet, not an error
		}
		return replyMsg(strings.TrimSpace(string(b)))
	})
}
