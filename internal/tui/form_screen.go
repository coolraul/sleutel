package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mms/sleutel/internal/model"
	"github.com/mms/sleutel/internal/vault"
)

var (
	styleFormTitle  = lipgloss.NewStyle().Bold(true).Foreground(accent).MarginBottom(1)
	styleLabel      = lipgloss.NewStyle().Foreground(fgDim).Width(12)
	styleRequired   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	styleFormHint   = lipgloss.NewStyle().Foreground(subtle).MarginLeft(2)
	styleFormError  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444"))
)

const (
	fldTitle = iota
	fldUsername
	fldPassword
	fldURL
	fldNotes
	fldTags
	fldCount
)

type submitAddMsg struct{ entry model.Entry }
type cancelFormMsg struct{}

type formScreen struct {
	fields  [fldCount]textinput.Model
	focused int
	showPw  bool
	errMsg  string
	width   int
	height  int
}

func newFormScreen(width, height int) formScreen {
	labels := []string{"Title", "Username", "Password", "URL", "Notes", "Tags"}
	placeholders := []string{"", "", "", "https://", "", "dev, work"}

	var fields [fldCount]textinput.Model
	for i := range fields {
		ti := textinput.New()
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 256
		ti.Width = 40
		ti.Prompt = styleLabel.Render(labels[i]+" ") + " "
		if i == fldPassword {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		fields[i] = ti
	}
	fields[fldTitle].Focus()

	return formScreen{fields: fields, width: width, height: height}
}

func (f formScreen) Init() tea.Cmd { return textinput.Blink }

func (f formScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return f, func() tea.Msg { return cancelFormMsg{} }

		case "ctrl+s":
			return f.submit()

		case "ctrl+g":
			pw, err := vault.GeneratePassword(24, true)
			if err == nil {
				f.fields[fldPassword].SetValue(pw)
			}
			return f, nil

		case "ctrl+p":
			f.showPw = !f.showPw
			if f.showPw {
				f.fields[fldPassword].EchoMode = textinput.EchoNormal
			} else {
				f.fields[fldPassword].EchoMode = textinput.EchoPassword
				f.fields[fldPassword].EchoCharacter = '•'
			}
			return f, nil

		case "tab", "down":
			f.fields[f.focused].Blur()
			f.focused = (f.focused + 1) % fldCount
			f.fields[f.focused].Focus()
			return f, textinput.Blink

		case "shift+tab", "up":
			f.fields[f.focused].Blur()
			f.focused = (f.focused - 1 + fldCount) % fldCount
			f.fields[f.focused].Focus()
			return f, textinput.Blink
		}
	}

	var cmd tea.Cmd
	f.fields[f.focused], cmd = f.fields[f.focused].Update(msg)
	return f, cmd
}

func (f formScreen) submit() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(f.fields[fldTitle].Value())
	if title == "" {
		f.errMsg = "Title is required"
		return f, nil
	}
	e := model.Entry{
		Title:    title,
		Username: strings.TrimSpace(f.fields[fldUsername].Value()),
		Password: f.fields[fldPassword].Value(),
		URL:      strings.TrimSpace(f.fields[fldURL].Value()),
		Notes:    strings.TrimSpace(f.fields[fldNotes].Value()),
		Tags:     parseTags(f.fields[fldTags].Value()),
	}
	return f, func() tea.Msg { return submitAddMsg{entry: e} }
}

func (f formScreen) View() string {
	if f.width == 0 {
		return ""
	}

	var b strings.Builder
	divider := styleDivider.Render(strings.Repeat("─", f.width))

	b.WriteString(styleFormTitle.Render("Add Entry"))
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n\n")

	for i, field := range f.fields {
		b.WriteString("  ")
		b.WriteString(field.View())

		switch i {
		case fldTitle:
			b.WriteString(styleRequired.Render(" *"))
		case fldPassword:
			hint := "ctrl+g generate"
			if f.showPw {
				hint += "  ctrl+p hide"
			} else {
				hint += "  ctrl+p show"
			}
			b.WriteString(styleFormHint.Render(hint))
		case fldTags:
			b.WriteString(styleFormHint.Render("comma-separated"))
		}
		b.WriteString("\n")
	}

	if f.errMsg != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", styleFormError.Render("✗ "+f.errMsg)))
	}

	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")
	b.WriteString(styleStatus.Render(
		styleKey.Render("tab") + " next   " +
			styleKey.Render("shift+tab") + " prev   " +
			styleKey.Render("ctrl+s") + " save   " +
			styleKey.Render("esc") + " cancel",
	))

	return b.String()
}

func parseTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
