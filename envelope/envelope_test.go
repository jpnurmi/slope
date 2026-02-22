package envelope

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEmpty(t *testing.T) {
	input := `{}` + "\n"
	env, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(env.Items))
	}
}

func TestParseWithItems(t *testing.T) {
	input := `{"event_id":"abc123"}` + "\n" +
		`{"type":"event","length":27}` + "\n" +
		`{"message":"hello world!!"}` + "\n" +
		`{"type":"attachment","length":4,"filename":"test.bin"}` + "\n" +
		"\x00\x01\x02\x03\n"

	env, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(env.Items))
	}

	if env.Items[0].Type != "event" {
		t.Errorf("item 0 type = %q, want %q", env.Items[0].Type, "event")
	}
	if string(env.Items[0].Payload) != `{"message":"hello world!!"}` {
		t.Errorf("item 0 payload = %q", string(env.Items[0].Payload))
	}

	if env.Items[1].Type != "attachment" {
		t.Errorf("item 1 type = %q, want %q", env.Items[1].Type, "attachment")
	}
	if env.Items[1].Filename != "test.bin" {
		t.Errorf("item 1 filename = %q, want %q", env.Items[1].Filename, "test.bin")
	}
	if len(env.Items[1].Payload) != 4 {
		t.Errorf("item 1 payload length = %d, want 4", len(env.Items[1].Payload))
	}
}

func TestParseNoLength(t *testing.T) {
	input := `{}` + "\n" +
		`{"type":"session"}` + "\n" +
		`{"sid":"abc","status":"ok"}` + "\n"

	env, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(env.Items))
	}
	if string(env.Items[0].Payload) != `{"sid":"abc","status":"ok"}` {
		t.Errorf("payload = %q", string(env.Items[0].Payload))
	}
}

func TestRoundTrip(t *testing.T) {
	input := `{"event_id":"abc123"}` + "\n" +
		`{"type":"event","length":27}` + "\n" +
		`{"message":"hello world!!"}` + "\n" +
		`{"type":"attachment","length":4,"filename":"test.bin"}` + "\n" +
		"\x00\x01\x02\x03\n"

	env, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	var buf bytes.Buffer
	if err := env.Serialize(&buf); err != nil {
		t.Fatalf("serialize error: %v", err)
	}

	env2, err := Parse(&buf)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}

	if len(env2.Items) != len(env.Items) {
		t.Fatalf("item count mismatch: %d vs %d", len(env2.Items), len(env.Items))
	}

	for i := range env.Items {
		if env2.Items[i].Type != env.Items[i].Type {
			t.Errorf("item %d type mismatch", i)
		}
		if !bytes.Equal(env2.Items[i].Payload, env.Items[i].Payload) {
			t.Errorf("item %d payload mismatch", i)
		}
	}
}

func TestIsBinary(t *testing.T) {
	if IsBinary([]byte(`{"hello":"world"}`)) {
		t.Error("JSON should not be binary")
	}
	if IsBinary([]byte("hello\nworld\t!")) {
		t.Error("text with tabs/newlines should not be binary")
	}
	if !IsBinary([]byte{0x00, 0x01, 0x02}) {
		t.Error("control chars should be binary")
	}
	if !IsBinary([]byte{0xff, 0xfe}) {
		t.Error("invalid UTF-8 should be binary")
	}
}

func TestParseTestdata(t *testing.T) {
	tests := []struct {
		file      string
		wantErr   bool
		itemCount int
		types     []string
	}{
		{"binary_attachment.envelope", false, 1, []string{"attachment"}},
		{"breakpad.envelope", false, 4, []string{"event", "session", "attachment", "attachment"}},
		{"empty_headers_eof.envelope", false, 1, []string{"session"}},
		{"implicit_length.envelope", false, 1, []string{"attachment"}},
		{"inproc.envelope", false, 2, []string{"event", "session"}},
		{"invalid.envelope", true, 0, nil},
		{"sigsegv.envelope", false, 1, []string{"event"}},
		{"two_empty_attachments.envelope", false, 2, []string{"attachment", "attachment"}},
		{"two_items.envelope", false, 2, []string{"attachment", "event"}},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", tt.file))
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}

			env, err := Parse(bytes.NewReader(data))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(env.Items) != tt.itemCount {
				t.Fatalf("item count = %d, want %d", len(env.Items), tt.itemCount)
			}
			for i, wantType := range tt.types {
				if env.Items[i].Type != wantType {
					t.Errorf("item %d type = %q, want %q", i, env.Items[i].Type, wantType)
				}
			}

			// Round-trip: Serialize → Parse
			var buf bytes.Buffer
			if err := env.Serialize(&buf); err != nil {
				t.Fatalf("serialize error: %v", err)
			}

			env2, err := Parse(&buf)
			if err != nil {
				t.Fatalf("re-parse error: %v", err)
			}
			if len(env2.Items) != len(env.Items) {
				t.Fatalf("round-trip item count = %d, want %d", len(env2.Items), len(env.Items))
			}
			for i := range env.Items {
				if env2.Items[i].Type != env.Items[i].Type {
					t.Errorf("round-trip item %d type = %q, want %q", i, env2.Items[i].Type, env.Items[i].Type)
				}
				if !bytes.Equal(env2.Items[i].Payload, env.Items[i].Payload) {
					t.Errorf("round-trip item %d payload mismatch (len %d vs %d)", i, len(env2.Items[i].Payload), len(env.Items[i].Payload))
				}
			}
		})
	}
}

func TestPrettyJSON(t *testing.T) {
	raw := json.RawMessage(`{"a":1,"b":"hello"}`)
	pretty := PrettyJSON(raw)
	expected := "{\n  \"a\": 1,\n  \"b\": \"hello\"\n}"
	if pretty != expected {
		t.Errorf("got:\n%s\nwant:\n%s", pretty, expected)
	}
}
