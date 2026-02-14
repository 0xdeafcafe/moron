package workingcopy

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/git"
	"github.com/0xdeafcafe/moron/internal/shared"
)

type section int

const (
	sectionStaged section = iota
	sectionUnstaged
	sectionCommit
)

// Model is the working copy panel model.
type Model struct {
	width         int
	height        int
	focused       bool
	repoDir       string
	currentBranch string
	staged        []git.FileStatus
	unstaged      []git.FileStatus
	cursor        int
	offset        int
	activeSection section
	commitMsg     string
	commitCursor  int
	amend         bool
	typing        bool
	returnCursor  int    // cursor position to return to on next navigate (-1 = none)
	fileView      string // "tree" or "list"
}

func New() Model { return Model{returnCursor: -1, fileView: "tree"} }

func (m *Model) SetSize(w, h int)            { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)           { m.focused = f; if !f { m.typing = false } }
func (m *Model) SetRepoDir(dir string)       { m.repoDir = dir }
func (m *Model) SetCurrentBranch(name string) { m.currentBranch = name }
func (m Model) IsTyping() bool               { return m.typing }
func (m *Model) SetFileView(v string)        { m.fileView = v }

// HintKeys returns context-sensitive shortcut hints.
func (m Model) HintKeys() string {
	if m.typing {
		return shared.HelpKeyStyle.Render("enter") + " commit  " +
			shared.HelpKeyStyle.Render("esc") + " cancel"
	}
	return shared.HelpKeyStyle.Render("s") + " stage  " +
		shared.HelpKeyStyle.Render("c") + " commit  " +
		shared.HelpKeyStyle.Render("i") + " ignore"
}

