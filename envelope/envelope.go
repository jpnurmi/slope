package envelope

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type Envelope struct {
	Header json.RawMessage
	Items  []Item
}

type Item struct {
	Header   json.RawMessage
	Payload  []byte
	Type     string
	Filename string
}

func Parse(r io.Reader) (*Envelope, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading envelope: %w", err)
	}

	buf := bytes.NewBuffer(data)
	env := &Envelope{}

	// First line: envelope header
	line, err := buf.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading envelope header: %w", err)
	}
	line = bytes.TrimRight(line, "\n")
	if len(line) == 0 {
		return nil, fmt.Errorf("empty envelope header")
	}
	if !json.Valid(line) {
		return nil, fmt.Errorf("invalid JSON in envelope header")
	}
	env.Header = json.RawMessage(line)

	// Parse items
	for buf.Len() > 0 {
		// Read item header line
		headerLine, err := buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("reading item header: %w", err)
		}
		headerLine = bytes.TrimRight(headerLine, "\n")
		if len(headerLine) == 0 {
			continue // skip empty lines (e.g., trailing newline)
		}
		if !json.Valid(headerLine) {
			return nil, fmt.Errorf("invalid JSON in item header: %s", headerLine)
		}

		item := Item{
			Header: json.RawMessage(headerLine),
		}

		// Extract type, length, filename from header
		var hdr struct {
			Type     string `json:"type"`
			Length   *int   `json:"length"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(headerLine, &hdr); err != nil {
			return nil, fmt.Errorf("parsing item header: %w", err)
		}
		item.Type = hdr.Type
		item.Filename = hdr.Filename

		// Read payload
		if hdr.Length != nil {
			length := *hdr.Length
			if buf.Len() < length {
				return nil, fmt.Errorf("payload truncated: expected %d bytes, got %d", length, buf.Len())
			}
			item.Payload = make([]byte, length)
			if _, err := io.ReadFull(buf, item.Payload); err != nil {
				return nil, fmt.Errorf("reading payload: %w", err)
			}
			// Consume trailing newline
			if buf.Len() > 0 {
				b, _ := buf.ReadByte()
				if b != '\n' {
					buf.UnreadByte()
				}
			}
		} else {
			// Read until newline
			payload, err := buf.ReadBytes('\n')
			if err != nil && err != io.EOF {
				return nil, fmt.Errorf("reading payload: %w", err)
			}
			item.Payload = bytes.TrimRight(payload, "\n")
		}

		env.Items = append(env.Items, item)
	}

	return env, nil
}

func (env *Envelope) Serialize(w io.Writer) error {
	bw := bufio.NewWriter(w)

	// Write envelope header (compact)
	compact, err := compactJSON(env.Header)
	if err != nil {
		return fmt.Errorf("compacting envelope header: %w", err)
	}
	bw.Write(compact)
	bw.WriteByte('\n')

	// Write items
	for _, item := range env.Items {
		// Update length in header
		header, err := UpdateLength(item.Header, len(item.Payload))
		if err != nil {
			return fmt.Errorf("updating item header: %w", err)
		}
		compact, err := compactJSON(header)
		if err != nil {
			return fmt.Errorf("compacting item header: %w", err)
		}
		bw.Write(compact)
		bw.WriteByte('\n')
		bw.Write(item.Payload)
		bw.WriteByte('\n')
	}

	return bw.Flush()
}

func IsBinary(data []byte) bool {
	if !utf8.Valid(data) {
		return true
	}
	for _, b := range data {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			return true
		}
	}
	return false
}

func PrettyJSON(data json.RawMessage) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return string(data)
	}
	return buf.String()
}

func OneLineJSON(data json.RawMessage) string {
	pretty := PrettyJSON(data)
	result := strings.Join(strings.Fields(pretty), " ")
	// Restore spaces after colons
	result = strings.ReplaceAll(result, ": ", ": ")
	return result
}

func compactJSON(data json.RawMessage) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UpdateLength(header json.RawMessage, length int) (json.RawMessage, error) {
	om := orderedmap.New[string, any]()
	if err := json.Unmarshal(header, om); err != nil {
		return nil, err
	}
	om.Set("length", length)
	return json.Marshal(om)
}
