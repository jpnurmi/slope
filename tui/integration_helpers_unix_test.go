//go:build !windows

package tui

import (
	"os"
	"testing"
)

const testPagerOK = "true"
const testPagerFail = "false"
const testEditorNoop = "true"
const testEditorFail = "false"

// createTestEditorScript creates a shell script that writes content to the
// file path passed as the first argument.
func createTestEditorScript(t *testing.T, content string) string {
	t.Helper()
	script, err := os.CreateTemp("", "slope-editor-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	script.WriteString("#!/bin/sh\nprintf '" + content + "' > \"$1\"\n")
	script.Close()
	os.Chmod(script.Name(), 0o755)
	return script.Name()
}
