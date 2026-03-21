package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/slope/envelope"
	"github.com/getsentry/slope/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: slope <file.envelope>\n")
		fmt.Fprintf(os.Stderr, "       slope header <file> [0-N]\n")
		fmt.Fprintf(os.Stderr, "       slope payload <file> [0-N]\n")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "header":
		err = cmdHeader(os.Args[2:])
	case "payload":
		err = cmdPayload(os.Args[2:])
	default:
		err = runTUI(os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
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
