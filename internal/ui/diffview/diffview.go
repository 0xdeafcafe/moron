package diffview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/0xdeafcafe/moron/internal/diff"
	"github.com/0xdeafcafe/moron/internal/git"
	"github.com/0xdeafcafe/moron/internal/shared"
)

// Model is the diff viewer panel model.
type Model struct {
	width         int
	height        int
	focused       bool
	repoDir       string
	filePath      string
	isCached      bool
	isUntracked   bool
	fileDiffs     []diff.FileDiff
	rendered      []diff.RenderedLine
	cursor        int
	offset        int
	lineSelect    bool
	selectStart   int
	selectedLines map[int]bool
	styles        diff.RenderStyles
}

func New() Model {
	return Model{
		styles:        diff.DefaultRenderStyles(),
		selectedLines: make(map[int]bool),
	}
}

func (m *Model) SetSize(w, h int)      { m.width = w; m.height = h }
func (m *Model) SetFocused(f bool)     { m.focused = f; if !f { m.lineSelect = false } }
func (m *Model) SetRepoDir(dir string) { m.repoDir = dir }

// HintKeys returns context-sensitive shortcut hints.
func (m Model) HintKeys() string {
	if m.lineSelect {
		return shared.HelpKeyStyle.Render("space") + " stage  " +
			shared.HelpKeyStyle.Render("u") + " unstage  " +
			shared.HelpKeyStyle.Render("v") + " exit select"
	}
	return shared.HelpKeyStyle.Render("space") + " stage hunk  " +
		shared.HelpKeyStyle.Render("v") + " select  " +
		shared.HelpKeyStyle.Render("d") + " discard"
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case shared.FileSelectedMsg:
		m.filePath = msg.Path
		m.isCached = msg.Staged
		m.isUntracked = msg.Untracked
		m.cursor = 0
		m.offset = 0
		m.lineSelect = false
		m.selectedLines = make(map[int]bool)
		return m, m.loadDiff()

	case shared.DiffUpdatedMsg:
		if msg.Err != nil {
			return m, nil
		}
		m.fileDiffs = msg.FileDiffs
		if len(m.fileDiffs) > 0 {
			m.rendered = diff.RenderFileDiffHighlighted(m.fileDiffs[0], m.styles, m.filePath)
		} else {
			m.rendered = nil
		}
		m.cursor = 0
		m.offset = 0
		return m, nil

	case shared.RefreshMsg:
		if m.filePath != "" {
			return m, m.loadDiff()
		}

	case tea.MouseMsg:
		if !m.focused {
			return m, nil
		}
		scrollAmount := 3
		switch msg.Type {
		case tea.MouseWheelUp:
			m.cursor -= scrollAmount
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case tea.MouseWheelDown:
			m.cursor += scrollAmount
			if m.cursor >= len(m.rendered) {
				m.cursor = len(m.rendered) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		}

	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}

		switch msg.String() {
		case shared.KeyJ, shared.KeyDown:
			if m.cursor < len(m.rendered)-1 {
				m.cursor++
				m.ensureVisible()
				if m.lineSelect {
					m.updateSelection()
				}
			}
		case shared.KeyK, shared.KeyUp:
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
				if m.lineSelect {
					m.updateSelection()
				}
			}
		case "ctrl+d":
			halfPage := (m.height - 2) / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor += halfPage
			if m.cursor >= len(m.rendered) {
				m.cursor = len(m.rendered) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case "ctrl+u":
			halfPage := (m.height - 2) / 2
			if halfPage < 1 {
				halfPage = 1
			}
			m.cursor -= halfPage
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case "g":
			m.cursor = 0
			m.ensureVisible()
		case "G":
			if len(m.rendered) > 0 {
				m.cursor = len(m.rendered) - 1
				m.ensureVisible()
			}
		case shared.KeyNextHunk:
			m.jumpToNextHunk()
		case shared.KeyPrevHunk:
			m.jumpToPrevHunk()
		case shared.KeyLineSelect:
			m.lineSelect = !m.lineSelect
			if m.lineSelect {
				m.selectStart = m.cursor
				m.selectedLines = make(map[int]bool)
				m.updateSelection()
			} else {
				m.selectedLines = make(map[int]bool)
			}
		case shared.KeySpace, shared.KeyStage:
			return m, m.stageAction(false)
		case shared.KeyUnstage:
			return m, m.stageAction(true)
		case shared.KeyDiscard:
			return m, m.discardAction()
		case "t":
			m.isCached = !m.isCached
			m.cursor = 0
			m.offset = 0
			return m, m.loadDiff()
		}
	}

	return m, nil
}

