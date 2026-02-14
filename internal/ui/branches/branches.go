package branches

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/git"
	"github.com/0xdeafcafe/moron/internal/shared"
)

type treeNode struct {
	Label      string
	Branch     *git.Branch
	Tag        *git.Tag
	Remote     *git.Remote
	Worktree   *git.Worktree
	IsGroup    bool
	IsExpanded bool
	Depth      int
}

// Model is the branches panel model.
type Model struct {
	width          int
	height         int
	focused        bool
	cursor         int
	offset         int
	nodes          []treeNode
	branches       []git.Branch
	remoteBranches []git.Branch
	tags           []git.Tag
	remotes        []git.Remote
	worktrees      []git.Worktree
	currentBranch  string
	repoDir        string
	firstBuild     bool
	// Branch creation input
	creating       bool
	createInput    string
	createCursor   int
}

func New() Model {
	return Model{firstBuild: true}
}

func (m *Model) SetSize(w, h int)       { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)      { m.focused = f; if !f { m.creating = false } }
func (m *Model) SetRepoDir(dir string)  { m.repoDir = dir }
func (m Model) IsTyping() bool          { return m.creating }

// HintKeys returns context-sensitive shortcut hints.
func (m Model) HintKeys() string {
	if m.creating {
		return shared.HelpKeyStyle.Render("enter") + " create  " +
			shared.HelpKeyStyle.Render("esc") + " cancel"
	}
	return shared.HelpKeyStyle.Render("enter") + " checkout  " +
		shared.HelpKeyStyle.Render("n") + " new  " +
		shared.HelpKeyStyle.Render("x") + " delete"
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.BranchesUpdatedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.branches = msg.Branches
		m.remoteBranches = msg.RemoteBranches
		m.tags = msg.Tags
		m.remotes = msg.Remotes
		m.worktrees = msg.Worktrees
		m.currentBranch = msg.CurrentBranch
		m.rebuildTree()
		return m, nil

	case tea.MouseMsg:
		if !m.focused || m.creating {
			return m, nil
		}
		switch msg.Type {
		case tea.MouseWheelUp:
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case tea.MouseWheelDown:
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
				m.ensureVisible()
			}
		}

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		if m.creating {
			return m.handleCreateInput(msg)
		}

		switch msg.String() {
		case shared.KeyJ, shared.KeyDown:
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case shared.KeyK, shared.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case shared.KeySpace:
			if m.cursor < len(m.nodes) && m.nodes[m.cursor].IsGroup {
				m.nodes[m.cursor].IsExpanded = !m.nodes[m.cursor].IsExpanded
				m.rebuildTree()
			}
		case shared.KeyEnter:
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.Branch != nil && !node.Branch.IsCurrent {
					branch := node.Branch.Name
					return m, func() tea.Msg {
						err := git.Checkout(m.repoDir, branch)
						return shared.CheckoutResultMsg{Branch: branch, Err: err}
					}
				}
			}
		case shared.KeyNewBranch:
			m.creating = true
			m.createInput = ""
			m.createCursor = 0
			return m, nil
		case shared.KeyDeleteBranch:
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.Branch != nil && !node.Branch.IsCurrent && !node.Branch.IsRemote {
					branchName := node.Branch.Name
					repoDir := m.repoDir
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Delete Branch",
							Message: fmt.Sprintf("Delete branch %s?", branchName),
							OnConfirm: func() tea.Msg {
								err := git.DeleteBranch(repoDir, branchName, false)
								return shared.DeleteBranchResultMsg{Branch: branchName, Err: err}
							},
						}
					}
				}
			}
		case shared.KeyPush:
			curBranch := m.currentBranch
			repoDir := m.repoDir
			return m, func() tea.Msg {
				return shared.ShowDialogMsg{
					Type:    shared.DialogConfirm,
					Title:   "Push",
					Message: fmt.Sprintf("Push %s to origin?", curBranch),
					OnConfirm: func() tea.Msg {
						err := git.Push(repoDir, "origin", curBranch, false)
						return shared.PushResultMsg{Err: err}
					},
				}
			}
		case shared.KeyForcePush:
			curBranch := m.currentBranch
			repoDir := m.repoDir
			return m, func() tea.Msg {
				return shared.ShowDialogMsg{
					Type:    shared.DialogConfirm,
					Title:   "Force Push",
					Message: fmt.Sprintf("Force push %s to origin?", curBranch),
					OnConfirm: func() tea.Msg {
						err := git.Push(repoDir, "origin", curBranch, true)
						return shared.PushResultMsg{Err: err}
					},
				}
			}
		case "f":
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.Fetch(repoDir, "origin")
				return shared.FetchResultMsg{Err: err}
			}
		case shared.KeyPull:
			curBranch := m.currentBranch
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.Pull(repoDir, "origin", curBranch)
				return shared.PullResultMsg{Err: err}
			}
		case shared.KeyRebase:
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.Branch != nil {
					target := node.Branch.Name
					repoDir := m.repoDir
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Rebase",
							Message: fmt.Sprintf("Rebase onto %s?", target),
							OnConfirm: func() tea.Msg {
								err := git.Rebase(repoDir, target)
								return shared.RebaseResultMsg{Err: err}
							},
						}
					}
				}
			}
		}
	}

	return m, nil
}

