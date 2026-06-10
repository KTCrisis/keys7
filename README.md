# keys7

A live harmony assistant for a MIDI piano (developed on a Yamaha P-125) — and
the first node of a personal "fleet" of physical surfaces wired to
[flux7-mesh](https://github.com/KTCrisis/flux7-mesh).

You play; keys7 reads the harmony back in real time and can hand it to an AI
that listens and suggests:

- **Chords** — triads through 13ths, fifth-less voicings, inversions as slash
  chords, dyads named with the harmonies they imply.
- **Key** — fixed, **auto-detected** from playing (Krumhansl-Schmuckler), or
  **drone** (tonic pinned to the bass, for pedal/modal playing).
- **Suggestions** — the diatonic palette with the current degree lit, cadence
  moves, neighbouring keys, and secondary dominants; a chord you play that
  fulfils a suggestion lights up.
- **Melody split** — a right-hand line over a held chord is separated from the
  harmonic core, so it doesn't pollute the chord name.
- **Notation** — letters and solfège shown together (A7 / La7).
- **AI bridge** — `--log` streams what's heard as JSONL for an assistant to read.

The theory is deterministic and local. The AI layer reads the log (today) and
will move onto the mesh (real-time) later.

## Architecture

Everything hangs off one interface, `midi.MidiSource`; the rest never knows
where events come from.

```
cmd/keys7/main.go     entry; flags --source --port --key --notation --log
cmd/play7/main.go     output twin: plays a JSON sequence on a MIDI out device
internal/midi/        source/output interfaces + events
  device_windows.go     //go:build windows  — WinMM input, pure Go (no CGO)
  device_other.go       //go:build !windows — clear error; use mock
  mock.go               synthetic source (default, for WSL/dev)
  out_windows.go        //go:build windows  — WinMM output (play7)
  out_other.go          //go:build !windows — clear error; use --out=mock
internal/theory/      pure pitch math: chords, dyads, keys/modes, cadences,
                      neighbours, secondary dominants, key detection, notation
internal/sequence/    JSON sequence parsing + scheduling (pure, like theory)
internal/ui/          Bubble Tea model + view (panels)
internal/session/     harmonic-event log (the AI bridge)
internal/mesh/        Forwarder seam (no-op; real-time transport later)
```

`deviceSource` reads the piano on Windows via `winmm.dll` in pure Go — syscalls
plus a `windows.NewCallback` — so `keys7.exe` cross-compiles from WSL with no
CGO, no RtMidi, no third-party MIDI module.

## Run

```bash
make run-mock            # synthetic source — works anywhere, incl. WSL
make build               # pure-Go build (mock), no CGO
make build-windows       # cross-compiles bin/keys7.exe from WSL, no toolchain
make test                # theory + midi + session tests
```

On Windows, with the P-125 connected (DAW closed — the USB-MIDI port is
exclusive):

```
keys7.exe --source=device --key auto --log "C:\…\session.jsonl"
```

- `--key` : `C`, `Am`, `F#m`, … · `auto` (infer) · `drone` (pin to bass)
- `--notation` : `letters` (C D E) · `solfege` (Do Ré Mi)
- `--log <file>` : append heard chords/keys as JSONL (the AI bridge)

## TUI keys

```
←/→  shift the tonic         m  cycle major / natural / harmonic / melodic minor
r    relative key            a  auto key-detection      d  drone (bass-pinned)
e    melody/harmony split    n  notation (letters↔solfège)
x    reset (forget playing)  q  quit
```

## The AI bridge

With `--log`, keys7 appends one JSON object per event: a `chord` (letters +
solfège, with its diatonic degree, or the secondary dominant it is if
chromatic), a `key` change (with detection confidence), or a `reset` marker.
An assistant reads the file to know what's being played and suggest over it —
the concrete realisation of the otherwise-dormant mesh `Forwarder` seam. Put the
log on a path both sides can see (e.g. under `/mnt/c/…` from WSL).

## play7 — playing *to* the piano

Where keys7 listens (resident TUI on MIDI in), play7 speaks: a silent one-shot
CLI that plays a JSON note/chord sequence on a MIDI output — so a machine (or
an assistant) can sound ideas on the piano instead of describing them.

```bash
make build-play7-windows   # cross-compiles bin/play7.exe, same no-CGO story
play7.exe --list           # name the output devices
play7.exe --port "P-125" sequence.json
echo '{"steps":[{"notes":["C4","E4","G4"],"beats":2}]}' | play7.exe
```

A sequence is `{tempo, channel, velocity, steps:[{notes, beats, velocity}]}` —
notes in scientific pitch ("A3", "F#4", chords as arrays, no notes = rest),
beats at the sequence tempo, step velocity overriding the sequence's. Defaults:
90 BPM, channel 1, velocity 80. `--out=mock` prints the messages instead of
playing (the WSL audition mode); Ctrl-C sends All Notes Off before exiting so
the piano never rings on.

## Cross-platform notes

- The P-125 is USB-MIDI **on Windows**; WSL doesn't see USB-MIDI natively, so
  device mode runs on the Windows host. WSL is the dev/mock environment.
- Single USB-MIDI port, exclusive on Windows: run keys7 with the DAW closed, or
  share via a virtual MIDI splitter (loopMIDI) — a later concern.
- **Any MIDI input works** — nothing is P-125-specific (WinMM opens any
  class-compliant device; `--port` matches by name, else the first input is
  used). The P-125 is just the device it was developed and validated on.

## Roadmap (v2)

- Real-time AI over the mesh (SSE/MCP, "like OBS") instead of the file pull, with
  style-aware coaching drawing on flux7-memory and the Renoise analyses corpus.
- Richer drone/modal detection (distinguish Dorian, Phrygian, … not just
  major/minor by the third).
- Pedal-aware chord segmentation (notes still sounding under sustain).
- UI-layer tests; packaging / a tagged release.
