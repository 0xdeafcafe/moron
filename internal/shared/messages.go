package shared

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/0xdeafcafe/moron/internal/diff"
	"github.com/0xdeafcafe/moron/internal/git"
)


// Panel identifies which panel is focused.
type Panel int

const (
	PanelBranches    Panel = 0
	PanelWorkingCopy Panel = 1
	PanelDiff        Panel = 2
)

// RefreshMsg triggers a full data refresh.
type RefreshMsg struct{}

// BackgroundFetchDoneMsg signals a background fetch completed.
type BackgroundFetchDoneMsg struct {
	Err error
}

// TickFetchMsg triggers a periodic background fetch.
type TickFetchMsg struct{}

// GitChangedMsg signals that the .git directory changed on disk.
type GitChangedMsg struct{}

// StatusUpdatedMsg carries refreshed status data.
type StatusUpdatedMsg struct {
	Files []git.FileStatus
	Err   error
}

// BranchesUpdatedMsg carries refreshed branch data.
type BranchesUpdatedMsg struct {
	Branches       []git.Branch
	RemoteBranches []git.Branch
	Tags           []git.Tag
	Remotes        []git.Remote
	Worktrees      []git.Worktree
	CurrentBranch  string
	Err            error
}

// DiffUpdatedMsg carries refreshed diff data.
type DiffUpdatedMsg struct {
	FileDiffs []diff.FileDiff
	File      string
	Cached    bool
	Err       error
}

// FileSelectedMsg is sent when a file is selected in the working copy.
type FileSelectedMsg struct {
	Path      string
	Staged    bool
	Untracked bool
}

// ErrorMsg represents an error to display.
type ErrorMsg struct {
	Err error
}

// CommitResultMsg carries the result of a commit operation.
type CommitResultMsg struct {
	Err error
}

// PushResultMsg carries the result of a push operation.
type PushResultMsg struct {
	Err error
}

// FetchResultMsg carries the result of a fetch operation.
type FetchResultMsg struct {
	Err error
}

// CheckoutResultMsg carries the result of a checkout operation.
type CheckoutResultMsg struct {
	Branch string
	Err    error
}

// RebaseResultMsg carries the result of a rebase operation.
type RebaseResultMsg struct {
	Err error
}

// PullResultMsg carries the result of a pull operation.
type PullResultMsg struct {
	Err error
}

// CreateBranchResultMsg carries the result of a branch creation.
type CreateBranchResultMsg struct {
	Branch string
	Err    error
}

// DeleteBranchResultMsg carries the result of a branch deletion.
type DeleteBranchResultMsg struct {
	Branch string
	Err    error
}

// AddRemoteResultMsg carries the result of adding a remote.
type AddRemoteResultMsg struct {
	Name string
	Err  error
}

// BranchLogMsg carries commit history for a selected branch.
type BranchLogMsg struct {
	Branch string
	Log    []git.LogEntry
	Err    error
}

// CreateTagResultMsg carries the result of creating a tag.
type CreateTagResultMsg struct {
	Tag string
	Err error
}

// DeleteTagResultMsg carries the result of deleting a tag.
type DeleteTagResultMsg struct {
	Tag string
	Err error
}

// StashResultMsg carries the result of a stash operation.
type StashResultMsg struct {
	Action string // "push", "pop", "apply", "drop"
	Err    error
}

// StashesUpdatedMsg carries refreshed stash data.
type StashesUpdatedMsg struct {
	Stashes []git.StashEntry
	Err     error
}

// FocusPanelMsg requests focus change to a specific panel.
type FocusPanelMsg struct {
	Panel Panel
}

// ShowDialogMsg displays a dialog overlay.
type ShowDialogMsg struct {
	Type      DialogType
	Title     string
	Message   string
	OnYes     func()     // deprecated: use OnConfirm
	OnConfirm func() tea.Msg // called on confirm, returns a result message
}

// CloseDialogMsg closes the current dialog.
type CloseDialogMsg struct{}

// DialogType identifies the kind of dialog.
type DialogType int

const (
	DialogConfirm DialogType = iota
	DialogError
	DialogPush
	DialogInput
)
