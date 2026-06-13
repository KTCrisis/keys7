# keys7

A live harmony assistant for a MIDI piano (developed on a Yamaha P-125) — and
the first node of a personal "fleet" of physical surfaces wired to
[flux7-mesh](https://github.com/KTCrisis/flux7-mesh).

You play; keys7 reads the harmony back in real time and can hand it to an AI
that listens and suggests:

- **Chords** — triads through 13ths, fifth-less voicings, inversions as slash
  chords, dyads named with the harmonies they imply.
- **Key** — fixed, **auto-detected** from playing (Krumhansl-Schmuckler), or
  **drone** (tonic pinned to the bass, for pedal/modal playing). In drone mode
  the colour over the bass names the mode — the seven diatonic modes (ionian
  through locrian), read from their characteristic tones, not just major/minor.
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
cmd/export7/main.go   transcribe a session journal (JSONL) into a .mid
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
internal/smf/         Standard MIDI File writer (pure Go), for export7
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
make install-windows     # build + deploy to Windows + Desktop shortcut (from WSL)
make test                # theory + midi + session tests
```

On Windows, with the P-125 connected (DAW closed — the USB-MIDI port is
exclusive):

```
keys7.exe --source=device --key auto --log "C:\…\session.jsonl"
```

**One-click launch.** `make install-windows` (run from WSL) cross-compiles
`keys7.exe` + `play7.exe`, copies them with `scripts/keys7.ps1` to
`%USERPROFILE%\Documents\keys7\`, and drops a **`keys7` shortcut on the
Desktop**. Double-clicking it opens a session in a PowerShell console: device
source, auto key, and a **fixed-path
journal** at `sessions\current.jsonl` — the previous session is rotated to a
timestamped archive at launch, so the assistant side always follows one stable
path (the launcher prints its WSL path for `watch-cue.sh`). The deploy hot-swaps
binaries, so it works even while a session is running (a running `.exe` is
renamed aside, not overwritten). Override the target dir with `WINDEST=…`.

- `--key` : `C`, `Am`, `F#m`, … · `auto` (infer) · `drone` (pin to bass)
- `--notation` : `letters` (C D E) · `solfege` (Do Ré Mi)
- `--log <file>` : append heard chords/keys/melody as JSONL (the AI bridge)
- `--reply <file>` : poll a text file and show it in an "assistant" panel —
  the bridge's return channel (polling, not fsnotify: change notifications
  don't cross the WSL/Windows mount; reads do)

## TUI keys

```
←/→  shift the tonic         m  cycle major / natural / harmonic / melodic minor
r    relative key            a  auto key-detection      d  drone (bass-pinned)
e    melody/harmony split    n  notation (letters↔solfège)
t    texture (free/block/arpeggio — declared intent, journaled)
x    reset (forget playing)  q  quit

cues (double-tap a signal-bar key):
A0 turn · A#0 replay · B0 transpose · C1 harmonise
```

## The AI bridge

With `--log`, keys7 appends one JSON object per event, in two layers. The
**faithful capture**: every `note` attack/release (name, number, velocity) and
`pedal` move, millisecond-stamped — the raw material a reader segments with
hindsight, where arpeggio-vs-line is easy (it is undecidable in real time).
A `texture` event records the player's declared mode (`t` key: free / block /
arpeggio) — intent as a fact, a strong prior for that segmentation. And the
**live interpretation**: a `chord` (letters +
solfège, with its diatonic degree, or the secondary dominant it is if
chromatic), a `key` change (with detection confidence), a `melody` onset (note
name, number, velocity, and a register: `"reg":"high"` for a line over the
chord, `"reg":"low"` for one walking under it — a left-hand melody beneath
right-hand chords. Low detection is temporal (a note landing well below an
already-sounding chord), so planted basses and slash chords stay harmonic),
a `reset` marker, or a `cue`. Timestamps carry milliseconds, so a reader
can reconstruct the melodic rhythm from inter-onset gaps. The split classifies
melody over a sounding chord (≥ 3 remaining notes) — held by fingers **or by
the sustain pedal**: notes released with the damper up stay in the harmonic
picture until the pedal comes up, so a pedaled chord under a hands-off line
journals like a held one. An assistant reads the file to know what's being played and suggest over
it — the concrete realisation of the otherwise-dormant mesh `Forwarder` seam.
Put the log on a path both sides can see (e.g. under `/mnt/c/…` from WSL).

**Cues** are signalling gestures on the four lowest keys — a "signal bar" below
any harmony register, kept out of the analysis. Double-tap one within 2 s and
keys7 logs `{"kind":"cue","cue":"…"}` (the header shows the gesture):

- **A0** — `turn`: your turn, answer now
- **A#0** — `replay`: play my last phrase back
- **B0** — `transpose`: move it
- **C1** — `harmonise`: add voices

keys7 only detects and journals the gesture; what the assistant does with each
is the session protocol's business. Combined with play7 on the same piano's MIDI
in, that's a full conversation without leaving the bench: you play, double-tap,
and the answer comes back through the instrument.

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

A sequence is one or more **voices**, each with its own steps and velocity,
all starting together — a melody can move, louder, over a chord the other
voice holds:

