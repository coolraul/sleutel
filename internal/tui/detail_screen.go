package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mms/sleutel/internal/clip"
	"github.com/mms/sleutel/internal/model"
)

var (
	styleDetailTitle = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	styleFieldLabel  = lipgloss.NewStyle().Foreground(fgDim).Width(12).Padding(0, 1)
	styleFieldValue  = lipgloss.NewStyle().Foreground(fg).Padding(0, 1)
	styleFieldMasked = lipgloss.NewStyle().Foreground(subtle).Padding(0, 1)
	styleConfirm     = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444")).Bold(true).Padding(0, 1)
)

type detailScreen struct {
	entry        model.Entry
	showPassword bool
	confirming   bool
	copied       bool // true briefly after a successful clipboard write
	width        int
	height       int
}

type clipClearMsg struct{}

func newDetailScreen(e model.Entry, width, height int) detailScreen {
	return detailScreen{entry: e, width: width, height: height}
}

type closeDetailMsg struct{}
type openEditMsg struct{ entry model.Entry }
type deleteEntryMsg struct{ id string }

func (d detailScreen) Init() tea.Cmd { return nil }

func (d detailScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if d.confirming {
			switch msg.String() {
			case "y", "Y":
				id := d.entry.ID
				return d, func() tea.Msg { return deleteEntryMsg{id: id} }
			default:
				d.confirming = false
			}
			return d, nil
		}

		switch msg.String() {
		case "esc", "q":
			return d, func() tea.Msg { return closeDetailMsg{} }
		case "p":
			d.showPassword = !d.showPassword
		case "e":
			e := d.entry
			return d, func() tea.Msg { return openEditMsg{entry: e} }
		case "d":
			d.confirming = true
		case "c":
			if d.entry.Password != "" {
				if err := clip.Write(d.entry.Password); err == nil {
					d.copied = true
					return d, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
						return clipClearMsg{}
					})
				}
			}
		}

	case clipClearMsg:
		d.copied = false
	}
	return d, nil
}

func (d detailScreen) View() string {
	if d.width == 0 {
		return ""
	}

	var b strings.Builder
	divider := styleDivider.Render(strings.Repeat("─", d.width))

	// Header
	b.WriteString(styleDetailTitle.Render(d.entry.Title))
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n\n")

	// Fields
	d.writeField(&b, "Username", d.entry.Username)
	d.writePassword(&b)
	d.writeField(&b, "URL", d.entry.URL)
	d.writeField(&b, "Notes", d.entry.Notes)
	if len(d.entry.Tags) > 0 {
		d.writeField(&b, "Tags", strings.Join(d.entry.Tags, ", "))
	}
	b.WriteString("\n")
	d.writeField(&b, "ID", d.entry.ID)
	d.writeField(&b, "Created", d.entry.CreatedAt.Format("2006-01-02 15:04:05"))
	d.writeField(&b, "Updated", d.entry.UpdatedAt.Format("2006-01-02 15:04:05"))

	// Status bar
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")
	switch {
	case d.confirming:
		b.WriteString(styleConfirm.Render(fmt.Sprintf(`delete "%s"? [y/N]`, d.entry.Title)))
	case d.copied:
		b.WriteString(styleConfirm.Render(fmt.Sprintf("copied — clipboard clears in %ds", int(clip.ClearDelay.Seconds()))))
	default:
		hint := ""
		if d.entry.Password != "" {
			if d.showPassword {
				hint = styleKey.Render("p") + styleStatus.Render(" hide password   ")
			} else {
				hint = styleKey.Render("p") + styleStatus.Render(" show password   ")
			}
			hint += styleKey.Render("c") + styleStatus.Render(" copy password   ")
		}
		hint += styleKey.Render("e") + styleStatus.Render(" edit   ")
		hint += styleKey.Render("d") + styleStatus.Render(" delete   ")
		hint += styleKey.Render("esc") + styleStatus.Render(" back")
		b.WriteString(styleStatus.Render(hint))
	}

	return b.String()
}

func (d detailScreen) writeField(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		styleFieldLabel.Render(label),
		styleFieldValue.Render(value),
	))
	b.WriteString("\n")
}

func (d detailScreen) writePassword(b *strings.Builder) {
	if d.entry.Password == "" {
		return
	}
	var val string
	if d.showPassword {
		val = styleFieldValue.Render(d.entry.Password)
	} else {
		val = styleFieldMasked.Render("••••••••")
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		styleFieldLabel.Render("Password"),
		val,
	))
	b.WriteString("\n")
}