// SelectedFile returns the currently selected file, whether it's staged, and whether it's untracked.
func (m Model) SelectedFile() (path string, staged bool, untracked bool) {
	if m.activeSection == sectionStaged {
		if m.cursor < len(m.staged) {
			return m.staged[m.cursor].Path, true, false
		}
	} else if m.activeSection == sectionUnstaged {
		idx := m.cursor - len(m.staged) - 1
		if idx >= 0 && idx < len(m.unstaged) {
			f := m.unstaged[idx]
			return f.Path, false, f.Unstaged == "?"
		}
	}
	return "", false, false
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.StatusUpdatedMsg:
		if msg.Err != nil {
			return m, nil
		}
		prevPath, _, _ := m.SelectedFile()
		m.staged = nil
		m.unstaged = nil
		for _, f := range msg.Files {
			if f.Staged != "" {
				m.staged = append(m.staged, f)
			}
			if f.Unstaged != "" {
				m.unstaged = append(m.unstaged, f)
			}
		}
		sort.Slice(m.staged, func(i, j int) bool { return m.staged[i].Path < m.staged[j].Path })
		sort.Slice(m.unstaged, func(i, j int) bool { return m.unstaged[i].Path < m.unstaged[j].Path })
		// Try to keep same file selected
		if prevPath != "" {
			m.selectByPath(prevPath)
		}
		m.clampCursor()
		m.skipSeparator(1) // ensure cursor isn't on section header
		m.updateSection()
		// Auto-select first file if we have one
		return m, m.emitFileSelected()

	case shared.BranchesUpdatedMsg:
		if msg.Err == nil {
			m.currentBranch = msg.CurrentBranch
		}
		return m, nil

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		if m.typing {
			return m.handleCommitInput(msg)
		}

		switch msg.String() {
		case shared.KeyDown:
			total := m.totalItems()
			if total > 0 {
				if m.returnCursor >= 0 {
					m.cursor = m.returnCursor
					m.returnCursor = -1
				} else {
					m.cursor++
					if m.cursor >= total {
						m.cursor = 0
					}
				}
				m.skipSeparator(1)
				m.clampCursor()
				m.updateSection()
				return m, m.emitFileSelected()
			}
		case shared.KeyUp:
			total := m.totalItems()
			if total > 0 {
				if m.returnCursor >= 0 {
					m.cursor = m.returnCursor
					m.returnCursor = -1
				} else {
					m.cursor--
					if m.cursor < 0 {
						m.cursor = total - 1
					}
				}
				m.skipSeparator(-1)
				m.clampCursor()
				m.updateSection()
				return m, m.emitFileSelected()
			}
		case shared.KeySpace, shared.KeyStage:
			m.returnCursor = m.cursor
			return m, m.stageUnstageFile()
		case shared.KeyStageAll:
			m.returnCursor = -1
			return m, m.stageAll()
		case shared.KeyUnstage:
			m.returnCursor = -1
			return m, m.unstageAll()
		case shared.KeyDiscard:
			path, staged, _ := m.SelectedFile()
			if path != "" && !staged {
				repoDir := m.repoDir
				return m, func() tea.Msg {
					return shared.ShowDialogMsg{
						Type:    shared.DialogConfirm,
						Title:   "Discard Changes",
						Message: "Discard changes to " + path + "?",
						OnConfirm: func() tea.Msg {
							err := git.DiscardFile(repoDir, path)
							if err != nil {
								return shared.ErrorMsg{Err: err}
							}
							return shared.RefreshMsg{}
						},
					}
				}
			}
		case shared.KeyIgnore:
			path, staged, _ := m.SelectedFile()
			if path != "" && !staged {
				repoDir := m.repoDir
				return m, func() tea.Msg {
					return shared.ShowDialogMsg{
						Type:    shared.DialogConfirm,
						Title:   "Add to .gitignore",
						Message: "Add " + path + " to .gitignore?",
						OnConfirm: func() tea.Msg {
							err := git.AddToGitignore(repoDir, path)
							if err != nil {
								return shared.ErrorMsg{Err: err}
							}
							return shared.RefreshMsg{}
						},
					}
				}
			}
		case shared.KeyCommit:
			m.typing = true
			m.commitCursor = len(m.commitMsg)
			m.activeSection = sectionCommit
			return m, nil
		case shared.KeyAmend:
			m.amend = !m.amend
			if m.amend {
				// Load the last commit message when enabling amend
				repoDir := m.repoDir
				return m, func() tea.Msg {
					msg, _ := git.LastCommitMessage(repoDir)
					return lastCommitMsg{Message: msg}
				}
			}
		case shared.KeyEnter:
			// Enter focuses the diff panel for chunk-level operations
			return m, tea.Batch(
				m.emitFileSelected(),
				func() tea.Msg { return shared.FocusPanelMsg{Panel: shared.PanelDiff} },
			)
		}

	case lastCommitMsg:
		if m.amend && m.commitMsg == "" {
			// Take only the first line (subject) for single-line input
			firstLine := strings.SplitN(msg.Message, "\n", 2)[0]
			m.commitMsg = strings.TrimSpace(firstLine)
			m.commitCursor = len(m.commitMsg)
		}
	}

	return m, nil
}

type lastCommitMsg struct {
	Message string
}

func (m Model) handleCommitInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case shared.KeyEscape:
		m.typing = false
	case shared.KeyCtrlEnter, shared.KeyEnter:
		if m.commitMsg != "" {
			commitMsg := m.commitMsg
			amend := m.amend
			repoDir := m.repoDir
			m.commitMsg = ""
			m.commitCursor = 0
			m.amend = false
			m.typing = false
			return m, func() tea.Msg {
				var err error
				if amend {
					err = git.CommitAmend(repoDir, commitMsg)
				} else {
					err = git.Commit(repoDir, commitMsg)
				}
				return shared.CommitResultMsg{Err: err}
			}
		}
	case "backspace":
		if m.commitCursor > 0 {
			m.commitMsg = m.commitMsg[:m.commitCursor-1] + m.commitMsg[m.commitCursor:]
			m.commitCursor--
		}
	case "delete":
		if m.commitCursor < len(m.commitMsg) {
			m.commitMsg = m.commitMsg[:m.commitCursor] + m.commitMsg[m.commitCursor+1:]
		}
	case "left":
		if m.commitCursor > 0 {
			m.commitCursor--
		}
	case "right":
		if m.commitCursor < len(m.commitMsg) {
			m.commitCursor++
		}
	case "alt+left", "alt+b":
		m.commitCursor = wordLeft(m.commitMsg, m.commitCursor)
	case "alt+right", "alt+f":
		m.commitCursor = wordRight(m.commitMsg, m.commitCursor)
	case "home", "ctrl+a":
		m.commitCursor = 0
	case "end", "ctrl+e":
		m.commitCursor = len(m.commitMsg)
	case "ctrl+k":
		m.commitMsg = m.commitMsg[:m.commitCursor]
	case "ctrl+u":
		m.commitMsg = m.commitMsg[m.commitCursor:]
		m.commitCursor = 0
	case "ctrl+w":
		// Delete word backwards
		newPos := wordLeft(m.commitMsg, m.commitCursor)
		m.commitMsg = m.commitMsg[:newPos] + m.commitMsg[m.commitCursor:]
		m.commitCursor = newPos
	default:
		if msg.Type == tea.KeySpace || msg.Type == tea.KeyRunes {
			var ch string
			if msg.Type == tea.KeySpace {
				ch = " "
			} else {
				ch = string(msg.Runes)
			}
			m.commitMsg = m.commitMsg[:m.commitCursor] + ch + m.commitMsg[m.commitCursor:]
			m.commitCursor += len(ch)
		}
	}
	return m, nil
}

