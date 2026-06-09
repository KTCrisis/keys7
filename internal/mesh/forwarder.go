package mesh

import "keys7/internal/midi"

// Forwarder is the seam toward flux7-mesh. In phase 1 it is a no-op; later it
// will push events to the mesh (SSE/HTTP daemon) so keys7 becomes a remote node
// of the stack — the P-125 lives on Windows while the mesh runs in WSL.
type Forwarder interface {
	Forward(ev midi.Event) error
}

// NopForwarder discards events. The default until the mesh transport is wired.
type NopForwarder struct{}

func (NopForwarder) Forward(midi.Event) error { return nil }
