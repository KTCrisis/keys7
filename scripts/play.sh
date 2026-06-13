#!/usr/bin/env bash
# Play a JSON sequence on the piano via play7, passing the sequence as an
# argument instead of stdin — so callers with prefix-based command allowlists
# (e.g. Claude Code permissions) can whitelist this script without wildcarding
# arbitrary pipes.
#
# Usage: play.sh '<sequence-json>' [port-match] [style]
#   style: straight (default) | ambient | orchestral | darksynth
set -euo pipefail

SEQ=${1:?usage: play.sh '<sequence-json>' [port-match] [style]}
PORT=${2:-Digital Piano}
STYLE=${3:-straight}
BIN="$(dirname "$0")/../bin/play7.exe"

printf '%s' "$SEQ" | "$BIN" --port "$PORT" --style "$STYLE"
