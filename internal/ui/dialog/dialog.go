package dialog

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/shared"
)

// Model is the dialog overlay model.
type Model struct {
	active    bool
	dtype     shared.DialogType
	title     string
	message   string
	yesActive bool
	input     string
	onConfirm func() tea.Cmd
	width     int
	height    int
}

func New() Model {
	return Model{}
}

func (m Model) Active() bool { return m.active }

func (m *Model) Show(dtype shared.DialogType, title, message string) {
	m.active = true
	m.dtype = dtype
	m.title = title
	m.message = message
	m.yesActive = false
	m.input = ""
}

func (m *Model) SetOnConfirm(fn func() tea.Cmd) {
	m.onConfirm = fn
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
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
		case "left", "h":
			m.yesActive = !m.yesActive
		case "right", "l":
			m.yesActive = !m.yesActive
		case "y":
			if m.dtype == shared.DialogConfirm || m.dtype == shared.DialogPush {
				m.active = false
				if m.onConfirm != nil {
					return m, m.onConfirm()
				}
			}
		case "n":
			m.active = false
			return m, nil
		case shared.KeyEnter:
			if m.dtype == shared.DialogError {
				m.active = false
				return m, nil
			}
			if m.yesActive {
				m.active = false
				if m.onConfirm != nil {
					return m, m.onConfirm()
				}
			} else {
				m.active = false
			}
			return m, nil
		case "backspace":
			if m.dtype == shared.DialogInput && len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if m.dtype == shared.DialogInput && msg.Type == tea.KeyRunes {
				m.input += string(msg.Runes)
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.active {
		return ""
	}

	var b strings.Builder

	b.WriteString(shared.DialogTitleStyle.Render(m.title))
	b.WriteString("\n\n")
	b.WriteString(m.message)
	b.WriteString("\n\n")

	if m.dtype == shared.DialogInput {
		inputStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(shared.ColorPrimary).
			Width(40).
			Padding(0, 1)
		b.WriteString(inputStyle.Render(m.input + "▌"))
		b.WriteString("\n\n")
	}

	switch m.dtype {
	case shared.DialogConfirm, shared.DialogPush:
		yesBtn := shared.ButtonStyle.Render(" Yes ")
		noBtn := shared.ButtonStyle.Render(" No ")
		if m.yesActive {
			yesBtn = shared.ActiveButtonStyle.Render(" Yes ")
		} else {
			noBtn = shared.ActiveButtonStyle.Render(" No ")
		}
		b.WriteString(yesBtn + "  " + noBtn)
	case shared.DialogError:
		b.WriteString(shared.ActiveButtonStyle.Render(" OK "))
	case shared.DialogInput:
		okBtn := shared.ButtonStyle.Render(" OK ")
		cancelBtn := shared.ActiveButtonStyle.Render(" Cancel ")
		if m.yesActive {
			okBtn = shared.ActiveButtonStyle.Render(" OK ")
			cancelBtn = shared.ButtonStyle.Render(" Cancel ")
		}
		b.WriteString(okBtn + "  " + cancelBtn)
	}

	return shared.DialogStyle.Render(b.String())
}