func wordLeft(s string, pos int) int {
	if pos <= 0 {
		return 0
	}
	i := pos - 1
	// Skip spaces
	for i > 0 && s[i] == ' ' {
		i--
	}
	// Skip word chars
	for i > 0 && s[i-1] != ' ' {
		i--
	}
	return i
}

func wordRight(s string, pos int) int {
	n := len(s)
	if pos >= n {
		return n
	}
	i := pos
	// Skip word chars
	for i < n && s[i] != ' ' {
		i++
	}
	// Skip spaces
	for i < n && s[i] == ' ' {
		i++
	}
	return i
}

func (m *Model) selectByPath(path string) {
	for i, f := range m.staged {
		if f.Path == path {
			m.cursor = i
			return
		}
	}
	for i, f := range m.unstaged {
		if f.Path == path {
			m.cursor = len(m.staged) + 1 + i
			return
		}
	}
}

func (m *Model) skipSeparator(dir int) {
	// The separator is at index len(m.staged) whenever unstaged files exist
	if len(m.unstaged) > 0 && m.cursor == len(m.staged) {
		m.cursor += dir
		total := m.totalItems()
		if m.cursor < 0 {
			m.cursor = total - 1
		} else if m.cursor >= total {
			m.cursor = 0
		}
	}
}

func (m *Model) updateSection() {
	if m.cursor < len(m.staged) {
		m.activeSection = sectionStaged
	} else {
		m.activeSection = sectionUnstaged
	}
}