```json
{"tempo": 65, "voices": [
  {"velocity": 90, "steps": [{"notes": ["D5"], "beats": 1}, {"notes": ["F5"], "beats": 2}]},
  {"velocity": 58, "steps": [{"notes": ["Bb2", "D3", "F3", "A3"], "beats": 3}]}
]}
```

Notes are scientific pitch ("A3", "F#4"), chords are arrays, no notes = rest;
beats run at the sequence tempo; velocity resolves step > voice > sequence.
A top-level `steps` array is shorthand for a single voice. Defaults: 90 BPM,
channel 1, velocity 80. `--out=mock` prints the messages instead of playing
(the WSL audition mode); Ctrl-C sends All Notes Off before exiting so the
piano never rings on.

A step can also move the **sustain pedal** with `"pedal": "down"` or `"up"`
(CC64 at the step's onset, ordered before the chord so it catches it):

```json
{"steps": [{"notes": ["C3","E3","G3"], "beats": 4, "pedal": "down"}, {"beats": 0, "pedal": "up"}]}
```

### Playing styles

`--style` applies a *feel* — humanisation, articulation and pedalling — so play7
doesn't sound mechanical. `straight` (default) is the identity; the others
loosen timing, roll chords, bend durations, vary velocity, and (ambient /
orchestral) sustain each chord automatically:

```bash
play7 --style ambient sequence.json
play7 --style darksynth --seed 7 sequence.json   # --seed makes a take reproducible
```

| `straight` | exact, plaqué, no pedal — byte-for-byte the parsed sequence |
| `ambient` | soft timing, rolled chords, legato, auto-pedal, softened velocity |
| `orchestral` | wider dynamics with breath, discreet rolls, legato, auto-pedal |
| `darksynth` | tight, plaqué, staccato, no pedal, marked steady velocity |

The randomness is seeded (`--seed`, default time-based), so a seed reproduces a
take. An explicit `pedal` in the sequence overrides a style's auto-pedal.

`--export out.mid` writes the sequence to a Standard MIDI File instead of
playing it (straight timing, no style — so it quantises into a readable score),
carrying the sequence tempo. Engrave it with MuseScore's CLI:

```bash
play7 --export take.mid sequence.json
"/mnt/c/Program Files/MuseScore 4/bin/MuseScore4.exe" -r 100 -o take.png take.mid
```

## Running a live session with an assistant

The bridge above is just files; what makes it a *conversation* is the loop on
the assistant's side. The reference setup (Claude Code, but any agent CLI with
background commands and command allowlists fits) has three pieces:

**1. Scripts, not inline shell** (`scripts/`):

- `watch-cue.sh <journal>` — polls the journal (2 s) until a `cue` lands, then
  prints every line added since it started and exits. Polling, not inotify:
  change notifications don't cross the WSL/Windows 9P mount; reads do.
- `play.sh '<sequence-json>' [port]` — plays a sequence via play7.
- `reply.sh <file> '<text>'` — writes the TUI reply panel file.

All three take **arguments instead of stdin/pipes**, deliberately: command
allowlists match on prefixes, and a pipe in the command line defeats them.

**2. An allowlist** so the loop runs without permission prompts — e.g. in
Claude Code's `settings.json`:

```json
{"permissions": {"allow": [
  "Bash(/path/to/keys7/scripts/watch-cue.sh *)",
  "Bash(/path/to/keys7/scripts/play.sh *)",
  "Bash(/path/to/keys7/scripts/reply.sh *)"
]}}
```

**3. A skill / standing instruction** encoding the protocol, so a session
starts from one phrase. The loop it describes:

1. Arm `watch-cue.sh` as a background command — **alone, never chained
   behind play/reply with `&&` or `&`** (an orphaned watcher reports to a
   finished task and never wakes anyone). One watcher at a time.
2. On wake: read the take, segment melody/harmony **from the raw `note` layer
   with hindsight** (chords cluster within ~30 ms; arpeggios accumulate under
   the pedal; melodies replace each other; `texture` events are the player's
   declared intent). The real-time `chord`/`melody` hints are corroboration,
   not truth.
3. Answer on both channels — `reply.sh` for the TUI panel, `play.sh` with
   voices (melody ~80-90 velocity over chords ~50-60) — then re-arm the
   watcher and yield.

The player's side of the protocol: play freely (pedal included), `t` to
declare texture, double-tap A0 to hand the turn over.

## Exporting a session

A journal is a faithful capture, so it transcribes straight to a `.mid` —
editable in Renoise, MuseScore, any DAW. Run `export7` from WSL (it reads the
journal on the Windows side under `/mnt/c`):

```bash
make build-export7
bin/export7 /mnt/c/Users/…/keys7/sessions/current.jsonl   # -> current.mid
bin/export7 -bpm 72 -o take.mid session.jsonl
```

Every `note` attack/release and `pedal` move becomes a MIDI event at its
recorded millisecond, with velocity — no re-quantisation. `-bpm` only sets the
tempo meta; absolute timing is preserved regardless.

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
- Packaging / a tagged release.
