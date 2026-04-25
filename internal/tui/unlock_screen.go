package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


var (
	styleUnlockTitle = lipgloss.NewStyle().Bold(true).Foreground(accent).MarginBottom(1)
	styleUnlockLabel = lipgloss.NewStyle().Foreground(fgDim)
	styleUnlockInput = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1).Width(36)
	styleError       = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).MarginTop(1)
	styleSpinner     = lipgloss.NewStyle().Foreground(fgDim).MarginTop(1)
)

type unlockScreen struct {
	input     textinput.Model
	err       error
	loading   bool
	submitted bool
	width     int
	height    int
}

func newUnlockScreen() unlockScreen {
	ti := textinput.New()
	ti.Placeholder = "master password"
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	ti.Width = 34
	ti.Focus()

	return unlockScreen{input: ti}
}

func (u unlockScreen) Init() tea.Cmd {
	return textinput.Blink
}

func (u unlockScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if u.loading {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "ctrl+c" {
			return u, tea.Quit
		}
		return u, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return u, tea.Quit
		case "enter":
			if strings.TrimSpace(u.input.Value()) != "" {
				u.submitted = true
				return u, nil
			}
		}
	}

	var cmd tea.Cmd
	u.input, cmd = u.input.Update(msg)
	return u, cmd
}

func (u unlockScreen) View() string {
	extra := ""
	switch {
	case u.loading:
		extra = styleSpinner.Render("unlocking vault…")
	case u.err != nil:
		extra = styleError.Render("✗ " + u.err.Error())
	}

	block := lipgloss.JoinVertical(lipgloss.Left,
		styleUnlockTitle.Render("sleutel"),
		styleUnlockLabel.Render("Master password"),
		styleUnlockInput.Render(u.input.View()),
		extra,
	)

	return lipgloss.Place(u.width, u.height, lipgloss.Center, lipgloss.Center, block)
}

func (u *unlockScreen) password() []byte {
	pw := []byte(u.input.Value())
	u.input.SetValue("")
	return pw
}

func (u *unlockScreen) setError(err error) { u.err = err }
func (u *unlockScreen) setLoading(v bool)  { u.loading = v }