func (m Model) handleCreateInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case shared.KeyEscape:
		m.creating = false
		m.createInput = ""
	case shared.KeyEnter:
		name := strings.TrimSpace(m.createInput)
		if name != "" {
			m.creating = false
			m.createInput = ""
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.CreateBranch(repoDir, name)
				return shared.CreateBranchResultMsg{Branch: name, Err: err}
			}
		}
	case "backspace":
		if m.createCursor > 0 {
			m.createInput = m.createInput[:m.createCursor-1] + m.createInput[m.createCursor:]
			m.createCursor--
		}
	case "left":
		if m.createCursor > 0 {
			m.createCursor--
		}
	case "right":
		if m.createCursor < len(m.createInput) {
			m.createCursor++
		}
	case "home", "ctrl+a":
		m.createCursor = 0
	case "end", "ctrl+e":
		m.createCursor = len(m.createInput)
	case "ctrl+u":
		m.createInput = m.createInput[m.createCursor:]
		m.createCursor = 0
	default:
		if msg.Type == tea.KeySpace || msg.Type == tea.KeyRunes {
			var ch string
			if msg.Type == tea.KeySpace {
				ch = "-" // spaces → dashes for branch names
			} else {
				ch = string(msg.Runes)
			}
			m.createInput = m.createInput[:m.createCursor] + ch + m.createInput[m.createCursor:]
			m.createCursor += len(ch)
		}
	}
	return m, nil
}

