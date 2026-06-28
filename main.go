package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/slope/envelope"
	"github.com/getsentry/slope/tui"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: slope <file>\n       slope header <file> [0-N]\n       slope payload <file> [0-N]")
	}
	switch args[0] {
	case "header":
		return cmdHeader(args[1:])
	case "payload":
		return cmdPayload(args[1:])
	default:
		return runTUI(args[0])
	}
}

func runTUI(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if strings.HasSuffix(strings.ToLower(path), ".dmp") {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m, err := tui.NewMinidumpViewer(data, path, fi.Size())
		if err != nil {
			return err
		}
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	}

	if strings.HasSuffix(strings.ToLower(path), ".json") {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m, err := tui.NewJSONViewer(data, path, fi.Size())
		if err != nil {
			return err
		}
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		m, err := tui.NewImageViewer(data, path, fi.Size())
		if err != nil {
			return err
		}
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	}

	env, err := envelope.Parse(f)
	if err != nil {
		return err
	}

	m := tui.NewModel(env, path, fi.Size())
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
