package palette

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/shared"
)

// Item is a selectable entry in the palette.
type Item struct {
	Label  string
	Detail string // secondary text (e.g. "remote" or "local")
	Value  string // the value to act on
}

// ResultMsg is sent when the user selects an item.
type ResultMsg struct {
	Action string
	Value  string
}

// Model is the command palette model.
type Model struct {
	active   bool
	action   string // what action to perform on selection
	title    string
	query    string
	cursor   int
	items    []Item
	filtered []Item
	width    int
	height   int
}

func New() Model { return Model{} }

func (m Model) Active() bool { return m.active }

func (m *Model) Show(title, action string, items []Item) {
	m.active = true
	m.title = title
	m.action = action
	m.query = ""
	m.cursor = 0
	m.items = items
	m.filter()
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) filter() {
	if m.query == "" {
		m.filtered = m.items
		return
	}
	q := strings.ToLower(m.query)
	m.filtered = nil
	for _, item := range m.items {
		if fuzzyMatch(strings.ToLower(item.Label), q) {
			m.filtered = append(m.filtered, item)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func fuzzyMatch(text, pattern string) bool {
	pi := 0
	for ti := 0; ti < len(text) && pi < len(pattern); ti++ {
		if text[ti] == pattern[pi] {
			pi++
		}
	}
	return pi == len(pattern)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case shared.KeyEscape:
			m.active = false
			return m, nil
		case shared.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				selected := m.filtered[m.cursor]
				action := m.action
				m.active = false
				return m, func() tea.Msg {
					return ResultMsg{Action: action, Value: selected.Value}
				}
			}
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "backspace":
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filter()
			}
		default:
			if msg.Type == tea.KeySpace || msg.Type == tea.KeyRunes {
				var ch string
				if msg.Type == tea.KeySpace {
					ch = " "
				} else {
					ch = string(msg.Runes)
				}
				m.query += ch
				m.filter()
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	paletteW := m.width * 60 / 100
	if paletteW < 30 {
		paletteW = 30
	}
	if paletteW > 80 {
		paletteW = 80
	}
	innerW := paletteW - 4

	var b strings.Builder

	// Title
	b.WriteString(shared.DialogTitleStyle.Render(m.title))
	b.WriteString("\n")

	// Input
	inputDisplay := m.query + "▌"
	inputStyle := shared.CommitInputActiveStyle
	b.WriteString(inputStyle.MaxWidth(innerW + 2).Width(innerW + 2).Render(inputDisplay))
	b.WriteString("\n")

	// Results
	maxResults := 10
	if len(m.filtered) == 0 {
		b.WriteString(shared.HelpDescStyle.Render("  No matches"))
		b.WriteString("\n")
	} else {
		start := 0
		if m.cursor >= maxResults {
			start = m.cursor - maxResults + 1
		}
		end := start + maxResults
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := start; i < end; i++ {
			item := m.filtered[i]
			line := "  " + item.Label
			if item.Detail != "" {
				line += "  " + shared.HelpDescStyle.Render(item.Detail)
			}
			line = lipgloss.NewStyle().MaxWidth(innerW).Render(line)
			if i == m.cursor {
				line = shared.CursorStyle.Width(innerW).Render(line)
			} else {
				line = lipgloss.NewStyle().Width(innerW).Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	content := shared.DialogStyle.Width(paletteW).Render(b.String())

	// Position at top center
	contentW := lipgloss.Width(content)
	padX := (m.width - contentW) / 2
	if padX < 0 {
		padX = 0
	}
	padY := 2

	return lipgloss.NewStyle().
		MarginLeft(padX).
		MarginTop(padY).
		Render(content)
}
