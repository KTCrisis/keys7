# keys7

MIDI capture node for a Yamaha P-125 — first node of a personal "fleet" of
physical surfaces wired to [flux7-mesh](https://github.com/KTCrisis/flux7-mesh).

**Phase 1 (this code): prove the wiring and the base UI, identically on Windows
and Linux.** No chord theory, no AI yet — capture MIDI and show it live.

## Architecture

Everything hangs off one interface, `midi.MidiSource`. The UI never knows where
events come from:

- **`deviceSource`** — a real MIDI port via RtMidi (WinMM on Windows, ALSA on
  Linux). Compiled only with `-tags midi_device`, which pulls in CGO.
- **`mockSource`** — a synthetic loop (a C-major triad + a pedalled scale run).
  The default. Pure Go, no CGO — this is what makes development on WSL, without
  the piano, possible and testable.

```
cmd/keys7/main.go        entry; flags --source=device|mock, --port
internal/midi/           source interface, event model, note naming
  device_rtmidi.go         //go:build midi_device  (real port, CGO)
  device_stub.go           //go:build !midi_device (returns a clear error)
  mock.go                  synthetic source (default)
internal/ui/             Bubble Tea model + view
internal/mesh/           Forwarder seam (no-op in phase 1)
```

## Run

```bash
make run-mock            # synthetic source — works anywhere, incl. WSL
make build               # pure-Go build (mock), no CGO
```

On Windows, with the P-125 connected:

```bash
make build-windows       # needs CGO + a C toolchain (mingw-w64) + RtMidi
keys7.exe --source=device --port "P-125"
```

## Cross-platform notes

- The P-125 is USB-MIDI **on Windows**; WSL doesn't see USB-MIDI natively, so the
  device mode runs on the Windows host. WSL is the dev/mock environment.
- The P-125 exposes a single USB-MIDI port. It's exclusive on Windows: if Renoise
  holds it, keys7 can't open it at the same time. A virtual MIDI splitter
  (loopMIDI) to let both read it is **phase 2**; for now, run with Renoise closed.
- The **device path is not yet validated against live hardware** — it's written
  against gomidi/midi v2 and must be confirmed on the Windows host, adjusting the
  gomidi calls if the installed API differs. The mock path is the verified one.

## Where this is going

Phase 1 is capture + UI. Next: chord recognition (deterministic), then
scale-locked cadence suggestions (deterministic), then style/memory-aware
coaching (the layer where AI earns its place, on the mesh).
