package git

import (
	"strings"
)

// StashEntry represents a single stash entry.
type StashEntry struct {
	Index   string // e.g. "stash@{0}"
	Message string
}

// StashList returns all stash entries.
func StashList(repoDir string) ([]StashEntry, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"stash", "list", "--format=%gd%x00%s"},
	})
	if err != nil {
		return nil, err
	}

	var entries []StashEntry
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) < 2 {
			continue
		}
		entries = append(entries, StashEntry{
			Index:   parts[0],
			Message: parts[1],
		})
	}

	return entries, nil
}

// StashPush creates a new stash entry.
func StashPush(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"stash", "push"},
	})
	return err
}

// StashPop applies and removes the top stash entry.
func StashPop(repoDir string, index string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"stash", "pop", index},
	})
	return err
}

// StashDrop removes a stash entry without applying.
func StashDrop(repoDir string, index string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"stash", "drop", index},
	})
	return err
}

// StashApply applies a stash entry without removing it.
func StashApply(repoDir string, index string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"stash", "apply", index},
	})
	return err
}

// IsCheckoutConflict checks if an error is due to uncommitted changes.
func IsCheckoutConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Your local changes") ||
		strings.Contains(msg, "would be overwritten") ||
		strings.Contains(msg, "Please commit your changes or stash them")
}
