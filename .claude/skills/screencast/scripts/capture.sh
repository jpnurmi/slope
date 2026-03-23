#!/bin/bash
set -euo pipefail

REPO_ROOT="${1:-$(cd "$(dirname "$0")/../../../.." && pwd)}"
SCRIPTS_DIR="$REPO_ROOT/.claude/skills/screencast/scripts"
MOV_FILE="$REPO_ROOT/screencast.mov"
GIF_FILE="$REPO_ROOT/screencast.gif"

send_keys() {
  local script='tell application "Ghostty" to activate
delay 0.1
tell application "System Events"
'
  for k in "$@"; do
    if [ "$k" = "enter" ]; then
      script+='  key code 36
'
    else
      script+='  keystroke "'"$k"'"
'
    fi
  done
  script+='end tell'
  osascript -e "$script"
}

scroll() {
  local key=$1 n=$2
  local script='tell application "Ghostty" to activate
delay 0.1
tell application "System Events"
'
  for _ in $(seq 1 "$n"); do
    script+='  keystroke "'"$key"'"
  delay 0.03
'
  done
  script+='end tell'
  osascript -e "$script"
}

# Record pre-existing Ghostty PIDs
PRE_PIDS=$(pgrep -x ghostty 2>/dev/null || true)

# Open a clean Ghostty window running the wrapper
open -na Ghostty --args \
  --window-save-state=never \
  --window-width=128 \
  --window-height=42 \
  --command="$SCRIPTS_DIR/run.sh"

sleep 3

# Find the slope window by title and capture its ID
read -r WINDOW_ID OWNER_PID < <(python3 -c "
import Quartz
windows = Quartz.CGWindowListCopyWindowInfo(Quartz.kCGWindowListOptionOnScreenOnly, Quartz.kCGNullWindowID)
for w in windows:
    if w.get('kCGWindowName') == 'slope breakpad.envelope':
        print(w['kCGWindowNumber'], w['kCGWindowOwnerPID'])
        break
")

# Move cursor off the Ghostty window
python3 -c "
import Quartz
Quartz.CGWarpMouseCursorPosition((0, 0))
"

# Start recording (25 seconds, silent)
screencapture -v -V 30 -l "$WINDOW_ID" -x "$MOV_FILE" &
RECORD_PID=$!
sleep 1

# Re-activate Ghostty after screencapture may have stolen focus
osascript -e 'tell application "Ghostty" to activate'

# Move cursor off-screen again (reactivation can pull it back)
python3 -c "
import Quartz
Quartz.CGWarpMouseCursorPosition((0, 0))
"

# Navigate the TUI
# 1. Browse the list to orient the viewer
sleep 0.5
osascript -e 'tell application "Ghostty" to activate' -e 'delay 0.1' -e '
tell application "System Events"
  keystroke "j"
  delay 0.2
  keystroke "j"
  delay 0.2
  keystroke "j"
  delay 0.2
  keystroke "k"
  delay 0.2
  keystroke "k"
  delay 0.2
  keystroke "k"
end tell'
sleep 0.3

# 2. View event (item 1 — now selected)
send_keys enter
sleep 0.5
scroll j 25
sleep 0.3
scroll k 25
sleep 0.3

# 3. Back to list
send_keys q
sleep 0.5

# 4. Move to minidump (item 3: down twice)
osascript -e 'tell application "Ghostty" to activate' -e 'delay 0.1' -e '
tell application "System Events"
  keystroke "j"
  delay 0.2
  keystroke "j"
end tell'
sleep 0.3

# 5. View minidump
send_keys enter
sleep 0.5
scroll j 120
sleep 0.3
scroll k 120
sleep 0.3

# 6. Back to list
send_keys q
sleep 0.5

# 7. Move to image (item 4: down once)
send_keys j
sleep 0.3

# 8. View image
send_keys enter
sleep 2

# 9. Back to list, navigate to first item for seamless loop
send_keys q || true
sleep 0.3
osascript -e 'tell application "Ghostty" to activate' -e 'delay 0.1' -e '
tell application "System Events"
  keystroke "k"
  delay 0.2
  keystroke "k"
  delay 0.2
  keystroke "k"
end tell' || true
sleep 0.5

# Wait for recording to finish
wait "$RECORD_PID" || true

# Convert to GIF: scale to logical pixels (half Retina), optimized palette
ffmpeg -y -i "$MOV_FILE" \
  -filter_complex "[0:v] fps=15,scale=iw/2:ih/2:flags=lanczos,split [a][b]; [a] palettegen=max_colors=128:stats_mode=diff [p]; [b][p] paletteuse=dither=bayer:bayer_scale=3" \
  -loop 0 \
  "$GIF_FILE"

# Kill the new Ghostty process only if it wasn't pre-existing
if ! echo "$PRE_PIDS" | grep -qw "$OWNER_PID"; then
  kill "$OWNER_PID"
fi

# Clean up
rm -f /tmp/slope /tmp/breakpad.envelope "$MOV_FILE"
