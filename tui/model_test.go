package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
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

func viewText(m Model) string {
	v := m.View()
	if ss, ok := v.Content.(*uv.StyledString); ok {
		return ss.Text
	}
	return ""
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

func TestListEnterEmpty(t *testing.T) {
	m := testModel(0)
	m = update(m, specialKey(tea.KeyEnter))
	if m.mode != modeList {
		t.Errorf("enter on empty: mode = %d, want modeList", m.mode)
	}
}

func TestListEditBinary(t *testing.T) {
	m := testModel(0)
	m.envelope.Items = []envelope.Item{{
		Header:  json.RawMessage(`{"type":"attachment","length":3}`),
		Payload: []byte{0x00, 0x01, 0x02},
		Type:    "attachment",
	}}
	m = update(m, key('e'))
	if m.mode != modeList {
		t.Errorf("edit binary: mode = %d, want modeList", m.mode)
	}
}

func TestListEnterViewsItem(t *testing.T) {
	m := testModel(1)
	_, cmd := m.Update(specialKey(tea.KeyEnter))
	if cmd == nil {
		t.Error("enter with items should return a cmd")
	}
}

func TestListEditText(t *testing.T) {
	m := testModel(1)
	_, cmd := m.Update(key('e'))
	if cmd == nil {
		t.Error("edit text item should return a cmd")
	}
}

func TestListExportEmpty(t *testing.T) {
	m := testModel(0)
	m = update(m, key('x'))
	if m.mode != modeList {
		t.Errorf("export on empty: mode = %d, want modeList", m.mode)
	}
}

func TestInit(t *testing.T) {
	m := testModel(1)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a non-nil cmd")
	}
}

