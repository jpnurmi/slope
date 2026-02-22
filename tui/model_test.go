package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/getsentry/slope/envelope"
)

func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r}
}

func specialKey(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}

func testModel(items int) Model {
	env := &envelope.Envelope{
		Header: json.RawMessage(`{"sdk":{"name":"test"}}`),
	}
	for i := 0; i < items; i++ {
		env.Items = append(env.Items, envelope.Item{
			Header:  json.RawMessage(`{"type":"event","length":2}`),
			Payload: []byte("{}"),
			Type:    "event",
		})
	}
	return NewModel(env, "test.envelope", 0)
}

func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	return ok
}

func update(m Model, msgs ...tea.Msg) Model {
	for _, msg := range msgs {
		next, _ := m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestModelNavigation(t *testing.T) {
	m := testModel(3)

	m = update(m, key('j'))
	if m.selected != 1 {
		t.Errorf("j: selected = %d, want 1", m.selected)
	}

	m = update(m, specialKey(tea.KeyDown))
	if m.selected != 2 {
		t.Errorf("down: selected = %d, want 2", m.selected)
	}

	m = update(m, key('j'))
	if m.selected != 2 {
		t.Errorf("j at bottom: selected = %d, want 2", m.selected)
	}

	m = update(m, key('k'))
	if m.selected != 1 {
		t.Errorf("k: selected = %d, want 1", m.selected)
	}

	m = update(m, specialKey(tea.KeyUp))
	if m.selected != 0 {
		t.Errorf("up: selected = %d, want 0", m.selected)
	}

	m = update(m, key('k'))
	if m.selected != 0 {
		t.Errorf("k at top: selected = %d, want 0", m.selected)
	}
}

func TestModelDelete(t *testing.T) {
	m := testModel(3)
	m = update(m, key('j'), key('j'))

	m = update(m, key('d'))
	if m.itemCount() != 2 {
		t.Fatalf("delete: itemCount = %d, want 2", m.itemCount())
	}
	if !m.dirty {
		t.Error("delete: dirty = false, want true")
	}
	if m.selected != 1 {
		t.Errorf("delete at end: selected = %d, want 1", m.selected)
	}

	m = update(m, key('k'))
	m = update(m, key('d'))
	if m.itemCount() != 1 {
		t.Fatalf("delete first: itemCount = %d, want 1", m.itemCount())
	}

	m = update(m, key('d'))
	if m.itemCount() != 0 {
		t.Fatalf("delete last: itemCount = %d, want 0", m.itemCount())
	}

	m = update(m, key('d'))
	if m.itemCount() != 0 {
		t.Fatalf("delete empty: itemCount = %d, want 0", m.itemCount())
	}
}

func TestModelQuitConfirmation(t *testing.T) {
	m := testModel(1)

	_, cmd := m.Update(key('q'))
	if !isQuitCmd(cmd) {
		t.Error("q when clean: expected quit cmd")
	}

	_, cmd = m.Update(ctrlKey('c'))
	if !isQuitCmd(cmd) {
		t.Error("ctrl+c when clean: expected quit cmd")
	}

	m.dirty = true
	m = update(m, key('q'))
	if m.mode != modeConfirmQuit {
		t.Errorf("q when dirty: mode = %d, want modeConfirmQuit", m.mode)
	}

	m = update(m, key('n'))
	if m.mode != modeList {
		t.Errorf("cancel quit: mode = %d, want modeList", m.mode)
	}

	m.dirty = true
	m = update(m, key('q'))
	_, cmd = m.Update(key('y'))
	if !isQuitCmd(cmd) {
		t.Error("y to confirm quit: expected quit cmd")
	}
}

func TestModelSave(t *testing.T) {
	tmp, err := os.CreateTemp("", "slope-test-*.envelope")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	tmp.Close()

	env := &envelope.Envelope{
		Header: json.RawMessage(`{"sdk":{"name":"test"}}`),
		Items: []envelope.Item{{
			Header:  json.RawMessage(`{"type":"event","length":13}`),
			Payload: []byte(`{"key":"val"}`),
			Type:    "event",
		}},
	}
	m := NewModel(env, tmp.Name(), 0)

	m = update(m, key('w'))
	data, _ := os.ReadFile(tmp.Name())
	if len(data) > 0 {
		t.Error("w when clean: file should be empty")
	}

	m.dirty = true
	m = update(m, key('w'))
	if m.dirty {
		t.Error("w when dirty: dirty should be false after save")
	}

	data, err = os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := envelope.Parse(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Items) != 1 {
		t.Fatalf("saved items = %d, want 1", len(parsed.Items))
	}
	if string(parsed.Items[0].Payload) != `{"key":"val"}` {
		t.Errorf("saved payload = %q, want %q", parsed.Items[0].Payload, `{"key":"val"}`)
	}

	fi, err := os.Stat(tmp.Name())
	if err != nil {
		t.Fatal(err)
	}
	if m.fileSize != fi.Size() {
		t.Errorf("fileSize = %d, want %d", m.fileSize, fi.Size())
	}
}

func TestModelSavePreservesFileOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.envelope")
	original := []byte("original content")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	env := &envelope.Envelope{
		Header: json.RawMessage(`not valid json`),
	}
	m := NewModel(env, path, int64(len(original)))
	m.dirty = true

	m = update(m, key('w'))
	if m.message == "" {
		t.Fatal("expected error message from failed save")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, original) {
		t.Errorf("file was modified after failed save: got %q", data)
	}
}

func TestModelEditResult(t *testing.T) {
	m := testModel(1)

	m = update(m, editResultMsg{index: 0, payload: []byte(`{"new":"data"}`)})
	if string(m.envelope.Items[0].Payload) != `{"new":"data"}` {
		t.Errorf("payload = %q, want %q", m.envelope.Items[0].Payload, `{"new":"data"}`)
	}
	if !m.dirty {
		t.Error("edit result: dirty = false, want true")
	}

	m.dirty = false
	m = update(m, editResultMsg{index: 0, payload: []byte(`{"new":"data"}`)})
	if m.dirty {
		t.Error("unchanged edit: dirty = true, want false")
	}

	m = update(m, editResultMsg{err: fmt.Errorf("test error")})
	if m.message == "" {
		t.Error("edit error: message should not be empty")
	}
}

func TestModelModeTransitions(t *testing.T) {
	m := testModel(1)

	m = update(m, key('a'))
	if m.mode != modeInput {
		t.Errorf("a: mode = %d, want modeInput", m.mode)
	}

	m = update(m, specialKey(tea.KeyEscape))
	if m.mode != modeList {
		t.Errorf("esc from input: mode = %d, want modeList", m.mode)
	}

	m = update(m, key('x'))
	if m.mode != modeExport {
		t.Errorf("x: mode = %d, want modeExport", m.mode)
	}

	m = update(m, specialKey(tea.KeyEscape))
	if m.mode != modeList {
		t.Errorf("esc from export: mode = %d, want modeList", m.mode)
	}
}

func TestModelWindowResize(t *testing.T) {
	m := testModel(0)
	m = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}

	m = update(m, tea.WindowSizeMsg{Width: 20, Height: 3})
	if m.picker.Height() < 1 {
		t.Errorf("picker height = %d with tiny terminal, want >= 1", m.picker.Height())
	}
}
