package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/getsentry/slope/envelope"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type editResultMsg struct {
	index   int
	payload []byte
	err     error
}

type viewMode int

const (
	modeList viewMode = iota
	modeInput
	modeExport
	modeConfirmQuit
)

type Model struct {
	envelope *envelope.Envelope
	filePath string
	fileSize int64
	selected int
	mode     viewMode
	picker   filepicker.Model
	export   textinput.Model
	dirty    bool
	message  string
	width    int
}

func NewModel(env *envelope.Envelope, filePath string, fileSize int64) Model {
	fp := filepicker.New()
	fp.CurrentDirectory, _ = os.Getwd()
	fp.FileAllowed = true
	fp.DirAllowed = false
	fp.ShowHidden = false
	fp.ShowSize = true
	fp.AutoHeight = false
	fp.SetHeight(20)
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	return Model{
		envelope: env,
		filePath: filePath,
		fileSize: fileSize,
		picker:   fp,
	}
}

func (m Model) itemCount() int {
	return len(m.envelope.Items)
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.printDump(), m.picker.Init())
}

func (m Model) buildDump() string {
	var b strings.Builder
	sep := m.separator()

	b.WriteString(labelStyle.Render(
		fmt.Sprintf("%s · %s", filepath.Base(m.filePath), formatSize(int(m.fileSize))),
	) + "\n")
	b.WriteString(sep + "\n")
	b.WriteString(formatHeader(m.envelope.Header, m.width) + "\n")

	for i, item := range m.envelope.Items {
		b.WriteString("\n" + labelStyle.Render(itemLabel(i, item)) + "\n")
		b.WriteString(sep + "\n")
		b.WriteString(formatHeader(item.Header, m.width) + "\n")
	}
	return b.String()
}

func (m Model) printDump() tea.Cmd {
	return tea.Println(m.buildDump())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.picker.SetHeight(max(msg.Height-5, 1))
	case editResultMsg:
		if msg.err != nil {
			m.message = errorStyle.Render("Error: " + msg.err.Error())
			return m, nil
		}
		item := &m.envelope.Items[msg.index]
		if bytes.Equal(item.Payload, msg.payload) {
			return m, nil
		}
		item.Payload = msg.payload
		header, err := envelope.UpdateLength(item.Header, len(msg.payload))
		if err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
			return m, nil
		}
		item.Header = header
		m.dirty = true
		m.message = savedStyle.Render("Payload updated")
		return m, m.printDump()
	case tea.KeyPressMsg:
		m.message = ""
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeInput:
			if msg.String() == keyEsc {
				m.mode = modeList
				return m, nil
			}
		case modeExport:
			return m.updateExport(msg)
		case modeConfirmQuit:
			switch msg.String() {
			case keyY:
				return m, tea.Quit
			default:
				m.mode = modeList
				return m, nil
			}
		}
	}

	if m.mode == modeInput {
		return m.updatePicker(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyUp, keyK:
		if m.selected > 0 {
			m.selected--
		}
	case keyDown, keyJ:
		if m.selected < m.itemCount()-1 {
			m.selected++
		}
	case keyEnter:
		if m.itemCount() > 0 {
			return m, m.viewInPager()
		}
	case keyE:
		if m.itemCount() > 0 && !envelope.IsBinary(m.envelope.Items[m.selected].Payload) {
			return m, m.editInEditor()
		}
	case keyD:
		if m.itemCount() > 0 {
			m.envelope.Items = append(m.envelope.Items[:m.selected], m.envelope.Items[m.selected+1:]...)
			if m.selected >= m.itemCount() && m.itemCount() > 0 {
				m.selected = m.itemCount() - 1
			}
			m.dirty = true
			m.message = "Item deleted"
			return m, m.printDump()
		}
	case keyX:
		if m.itemCount() > 0 {
			m.export = textinput.New()
			m.export.SetValue(m.defaultExportFilename())
			m.mode = modeExport
			return m, m.export.Focus()
		}
	case keyA:
		m.mode = modeInput
		return m, m.picker.Init()
	case keyW:
		if !m.dirty {
			return m, nil
		}
		size, err := m.writeFile()
		if err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
		} else {
			m.dirty = false
			m.fileSize = size
			m.message = savedStyle.Render("Saved " + m.filePath)
			return m, m.printDump()
		}
	case keyQ, keyCtrlC:
		if m.dirty {
			m.mode = modeConfirmQuit
			return m, nil
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewInPager() tea.Cmd {
	item := m.envelope.Items[m.selected]
	var content string

	if len(item.Payload) == 0 {
		content = "(empty payload)\n"
	} else if envelope.IsBinary(item.Payload) {
		content = hexDump(item.Payload)
	} else if json.Valid(item.Payload) {
		content = highlightJSON(envelope.PrettyJSON(json.RawMessage(item.Payload))) + "\n"
	} else {
		content = string(item.Payload) + "\n"
	}

	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}
	c := exec.Command("sh", "-c", pager, "--")
	c.Stdin = strings.NewReader(content)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return editResultMsg{err: fmt.Errorf("pager: %w", err)}
		}
		return nil
	})
}