func (m *Model) ensureVisible() {
	viewHeight := m.height - 2 - 1 // borders + title
	if viewHeight < 1 {
		viewHeight = 1
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+viewHeight {
		m.offset = m.cursor - viewHeight + 1
	}
}

func (m *Model) rebuildTree() {
	expandState := make(map[string]bool)
	for _, n := range m.nodes {
		if n.IsGroup {
			expandState[n.Label] = n.IsExpanded
		}
	}

	var nodes []treeNode

	// --- Branches ---
	groups := make(map[string][]git.Branch)
	var ungrouped []git.Branch
	for _, b := range m.branches {
		if idx := strings.Index(b.Name, "/"); idx > 0 {
			prefix := b.Name[:idx]
			groups[prefix] = append(groups[prefix], b)
		} else {
			ungrouped = append(ungrouped, b)
		}
	}

	branchLabel := fmt.Sprintf("Branches (%d)", len(m.branches))
	branchesExpanded := true
	if v, ok := expandState[branchLabel]; ok {
		branchesExpanded = v
	} else if v, ok := expandState[m.findOldLabel("Branches")]; ok {
		branchesExpanded = v
	}
	nodes = append(nodes, treeNode{Label: branchLabel, IsGroup: true, IsExpanded: branchesExpanded})

	if branchesExpanded {
		for i := range ungrouped {
			b := ungrouped[i]
			nodes = append(nodes, treeNode{Label: b.Name, Branch: &b, Depth: 1})
		}

		for prefix, brs := range groups {
			expanded := true
			if v, ok := expandState[prefix+"/"]; ok {
				expanded = v
			}
			nodes = append(nodes, treeNode{Label: prefix + "/", IsGroup: true, IsExpanded: expanded, Depth: 1})
			if expanded {
				for i := range brs {
					b := brs[i]
					shortName := strings.TrimPrefix(b.Name, prefix+"/")
					nodes = append(nodes, treeNode{Label: shortName, Branch: &b, Depth: 2})
				}
			}
		}
	}

	// --- Tags (collapsed by default) ---
	if len(m.tags) > 0 {
		tagLabel := fmt.Sprintf("Tags (%d)", len(m.tags))
		tagsExpanded := false
		if v, ok := expandState[tagLabel]; ok {
			tagsExpanded = v
		} else if v, ok := expandState[m.findOldLabel("Tags")]; ok {
			tagsExpanded = v
		} else if !m.firstBuild {
			tagsExpanded = false
		}
		nodes = append(nodes, treeNode{Label: tagLabel, IsGroup: true, IsExpanded: tagsExpanded})
		if tagsExpanded {
			for i := range m.tags {
				t := m.tags[i]
				nodes = append(nodes, treeNode{Label: t.Name, Tag: &t, Depth: 1})
			}
		}
	}

	// --- Remotes ---
	if len(m.remotes) > 0 {
		remoteBranchCount := len(m.remoteBranches)
		remoteLabel := fmt.Sprintf("Remotes (%d)", remoteBranchCount)
		remotesExpanded := true
		if v, ok := expandState[remoteLabel]; ok {
			remotesExpanded = v
		} else if v, ok := expandState[m.findOldLabel("Remotes")]; ok {
			remotesExpanded = v
		}
		nodes = append(nodes, treeNode{Label: remoteLabel, IsGroup: true, IsExpanded: remotesExpanded})
		if remotesExpanded {
			for _, remote := range m.remotes {
				var count int
				for _, rb := range m.remoteBranches {
					if strings.HasPrefix(rb.Name, remote.Name+"/") {
						count++
					}
				}
				remoteNodeLabel := fmt.Sprintf("%s (%d)", remote.Name, count)
				rExpanded := false
				if v, ok := expandState[remoteNodeLabel]; ok {
					rExpanded = v
				} else if v, ok := expandState[remote.Name]; ok {
					rExpanded = v
				}
				r := remote
				nodes = append(nodes, treeNode{
					Label: remoteNodeLabel, Remote: &r, IsGroup: true, IsExpanded: rExpanded, Depth: 1,
				})
				if rExpanded {
					for i := range m.remoteBranches {
						rb := m.remoteBranches[i]
						if strings.HasPrefix(rb.Name, remote.Name+"/") {
							shortName := strings.TrimPrefix(rb.Name, remote.Name+"/")
							nodes = append(nodes, treeNode{Label: shortName, Branch: &rb, Depth: 2})
						}
					}
				}
			}
		}
	}

	// --- Worktrees ---
	if len(m.worktrees) > 1 {
		wtLabel := fmt.Sprintf("Worktrees (%d)", len(m.worktrees))
		wtExpanded := true
		if v, ok := expandState[wtLabel]; ok {
			wtExpanded = v
		} else if v, ok := expandState[m.findOldLabel("Worktrees")]; ok {
			wtExpanded = v
		}
		nodes = append(nodes, treeNode{Label: wtLabel, IsGroup: true, IsExpanded: wtExpanded})
		if wtExpanded {
			for i := range m.worktrees {
				wt := m.worktrees[i]
				label := wt.Path
				if wt.Branch != "" {
					label = wt.Branch + " → " + wt.Path
				}
				if wt.IsBare {
					label += " (bare)"
				}
				nodes = append(nodes, treeNode{Label: label, Worktree: &wt, Depth: 1})
			}
		}
	}

	m.firstBuild = false
	m.nodes = nodes
	if m.cursor >= len(m.nodes) {
		m.cursor = len(m.nodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) findOldLabel(prefix string) string {
	for _, n := range m.nodes {
		if n.IsGroup && strings.HasPrefix(n.Label, prefix) {
			return n.Label
		}
	}
	return ""
}

func (m Model) View() string {
	style := shared.PanelStyle
	titleStyle := shared.PanelTitleStyle
	if m.focused {
		style = shared.ActivePanelStyle
		titleStyle = shared.ActivePanelTitleStyle
	}

	innerW := m.width - 2
	innerH := m.height - 2

	titleText := "Branches"
	if m.currentBranch != "" {
		titleText = "● " + m.currentBranch
	}
	title := titleStyle.MaxWidth(innerW).Render(titleText)

	contentHeight := innerH - 1 // subtract title line
	if contentHeight < 0 {
		contentHeight = 0
	}

	// If creating a branch, reserve 1 line for the input at the top
	var createLine string
	if m.creating {
		inputStyle := shared.CommitInputActiveStyle
		display := m.createInput
		prefix := "New branch: "
		maxInputW := innerW - 2 - len(prefix)
		if maxInputW < 1 {
			maxInputW = 1
		}
		pos := m.createCursor
		if pos > len(display) {
			pos = len(display)
		}
		viewStart := 0
		if pos > maxInputW-1 {
			viewStart = pos - maxInputW + 1
		}
		if viewStart < 0 {
			viewStart = 0
		}
		viewEnd := viewStart + maxInputW
		if viewEnd > len(display) {
			viewEnd = len(display)
		}
		visible := display[viewStart:viewEnd]
		cursorInView := pos - viewStart
		if cursorInView > len(visible) {
			cursorInView = len(visible)
		}
		display = visible[:cursorInView] + "▌" + visible[cursorInView:]
		createLine = inputStyle.MaxWidth(innerW).Render(prefix + display)
		contentHeight-- // take 1 line from tree
	}

	var lines []string
	visibleEnd := m.offset + contentHeight
	if visibleEnd > len(m.nodes) {
		visibleEnd = len(m.nodes)
	}

	if len(m.nodes) == 0 {
		lines = append(lines, shared.HelpDescStyle.Render("  No branches"))
	}

	for i := m.offset; i < visibleEnd; i++ {
		node := m.nodes[i]
		indent := strings.Repeat("  ", node.Depth)

		var line string
		if node.IsGroup {
			arrow := "▶"
			if node.IsExpanded {
				arrow = "▼"
			}
			line = indent + shared.BranchGroupStyle.Render(arrow+" "+node.Label)
		} else if node.Branch != nil {
			if node.Branch.IsCurrent {
				line = indent + shared.BranchCurrentStyle.Render("● "+node.Label)
			} else {
				line = indent + shared.BranchStyle.Render("  "+node.Label)
			}
		} else if node.Tag != nil {
			line = indent + shared.BranchStyle.Render("  "+node.Label)
		} else if node.Worktree != nil {
			line = indent + shared.BranchStyle.Render("  "+node.Label)
		} else {
			line = indent + node.Label
		}

		line = lipgloss.NewStyle().MaxWidth(innerW).Render(line)
		if i == m.cursor && m.focused && !m.creating {
			line = shared.CursorStyle.Width(innerW).Render(line)
		} else {
			line = lipgloss.NewStyle().Width(innerW).Render(line)
		}

		lines = append(lines, line)
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	// Scroll indicators
	if m.offset > 0 && len(lines) > 0 {
		lines[0] = lipgloss.NewStyle().Width(innerW).Render(shared.HelpDescStyle.Render("↑ more"))
	}
	if visibleEnd < len(m.nodes) && contentHeight > 1 {
		lines[contentHeight-1] = lipgloss.NewStyle().Width(innerW).Render(shared.HelpDescStyle.Render("↓ more"))
	}

	var body string
	if m.creating {
		body = createLine + "\n" + strings.Join(lines, "\n")
	} else {
		body = strings.Join(lines, "\n")
	}
	return style.Width(innerW).Height(innerH).Render(title + "\n" + body)
}
