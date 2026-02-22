package tui

import (
	"testing"

	"github.com/getsentry/slope/envelope"
)

func TestItemLabel(t *testing.T) {
	tests := []struct {
		name string
		idx  int
		item envelope.Item
		want string
	}{
		{
			"basic",
			0,
			envelope.Item{Type: "event", Payload: make([]byte, 100)},
			"1. EVENT · 100 B",
		},
		{
			"with filename",
			2,
			envelope.Item{Type: "attachment", Payload: make([]byte, 50), Filename: "test.bin"},
			"3. ATTACHMENT · 50 B · test.bin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := itemLabel(tt.idx, tt.item)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		in   int
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.in)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