func (m Model) editInEditor() tea.Cmd {
	item := m.envelope.Items[m.selected]
	index := m.selected
	isJSON := json.Valid(item.Payload)

	ext := ".bin"
	if isJSON {
		ext = ".json"
	}

	tmpFile, err := os.CreateTemp("", "slope-*"+ext)
	if err != nil {
		return func() tea.Msg { return editResultMsg{err: err} }
	}
	tmpPath := tmpFile.Name()

	payload := item.Payload
	if isJSON {
		var buf bytes.Buffer
		json.Indent(&buf, payload, "", "  ")
		payload = buf.Bytes()
	}

	if _, err := tmpFile.Write(payload); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return func() tea.Msg { return editResultMsg{err: err} }
	}
	tmpFile.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	c := exec.Command("sh", "-c", editor+" \"$1\"", "--", tmpPath)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		defer os.Remove(tmpPath)
		if err != nil {
			return editResultMsg{err: err}
		}
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return editResultMsg{err: err}
		}
		if json.Valid(data) {
			var buf bytes.Buffer
			json.Compact(&buf, data)
			data = buf.Bytes()
		}
		return editResultMsg{index: index, payload: data}
	})
}

func (m Model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if m.picker.Path != "" {
		path := m.picker.Path
		m.picker.Path = ""
		if err := m.addAttachment(path); err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
			m.mode = modeList
			return m, nil
		}
		m.dirty = true
		m.message = savedStyle.Render("Added " + filepath.Base(path))
		m.mode = modeList
		return m, m.printDump()
	}

	return m, cmd
}

func (m Model) defaultExportFilename() string {
	item := m.envelope.Items[m.selected]
	if item.Filename != "" {
		return item.Filename
	}
	typ := item.Type
	if typ == "" {
		typ = "item"
	}
	if json.Valid(item.Payload) {
		return typ + ".json"
	}
	return typ + ".bin"
}

func (m Model) updateExport(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		filename := m.export.Value()
		if filename == "" {
			m.mode = modeList
			return m, nil
		}
		item := m.envelope.Items[m.selected]
		if err := os.WriteFile(filename, item.Payload, 0o644); err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
		} else {
			m.message = savedStyle.Render("Exported " + filename)
		}
		m.mode = modeList
		return m, nil
	case keyEsc:
		m.mode = modeList
		return m, nil
	}
	var cmd tea.Cmd
	m.export, cmd = m.export.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var b strings.Builder

	switch m.mode {
	case modeList:
		if m.itemCount() > 0 {
			for i, item := range m.envelope.Items {
				label := itemLabel(i, item)
				if i == m.selected {
					b.WriteString("> " + selectedLabelStyle.Render(label) + "\n")
				} else {
					b.WriteString("  " + label + "\n")
				}
			}
		}
	case modeInput:
		b.WriteString(labelStyle.Render("Select file to attach") + "\n\n")
		b.WriteString(m.picker.View() + "\n")
	case modeExport:
		b.WriteString(labelStyle.Render("Export to: ") + m.export.View() + "\n")
	case modeConfirmQuit:
		b.WriteString(errorStyle.Render("Unsaved changes. Quit anyway?") + "\n")
	}

	if m.message != "" {
		b.WriteString("\n" + m.message + "\n")
	}

	b.WriteString("\n" + m.helpText() + "\n")
	return tea.NewView(b.String())
}

func (m Model) separator() string {
	w := m.width
	if w <= 0 {
		w = 40
	}
	return separatorStyle.Render(strings.Repeat("─", w))
}

func (m Model) helpText() string {
	switch m.mode {
	case modeInput:
		return helpStyle.Render("↑/↓ navigate · enter select · esc cancel")
	case modeExport:
		return helpStyle.Render("enter confirm · esc cancel")
	case modeConfirmQuit:
		return helpStyle.Render("y quit · any key cancel")
	default:
		dirty := ""
		if m.dirty {
			dirty = " · (modified)"
		}
		editStyle := helpStyle
		if m.itemCount() == 0 || envelope.IsBinary(m.envelope.Items[m.selected].Payload) {
			editStyle = helpDisabledStyle
		}
		saveStyle := helpStyle
		if !m.dirty {
			saveStyle = helpDisabledStyle
		}
		return helpStyle.Render("↑/↓ navigate · enter view · a add") +
			editStyle.Render(" · e edit") +
			helpStyle.Render(" · x export · d delete") +
			saveStyle.Render(" · w save") +
			helpStyle.Render(" · q quit"+dirty)
	}
}

func (m *Model) writeFile() (int64, error) {
	dir := filepath.Dir(m.filePath)
	tmp, err := os.CreateTemp(dir, ".slope-*")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()

	if err := m.envelope.Serialize(tmp); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return 0, err
	}
	fi, err := tmp.Stat()
	if err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return 0, err
	}
	size := fi.Size()
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return 0, err
	}
	return size, os.Rename(tmpPath, m.filePath)
}

func (m *Model) addAttachment(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	basename := filepath.Base(path)
	om := orderedmap.New[string, any]()
	om.Set("type", "attachment")
	om.Set("length", len(data))
	om.Set("filename", basename)
	if ct := mime.TypeByExtension(filepath.Ext(path)); ct != "" {
		om.Set("content_type", ct)
	}

	header, err := json.Marshal(om)
	if err != nil {
		return fmt.Errorf("marshaling header: %w", err)
	}

	m.envelope.Items = append(m.envelope.Items, envelope.Item{
		Header:   json.RawMessage(header),
		Payload:  data,
		Type:     "attachment",
		Filename: basename,
	})
	return nil
}
