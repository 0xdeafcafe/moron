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
	Stash      *git.StashEntry
	IsGroup    bool
	IsExpanded bool
	Depth      int
}

type inputMode int

const (
	inputNone inputMode = iota
	inputCreateBranch
	inputAddRemoteName
	inputAddRemoteURL
	inputCreateTag
)

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
	stashes        []git.StashEntry
	currentBranch  string
	repoDir        string
	firstBuild     bool
	stagedCount    int
	unstagedCount  int
	// Input mode
	mode         inputMode
	inputText    string
	inputCursor  int
	remoteName   string // stored between remote name and URL steps
	tagRef       string // ref for tag creation (empty = HEAD)
}

func New() Model {
	return Model{firstBuild: true}
}

func (m *Model) SetSize(w, h int)       { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool) tea.Cmd {
	m.focused = f
	if !f {
		m.mode = inputNone
		return nil
	}
	return m.emitLogRequest()
}
func (m *Model) SetRepoDir(dir string)  { m.repoDir = dir }
func (m Model) IsTyping() bool          { return m.mode != inputNone }

func (m Model) CurrentBranch() string        { return m.currentBranch }
func (m Model) AllBranches() []git.Branch    { return m.branches }
func (m Model) AllRemoteBranches() []git.Branch { return m.remoteBranches }
func (m *Model) StartCreateBranch() { m.mode = inputCreateBranch; m.inputText = ""; m.inputCursor = 0 }
func (m *Model) StartAddRemote()    { m.mode = inputAddRemoteName; m.inputText = ""; m.inputCursor = 0 }
func (m *Model) StartCreateTag(ref string) { m.mode = inputCreateTag; m.inputText = ""; m.inputCursor = 0; m.tagRef = ref }

// PushRemote returns the remote name to push to for the current branch.
// Uses the upstream remote if set, otherwise falls back to the first remote.
func (m Model) PushRemote() string {
	for _, b := range m.branches {
		if b.IsCurrent && b.Upstream != "" {
			// Upstream is like "origin/main" — extract the remote name
			if idx := strings.Index(b.Upstream, "/"); idx > 0 {
				return b.Upstream[:idx]
			}
		}
	}
	if len(m.remotes) > 0 {
		return m.remotes[0].Name
	}
	return "origin"
}

// HintKeys returns context-sensitive shortcut hints.
func (m Model) HintKeys() string {
	if m.mode != inputNone {
		return shared.HelpKeyStyle.Render("enter") + " confirm  " +
			shared.HelpKeyStyle.Render("esc") + " cancel"
	}
	return shared.HelpKeyStyle.Render("enter") + " checkout  " +
		shared.HelpKeyStyle.Render("n") + " new  " +
		shared.HelpKeyStyle.Render("x") + " delete"
}

// SelectedBranch returns the branch name under the cursor, if any.
func (m Model) SelectedBranch() string {
	if m.cursor < len(m.nodes) && m.nodes[m.cursor].Branch != nil {
		return m.nodes[m.cursor].Branch.Name
	}
	return ""
}

// SelectedTag returns the tag name under the cursor, if any.
func (m Model) SelectedTag() string {
	if m.cursor < len(m.nodes) && m.nodes[m.cursor].Tag != nil {
		return m.nodes[m.cursor].Tag.Name
	}
	return ""
}

