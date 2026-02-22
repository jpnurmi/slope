---
user-invocable: true
description: Take a screenshot of slope for the README
---

# Screenshot

Take a screenshot of slope running in a fresh Ghostty terminal window.

## Steps

1. Build and copy files to /tmp:
   ```
   go build -o /tmp/slope . && cp envelope/testdata/breakpad.envelope /tmp/breakpad.envelope
   ```
2. Capture the screenshot:
   ```
   .claude/skills/screenshot/scripts/capture.sh
   ```
3. Verify the result by reading `screenshot.png`.
