---
user-invocable: true
description: Record a screencast GIF of slope for the README
---

# Screencast

Record a short screencast of slope navigating through different item types
in a fresh Ghostty terminal window. The recording demonstrates viewing an
event (JSON), a minidump (structured data), and an image attachment
(half-block art).

## Steps

1. Build and copy files to /tmp:
   ```
   go build -o /tmp/slope . && cp envelope/testdata/breakpad.envelope /tmp/breakpad.envelope
   ```
2. Record the screencast:
   ```
   .claude/skills/screencast/scripts/capture.sh
   ```
3. Verify the result by reading `screencast.gif`.
