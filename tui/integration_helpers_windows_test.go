//go:build windows

package tui

import (
	"os"
	"testing"
)

const testPagerOK = "exit 0"
const testPagerFail = "exit 1"
const testEditorNoop = "rem"
const testEditorFail = "exit 1"

// createTestEditorScript creates a batch script that copies a pre-written
// content file over the file path passed as the first argument.
func createTestEditorScript(t *testing.T, content string) string {
	t.Helper()

	// Write the desired content to a temporary file to avoid batch escaping issues.
	contentFile, err := os.CreateTemp("", "slope-content-*")
	if err != nil {
		t.Fatal(err)
	}
	contentFile.WriteString(content)
	contentFile.Close()
	t.Cleanup(func() { os.Remove(contentFile.Name()) })

	script, err := os.CreateTemp("", "slope-editor-*.bat")
	if err != nil {
		t.Fatal(err)
	}
	// copy /y overwrites without prompting; %~1 strips surrounding quotes
	script.WriteString("@echo off\r\ncopy /y \"" + contentFile.Name() + "\" %~1 >nul\r\n")
	script.Close()
	return script.Name()
}
