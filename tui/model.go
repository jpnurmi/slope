package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/getsentry/slope/envelope"
)

type viewMode int

const (
	modeList viewMode = iota
	modeInput
)

type Model struct {
	envelope *envelope.Envelope
	filePath string
	fileSize int64
	selected int
	mode     viewMode
	picker   filepicker.Model
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
	return m.picker.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.picker.SetHeight(msg.Height - 5)
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
	case keyD:
		if m.itemCount() > 0 {
			m.envelope.Items = append(m.envelope.Items[:m.selected], m.envelope.Items[m.selected+1:]...)
			if m.selected >= m.itemCount() && m.itemCount() > 0 {
				m.selected = m.itemCount() - 1
			}
			m.dirty = true
			m.message = "Item deleted"
		}
	case keyA:
		m.mode = modeInput
		return m, m.picker.Init()
	case keyW:
		if err := m.writeFile(); err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
		} else {
			m.dirty = false
			m.message = savedStyle.Render("Saved " + m.filePath)
		}
	case keyQ, keyCtrlC:
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

	c := exec.Command("less", "-R")
	c.Stdin = strings.NewReader(content)
	return tea.ExecProcess(c, func(err error) tea.Msg { return nil })
}

func (m Model) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)

	if m.picker.Path != "" {
		path := m.picker.Path
		m.picker.Path = ""
		if err := m.addAttachment(path); err != nil {
			m.message = errorStyle.Render("Error: " + err.Error())
		} else {
			m.dirty = true
			m.message = savedStyle.Render("Added " + filepath.Base(path))
		}
		m.mode = modeList
		return m, nil
	}

	return m, cmd
}

func (m Model) View() tea.View {
	var b strings.Builder

	switch m.mode {
	case modeList:
		m.viewList(&b)
	case modeInput:
		b.WriteString(labelStyle.Render("Select file to attach") + "\n\n")
		b.WriteString(m.picker.View() + "\n")
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

func (m Model) viewList(b *strings.Builder) {
	sep := m.separator()

	// Envelope header
	b.WriteString(labelStyle.Render(
		fmt.Sprintf("%s · %s", filepath.Base(m.filePath), formatSize(int(m.fileSize))),
	) + "\n")
	b.WriteString(sep + "\n")
	b.WriteString(formatHeader(m.envelope.Header) + "\n")

	// Item headers (static dump, fully copy-pasteable)
	for i, item := range m.envelope.Items {
		b.WriteString("\n" + labelStyle.Render(itemLabel(i, item)) + "\n")
		b.WriteString(sep + "\n")
		b.WriteString(formatHeader(item.Header) + "\n")
	}

	// Selection list at bottom
	if m.itemCount() > 0 {
		b.WriteString("\n")
		for i, item := range m.envelope.Items {
			label := itemLabel(i, item)
			if i == m.selected {
				b.WriteString("> " + selectedLabelStyle.Render(label) + "\n")
			} else {
				b.WriteString("  " + label + "\n")
			}
		}
	}
}


func (m Model) helpText() string {
	switch m.mode {
	case modeInput:
		return helpStyle.Render("↑/↓ navigate · enter select · esc cancel")
	default:
		dirty := ""
		if m.dirty {
			dirty = " · (modified)"
		}
		return helpStyle.Render("↑/↓ navigate · enter select · d delete · a add · w save · q quit" + dirty)
	}
}

func (m *Model) writeFile() error {
	f, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return m.envelope.Serialize(f)
}

func (m *Model) addAttachment(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	basename := filepath.Base(path)

	header, _ := json.Marshal(map[string]any{
		"type":         "attachment",
		"length":       len(data),
		"filename":     basename,
		"content_type": "application/octet-stream",
	})

	m.envelope.Items = append(m.envelope.Items, envelope.Item{
		Header:   json.RawMessage(header),
		Payload:  data,
		Type:     "attachment",
		Filename: basename,
	})
	return nil
}