func TestDefaultExportFilename(t *testing.T) {
	tests := []struct {
		name string
		item envelope.Item
		want string
	}{
		{"filename set", envelope.Item{Filename: "crash.txt", Type: "attachment", Payload: []byte("data")}, "crash.txt"},
		{"type+json", envelope.Item{Type: "event", Payload: []byte(`{}`)}, "event.json"},
		{"type+binary", envelope.Item{Type: "event", Payload: []byte{0x00}}, "event.bin"},
		{"no type+json", envelope.Item{Payload: []byte(`{}`)}, "item.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel(0)
			m.envelope.Items = []envelope.Item{tt.item}
			got := m.defaultExportFilename()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAddAttachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := testModel(0)
	if err := m.addAttachment(path); err != nil {
		t.Fatal(err)
	}

	if len(m.envelope.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(m.envelope.Items))
	}

	item := m.envelope.Items[0]
	if item.Type != "attachment" {
		t.Errorf("type = %q, want attachment", item.Type)
	}
	if item.Filename != "hello.txt" {
		t.Errorf("filename = %q, want hello.txt", item.Filename)
	}
	if !bytes.Equal(item.Payload, []byte("hello")) {
		t.Errorf("payload = %q, want %q", item.Payload, "hello")
	}

	var hdr struct {
		Type        string `json:"type"`
		Length      int    `json:"length"`
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	}
	if err := json.Unmarshal(item.Header, &hdr); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if hdr.Type != "attachment" {
		t.Errorf("header type = %q, want attachment", hdr.Type)
	}
	if hdr.Length != 5 {
		t.Errorf("header length = %d, want 5", hdr.Length)
	}
	if hdr.Filename != "hello.txt" {
		t.Errorf("header filename = %q, want hello.txt", hdr.Filename)
	}
	if hdr.ContentType == "" {
		t.Error("header content_type should be set for .txt")
	}
}

func TestUpdateExport(t *testing.T) {
	t.Run("enter with filename", func(t *testing.T) {
		dir := t.TempDir()
		m := testModel(1)
		m = update(m, key('x'))

		path := filepath.Join(dir, "out.json")
		m.export.SetValue(path)

		m = update(m, specialKey(tea.KeyEnter))
		if m.mode != modeList {
			t.Errorf("mode = %d, want modeList", m.mode)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "{}" {
			t.Errorf("exported = %q, want %q", data, "{}")
		}
	})

	t.Run("enter empty", func(t *testing.T) {
		m := testModel(1)
		m = update(m, key('x'))
		m.export.SetValue("")

		m = update(m, specialKey(tea.KeyEnter))
		if m.mode != modeList {
			t.Errorf("mode = %d, want modeList", m.mode)
		}
	})

	t.Run("esc", func(t *testing.T) {
		m := testModel(1)
		m = update(m, key('x'))

		m = update(m, specialKey(tea.KeyEscape))
		if m.mode != modeList {
			t.Errorf("mode = %d, want modeList", m.mode)
		}
	})

	t.Run("typing updates input", func(t *testing.T) {
		m := testModel(1)
		m = update(m, key('x'))
		m = update(m, key('z'))
		if m.mode != modeExport {
			t.Errorf("mode = %d, want modeExport", m.mode)
		}
	})
}

func TestHelpText(t *testing.T) {
	tests := []struct {
		name  string
		mode  viewMode
		dirty bool
	}{
		{"input", modeInput, false},
		{"export", modeExport, false},
		{"confirmQuit", modeConfirmQuit, false},
		{"list clean", modeList, false},
		{"list dirty", modeList, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel(1)
			m.mode = tt.mode
			m.dirty = tt.dirty
			if m.helpText() == "" {
				t.Error("helpText() is empty")
			}
		})
	}
}

func TestSeparator(t *testing.T) {
	m := testModel(0)

	s := m.separator()
	if strings.Count(s, "─") != 40 {
		t.Errorf("default separator: got %d '─' chars, want 40", strings.Count(s, "─"))
	}

	m.width = 80
	s = m.separator()
	if strings.Count(s, "─") != 80 {
		t.Errorf("80-width separator: got %d '─' chars, want 80", strings.Count(s, "─"))
	}
}

func TestBuildDump(t *testing.T) {
	m := testModel(2)
	m.width = 80
	dump := m.buildDump()

	if !strings.Contains(dump, "test.envelope") {
		t.Error("dump should contain filename")
	}
	if !strings.Contains(dump, "EVENT") {
		t.Error("dump should contain item type")
	}
}

func TestView(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		m := testModel(1)
		v := viewText(m)
		if !strings.Contains(v, ">") {
			t.Error("list view should contain '>'")
		}
	})

	t.Run("list empty", func(t *testing.T) {
		m := testModel(0)
		v := viewText(m)
		if strings.Contains(v, ">") {
			t.Error("empty list view should not contain '>'")
		}
	})

	t.Run("input", func(t *testing.T) {
		m := testModel(1)
		m = update(m, key('a'))
		v := viewText(m)
		if !strings.Contains(v, "Select file") {
			t.Error("input view should contain 'Select file'")
		}
	})

	t.Run("export", func(t *testing.T) {
		m := testModel(1)
		m = update(m, key('x'))
		v := viewText(m)
		if !strings.Contains(v, "Export to") {
			t.Error("export view should contain 'Export to'")
		}
	})

	t.Run("confirmQuit", func(t *testing.T) {
		m := testModel(1)
		m.dirty = true
		m = update(m, key('q'))
		v := viewText(m)
		if !strings.Contains(v, "Unsaved") {
			t.Error("confirm quit view should contain 'Unsaved'")
		}
	})

	t.Run("message", func(t *testing.T) {
		m := testModel(1)
		m.message = "test message"
		v := viewText(m)
		if !strings.Contains(v, "test message") {
			t.Error("view should contain message")
		}
	})
}

func TestFormatHeader(t *testing.T) {
	short := json.RawMessage(`{"a":1}`)
	long := json.RawMessage(`{"key1":"value1","key2":"value2","key3":"value3","key4":"value4"}`)

	result := formatHeader(short, 200)
	if strings.Contains(result, "\n") {
		t.Errorf("short JSON wide width should be one-line, got %q", result)
	}

	result = formatHeader(long, 10)
	if !strings.Contains(result, "\n") {
		t.Errorf("long JSON narrow width should be multi-line, got %q", result)
	}
}

func TestWriteFileError(t *testing.T) {
	m := testModel(1)
	m.filePath = "/nonexistent/dir/file.envelope"
	m.dirty = true

	m = update(m, key('w'))
	if !strings.Contains(m.message, "Error") {
		t.Errorf("expected error message, got %q", m.message)
	}
}

func TestAddAttachmentError(t *testing.T) {
	m := testModel(0)
	err := m.addAttachment("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEditResultUpdateLengthError(t *testing.T) {
	m := testModel(0)
	m.envelope.Items = []envelope.Item{{
		Header:  json.RawMessage("not json"),
		Payload: []byte("old"),
	}}

	m = update(m, editResultMsg{index: 0, payload: []byte("new")})
	if !strings.Contains(m.message, "Error") {
		t.Errorf("expected error message, got %q", m.message)
	}
}