func (m Model) emitLogRequest() tea.Cmd {
	branch := m.SelectedBranch()
	if branch == "" {
		return nil
	}
	repoDir := m.repoDir
	return func() tea.Msg {
		entries, err := git.Log(repoDir, 50, branch)
		return shared.BranchLogMsg{Branch: branch, Log: entries, Err: err}
	}
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
		if m.focused {
			return m, m.emitLogRequest()
		}
		return m, nil

	case shared.StashesUpdatedMsg:
		if msg.Err == nil {
			m.stashes = msg.Stashes
			m.rebuildTree()
		}
		return m, nil

	case shared.StatusUpdatedMsg:
		if msg.Err == nil {
			m.stagedCount = 0
			m.unstagedCount = 0
			for _, f := range msg.Files {
				if f.Staged != "" {
					m.stagedCount++
				}
				if f.Unstaged != "" {
					m.unstagedCount++
				}
			}
		}
		return m, nil

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		if m.mode != inputNone {
			return m.handleInput(msg)
		}

		switch msg.String() {
		case shared.KeyDown:
			if len(m.nodes) > 0 {
				m.cursor++
				if m.cursor >= len(m.nodes) {
					m.cursor = 0
				}
				m.ensureVisible()
				return m, m.emitLogRequest()
			}
		case shared.KeyUp:
			if len(m.nodes) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.nodes) - 1
				}
				m.ensureVisible()
				return m, m.emitLogRequest()
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
				if node.Stash != nil {
					stashIdx := node.Stash.Index
					repoDir := m.repoDir
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Pop Stash",
							Message: fmt.Sprintf("Pop %s?", stashIdx),
							OnConfirm: func() tea.Msg {
								err := git.StashPop(repoDir, stashIdx)
								return shared.StashResultMsg{Action: "pop", Err: err}
							},
						}
					}
				}
			}
		case shared.KeyCreateTag:
			m.StartCreateTag("")
			return m, nil
		case shared.KeyDeleteBranch:
			if m.cursor < len(m.nodes) {
				node := m.nodes[m.cursor]
				if node.Tag != nil {
					tagName := node.Tag.Name
					repoDir := m.repoDir
					remote := m.PushRemote()
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Delete Tag",
							Message: fmt.Sprintf("Delete tag %s locally and from %s?", tagName, remote),
							OnConfirm: func() tea.Msg {
								err := git.DeleteTag(repoDir, tagName)
								if err != nil {
									return shared.DeleteTagResultMsg{Tag: tagName, Err: err}
								}
								// Best-effort remote deletion — ignore errors (tag may not exist on remote)
								_ = git.DeleteRemoteTag(repoDir, remote, tagName)
								return shared.DeleteTagResultMsg{Tag: tagName}
							},
						}
					}
				}
				if node.Stash != nil {
					stashIdx := node.Stash.Index
					repoDir := m.repoDir
					return m, func() tea.Msg {
						return shared.ShowDialogMsg{
							Type:    shared.DialogConfirm,
							Title:   "Drop Stash",
							Message: fmt.Sprintf("Drop %s? This cannot be undone.", stashIdx),
							OnConfirm: func() tea.Msg {
								err := git.StashDrop(repoDir, stashIdx)
								return shared.StashResultMsg{Action: "drop", Err: err}
							},
						}
					}
				}
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

