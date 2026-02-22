package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/slope/envelope"
)

// execDoneMsg is a sentinel sent after the exec-triggering key. Since
// ExecProcess blocks the event loop, this message is queued and delivered
// only after the external process completes.
type execDoneMsg struct{}

// testHarness wraps Model to quit deterministically after exec completes.
type testHarness struct {
	Model
}

func (h *testHarness) Init() tea.Cmd {
	return h.Model.Init()
}

func (h *testHarness) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(editResultMsg); ok {
		next, cmd := h.Model.Update(msg)
		h.Model = next.(Model)
		return h, tea.Batch(cmd, tea.Quit)
	}
	if _, ok := msg.(execDoneMsg); ok {
		return h, tea.Quit
	}
	next, cmd := h.Model.Update(msg)
	h.Model = next.(Model)
	return h, cmd
}

func (h *testHarness) View() tea.View {
	return h.Model.View()
}

func newTestProgram(m tea.Model) *tea.Program {
	r, w, _ := os.Pipe()
	w.Close()
	return tea.NewProgram(m, tea.WithInput(r), tea.WithoutRenderer())
}

func integrationModel(payload []byte) Model {
	env := &envelope.Envelope{
		Header: json.RawMessage(`{"sdk":{"name":"test"}}`),
		Items: []envelope.Item{{
			Header:  json.RawMessage(fmt.Sprintf(`{"type":"event","length":%d}`, len(payload))),
			Payload: payload,
			Type:    "event",
		}},
	}
	return NewModel(env, "test.envelope", 0)
}

func TestIntegrationPager(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	tests := []struct {
		name    string
		payload []byte
	}{
		{"json", []byte(`{"key":"value"}`)},
		{"binary", []byte{0x00, 0x01, 0x02}},
		{"empty", []byte{}},
		{"text", []byte("hello world")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PAGER", "true")
			h := &testHarness{Model: integrationModel(tt.payload)}
			p := newTestProgram(h)

			timer := time.AfterFunc(10*time.Second, func() { p.Quit() })
			defer timer.Stop()

			go func() {
				p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
				p.Send(execDoneMsg{})
			}()

			if _, err := p.Run(); err != nil {
				t.Fatalf("program error: %v", err)
			}
		})
	}
}

func TestIntegrationPagerError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	t.Setenv("PAGER", "false")
	h := &testHarness{Model: integrationModel([]byte(`{"key":"value"}`))}
	p := newTestProgram(h)

	timer := time.AfterFunc(10*time.Second, func() { p.Quit() })
	defer timer.Stop()

	go func() {
		p.Send(tea.KeyPressMsg{Code: tea.KeyEnter})
	}()

	final, err := p.Run()
	if err != nil {
		t.Fatalf("program error: %v", err)
	}

	fm := final.(*testHarness).Model
	if !strings.Contains(fm.message, "pager") {
		t.Errorf("expected error message containing 'pager', got %q", fm.message)
	}
}

func TestIntegrationEditor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	script, err := os.CreateTemp("", "slope-editor-*.sh")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(script.Name())

	script.WriteString("#!/bin/sh\nprintf '{\"edited\":true}' > \"$1\"\n")
	script.Close()
	os.Chmod(script.Name(), 0o755)

	t.Setenv("EDITOR", script.Name())
	h := &testHarness{Model: integrationModel([]byte(`{"original":true}`))}
	p := newTestProgram(h)

	timer := time.AfterFunc(10*time.Second, func() { p.Quit() })
	defer timer.Stop()

	go func() {
		p.Send(tea.KeyPressMsg{Code: 'e'})
	}()

	final, err := p.Run()
	if err != nil {
		t.Fatalf("program error: %v", err)
	}

	fm := final.(*testHarness).Model
	if !fm.dirty {
		t.Error("expected dirty=true after editor changed payload")
	}
	if string(fm.envelope.Items[0].Payload) != `{"edited":true}` {
		t.Errorf("payload = %q, want %q", fm.envelope.Items[0].Payload, `{"edited":true}`)
	}
}

func TestIntegrationEditorNoChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	t.Setenv("EDITOR", "true")
	original := []byte(`{"original":true}`)
	h := &testHarness{Model: integrationModel(original)}
	p := newTestProgram(h)

	timer := time.AfterFunc(10*time.Second, func() { p.Quit() })
	defer timer.Stop()

	go func() {
		p.Send(tea.KeyPressMsg{Code: 'e'})
	}()

	final, err := p.Run()
	if err != nil {
		t.Fatalf("program error: %v", err)
	}

	fm := final.(*testHarness).Model
	if fm.dirty {
		t.Error("expected dirty=false when editor makes no changes")
	}
	if string(fm.envelope.Items[0].Payload) != `{"original":true}` {
		t.Errorf("payload = %q, want %q", fm.envelope.Items[0].Payload, `{"original":true}`)
	}
}

func TestIntegrationEditorError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	t.Setenv("EDITOR", "false")
	h := &testHarness{Model: integrationModel([]byte(`{"key":"value"}`))}
	p := newTestProgram(h)

	timer := time.AfterFunc(10*time.Second, func() { p.Quit() })
	defer timer.Stop()

	go func() {
		p.Send(tea.KeyPressMsg{Code: 'e'})
	}()

	final, err := p.Run()
	if err != nil {
		t.Fatalf("program error: %v", err)
	}

	fm := final.(*testHarness).Model
	if fm.message == "" {
		t.Error("expected error message after editor failure")
	}
}
