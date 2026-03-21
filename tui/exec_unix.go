//go:build !windows

package tui

import "os/exec"

const defaultPager = "less -R"
const defaultEditor = "nano"

// pagerCommand wraps a pager command string via sh -c.
func pagerCommand(cmdline string) *exec.Cmd {
	return exec.Command("sh", "-c", cmdline+" --")
}

// editorCommand wraps an editor command string via sh -c, safely passing
// the file path as a positional parameter.
func editorCommand(cmdline, filePath string) *exec.Cmd {
	return exec.Command("sh", "-c", cmdline+` "$1"`, "--", filePath)
}
