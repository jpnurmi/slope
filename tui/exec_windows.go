//go:build windows

package tui

import "os/exec"

const defaultPager = "" // use built-in pager
const defaultEditor = "notepad"

// pagerCommand wraps a pager command string via cmd.exe /c.
func pagerCommand(cmdline string) *exec.Cmd {
	return exec.Command("cmd.exe", "/c", cmdline)
}

// editorCommand wraps an editor command string via cmd.exe /c, appending
// the file path as a quoted argument.
func editorCommand(cmdline, filePath string) *exec.Cmd {
	return exec.Command("cmd.exe", "/c", cmdline, filePath)
}