func (m *Model) clampCursor() {
	total := m.totalItems()
	if m.cursor >= total {
		m.cursor = total - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureFileVisible()
}

func (m *Model) ensureFileVisible() {
	// File list area: contentHeight minus 2 fixed lines (commit box + blank)
	contentHeight := m.height - 2 - 1 // innerH - title
	fileListHeight := contentHeight - 2 // minus commit box + blank line
	if fileListHeight < 1 {
		fileListHeight = 1
	}
	// Convert cursor to a row index (accounting for section headers)
	row := m.cursorToRow()
	if row < m.offset {
		m.offset = row
	}
	if row >= m.offset+fileListHeight {
		m.offset = row - fileListHeight + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) cursorToRow() int {
	row := 0
	lineIdx := 0
	isTree := m.fileView == "tree"

	if len(m.staged) > 0 {
		row++ // staged header
		lastDir := ""
		for _, f := range m.staged {
			if isTree {
				dir := filepath.Dir(f.Path)
				if dir != "." && dir != lastDir {
					row++ // directory header
					lastDir = dir
				} else if dir == "." {
					lastDir = ""
				}
			}
			if lineIdx == m.cursor {
				return row
			}
			row++
			lineIdx++
		}
	}

	if len(m.unstaged) > 0 {
		row++ // unstaged header
		lineIdx++ // separator line index
		lastDir := ""
		for _, f := range m.unstaged {
			if isTree {
				dir := filepath.Dir(f.Path)
				if dir != "." && dir != lastDir {
					row++ // directory header
					lastDir = dir
				} else if dir == "." {
					lastDir = ""
				}
			}
			if lineIdx == m.cursor {
				return row
			}
			row++
			lineIdx++
		}
	}

	return row
}

func (m Model) totalItems() int {
	total := len(m.staged)
	if len(m.unstaged) > 0 {
		total += 1 + len(m.unstaged)
	}
	if total == 0 {
		return 0
	}
	return total
}

func (m Model) emitFileSelected() tea.Cmd {
	path, staged, untracked := m.SelectedFile()
	return func() tea.Msg {
		return shared.FileSelectedMsg{Path: path, Staged: staged, Untracked: untracked}
	}
}

func (m Model) stageUnstageFile() tea.Cmd {
	path, staged, _ := m.SelectedFile()
	if path == "" {
		return nil
	}
	repoDir := m.repoDir
	return func() tea.Msg {
		var err error
		if staged {
			err = git.UnstageFile(repoDir, path)
		} else {
			err = git.StageFile(repoDir, path)
		}
		if err != nil {
			return shared.ErrorMsg{Err: err}
		}
		return shared.RefreshMsg{}
	}
}

func (m Model) stageAll() tea.Cmd {
	repoDir := m.repoDir
	return func() tea.Msg {
		if err := git.StageAll(repoDir); err != nil {
			return shared.ErrorMsg{Err: err}
		}
		return shared.RefreshMsg{}
	}
}

func (m Model) unstageAll() tea.Cmd {
	repoDir := m.repoDir
	return func() tea.Msg {
		if err := git.UnstageAll(repoDir); err != nil {
			return shared.ErrorMsg{Err: err}
		}
		return shared.RefreshMsg{}
	}
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

	titleText := "Working Copy"
	if m.currentBranch != "" {
		titleText = m.currentBranch
	}
	changeCount := len(m.staged) + len(m.unstaged)
	if changeCount > 0 {
		titleText += fmt.Sprintf(" (%d)", changeCount)
	}
	title := titleStyle.MaxWidth(innerW).Render(titleText)

	contentHeight := innerH - 1 // subtract title line
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Commit input area (fixed at top, exactly 1 rendered line + blank = 2 lines)
	commitStyle := shared.CommitInputStyle
	if m.typing {
		commitStyle = shared.CommitInputActiveStyle
	}
	commitLabel := "Commit"
	if m.amend {
		commitLabel = "Amend"
	}
	prefix := commitLabel + ": "
	commitDisplay := m.commitMsg
	if commitDisplay == "" && !m.typing {
		commitDisplay = "Press 'c' to compose"
	}
	maxW := innerW - 2 - len(prefix) // padding(0,1) = 2 chars
	if maxW < 1 {
		maxW = 1
	}
	if m.typing {
		pos := m.commitCursor
		if pos > len(commitDisplay) {
			pos = len(commitDisplay)
		}
		// Scroll to keep cursor visible
		viewStart := 0
		if pos > maxW-1 {
			viewStart = pos - maxW + 1
		}
		if viewStart < 0 {
			viewStart = 0
		}
		viewEnd := viewStart + maxW
		if viewEnd > len(commitDisplay) {
			viewEnd = len(commitDisplay)
		}
		visible := commitDisplay[viewStart:viewEnd]
		cursorInView := pos - viewStart
		if cursorInView > len(visible) {
			cursorInView = len(visible)
		}
		commitDisplay = visible[:cursorInView] + "▌" + visible[cursorInView:]
	} else if len(commitDisplay) > maxW {
		commitDisplay = commitDisplay[:maxW]
	}
	commitBox := commitStyle.MaxWidth(innerW).Render(prefix + commitDisplay)

	var fixedLines []string
	fixedLines = append(fixedLines, commitBox)
	fixedLines = append(fixedLines, "")

	// Build file rows (tree or list view)
	var rows []fileRow
	lineIdx := 0

	if len(m.staged) > 0 {
		stagedHeader := fmt.Sprintf("  Staged (%d)", len(m.staged))
		rows = append(rows, fileRow{text: shared.StatusStagedStyle.Render(stagedHeader), isHeader: true, lineIdx: -1})
		if m.fileView == "tree" {
			rows = append(rows, buildTreeRows(m.staged, shared.StatusStagedStyle, func(f git.FileStatus) string { return shared.StatusStagedStyle.Render(f.Staged) }, innerW, &lineIdx)...)
		} else {
			rows = append(rows, buildListRows(m.staged, func(f git.FileStatus) string { return shared.StatusStagedStyle.Render(f.Staged) }, innerW, &lineIdx)...)
		}
	}

	if len(m.unstaged) > 0 {
		unstagedHeader := fmt.Sprintf("  Unstaged (%d)", len(m.unstaged))
		rows = append(rows, fileRow{text: shared.StatusUnstagedStyle.Render(unstagedHeader), isHeader: true, lineIdx: lineIdx})
		lineIdx++ // separator counts as a line index
		statusFn := func(f git.FileStatus) string {
			if f.Unstaged == "?" {
				return shared.StatusUntrackedStyle.Render("?")
			}
			return shared.StatusUnstagedStyle.Render(f.Unstaged)
		}
		if m.fileView == "tree" {
			rows = append(rows, buildTreeRows(m.unstaged, shared.StatusUnstagedStyle, statusFn, innerW, &lineIdx)...)
		} else {
			rows = append(rows, buildListRows(m.unstaged, statusFn, innerW, &lineIdx)...)
		}
	}

	// File list area height
	fileListHeight := contentHeight - len(fixedLines)
	if fileListHeight < 0 {
		fileListHeight = 0
	}

	var fileLines []string
	if len(rows) == 0 {
		fileLines = append(fileLines, shared.HelpDescStyle.Render("  No changes"))
	} else {
		end := m.offset + fileListHeight
		if end > len(rows) {
			end = len(rows)
		}
		for i := m.offset; i < end; i++ {
			r := rows[i]
			line := lipgloss.NewStyle().MaxWidth(innerW).Render(r.text)
			if !r.isHeader && r.lineIdx == m.cursor && m.focused && !m.typing {
				line = shared.CursorStyle.Width(innerW).Render(line)
			} else {
				line = lipgloss.NewStyle().Width(innerW).Render(line)
			}
			fileLines = append(fileLines, line)
		}
	}

	// Combine fixed + file list, pad to fill
	var lines []string
	lines = append(lines, fixedLines...)
	lines = append(lines, fileLines...)

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := strings.Join(lines, "\n")
	return style.Width(innerW).Height(innerH).Render(title + "\n" + content)
}

type fileRow struct {
	text     string
	isHeader bool
	lineIdx  int // -1 for headers/dir groups
}

// buildListRows produces flat file rows (no grouping).
func buildListRows(files []git.FileStatus, statusFn func(git.FileStatus) string, innerW int, lineIdx *int) []fileRow {
	var rows []fileRow
	for _, f := range files {
		status := statusFn(f)
		path := f.Path
		maxW := innerW - 6
		if maxW < 4 {
			maxW = 4
		}
		if len(path) > maxW {
			path = "…" + path[len(path)-maxW+1:]
		}
		label := status + " " + path
		rows = append(rows, fileRow{text: "  " + label, lineIdx: *lineIdx})
		*lineIdx++
	}
	return rows
}

// buildTreeRows groups files by directory and produces indented tree rows.
func buildTreeRows(files []git.FileStatus, defaultStyle lipgloss.Style, statusFn func(git.FileStatus) string, innerW int, lineIdx *int) []fileRow {
	var rows []fileRow
	lastDir := ""

	for _, f := range files {
		dir := filepath.Dir(f.Path)
		name := filepath.Base(f.Path)

		if dir != "." && dir != lastDir {
			// New directory group
			dirLabel := shared.HelpDescStyle.Render("  " + dir + "/")
			rows = append(rows, fileRow{text: dirLabel, isHeader: true, lineIdx: -1})
			lastDir = dir
		} else if dir == "." {
			lastDir = ""
		}

		indent := "  "
		if dir != "." {
			indent = "    "
		} else {
			name = f.Path
		}

		status := statusFn(f)
		label := status + " " + name
		rows = append(rows, fileRow{text: indent + label, lineIdx: *lineIdx})
		*lineIdx++
	}

	return rows
}
