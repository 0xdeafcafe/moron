package app

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/config"
	"github.com/0xdeafcafe/moron/internal/git"
	"github.com/0xdeafcafe/moron/internal/shared"
	"github.com/0xdeafcafe/moron/internal/ui/branches"
	"github.com/0xdeafcafe/moron/internal/ui/dialog"
	"github.com/0xdeafcafe/moron/internal/ui/diffview"
	"github.com/0xdeafcafe/moron/internal/ui/palette"
	"github.com/0xdeafcafe/moron/internal/ui/workingcopy"
)

const fetchInterval = 60 * time.Second

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
	palette     palette.Model

	settings    config.Settings
	statusBar   string
	showHelp    bool
	watcherDone chan struct{}
	watcherCh   chan struct{}
	lastRefresh time.Time
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
	p := palette.New()
	s := config.LoadSettings()

	wc.SetFileView(string(s.FileView))

	return Model{
		repoDir:     repoDir,
		activePanel: shared.PanelBranches,
		branches:    b,
		workingCopy: wc,
		diffView:    dv,
		dialog:      d,
		palette:     p,
		settings:    s,
		watcherDone: make(chan struct{}),
		watcherCh:   make(chan struct{}, 1),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.backgroundFetch(),
		m.refreshAll(),
		m.scheduleFetch(),
		m.startWatcher(),
	)
}

func (m Model) backgroundFetch() tea.Cmd {
	repoDir := m.repoDir
	return func() tea.Msg {
		err := git.FetchAll(repoDir)
		return shared.BackgroundFetchDoneMsg{Err: err}
	}
}

func (m Model) scheduleFetch() tea.Cmd {
	return tea.Tick(fetchInterval, func(time.Time) tea.Msg {
		return shared.TickFetchMsg{}
	})
}

func (m Model) startWatcher() tea.Cmd {
	repoDir := m.repoDir
	done := m.watcherDone
	ch := m.watcherCh

	go git.WatchGitDir(repoDir, func() {
		select {
		case ch <- struct{}{}:
		default:
		}
	}, done)

	return m.listenForGitChange()
}

func (m Model) listenForGitChange() tea.Cmd {
	ch := m.watcherCh
	return func() tea.Msg {
		<-ch
		return shared.GitChangedMsg{}
	}
}

