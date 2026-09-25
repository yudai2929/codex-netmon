#!/bin/sh
set -eu

label=io.github.yudai2929.codex-netmon
domain="gui/$(id -u)"
if launchctl print "$domain/$label" >/dev/null 2>&1; then
  launchctl bootout "$domain/$label"
fi
