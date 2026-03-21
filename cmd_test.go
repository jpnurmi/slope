package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

const testEnvelope = "envelope/testdata/breakpad.envelope"

func TestRunNoArgs(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestRunHeader(t *testing.T) {
	out := captureStdout(t, func() {
		if err := run([]string{"header", testEnvelope}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, `"dsn"`) {
		t.Errorf("expected dsn in output, got %q", out[:min(len(out), 100)])
	}
}

func TestRunPayload(t *testing.T) {
	out := captureStdout(t, func() {
		if err := run([]string{"payload", testEnvelope, "1"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "event_id") {
		t.Errorf("expected event payload, got %q", out[:min(len(out), 100)])
	}
}

func TestRunTUIBadFile(t *testing.T) {
	if err := run([]string{"nonexistent"}); err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestCmdHeaderAll(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdHeader([]string{testEnvelope}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines (1 envelope + 4 items), got %d: %s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "0: ") || !strings.Contains(lines[0], `"dsn"`) {
		t.Errorf("expected envelope header at index 0, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "1: ") {
		t.Errorf("expected line to start with '1: ', got %q", lines[1])
	}
	if !strings.Contains(lines[3], `"filename":"minidump.dmp"`) {
		t.Errorf("expected filename in item 3, got %q", lines[3])
	}
}

func TestCmdHeaderEnvelopeHeader(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdHeader([]string{testEnvelope, "0"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, `"dsn"`) {
		t.Errorf("expected dsn in envelope header, got %q", out)
	}
}

func TestCmdHeaderByIndex(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdHeader([]string{testEnvelope, "1"}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected single line, got %d lines", len(lines))
	}
	if !strings.Contains(out, `"type":"event"`) {
		t.Errorf("expected event type in header, got %q", out)
	}
}

func TestCmdHeaderOutOfRange(t *testing.T) {
	if err := cmdHeader([]string{testEnvelope, "99"}); err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestCmdHeaderMissingFile(t *testing.T) {
	if err := cmdHeader([]string{"nonexistent"}); err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCmdHeaderNegativeIndex(t *testing.T) {
	if err := cmdHeader([]string{testEnvelope, "-1"}); err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestCmdHeaderInvalidIndex(t *testing.T) {
	if err := cmdHeader([]string{testEnvelope, "abc"}); err == nil {
		t.Fatal("expected error for non-numeric index")
	}
}

func TestCmdHeaderNoArgs(t *testing.T) {
	if err := cmdHeader(nil); err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestCmdHeaderTooManyArgs(t *testing.T) {
	if err := cmdHeader([]string{testEnvelope, "0", "extra"}); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestCmdPayload(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdPayload([]string{testEnvelope, "1"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "event_id") {
		t.Errorf("expected event payload, got %q", out[:min(len(out), 100)])
	}
}

func TestCmdPayloadOutOfRange(t *testing.T) {
	if err := cmdPayload([]string{testEnvelope, "99"}); err == nil {
		t.Fatal("expected error for out-of-range index")
	}
}

func TestCmdPayloadSummary(t *testing.T) {
	out := captureStdout(t, func() {
		if err := cmdPayload([]string{testEnvelope}); err != nil {
			t.Fatal(err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "0: <") {
		t.Errorf("expected envelope summary at index 0, got %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "1: ") {
		t.Errorf("expected line to start with '1: ', got %q", lines[1])
	}
	if !strings.Contains(lines[3], "<binary") {
		t.Errorf("expected binary marker in line 4, got %q", lines[3])
	}
}

func TestCmdPayloadEnvelope(t *testing.T) {
	env, err := parseEnvelopeFile(testEnvelope)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := env.SerializeItems(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"type":"event"`) {
		t.Error("expected item headers in serialized items")
	}
}

func TestCmdPayloadNegativeIndex(t *testing.T) {
	if err := cmdPayload([]string{testEnvelope, "-1"}); err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestCmdPayloadMissingFile(t *testing.T) {
	if err := cmdPayload([]string{"nonexistent", "0"}); err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCmdPayloadTooManyArgs(t *testing.T) {
	if err := cmdPayload([]string{testEnvelope, "0", "extra"}); err == nil {
		t.Fatal("expected error for too many args")
	}
}

func TestCmdPayloadInvalidIndex(t *testing.T) {
	if err := cmdPayload([]string{testEnvelope, "abc"}); err == nil {
		t.Fatal("expected error for non-numeric index")
	}
}

func TestCmdPayloadNoArgs(t *testing.T) {
	if err := cmdPayload(nil); err == nil {
		t.Fatal("expected error for no args")
	}
}
