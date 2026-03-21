package main

import (
	"fmt"
	"os"

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
		return fmt.Errorf("usage: slope <file.envelope>\n       slope header <file> [0-N]\n       slope payload <file> [0-N]")
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

	env, err := envelope.Parse(f)
	if err != nil {
		return err
	}

	m := tui.NewModel(env, path, fi.Size())
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}
