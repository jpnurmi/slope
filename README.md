# slope

[![Go](https://img.shields.io/github/go-mod/go-version/jpnurmi/slope)](https://go.dev/)
[![CI](https://github.com/jpnurmi/slope/actions/workflows/ci.yml/badge.svg)](https://github.com/jpnurmi/slope/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jpnurmi/slope/branch/main/graph/badge.svg)](https://codecov.io/gh/jpnurmi/slope)

A TUI viewer and editor for [Sentry envelopes](https://develop.sentry.dev/sdk/foundations/data-model/envelopes/).

![screenshot](screenshot.png)

## Features

- Pretty-formatted, syntax-highlighted JSON headers
- Selectable item list with payload viewing via pager
- JSON payloads are pretty-printed and highlighted
- Binary payloads are shown as hex dump
- Add, delete, and export envelope items
- Save modified envelopes back to file

## Install

```
go install github.com/jpnurmi/slope@latest
```

Or build from source:

```
go build -o slope .
```

## Usage

```
slope <file.envelope>
```

### Key bindings

| Key | Action |
|-----|--------|
| `j` / `k` / `Up` / `Down` | Navigate items |
| `Enter` | View item payload in pager |
| `e` | Edit item payload in `$EDITOR` |
| `a` | Add attachment |
| `x` | Export item payload to file |
| `d` | Delete selected item |
| `w` | Save to file |
| `q` | Quit |
