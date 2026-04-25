package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mms/sleutel/internal/model"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#555555"}
	accent    = lipgloss.AdaptiveColor{Light: "#3c6ef0", Dark: "#5f87ff"}
	fg        = lipgloss.AdaptiveColor{Light: "#111111", Dark: "#eeeeee"}
	fgDim     = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#777777"}
	bgRow     = lipgloss.AdaptiveColor{Light: "#e8eeff", Dark: "#1e2030"}
)

var (
	styleHeader   = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	styleCount    = lipgloss.NewStyle().Foreground(fgDim).Padding(0, 1)
	styleDivider  = lipgloss.NewStyle().Foreground(subtle)
	styleColHead  = lipgloss.NewStyle().Bold(true).Foreground(fgDim).Padding(0, 1)
	styleRow      = lipgloss.NewStyle().Foreground(fg).Padding(0, 1)
	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(accent).Background(bgRow).Padding(0, 1)
	styleStatus   = lipgloss.NewStyle().Foreground(fgDim).Padding(0, 1)
	styleKey      = lipgloss.NewStyle().Foreground(accent).Bold(true)
)

type MainScreen struct {
	entries  []model.Entry
	filtered []model.Entry
	search   textinput.Model
	cursor   int
	width    int
	height   int
}

func NewMainScreen(entries []model.Entry) MainScreen {
	ti := textinput.New()
	ti.Placeholder = "type to filter…"
	ti.CharLimit = 80

	return MainScreen{
		entries:  entries,
		filtered: entries,
		search:   ti,
	}
}

func (m MainScreen) Init() tea.Cmd { return nil }

func (m MainScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.search.Focused() {
			switch msg.String() {
			case "esc":
				m.search.Blur()
				m.search.SetValue("")
				m.filtered = m.entries
				m.cursor = 0
				return m, nil
			case "enter":
				m.search.Blur()
				return m, nil
			default:
				m.search, cmd = m.search.Update(msg)
				m.applyFilter()
				m.cursor = 0
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "/":
			m.search.Focus()
			return m, textinput.Blink
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.filtered) > 0 {
				e := m.filtered[m.cursor]
				return m, func() tea.Msg { return openDetailMsg{entry: e} }
			}
		case "n":
			return m, func() tea.Msg { return openAddMsg{} }
		}
	}
	return m, nil
}

type openDetailMsg struct{ entry model.Entry }
type openAddMsg struct{}

func (m *MainScreen) applyFilter() {
	q := strings.ToLower(m.search.Value())
	if q == "" {
		m.filtered = m.entries
		return
	}
	var out []model.Entry
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Title), q) ||
			strings.Contains(strings.ToLower(e.Username), q) ||
			strings.Contains(strings.ToLower(e.URL), q) ||
			strings.Contains(strings.ToLower(e.Notes), q) ||
			containsTag(e.Tags, q) {
			out = append(out, e)
		}
	}
	m.filtered = out
}

func containsTag(tags []string, q string) bool {
	for _, t := range tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

func (m MainScreen) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder
	divider := styleDivider.Render(strings.Repeat("─", m.width))

	// Header
	count := len(m.filtered)
	countStr := fmt.Sprintf("%d entries", len(m.entries))
	if m.search.Value() != "" {
		countStr = fmt.Sprintf("%d / %d entries", count, len(m.entries))
	}
	b.WriteString(styleHeader.Render("sleutel"))
	b.WriteString(styleCount.Render(countStr))
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")

	// Search bar
	b.WriteString(m.renderSearch())
	b.WriteString("\n")

	// Column headers
	c := m.cols()
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
		styleColHead.Width(c.cursor).Render(""),
		styleColHead.Width(c.title).Render("TITLE"),
		styleColHead.Width(c.user).Render("USERNAME"),
		styleColHead.Width(c.url).Render("URL"),
		styleColHead.Width(c.age).Render("UPDATED"),
	))
	b.WriteString("\n")

	// Rows
	if len(m.filtered) == 0 {
		if len(m.entries) == 0 {
			b.WriteString(styleStatus.Render("  no entries — use 'sleutel add' to create one"))
		} else {
			b.WriteString(styleStatus.Render("  no matches"))
		}
		b.WriteString("\n")
	} else {
		maxRows := m.height - 8 // header(2) + divider(1) + search(1) + gap(1) + colhead(1) + divider(1) + status(1)
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if m.cursor >= maxRows {
			start = m.cursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(m.filtered) {
			end = len(m.filtered)
		}

		for i := start; i < end; i++ {
			e := m.filtered[i]
			s := styleRow
			cur := "  "
			if i == m.cursor {
				s = styleSelected
				cur = "▶ "
			}
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
				s.Width(c.cursor).Render(cur),
				s.Width(c.title).Render(trunc(e.Title, c.title-2)),
				s.Width(c.user).Render(trunc(e.Username, c.user-2)),
				s.Width(c.url).Render(trunc(e.URL, c.url-2)),
				s.Width(c.age).Render(fmtAge(e.UpdatedAt)),
			))
			b.WriteString("\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(divider)
	b.WriteString("\n")
	if m.search.Focused() {
		b.WriteString(styleStatus.Render(
			styleKey.Render("enter") + " confirm   " +
				styleKey.Render("esc") + " clear",
		))
	} else {
		b.WriteString(styleStatus.Render(
			styleKey.Render("↑↓/jk") + " navigate   " +
				styleKey.Render("enter") + " open   " +
				styleKey.Render("n") + " new   " +
				styleKey.Render("/") + " search   " +
				styleKey.Render("q") + " quit",
		))
	}

	return b.String()
}

func (m MainScreen) renderSearch() string {
	prefix := styleKey.Render("/") + " "
	if m.search.Focused() {
		return styleStatus.Render(prefix + m.search.View())
	}
	if m.search.Value() != "" {
		return styleStatus.Render(prefix + styleKey.Render(m.search.Value()))
	}
	return styleStatus.Render(prefix + styleDivider.Render("search"))
}

type cols struct{ cursor, title, user, url, age int }

func (m MainScreen) cols() cols {
	w := m.width
	if w < 60 {
		w = 60
	}
	cursor := 4
	ageW := 10
	rest := w - cursor - ageW
	title := rest / 3
	user := rest / 3
	url := rest - title - user
	return cols{cursor, title, user, url, ageW}
}

func fmtAge(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func trunc(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
