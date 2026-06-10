//go:build !windows

package midi

import "errors"

// Like the input side, the device output is Windows-only (WinMM). On other
// platforms (e.g. WSL), use --out=mock to audition a sequence as text.
var errNoOutDeviceSupport = errors.New(
	"play7 device output is Windows-only (WinMM); use --out=mock on this platform")

func newDeviceOut(string) (MidiOut, error) { return nil, errNoOutDeviceSupport }

// ListOutputs is Windows-only for the same reason.
func ListOutputs() ([]string, error) { return nil, errNoOutDeviceSupport }
