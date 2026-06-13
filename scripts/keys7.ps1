# keys7.ps1 - launch a live keys7 session on a connected MIDI piano.
#
# The live journal is always sessions\current.jsonl (a stable path the assistant
# side follows with watch-cue.sh); the previous session is rotated out to a
# timestamped archive at launch. Prefers Windows Terminal for rendering, falling
# back to the plain console.
#
# Meant to be launched by the Desktop shortcut, but works from any terminal:
#   .\keys7.ps1                       # device, auto key, letters
#   .\keys7.ps1 -Key Am -Notation solfege
#   .\keys7.ps1 -Port "P-125"         # match a specific input by name

param(
    [string]$Key      = "auto",        # C | Am | F#m | auto | drone
    [string]$Port     = "",            # input name match; empty = first input
    [string]$Notation = "letters"      # letters | solfege
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSCommandPath

# Runs in the PowerShell console the shortcut opens - it renders the TUI's
# ANSI/256-colour output fine. (An earlier auto-relaunch into Windows Terminal
# was dropped: the wt.exe execution alias can't be started reliably from a
# script. To use WT, just run this .ps1 from a WT tab.)
$exe = Join-Path $root "keys7.exe"
if (-not (Test-Path $exe)) {
    throw "keys7.exe not found next to this script ($root). Run 'make install-windows' from the repo."
}

$sessions = Join-Path $root "sessions"
New-Item -ItemType Directory -Force -Path $sessions | Out-Null

# Fixed-path journal: the live session is always current.jsonl. Rotate the
# previous one out to an archive named by its own last-write time, so the
# assistant side never has to learn a new path between sessions.
$log = Join-Path $sessions "current.jsonl"
if ((Test-Path $log) -and (Get-Item $log).Length -gt 0) {
    $arch = Join-Path $sessions ((Get-Item $log).LastWriteTime.ToString("yyyy-MM-dd_HHmm") + ".jsonl")
    Move-Item -Force -Path $log -Destination $arch
}
$reply = Join-Path $sessions "reply.txt"
if (-not (Test-Path $reply)) { New-Item -ItemType File -Path $reply | Out-Null }

# WSL view of the fixed journal, for the assistant loop (watch-cue.sh <path>).
# C:\Users\x\... -> /mnt/c/Users/x/...
$wsl = "/mnt/" + $log.Substring(0, 1).ToLower() + ($log.Substring(2) -replace '\\', '/')

Write-Host ""
Write-Host "  keys7 - live session" -ForegroundColor Magenta
Write-Host "  journal (assistant/WSL): " -NoNewline; Write-Host $wsl -ForegroundColor Cyan
Write-Host "  reply panel file:        " -NoNewline; Write-Host $reply -ForegroundColor DarkGray
Write-Host ""

$keysArgs = @("--source=device", "--key", $Key, "--notation", $Notation, "--log", $log, "--reply", $reply)
if ($Port) { $keysArgs += @("--port", $Port) }

& $exe @keysArgs
