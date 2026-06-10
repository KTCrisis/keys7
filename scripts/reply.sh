#!/usr/bin/env bash
# Write the assistant's reply into the file keys7 polls for its TUI panel
# (--reply). Argument-based for the same allowlist reason as play.sh.
#
# Usage: reply.sh <reply-file> '<text>'
set -euo pipefail

FILE=${1:?usage: reply.sh <reply-file> '<text>'}
TEXT=${2:?usage: reply.sh <reply-file> '<text>'}

printf '%s' "$TEXT" > "$FILE"
