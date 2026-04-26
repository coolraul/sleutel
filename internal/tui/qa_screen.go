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

var styleQAQuestion = lipgloss.NewStyle().Foreground(fg).Padding(0, 1)
var styleQAAnswer   = lipgloss.NewStyle().Foreground(subtle).Padding(0, 1)

type openQAMsg struct{ entry model.Entry }
type closeQAMsg struct{ entry model.Entry }
type deleteQAMsg struct{ entryID string; idx int }

type qaScreen struct {
	entry      model.Entry
	cursor     int
	confirming bool
	copied     bool
	width      int
	height     int
}

func newQAScreen(e model.Entry, width, height int) qaScreen {
	return qaScreen{entry: e, width: width, height: height}
}

func (q qaScreen) Init() tea.Cmd { return nil }

func (q qaScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	sqs := q.entry.SecurityQuestions

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if q.confirming {
			if msg.String() == "y" || msg.String() == "Y" {
				id, idx := q.entry.ID, q.cursor
				q.confirming = false
				return q, func() tea.Msg { return deleteQAMsg{entryID: id, idx: idx} }
			}
			q.confirming = false
			return q, nil
		}

		switch msg.String() {
		case "esc", "q":
			e := q.entry
			return q, func() tea.Msg { return closeQAMsg{entry: e} }
		case "up", "k":
			if q.cursor > 0 {
				q.cursor--
			}
		case "down", "j":
			if q.cursor < len(sqs)-1 {
				q.cursor++
			}
		case "a":
			id := q.entry.ID
			return q, func() tea.Msg {
				return openQAFormMsg{entryID: id, idx: -1, sq: model.SecurityQuestion{}}
			}
		case "e":
			if len(sqs) > 0 {
				id, idx, sq := q.entry.ID, q.cursor, sqs[q.cursor]
				return q, func() tea.Msg { return openQAFormMsg{entryID: id, idx: idx, sq: sq} }
			}
		case "d":
			if len(sqs) > 0 {
				q.confirming = true
			}
		case "c":
			if len(sqs) > 0 && sqs[q.cursor].Answer != "" {
				if err := clip.Write(sqs[q.cursor].Answer); err == nil {
					q.copied = true
					return q, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
						return clipClearMsg{}
					})
				}
			}
		}

	case clipClearMsg:
		q.copied = false
	}

	return q, nil
}

type openQAFormMsg struct {
	entryID string
	idx     int
	sq      model.SecurityQuestion
}

func (q qaScreen) View() string {
	if q.width == 0 {
		return ""
	}

	sqs := q.entry.SecurityQuestions
	var b strings.Builder
	divider := styleDivider.Render(strings.Repeat("─", q.width))

	// Header
	title := fmt.Sprintf("%s — Security Questions", q.entry.Title)
	if len(sqs) > 0 {
		title += fmt.Sprintf("  (%d)", len(sqs))
	}
	b.WriteString(styleDetailTitle.Render(title))
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n\n")

	// List
	if len(sqs) == 0 {
		b.WriteString(styleStatus.Render("  No security questions — press a to add one."))
		b.WriteString("\n")
	} else {
		qWidth := q.width - 20
		if qWidth < 20 {
			qWidth = 20
		}
		for i, sq := range sqs {
			s := styleRow
			cur := "  "
			if i == q.cursor {
				s = styleSelected
				cur = "▶ "
			}
			question := trunc(sq.Question, qWidth-4)
			answer := styleQAAnswer.Render("••••••••")
			if i == q.cursor {
				answer = styleFieldMasked.Render("••••••••")
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
				s.Padding(0, 1).Render(cur),
				s.Width(qWidth).Render(question),
				answer,
			))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")

	switch {
	case q.confirming:
		b.WriteString(styleConfirm.Render(
			fmt.Sprintf(`delete "%s"? [y/N]`, trunc(sqs[q.cursor].Question, 40)),
		))
	case q.copied:
		b.WriteString(styleConfirm.Render(
			fmt.Sprintf("answer copied — clipboard clears in %ds", int(clip.ClearDelay.Seconds())),
		))
	default:
		hint := styleKey.Render("↑↓/jk") + styleStatus.Render(" navigate   ")
		if len(sqs) > 0 {
			hint += styleKey.Render("c") + styleStatus.Render(" copy answer   ")
			hint += styleKey.Render("e") + styleStatus.Render(" edit   ")
			hint += styleKey.Render("d") + styleStatus.Render(" delete   ")
		}
		hint += styleKey.Render("a") + styleStatus.Render(" add   ")
		hint += styleKey.Render("esc") + styleStatus.Render(" back")
		b.WriteString(hint)
	}

	return b.String()
}
