package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getsentry/slope/envelope"
)

func itemLabel(idx int, item envelope.Item) string {
	parts := []string{fmt.Sprintf("%d. %s", idx+1, strings.ToUpper(item.Type))}
	parts = append(parts, formatSize(len(item.Payload)))
	if item.Filename != "" {
		parts = append(parts, item.Filename)
	}
	return strings.Join(parts, " · ")
}

func formatHeader(header json.RawMessage) string {
	if envelope.JSONFieldCount(header) <= 2 {
		return highlightJSON(envelope.OneLineJSON(header))
	}
	return highlightJSON(envelope.PrettyJSON(header))
}

func formatSize(b int) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}
