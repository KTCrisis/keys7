//go:build !windows

package midi

import "errors"

// The device source is Windows-only (WinMM). On other platforms (e.g. WSL for
// development), use --source=mock. A Linux/ALSA backend could be added later if
// the P-125 is ever read directly from WSL via USB passthrough.
var errNoDeviceSupport = errors.New(
	"keys7 device source is Windows-only (WinMM); use --source=mock on this platform")

func newDeviceSource(string) (MidiSource, error) { return nil, errNoDeviceSupport }
