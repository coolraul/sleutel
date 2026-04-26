package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mms/sleutel/internal/model"
)

const (
	qaFldQuestion = iota
	qaFldAnswer
	qaFldCount
)

type submitQAFormMsg struct {
	entryID string
	idx     int // -1 = add
	sq      model.SecurityQuestion
}
type cancelQAFormMsg struct{}

type qaFormScreen struct {
	entryID string
	idx     int // -1 = add
	fields  [qaFldCount]textinput.Model
	showAns bool
	errMsg  string
	width   int
	height  int
}

func newQAFormScreen(entryID string, idx int, sq model.SecurityQuestion, width, height int) qaFormScreen {
	labels := []string{"Question", "Answer"}
	values := []string{sq.Question, sq.Answer}

	var fields [qaFldCount]textinput.Model
	for i := range fields {
		ti := textinput.New()
		ti.CharLimit = 512
		ti.Width = 50
		ti.Prompt = styleLabel.Render(labels[i]+" ") + " "
		ti.SetValue(values[i])
		if i == qaFldAnswer {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		fields[i] = ti
	}
	fields[qaFldQuestion].Focus()

	return qaFormScreen{entryID: entryID, idx: idx, fields: fields, width: width, height: height}
}

func (f qaFormScreen) Init() tea.Cmd { return textinput.Blink }

func (f qaFormScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return f, func() tea.Msg { return cancelQAFormMsg{} }
		case "ctrl+s":
			return f.submit()
		case "ctrl+p":
			f.showAns = !f.showAns
			if f.showAns {
				f.fields[qaFldAnswer].EchoMode = textinput.EchoNormal
			} else {
				f.fields[qaFldAnswer].EchoMode = textinput.EchoPassword
				f.fields[qaFldAnswer].EchoCharacter = '•'
			}
			return f, nil
		case "tab", "down":
			f.fields[f.focused()].Blur()
			f.fields[f.next()].Focus()
			return f, textinput.Blink
		case "shift+tab", "up":
			f.fields[f.focused()].Blur()
			f.fields[f.prev()].Focus()
			return f, textinput.Blink
		}
	}

	var cmd tea.Cmd
	f.fields[f.focused()], cmd = f.fields[f.focused()].Update(msg)
	return f, cmd
}

func (f qaFormScreen) submit() (tea.Model, tea.Cmd) {
	q := strings.TrimSpace(f.fields[qaFldQuestion].Value())
	a := f.fields[qaFldAnswer].Value()
	if q == "" {
		f.errMsg = "Question is required"
		return f, nil
	}
	sq := model.SecurityQuestion{Question: q, Answer: a}
	id, idx := f.entryID, f.idx
	return f, func() tea.Msg { return submitQAFormMsg{entryID: id, idx: idx, sq: sq} }
}

func (f qaFormScreen) View() string {
	if f.width == 0 {
		return ""
	}

	var b strings.Builder
	divider := styleDivider.Render(strings.Repeat("─", f.width))

	title := "Add Security Question"
	if f.idx >= 0 {
		title = "Edit Security Question"
	}
	b.WriteString(styleFormTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n\n")

	for i, field := range f.fields {
		b.WriteString("  ")
		b.WriteString(field.View())
		if i == qaFldQuestion {
			b.WriteString(styleRequired.Render(" *"))
		}
		if i == qaFldAnswer {
			if f.showAns {
				b.WriteString(styleFormHint.Render("ctrl+p hide"))
			} else {
				b.WriteString(styleFormHint.Render("ctrl+p show"))
			}
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
			styleKey.Render("ctrl+s") + " save   " +
			styleKey.Render("esc") + " cancel",
	))

	return b.String()
}

func (f qaFormScreen) focused() int {
	for i, field := range f.fields {
		if field.Focused() {
			return i
		}
	}
	return 0
}

func (f qaFormScreen) next() int { return (f.focused() + 1) % qaFldCount }
func (f qaFormScreen) prev() int { return (f.focused() - 1 + qaFldCount) % qaFldCount }
