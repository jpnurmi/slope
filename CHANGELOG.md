# Changelog

## 0.3.2

### Features

- Recognize well-known stream types (#36)

### Dependencies

- Bump the charm group with 3 updates (#31)
- Bump golang.org/x/image from 0.37.0 to 0.44.0 (#34)
- Bump actions/setup-go from 6 to 7 (#35)
- Bump the charm group across 1 directory with 2 updates (#33)
- Bump charm.land/lipgloss/v2 from 2.0.5 to 2.0.6 in the charm group (#37)
- Bump golang.org/x/image from 0.44.0 to 0.45.0 (#38)

## 0.3.1

### Fixes

- Update module path for "go install" compatibility (#29)

## 0.3.0

### Features

- Standalone minidump viewer mode (#23)
- Standalone JSON viewer mode (#24)
- Standalone image viewer mode (#25)
- Standalone text/binary viewer mode (#26)

### Dependencies

- Bump actions/checkout from 6 to 7 (#21)
- Bump codecov/codecov-action from 5 to 7 (#20)
- Bump the charm group across 1 directory with 3 updates (#17)
- Bump github.com/alecthomas/chroma/v2 from 2.23.1 to 2.27.0 (#22)

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