// StopWatcher cleans up the file watcher.
func (m Model) StopWatcher() {
	if m.watcherDone != nil {
		close(m.watcherDone)
	}
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
		// Palette captures all input when active
		if m.palette.Active() {
			var cmd tea.Cmd
			m.palette, cmd = m.palette.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

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
			case shared.KeyTab:
				m.activePanel = (m.activePanel + 1) % 3
				return m, m.updateFocus()
			case shared.KeyShiftTab:
				m.activePanel = (m.activePanel + 2) % 3
				return m, m.updateFocus()
			case "right":
				if m.activePanel == shared.PanelBranches {
					m.branches.ExpandSelected()
				} else {
					m.activePanel = (m.activePanel + 1) % 3
					return m, m.updateFocus()
				}
			case "left":
				if m.activePanel == shared.PanelBranches {
					m.branches.CollapseSelected()
				} else {
					m.activePanel = (m.activePanel + 2) % 3
					return m, m.updateFocus()
				}
			case shared.KeyPanel1:
				m.activePanel = shared.PanelBranches
				return m, m.updateFocus()
			case shared.KeyPanel2:
				m.activePanel = shared.PanelWorkingCopy
				return m, m.updateFocus()
			case shared.KeyPanel3:
				m.activePanel = shared.PanelDiff
				return m, m.updateFocus()
			case shared.KeyNewBranch:
				ref := m.branches.SelectedBranch()
				m.activePanel = shared.PanelBranches
				m.updateFocus()
				m.branches.StartCreateBranch(ref)
				return m, nil
			case shared.KeyAddRemote:
				m.activePanel = shared.PanelBranches
				m.updateFocus()
				m.branches.StartAddRemote()
				return m, nil
			case shared.KeyPush:
				remote := m.branches.PushRemote()
				repoDir := m.repoDir
				if tag := m.branches.SelectedTag(); tag != "" {
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Push Tag",
							Message: fmt.Sprintf("Push tag %s to %s?", tag, remote),
							OnConfirm: func() tea.Msg {
								err := git.PushTag(repoDir, remote, tag)
								return shared.PushResultMsg{Err: err}
							},
						}
					}
				}
				curBranch := m.branches.CurrentBranch()
				return m, func() tea.Msg {
					return shared.ShowDialogMsg{
						Type:    shared.DialogConfirm,
						Title:   "Push",
						Message: fmt.Sprintf("Push %s to %s?", curBranch, remote),
						OnConfirm: func() tea.Msg {
							err := git.Push(repoDir, remote, curBranch, false)
							return shared.PushResultMsg{Err: err}
						},
					}
				}
			case shared.KeyForcePush:
				remote := m.branches.PushRemote()
				curBranch := m.branches.CurrentBranch()
				repoDir := m.repoDir
				return m, func() tea.Msg {
					return shared.ShowDialogMsg{
						Type:    shared.DialogConfirm,
						Title:   "Force Push",
						Message: fmt.Sprintf("Force push %s to %s?", curBranch, remote),
						OnConfirm: func() tea.Msg {
							err := git.Push(repoDir, remote, curBranch, true)
							return shared.PushResultMsg{Err: err}
						},
					}
				}
			case shared.KeyFetch:
				repoDir := m.repoDir
				return m, func() tea.Msg {
					err := git.FetchAll(repoDir)
					return shared.FetchResultMsg{Err: err}
				}
			case "ctrl+p":
				// Command palette — switch branch
				var items []palette.Item
				for _, b := range m.branches.AllBranches() {
					detail := "local"
					if b.IsCurrent {
						detail = "current"
					}
					items = append(items, palette.Item{Label: b.Name, Detail: detail, Value: b.Name})
				}
				for _, rb := range m.branches.AllRemoteBranches() {
					items = append(items, palette.Item{Label: rb.Name, Detail: "remote", Value: rb.Name})
				}
				m.palette.Show("Switch Branch", "checkout", items)
				return m, nil
			case "ctrl+k":
				// Settings palette
				items := []palette.Item{
					{Label: "File View: Tree", Detail: "group files by directory", Value: "tree"},
					{Label: "File View: List", Detail: "flat file list", Value: "list"},
				}
				m.palette.Show("Settings", "fileview", items)
				return m, nil
			case shared.KeyPull:
				remote := m.branches.PushRemote()
				curBranch := m.branches.CurrentBranch()
				repoDir := m.repoDir
				return m, func() tea.Msg {
					err := git.Pull(repoDir, remote, curBranch)
					return shared.PullResultMsg{Err: err}
				}
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
			// Intercept tag creation from log view
			if msg.String() == shared.KeyCreateTag {
				if hash := m.diffView.SelectedLogHash(); hash != "" {
					m.activePanel = shared.PanelBranches
					m.updateFocus()
					m.branches.StartCreateTag(hash)
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.diffView, cmd = m.diffView.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case shared.FocusPanelMsg:
		m.activePanel = msg.Panel
		return m, m.updateFocus()

	case shared.RefreshMsg:
		cmds = append(cmds, m.refreshAll())

	case shared.StatusUpdatedMsg:
		var cmd tea.Cmd
		m.workingCopy, cmd = m.workingCopy.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		m.branches, _ = m.branches.Update(msg)
		if msg.Err != nil {
			m.statusBar = "Error: " + msg.Err.Error()
		}

	case shared.BranchesUpdatedMsg:
		m.branches, _ = m.branches.Update(msg)
		m.workingCopy, _ = m.workingCopy.Update(msg)
		if msg.Err != nil {
			m.statusBar = "Error: " + msg.Err.Error()
		}

	case shared.BranchLogMsg:
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
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
			if git.IsCheckoutConflict(msg.Err) {
				branch := msg.Branch
				repoDir := m.repoDir
				return m, func() tea.Msg {
					return shared.ShowDialogMsg{
						Type:    shared.DialogConfirm,
						Title:   "Stash & Switch",
						Message: fmt.Sprintf("You have uncommitted changes. Stash and switch to %s?", branch),
						OnConfirm: func() tea.Msg {
							if err := git.StashPush(repoDir); err != nil {
								return shared.CheckoutResultMsg{Branch: branch, Err: fmt.Errorf("stash failed: %w", err)}
							}
							err := git.Checkout(repoDir, branch)
							return shared.CheckoutResultMsg{Branch: branch, Err: err}
						},
					}
				}
			}
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

	case shared.BackgroundFetchDoneMsg:
		// Silent fetch completed — watcher will pick up changes

	case shared.TickFetchMsg:
		cmds = append(cmds, m.backgroundFetch(), m.scheduleFetch())

	case shared.GitChangedMsg:
		cmds = append(cmds, m.listenForGitChange())
		// Skip if we recently refreshed from our own action
		if time.Since(m.lastRefresh) > 300*time.Millisecond {
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

	case shared.StashesUpdatedMsg:
		m.branches, _ = m.branches.Update(msg)

	case shared.StashResultMsg:
		if msg.Err != nil {
			m.statusBar = fmt.Sprintf("Stash %s failed: %s", msg.Action, msg.Err.Error())
		} else {
			m.statusBar = fmt.Sprintf("Stash %s successful", msg.Action)
			cmds = append(cmds, m.refreshAll())
		}

	case shared.CreateTagResultMsg:
		if msg.Err != nil {
			m.statusBar = "Create tag failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Created tag " + msg.Tag
			cmds = append(cmds, m.refreshAll())
		}

	case shared.DeleteTagResultMsg:
		if msg.Err != nil {
			m.statusBar = "Delete tag failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Deleted tag " + msg.Tag
			cmds = append(cmds, m.refreshAll())
		}

	case shared.AddRemoteResultMsg:
		if msg.Err != nil {
			m.statusBar = "Add remote failed: " + msg.Err.Error()
		} else {
			m.statusBar = "Added remote " + msg.Name
			cmds = append(cmds, m.refreshAll())
		}

	case palette.ResultMsg:
		switch msg.Action {
		case "checkout":
			branch := msg.Value
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.Checkout(repoDir, branch)
				return shared.CheckoutResultMsg{Branch: branch, Err: err}
			}
		case "fileview":
			m.settings.FileView = config.FileView(msg.Value)
			m.workingCopy.SetFileView(msg.Value)
			_ = config.SaveSettings(m.settings)
			m.statusBar = "File view: " + msg.Value
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

func (m *Model) updateFocus() tea.Cmd {
	cmd := m.branches.SetFocused(m.activePanel == shared.PanelBranches)
	m.workingCopy.SetFocused(m.activePanel == shared.PanelWorkingCopy)
	m.diffView.SetFocused(m.activePanel == shared.PanelDiff)
	return cmd
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
	m.palette.SetSize(m.width, m.height)
}

func (m *Model) refreshAll() tea.Cmd {
	m.lastRefresh = time.Now()
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
		func() tea.Msg {
			stashes, err := git.StashList(repoDir)
			return shared.StashesUpdatedMsg{Stashes: stashes, Err: err}
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

	if m.palette.Active() {
		view = m.overlayDialog(view, m.palette.View())
	}

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
		{"ctrl+p", "Switch branch palette"},
		{"ctrl+k", "Settings palette"},
		{"", ""},
		{"--- Branches ---", ""},
		{"Enter", "Checkout branch"},
		{"Space", "Expand/collapse group"},
		{"n", "Create new branch"},
		{"t", "Create tag"},
		{"x", "Delete branch/tag/stash"},
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
		{"--- Diff / Log ---", ""},
		{"Space / s", "Stage hunk (or selection)"},
		{"u", "Unstage hunk (or selection)"},
		{"d", "Discard hunk"},
		{"v", "Select lines"},
		{"t", "Create tag at commit"},
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
