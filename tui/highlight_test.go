package tui

import "testing"

func TestHighlightJSON(t *testing.T) {
	input := `{ "key": "value" }`
	output := highlightJSON(input)
	if output == "" {
		t.Error("output is empty")
	}
	if output == input {
		t.Error("output should differ from input (ANSI codes added)")
	}
}
