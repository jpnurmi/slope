# Changelog

## 0.2.0

### Features

- Render image attachments in the payload viewer (#8)
- Render minidump attachments in the payload viewer (#9)
- Parse and visualize minidump stacktraces (#11)
- Demangle C++ symbols in stacktraces (#12)

### Dependencies

- Bump goreleaser/goreleaser-action from 6 to 7 (#4)
- Bump charm.land/bubbles/v2 from 2.0.0-rc.1 to 2.0.0 (#5)
- Bump charm.land/lipgloss/v2 from 2.0.0-beta.3.0.20251106192539-4b304240aab7 to 2.0.2 (#6)
- Bump charm.land/bubbletea/v2 from 2.0.0-rc.2 to 2.0.2 (#7)

## 0.1.0

### Features

- Interactive TUI for viewing and editing Sentry envelopes
  - Pretty-formatted, syntax-highlighted JSON headers and payloads
  - Binary payloads shown as hex dump
  - Add, edit, delete, and export envelope items
  - Save modified envelopes back to file
- Non-interactive CLI (`slope header`, `slope payload`) for scripting and agents
- Windows, macOS, and Linux support