func (m *Model) ensureVisible() {
	reserved := 2 // title + footer
	viewHeight := m.height - 2 - reserved
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

func (m *Model) updateSelection() {
	m.selectedLines = make(map[int]bool)
	start, end := m.selectStart, m.cursor
	if start > end {
		start, end = end, start
	}
	for i := start; i <= end; i++ {
		if i < len(m.rendered) && !m.rendered[i].IsHunkHead && m.rendered[i].LineIdx >= 0 {
			m.selectedLines[i] = true
		}
	}
}

func (m *Model) jumpToNextHunk() {
	for i := m.cursor + 1; i < len(m.rendered); i++ {
		if m.rendered[i].IsHunkHead {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

func (m *Model) jumpToPrevHunk() {
	for i := m.cursor - 1; i >= 0; i-- {
		if m.rendered[i].IsHunkHead {
			m.cursor = i
			m.ensureVisible()
			return
		}
	}
}

func (m Model) loadDiff() tea.Cmd {
	repoDir := m.repoDir
	filePath := m.filePath
	cached := m.isCached
	untracked := m.isUntracked
	return func() tea.Msg {
		var raw string
		var err error
		if untracked {
			raw, err = git.DiffUntracked(repoDir, filePath)
		} else {
			raw, err = git.Diff(repoDir, git.DiffOptions{
				Cached: cached,
				File:   filePath,
			})
		}
		if err != nil {
			return shared.DiffUpdatedMsg{Err: err}
		}
		diffs := diff.Parse(raw)
		return shared.DiffUpdatedMsg{
			FileDiffs: diffs,
			File:      filePath,
			Cached:    cached,
		}
	}
}

func (m Model) stageAction(reverse bool) tea.Cmd {
	if len(m.fileDiffs) == 0 {
		return nil
	}
	fd := m.fileDiffs[0]
	repoDir := m.repoDir

	hunkIdx := -1
	if m.cursor < len(m.rendered) {
		hunkIdx = m.rendered[m.cursor].HunkIdx
	}
	if hunkIdx < 0 || hunkIdx >= len(fd.Hunks) {
		return nil
	}

	var patch string
	if m.lineSelect && len(m.selectedLines) > 0 {
		hunkLines := make(map[int]bool)
		for ri := range m.selectedLines {
			if ri < len(m.rendered) {
				rl := m.rendered[ri]
				if rl.HunkIdx == hunkIdx && rl.LineIdx >= 0 {
					hunkLines[rl.LineIdx] = true
				}
			}
		}
		patch = diff.BuildSelectedLinesPatch(fd, hunkIdx, hunkLines)
	} else {
		patch = diff.BuildHunkPatch(fd, hunkIdx)
	}

	if patch == "" {
		return nil
	}

	return func() tea.Msg {
		var err error
		if reverse {
			err = git.ApplyPatchReverseCached(repoDir, patch)
		} else {
			err = git.ApplyPatchCached(repoDir, patch)
		}
		if err != nil {
			return shared.ErrorMsg{Err: err}
		}
		return shared.RefreshMsg{}
	}
}

func (m Model) discardAction() tea.Cmd {
	if len(m.fileDiffs) == 0 {
		return nil
	}
	fd := m.fileDiffs[0]

	hunkIdx := -1
	if m.cursor < len(m.rendered) {
		hunkIdx = m.rendered[m.cursor].HunkIdx
	}
	if hunkIdx < 0 || hunkIdx >= len(fd.Hunks) {
		return nil
	}

	patch := diff.BuildHunkPatch(fd, hunkIdx)
	if patch == "" {
		return nil
	}

	repoDir := m.repoDir
	return func() tea.Msg {
		return shared.ShowDialogMsg{
			Type:    shared.DialogConfirm,
			Title:   "Discard Hunk",
			Message: "Discard this hunk? This cannot be undone.",
			OnConfirm: func() tea.Msg {
				err := git.ApplyPatchReverse(repoDir, patch)
				if err != nil {
					return shared.ErrorMsg{Err: err}
				}
				return shared.RefreshMsg{}
			},
		}
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

	titleText := "Diff"
	if m.filePath != "" {
		short := m.filePath
		if idx := strings.LastIndex(short, "/"); idx >= 0 {
			short = short[idx+1:]
		}
		titleText = short
		if m.isCached {
			titleText += " [Staged]"
		} else {
			titleText += " [Unstaged]"
		}
		if len(m.rendered) > 0 {
			titleText += fmt.Sprintf(" %d/%d", m.cursor+1, len(m.rendered))
		}
	}
	title := titleStyle.MaxWidth(innerW).Render(titleText)

	// Reserve lines for title and optional footer
	hasFooter := m.focused && len(m.rendered) > 0
	reserved := 1 // title
	if hasFooter {
		reserved++
	}
	contentHeight := innerH - reserved
	if contentHeight < 0 {
		contentHeight = 0
	}

	var lines []string

	if len(m.rendered) == 0 {
		if m.filePath == "" {
			lines = append(lines, shared.HelpDescStyle.Render("  Select a file to view diff"))
		} else {
			lines = append(lines, shared.HelpDescStyle.Render("  No diff for "+m.filePath))
		}
	} else {
		visibleEnd := m.offset + contentHeight
		if visibleEnd > len(m.rendered) {
			visibleEnd = len(m.rendered)
		}

		for i := m.offset; i < visibleEnd; i++ {
			rl := m.rendered[i]
			line := rl.Text

			if m.selectedLines[i] {
				line = m.styles.Selected.Render(line)
			}

			// Truncate first (MaxWidth), then pad (Width) separately
			// Width() wraps text, so we must truncate before padding
			line = lipgloss.NewStyle().MaxWidth(innerW).Render(line)
			if i == m.cursor && m.focused {
				line = shared.CursorStyle.Width(innerW).Render(line)
			} else {
				line = lipgloss.NewStyle().Width(innerW).Render(line)
			}

			lines = append(lines, line)
		}
	}

	for len(lines) < contentHeight {
		lines = append(lines, "")
	}
	if len(lines) > contentHeight {
		lines = lines[:contentHeight]
	}

	content := strings.Join(lines, "\n")

	body := title + "\n" + content
	if hasFooter {
		// Count added/removed lines
		var added, removed int
		for _, fd := range m.fileDiffs {
			for _, h := range fd.Hunks {
				for _, l := range h.Lines {
					switch l.Type {
					case diff.LineAdded:
						added++
					case diff.LineRemoved:
						removed++
					}
				}
			}
		}
		stats := shared.DiffAddedStyle.Render(fmt.Sprintf("+%d", added)) + " " +
			shared.DiffRemovedStyle.Render(fmt.Sprintf("-%d", removed))
		body += "\n" + shared.StatusBarStyle.MaxWidth(innerW).Render(stats)
	}

	return style.Width(innerW).Height(innerH).Render(body)
}