func (m Model) handleInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case shared.KeyEscape:
		m.mode = inputNone
		m.inputText = ""
	case shared.KeyEnter:
		text := strings.TrimSpace(m.inputText)
		if text == "" {
			return m, nil
		}
		switch m.mode {
		case inputCreateBranch:
			m.mode = inputNone
			m.inputText = ""
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.CreateBranch(repoDir, text)
				return shared.CreateBranchResultMsg{Branch: text, Err: err}
			}
		case inputAddRemoteName:
			m.remoteName = text
			m.mode = inputAddRemoteURL
			m.inputText = ""
			m.inputCursor = 0
			return m, nil
		case inputAddRemoteURL:
			remoteName := m.remoteName
			m.mode = inputNone
			m.inputText = ""
			m.remoteName = ""
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.AddRemote(repoDir, remoteName, text)
				return shared.AddRemoteResultMsg{Name: remoteName, Err: err}
			}
		case inputCreateTag:
			ref := m.tagRef
			m.mode = inputNone
			m.inputText = ""
			m.tagRef = ""
			repoDir := m.repoDir
			return m, func() tea.Msg {
				err := git.CreateTag(repoDir, text, ref)
				return shared.CreateTagResultMsg{Tag: text, Err: err}
			}
		}
	case "backspace":
		if m.inputCursor > 0 {
			m.inputText = m.inputText[:m.inputCursor-1] + m.inputText[m.inputCursor:]
			m.inputCursor--
		}
	case "left":
		if m.inputCursor > 0 {
			m.inputCursor--
		}
	case "right":
		if m.inputCursor < len(m.inputText) {
			m.inputCursor++
		}
	case "home", "ctrl+a":
		m.inputCursor = 0
	case "end", "ctrl+e":
		m.inputCursor = len(m.inputText)
	case "ctrl+u":
		m.inputText = m.inputText[m.inputCursor:]
		m.inputCursor = 0
	default:
		if msg.Type == tea.KeySpace || msg.Type == tea.KeyRunes {
			var ch string
			if msg.Type == tea.KeySpace {
				if m.mode == inputCreateBranch {
					ch = "-" // spaces → dashes for branch names
				} else {
					ch = " "
				}
			} else {
				ch = string(msg.Runes)
			}
			m.inputText = m.inputText[:m.inputCursor] + ch + m.inputText[m.inputCursor:]
			m.inputCursor += len(ch)
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
	tagLabel := fmt.Sprintf("Tags (%d)", len(m.tags))
	tagsExpanded := false
	if v, ok := expandState[tagLabel]; ok {
		tagsExpanded = v
	} else if v, ok := expandState[m.findOldLabel("Tags")]; ok {
		tagsExpanded = v
	}
	nodes = append(nodes, treeNode{Label: tagLabel, IsGroup: true, IsExpanded: tagsExpanded})
	if tagsExpanded {
		for i := range m.tags {
			t := m.tags[i]
			nodes = append(nodes, treeNode{Label: t.Name, Tag: &t, Depth: 1})
		}
	}

	// --- Remotes ---
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

	// --- Stashes ---
	stashLabel := fmt.Sprintf("Stashes (%d)", len(m.stashes))
	stashExpanded := len(m.stashes) > 0
	if v, ok := expandState[stashLabel]; ok {
		stashExpanded = v
	} else if v, ok := expandState[m.findOldLabel("Stashes")]; ok {
		stashExpanded = v
	}
	nodes = append(nodes, treeNode{Label: stashLabel, IsGroup: true, IsExpanded: stashExpanded})
	if stashExpanded {
		for i := range m.stashes {
			s := m.stashes[i]
			nodes = append(nodes, treeNode{Label: s.Message, Stash: &s, Depth: 1})
		}
	}

	// --- Worktrees (exclude main worktree) ---
	var extraWorktrees []git.Worktree
	for _, wt := range m.worktrees {
		if wt.Path != m.repoDir {
			extraWorktrees = append(extraWorktrees, wt)
		}
	}
	wtLabel := fmt.Sprintf("Worktrees (%d)", len(extraWorktrees))
	wtExpanded := len(extraWorktrees) > 0
	if v, ok := expandState[wtLabel]; ok {
		wtExpanded = v
	} else if v, ok := expandState[m.findOldLabel("Worktrees")]; ok {
		wtExpanded = v
	}
	nodes = append(nodes, treeNode{Label: wtLabel, IsGroup: true, IsExpanded: wtExpanded})
	if wtExpanded {
		for i := range extraWorktrees {
			wt := extraWorktrees[i]
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// branchTrackInfo returns a short ahead/behind string for a branch.
func branchTrackInfo(b *git.Branch) string {
	if b == nil || (b.Ahead == 0 && b.Behind == 0) {
		return ""
	}
	var parts []string
	if b.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("↑%d", b.Ahead))
	}
	if b.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↓%d", b.Behind))
	}
	return " " + strings.Join(parts, " ")
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
		// Show ahead/behind for current branch in title
		for i := range m.branches {
			if m.branches[i].IsCurrent {
				titleText += branchTrackInfo(&m.branches[i])
				break
			}
		}
	}
	title := titleStyle.MaxWidth(innerW).Render(titleText)

	contentHeight := innerH - 1 // subtract title line
	if contentHeight < 0 {
		contentHeight = 0
	}

	// If in input mode, reserve 1 line for the input at the top
	var inputLine string
	if m.mode != inputNone {
		inputStyle := shared.CommitInputActiveStyle
		display := m.inputText
		var prefix string
		switch m.mode {
		case inputCreateBranch:
			prefix = "New branch: "
		case inputAddRemoteName:
			prefix = "Remote name: "
		case inputAddRemoteURL:
			prefix = fmt.Sprintf("URL for %s: ", m.remoteName)
		case inputCreateTag:
			if m.tagRef != "" {
				prefix = fmt.Sprintf("Tag at %s: ", m.tagRef[:minInt(7, len(m.tagRef))])
			} else {
				prefix = "New tag: "
			}
		}
		maxInputW := innerW - 2 - len(prefix)
		if maxInputW < 1 {
			maxInputW = 1
		}
		pos := m.inputCursor
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
		inputLine = inputStyle.MaxWidth(innerW).Render(prefix + display)
		contentHeight-- // take 1 line from tree
	}

	// Working changes summary
	var changeLine string
	if m.stagedCount > 0 || m.unstagedCount > 0 {
		changeLine = "  " +
			shared.StatusStagedStyle.Render(fmt.Sprintf("●%d", m.stagedCount)) + " " +
			shared.StatusUnstagedStyle.Render(fmt.Sprintf("●%d", m.unstagedCount))
		changeLine = lipgloss.NewStyle().MaxWidth(innerW).Width(innerW).Render(changeLine)
		contentHeight-- // reserve line for changes
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
			label := node.Label
			track := branchTrackInfo(node.Branch)
			if node.Branch.IsCurrent {
				line = indent + shared.BranchCurrentStyle.Render("● "+label) + shared.HelpDescStyle.Render(track)
			} else {
				line = indent + shared.BranchStyle.Render("  "+label) + shared.HelpDescStyle.Render(track)
			}
		} else if node.Tag != nil {
			line = indent + shared.BranchStyle.Render("  "+node.Label)
		} else if node.Stash != nil {
			line = indent + shared.BranchStyle.Render("  "+node.Label)
		} else if node.Worktree != nil {
			line = indent + shared.BranchStyle.Render("  "+node.Label)
		} else {
			line = indent + node.Label
		}

		line = lipgloss.NewStyle().MaxWidth(innerW).Render(line)
		if i == m.cursor && m.focused && m.mode == inputNone {
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

	var bodyParts []string
	if changeLine != "" {
		bodyParts = append(bodyParts, changeLine)
	}
	if m.mode != inputNone {
		bodyParts = append(bodyParts, inputLine)
	}
	bodyParts = append(bodyParts, strings.Join(lines, "\n"))
	body := strings.Join(bodyParts, "\n")
	return style.Width(innerW).Height(innerH).Render(title + "\n" + body)
}
