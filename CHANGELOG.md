# Changelog

## 0.2.0

### Dependencies

- Bump goreleaser/goreleaser-action from 6 to 7 (#4)
- Bump charm.land/bubbles/v2 from 2.0.0-rc.1 to 2.0.0 (#5)
- Bump charm.land/lipgloss/v2 from 2.0.0-beta.3.0.20251106192539-4b304240aab7 to 2.0.2 (#6)
- Bump charm.land/bubbletea/v2 from 2.0.0-rc.2 to 2.0.2 (#7)

### Features

- Render image attachments in the payload viewer (#8)
- Minidump (#9)
- Parse and visualize minidump stacktraces (#11)

### Fixes

- Adapt viewText to bubbletea v2 View.Content string type
- Skip publish job for non-release PR merges
## 0.1.0

### Features

- Initial sentry envelope viewer
- Edit item payload in external editor
- Add /screenshot skill for automated README screenshots
- Pretty-format JSON payloads for editing
- Add x key to export item payload to file
- Add Windows support (#1)
- Add non-interactive CLI (#2)

### Fixes

- Use tea.Println for scrollable envelope dump
- Use terminal width for JSON single/multi-line formatting
- Disable edit action for binary payloads
- Reorder help bar actions to add, edit, delete
- Disable save action when envelope is not modified
- Remove no-op ReplaceAll in OneLineJSON
- Shell-parse $EDITOR for multi-word values
- Respect $PAGER and handle pager errors
- Warn before quitting with unsaved changes
- Propagate json.Marshal error in addAttachment
- Use json.Valid for temp file extension in editor
- Use atomic write to prevent data loss on save
- Preserve whitespace in JSON strings in OneLineJSON
- Update fileSize after save
- Clamp file picker height to minimum of 1

### Improvements

- Update README
- Consolidate screenshot skill into capture script
- Add export action to README
- Remove unused IsCompactJSON
- Inline hexDump wrapper

### Release

- V0.1.0 (#3)
