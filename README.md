# keys7

MIDI capture node for a Yamaha P-125 — first node of a personal "fleet" of
physical surfaces wired to [flux7-mesh](https://github.com/KTCrisis/flux7-mesh).

It captures what you play and reads the harmony back in real time: the chord
(extended chords and fifth-less voicings included), implied harmony from a
dyad, the diatonic palette and cadence suggestions for a key — fixed or
**auto-detected** from your playing — plus neighbouring keys and secondary
dominants. Melody is separated from the harmonic core so a right-hand line
doesn't pollute the chord name.

All the theory is deterministic and local (no AI yet); that layer comes later,
over flux7-mesh.

## Architecture

Everything hangs off one interface, `midi.MidiSource`. The UI never knows where
events come from:

- **`deviceSource`** — the real P-125, read on Windows via WinMM (`winmm.dll`)
  in **pure Go**: syscalls plus a callback created with `windows.NewCallback`.
  No CGO, no RtMidi, no third-party MIDI module — so `keys7.exe` cross-compiles
  straight from WSL.
- **`mockSource`** — a synthetic loop (a C-major triad + a pedalled scale run).
  The default. Lets development and tests run on WSL with no piano attached.

```
cmd/keys7/main.go        entry; flags --source=device|mock, --port
internal/midi/           source interface, event model, note naming
  device_windows.go        //go:build windows  (WinMM, pure Go)
  device_other.go          //go:build !windows (returns a clear error)
  mock.go                  synthetic source (default)
internal/ui/             Bubble Tea model + view
internal/mesh/           Forwarder seam (no-op for now)
```

## Run

```bash
make run-mock            # synthetic source — works anywhere, incl. WSL
make build               # pure-Go build (mock), no CGO
```

For the real piano, build the Windows binary (from WSL — no toolchain needed)
and run it on Windows with the P-125 connected:

```bash
make build-windows                      # cross-compiles bin/keys7.exe, no CGO
# then, on Windows:
keys7.exe --source=device --port "P-125"
```

## Cross-platform notes

- The P-125 is USB-MIDI **on Windows**; WSL doesn't see USB-MIDI natively, so the
  device mode runs on the Windows host. WSL is the dev/mock environment.
- The P-125 exposes a single USB-MIDI port. It's exclusive on Windows: if another
  app (e.g. a DAW) holds it, keys7 can't open it at the same time. Run it solo,
  or use a virtual MIDI splitter (loopMIDI) to share — that's a later concern.
- The **device path compiles but is not yet validated against live hardware** —
  run `keys7.exe --source=device` on Windows with the P-125 to confirm the WinMM
  callback and device matching. The mock path is the verified one.

## Keys (in the TUI)

```
←/→  shift the key's tonic        m  cycle major / nat / harm / mel minor
r    jump to the relative key     a  toggle auto key-detection
e    toggle melody/harmony split  q  quit
```

`--key` accepts `C`, `Am`, `F#m`, … or `auto` to infer the key from playing.

## Where this is going

Deterministic theory is in (chords, dyads, cadences, key detection, neighbours,
secondary dominants). Next: style/memory-aware coaching — the layer where AI
earns its place — over flux7-mesh, drawing on what you've played.
