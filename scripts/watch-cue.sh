#!/usr/bin/env bash
# Poll a keys7 session journal until a cue event lands, then print every line
# added since this script started and exit 0. Polling, not inotify: change
# notifications don't cross the WSL/Windows 9P mount; reads do.
#
# Usage: watch-cue.sh <journal.jsonl> [poll-seconds]
# Typical use: run in the background; the runner is notified on exit and reads
# the take from stdout.
set -euo pipefail

F=${1:?usage: watch-cue.sh <journal.jsonl> [poll-seconds]}
P=${2:-2}

BASE=$(wc -l < "$F")
while sleep "$P"; do
  N=$(wc -l < "$F")
  if [ "$N" -gt "$BASE" ] && sed -n "$((BASE + 1)),\$p" "$F" | grep -qF '"kind":"cue"'; then
    sed -n "$((BASE + 1)),\$p" "$F"
    exit 0
  fi
done
