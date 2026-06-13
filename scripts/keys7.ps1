# keys7.ps1 - launch a live keys7 session on a connected MIDI piano.
#
# Encodes the session defaults (device source, auto key) and, crucially, opens a
# fresh timestamped journal under sessions\ each run - so takes accumulate
# instead of overwriting - and prints that journal's WSL path, which is what the
# assistant side (watch-cue.sh) needs to follow along.
#
# Meant to be launched by the Desktop shortcut (which runs it in a PowerShell
# console so the TUI renders), but works from any terminal:
#   .\keys7.ps1                       # device, auto key, letters
#   .\keys7.ps1 -Key Am -Notation solfege
#   .\keys7.ps1 -Port "P-125"         # match a specific input by name

param(
    [string]$Key      = "auto",        # C | Am | F#m | auto | drone
    [string]$Port     = "",            # input name match; empty = first input
    [string]$Notation = "letters"      # letters | solfege
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

$exe = Join-Path $root "keys7.exe"
if (-not (Test-Path $exe)) {
    throw "keys7.exe not found next to this script ($root). Run 'make install-windows' from the repo."
}

# One journal per session, timestamped; reply file is reused (last reply shown).
$sessions = Join-Path $root "sessions"
New-Item -ItemType Directory -Force -Path $sessions | Out-Null
$stamp = Get-Date -Format "yyyy-MM-dd_HHmm"
$log   = Join-Path $sessions "$stamp.jsonl"
$reply = Join-Path $sessions "reply.txt"
if (-not (Test-Path $reply)) { New-Item -ItemType File -Path $reply | Out-Null }

# WSL view of the journal, for the assistant loop (watch-cue.sh <this path>).
# C:\Users\x\... -> /mnt/c/Users/x/...
$wsl = "/mnt/" + $log.Substring(0,1).ToLower() + ($log.Substring(2) -replace '\\','/')

Write-Host ""
Write-Host "  keys7 - live session" -ForegroundColor Magenta
Write-Host "  journal (assistant/WSL): " -NoNewline; Write-Host $wsl -ForegroundColor Cyan
Write-Host "  reply panel file:        " -NoNewline; Write-Host $reply -ForegroundColor DarkGray
Write-Host ""

$keysArgs = @("--source=device", "--key", $Key, "--notation", $Notation, "--log", $log, "--reply", $reply)
if ($Port) { $keysArgs += @("--port", $Port) }

& $exe @keysArgs
