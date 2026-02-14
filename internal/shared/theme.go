package shared

import "github.com/charmbracelet/lipgloss"

// Colors used throughout the app.
var (
	ColorPrimary    = lipgloss.Color("4")  // blue
	ColorSecondary  = lipgloss.Color("6")  // cyan
	ColorSuccess    = lipgloss.Color("2")  // green
	ColorDanger     = lipgloss.Color("1")  // red
	ColorWarning    = lipgloss.Color("3")  // yellow
	ColorMuted      = lipgloss.Color("8")  // gray
	ColorText       = lipgloss.Color("7")  // white
	ColorBrightText = lipgloss.Color("15") // bright white
	ColorBg         = lipgloss.Color("0")  // black
	ColorHighlight  = lipgloss.Color("5")  // magenta
)

// Styles used by the app.
var (
	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)

	ActivePanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	PanelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBrightText).
			Padding(0, 1)

	ActivePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary).
				Padding(0, 1)

	BranchCurrentStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	BranchStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	BranchGroupStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Bold(true)

	StatusStagedStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	StatusUnstagedStyle = lipgloss.NewStyle().
				Foreground(ColorDanger)

	StatusUntrackedStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)

	DiffAddedStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	DiffRemovedStyle = lipgloss.NewStyle().
				Foreground(ColorDanger)

	DiffContextStyle = lipgloss.NewStyle().
				Foreground(ColorText)

	DiffHunkHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorSecondary)

	DiffLineNumStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	SelectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237"))

	CursorStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Bold(true)

	DialogStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.DoubleBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2).
			Width(50)

	DialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWarning)

	ButtonStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(lipgloss.Color("237")).
			Padding(0, 2)

	ActiveButtonStyle = lipgloss.NewStyle().
				Foreground(ColorBg).
				Background(ColorPrimary).
				Padding(0, 2).
				Bold(true)

	CommitInputStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	CommitInputActiveStyle = lipgloss.NewStyle().
				Foreground(ColorBrightText).
				Background(lipgloss.Color("235")).
				Padding(0, 1)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
