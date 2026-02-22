---
user-invocable: true
description: Take a screenshot of slope for the README
---

# Screenshot

Take a screenshot of slope running in a fresh Ghostty terminal window.

## Steps

1. Build and copy files to /tmp:
   ```
   go build -o /tmp/slope .
   cp envelope/testdata/breakpad.envelope /tmp/breakpad.envelope
   ```
2. Create a wrapper script at `/tmp/slope-screenshot.sh`:
   ```bash
   #!/bin/bash
   echo -ne "\033]0;slope breakpad.envelope\007"
   export PS1="$ "
   clear
   echo "$ slope breakpad.envelope"
   exec /tmp/slope /tmp/breakpad.envelope
   ```
   Make it executable: `chmod +x /tmp/slope-screenshot.sh`
3. Open a clean Ghostty window running the wrapper with a preset size:
   ```
   open -na Ghostty --args --window-save-state=never --window-width=160 --window-height=50 --command=/tmp/slope-screenshot.sh
   ```
4. Sleep 3 seconds for the TUI to render: `sleep 3`
5. Find the new Ghostty window ID via CoreGraphics and capture the screenshot.
   The new instance will have a different PID from the main Ghostty process.
   ```
   python3 -c "
   import Quartz
   windows = Quartz.CGWindowListCopyWindowInfo(Quartz.kCGWindowListOptionOnScreenOnly, Quartz.kCGNullWindowID)
   ghostty = [(w['kCGWindowNumber'], w['kCGWindowOwnerPID']) for w in windows
              if 'ghostty' in w.get('kCGWindowOwnerName', '').lower() and w.get('kCGWindowLayer', -1) == 0]
   # find the smallest PID group (main Ghostty) and pick the other one
   from collections import Counter
   pid_counts = Counter(pid for _, pid in ghostty)
   main_pid = pid_counts.most_common()[-1][0] if len(pid_counts) > 1 else None
   for wid, pid in ghostty:
       if pid != main_pid or main_pid is None:
           print(wid)
           break
   "
   ```
   Then: `screencapture -l <WINDOW_ID> screenshot.png`
6. Close the slope Ghostty process: `kill <PID>`
7. Clean up: `rm /tmp/slope /tmp/breakpad.envelope /tmp/slope-screenshot.sh`

## Notes

- Each step must be run separately. Wait for confirmation before proceeding to the next.
- The resulting `screenshot.png` is in the repo root.
- IMPORTANT: Never resize or interact with the main Ghostty window (the one running Claude Code).
  Always identify the new instance by its different PID.
