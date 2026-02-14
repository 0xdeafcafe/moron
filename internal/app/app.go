package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/git"
	"github.com/0xdeafcafe/moron/internal/shared"
	"github.com/0xdeafcafe/moron/internal/ui/branches"
	"github.com/0xdeafcafe/moron/internal/ui/dialog"
	"github.com/0xdeafcafe/moron/internal/ui/diffview"
	"github.com/0xdeafcafe/moron/internal/ui/workingcopy"
)

// Model is the root application model.
type Model struct {
	repoDir     string
	width       int
	height      int
	activePanel shared.Panel

	branches    branches.Model
	workingCopy workingcopy.Model
	diffView    diffview.Model
	dialog      dialog.Model

	statusBar string
	showHelp  bool
}

// New creates a new application model.
func New(repoDir string) Model {
	b := branches.New()
	b.SetRepoDir(repoDir)
	b.SetFocused(true)

	wc := workingcopy.New()
	wc.SetRepoDir(repoDir)

	dv := diffview.New()
	dv.SetRepoDir(repoDir)

	d := dialog.New()

	return Model{
		repoDir:     repoDir,
		activePanel: shared.PanelBranches,
		branches:    b,
		workingCopy: wc,
		diffView:    dv,
		dialog:      d,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.refreshAll(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		return m, nil

	case tea.MouseMsg:
		// Only route mouse scroll to the diff panel
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Dialog captures all input when active
		if m.dialog.Active() {
			var cmd tea.Cmd
			m.dialog, cmd = m.dialog.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Help overlay
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Global keys (unless typing in commit input or branch creation)
		if !m.workingCopy.IsTyping() && !m.branches.IsTyping() {
			switch msg.String() {
			case shared.KeyQuit:
				return m, tea.Quit
			case shared.KeyForceQuit:
				return m, tea.Quit
			case shared.KeyHelp:
				m.showHelp = true
				return m, nil
			case shared.KeyTab, "right":
				m.activePanel = (m.activePanel + 1) % 3
				m.updateFocus()
				return m, nil
			case shared.KeyShiftTab, "left":
				m.activePanel = (m.activePanel + 2) % 3
				m.updateFocus()
				return m, nil
			case shared.KeyPanel1:
				m.activePanel = shared.PanelBranches
				m.updateFocus()
				return m, nil
			case shared.KeyPanel2:
				m.activePanel = shared.PanelWorkingCopy
				m.updateFocus()
				return m, nil
			case shared.KeyPanel3:
				m.activePanel = shared.PanelDiff
				m.updateFocus()
				return m, nil
			}
		}

		// Forward to active panel
		switch m.activePanel {
		case shared.PanelBranches:
			var cmd tea.Cmd
			m.branches, cmd = m.branches.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case shared.PanelWorkingCopy:
			var cmd tea.Cmd
			m.workingCopy, cmd = m.workingCopy.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case shared.PanelDiff:
			var cmd tea.Cmd
			m.diffView, cmd = m.diffView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case shared.FocusPanelMsg:
		m.activePanel = msg.Panel
		m.updateFocus()
		return m, nil

	case shared.RefreshMsg:
		cmds = append(cmds, m.refreshAll())

	case shared.StatusUpdatedMsg:
		m.workingCopy, _ = m.workingCopy.Update(msg)
		if msg.Err != nil {
			m.statusBar = "Error: " + msg.Err.Error()
		}

	case shared.BranchesUpdatedMsg:
		m.branches, _ = m.branches.Update(msg)
		m.workingCopy, _ = m.workingCopy.Update(msg)
		if msg.Err != nil {
			m.statusBar = "Error: " + msg.Err.Error()
		}

	case shared.DiffUpdatedMsg:
		m.diffView, _ = m.diffView.Update(msg)

	case shared.FileSelectedMsg:
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case shared.CommitResultMsg:
		if msg.Err != nil {
			m.statusBar = "Commit failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Committed successfully"
			cmds = append(cmds, m.refreshAll())
		}

	case shared.CheckoutResultMsg:
		if msg.Err != nil {
			m.statusBar = "Checkout failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Switched to " + msg.Branch
			cmds = append(cmds, m.refreshAll())
		}

	case shared.PushResultMsg:
		if msg.Err != nil {
			m.statusBar = "Push failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Pushed successfully"
			cmds = append(cmds, m.refreshAll())
		}

	case shared.FetchResultMsg:
		if msg.Err != nil {
			m.statusBar = "Fetch failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Fetched successfully"
			cmds = append(cmds, m.refreshAll())
		}

	case shared.RebaseResultMsg:
		if msg.Err != nil {
			m.statusBar = "Rebase failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Rebased successfully"
			cmds = append(cmds, m.refreshAll())
		}

	case shared.PullResultMsg:
		if msg.Err != nil {
			m.statusBar = "Pull failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Pulled successfully"
			cmds = append(cmds, m.refreshAll())
		}

	case shared.CreateBranchResultMsg:
		if msg.Err != nil {
			m.statusBar = "Create branch failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Created and switched to " + msg.Branch
			cmds = append(cmds, m.refreshAll())
		}

	case shared.DeleteBranchResultMsg:
		if msg.Err != nil {
			m.statusBar = "Delete branch failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Deleted " + msg.Branch
			cmds = append(cmds, m.refreshAll())
		}

	case shared.AddRemoteResultMsg:
		if msg.Err != nil {
			m.statusBar = "Add remote failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Added remote " + msg.Name
			cmds = append(cmds, m.refreshAll())
		}

	case shared.ErrorMsg:
		m.dialog.Show(shared.DialogError, "Error", msg.Err.Error())

	case shared.ShowDialogMsg:
		m.dialog.Show(msg.Type, msg.Title, msg.Message)
		if msg.OnConfirm != nil {
			onConfirm := msg.OnConfirm
			m.dialog.SetOnConfirm(func() tea.Cmd {
				return onConfirm
			})
		} else if msg.OnYes != nil {
			onYes := msg.OnYes
			m.dialog.SetOnConfirm(func() tea.Cmd {
				return func() tea.Msg {
					onYes()
					return shared.RefreshMsg{}
				}
			})
		}

	case shared.CloseDialogMsg:
		m.dialog = dialog.New()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateFocus() {
	m.branches.SetFocused(m.activePanel == shared.PanelBranches)
	m.workingCopy.SetFocused(m.activePanel == shared.PanelWorkingCopy)
	m.diffView.SetFocused(m.activePanel == shared.PanelDiff)
}

func (m *Model) updateLayout() {
	if m.width == 0 || m.height == 0 {
		return
	}

	statusHeight := 1
	contentHeight := m.height - statusHeight

	leftW := m.width * 20 / 100
	centerW := m.width * 30 / 100

	if leftW < 15 {
		leftW = 15
	}
	if centerW < 20 {
		centerW = 20
	}

	rightW := m.width - leftW - centerW
	if rightW < 10 {
		// Terminal too narrow — shrink center to make room
		rightW = 10
		centerW = m.width - leftW - rightW
		if centerW < 10 {
			centerW = 10
			leftW = m.width - centerW - rightW
			if leftW < 5 {
				leftW = 5
			}
		}
	}

	m.branches.SetSize(leftW, contentHeight)
	m.workingCopy.SetSize(centerW, contentHeight)
	m.diffView.SetSize(rightW, contentHeight)
	m.dialog.SetSize(m.width, m.height)
}

func (m Model) refreshAll() tea.Cmd {
	repoDir := m.repoDir
	return tea.Batch(
		func() tea.Msg {
			files, err := git.Status(repoDir)
			return shared.StatusUpdatedMsg{Files: files, Err: err}
		},
		func() tea.Msg {
			brs, _ := git.ListBranches(repoDir)
			remoteBranches, _ := git.ListRemoteBranches(repoDir)
			tags, _ := git.ListTags(repoDir)
			remotes, _ := git.ListRemotes(repoDir)
			worktrees, _ := git.ListWorktrees(repoDir)
			currentBranch, _ := git.CurrentBranch(repoDir)

			// On repos with no commits, git branch returns empty but
			// we still know the HEAD branch name from symbolic-ref.
			// Add it as a synthetic branch so the UI isn't empty.
			if len(brs) == 0 && currentBranch != "" {
				brs = []git.Branch{{Name: currentBranch, IsCurrent: true}}
			}

			return shared.BranchesUpdatedMsg{
				Branches:       brs,
				RemoteBranches: remoteBranches,
				Tags:           tags,
				Remotes:        remotes,
				Worktrees:      worktrees,
				CurrentBranch:  currentBranch,
			}
		},
	)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	left := m.branches.View()
	center := m.workingCopy.View()
	right := m.diffView.View()

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	// Left side: status message or panel name
	leftText := m.statusBar
	if leftText == "" {
		panelNames := []string{"branches", "working copy", "diff"}
		activeName := panelNames[m.activePanel]
		leftText = fmt.Sprintf(" [%s]  ?: help  q: quit", activeName)
	}

	// Right side: context-sensitive hints
	var hints string
	switch m.activePanel {
	case shared.PanelBranches:
		hints = m.branches.HintKeys()
	case shared.PanelWorkingCopy:
		hints = m.workingCopy.HintKeys()
	case shared.PanelDiff:
		hints = m.diffView.HintKeys()
	}

	// Render: left text + right-aligned hints
	leftRendered := shared.StatusBarStyle.Render(leftText)
	leftWidth := lipgloss.Width(leftRendered)
	hintsRendered := shared.StatusBarStyle.Render(hints + " ")
	hintsWidth := lipgloss.Width(hintsRendered)
	gap := m.width - leftWidth - hintsWidth
	if gap < 1 {
		gap = 1
	}
	statusLine := leftRendered + strings.Repeat(" ", gap) + hintsRendered
	statusLine = lipgloss.NewStyle().MaxWidth(m.width).Width(m.width).Render(statusLine)

	view := mainContent + "\n" + statusLine

	if m.dialog.Active() {
		overlay := m.dialog.View()
		view = m.overlayDialog(view, overlay)
	}

	if m.showHelp {
		view = m.overlayDialog(view, m.renderHelp())
	}

	return view
}

func (m Model) overlayDialog(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	startY := (len(baseLines) - len(overlayLines)) / 2
	if startY < 0 {
		startY = 0
	}

	for i, ol := range overlayLines {
		idx := startY + i
		if idx < len(baseLines) {
			olWidth := lipgloss.Width(ol)
			padLeft := (m.width - olWidth) / 2
			if padLeft < 0 {
				padLeft = 0
			}
			baseLines[idx] = strings.Repeat(" ", padLeft) + ol
		}
	}

	return strings.Join(baseLines, "\n")
}

func (m Model) renderHelp() string {
	help := []struct{ key, desc string }{
		{"q / ctrl+c", "Quit"},
		{"← / → / Tab", "Switch panel"},
		{"1 / 2 / 3", "Focus panel"},
		{"↑ / ↓", "Navigate"},
		{"", ""},
		{"--- Branches ---", ""},
		{"Enter", "Checkout branch"},
		{"Space", "Expand/collapse group"},
		{"n", "Create new branch"},
		{"x", "Delete branch"},
		{"A", "Add remote"},
		{"p / P", "Push / Force push"},
		{"l", "Pull from remote"},
		{"f", "Fetch from remote"},
		{"r", "Rebase onto branch"},
		{"", ""},
		{"--- Files ---", ""},
		{"Space / s", "Stage/unstage file"},
		{"Enter", "Open diff for hunk staging"},
		{"S", "Stage all files"},
		{"u", "Unstage all files"},
		{"d", "Discard file changes"},
		{"i", "Add to .gitignore"},
		{"c", "Compose commit message"},
		{"enter", "Submit commit"},
		{"a", "Toggle amend"},
		{"", ""},
		{"--- Diff ---", ""},
		{"Space / s", "Stage hunk (or selection)"},
		{"u", "Unstage hunk (or selection)"},
		{"d", "Discard hunk"},
		{"v", "Select lines"},
		{"t", "Toggle staged/unstaged"},
		{"J / K", "Next/prev hunk"},
		{"ctrl+d / ctrl+u", "Half-page scroll"},
		{"?", "Toggle this help"},
	}

	var b strings.Builder
	b.WriteString(shared.DialogTitleStyle.Render("Keybindings"))
	b.WriteString("\n\n")
	for _, h := range help {
		if h.key == "" && h.desc == "" {
			b.WriteString("\n")
			continue
		}
		if strings.HasPrefix(h.key, "---") {
			b.WriteString(shared.HelpDescStyle.Render(h.key))
			b.WriteString("\n")
			continue
		}
		b.WriteString(shared.HelpKeyStyle.Render(fmt.Sprintf("%17s", h.key)))
		b.WriteString("  ")
		b.WriteString(shared.HelpDescStyle.Render(h.desc))
		b.WriteString("\n")
	}

	return shared.DialogStyle.Width(50).Render(b.String())
}
