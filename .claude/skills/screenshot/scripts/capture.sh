#!/bin/bash
set -euo pipefail

REPO_ROOT="${1:-$(cd "$(dirname "$0")/../../../.." && pwd)}"
SCRIPTS_DIR="$REPO_ROOT/.claude/skills/screenshot/scripts"

# Record pre-existing Ghostty PIDs
PRE_PIDS=$(pgrep -x ghostty 2>/dev/null || true)

# Open a clean Ghostty window running the wrapper
open -na Ghostty --args \
  --window-save-state=never \
  --window-width=124 \
  --window-height=42 \
  --command="$SCRIPTS_DIR/run.sh"

sleep 3

# Find the slope window by title and capture screenshot
read -r WINDOW_ID OWNER_PID < <(python3 -c "
import Quartz
windows = Quartz.CGWindowListCopyWindowInfo(Quartz.kCGWindowListOptionOnScreenOnly, Quartz.kCGNullWindowID)
for w in windows:
    if w.get('kCGWindowName') == 'slope breakpad.envelope':
        print(w['kCGWindowNumber'], w['kCGWindowOwnerPID'])
        break
")

screencapture -l "$WINDOW_ID" "$REPO_ROOT/screenshot.png"

# Kill the new Ghostty process only if it wasn't pre-existing
if ! echo "$PRE_PIDS" | grep -qw "$OWNER_PID"; then
  kill "$OWNER_PID"
fi

# Clean up
rm -f /tmp/slope /tmp/breakpad.envelope
