package tui

import "encoding/hex"

func hexDump(data []byte) string {
	return hex.Dump(data)
}
