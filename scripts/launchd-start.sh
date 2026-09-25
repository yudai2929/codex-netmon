#!/bin/sh
set -eu

mkdir -p .run
label=io.github.yudai2929.codex-netmon
domain="gui/$(id -u)"
if launchctl print "$domain/$label" >/dev/null 2>&1; then
  echo "Codex network monitor is already running"
  exit 0
fi

go build -o .run/codex-netmon .
workspace=$(pwd -P)
cat > .run/netmon.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key><array><string>$workspace/.run/codex-netmon</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>ThrottleInterval</key><integer>10</integer>
  <key>StandardOutPath</key><string>$workspace/.run/netmon.log</string>
  <key>StandardErrorPath</key><string>$workspace/.run/netmon.err.log</string>
</dict></plist>
EOF
launchctl bootstrap "$domain" "$workspace/.run/netmon.plist"
echo "codex-netmon started with launchd"
