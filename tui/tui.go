// Package tui exposes the moron terminal UI for embedding in other tools.
package tui

import (
	"github.com/0xdeafcafe/moron/internal/app"
	"github.com/0xdeafcafe/moron/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

// Run opens the moron TUI for the repository containing dir and blocks
// until the user quits. The terminal must be a TTY.
func Run(dir string) error {
	repoDir, err := git.FindRepo(dir)
	if err != nil {
		return err
	}

	model := app.New(repoDir)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
