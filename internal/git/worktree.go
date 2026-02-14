package git

import "strings"

// Worktree represents a git worktree.
type Worktree struct {
	Path   string
	Branch string
	IsBare bool
}

// ListWorktrees returns all worktrees for the repository.
func ListWorktrees(repoDir string) ([]Worktree, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"worktree", "list", "--porcelain"},
	})
	if err != nil {
		return nil, err
	}

	var worktrees []Worktree
	var current Worktree
	for _, line := range strings.Split(result.Stdout, "\n") {
		if line == "" {
			if current.Path != "" {
				worktrees = append(worktrees, current)
				current = Worktree{}
			}
			continue
		}
		if strings.HasPrefix(line, "worktree ") {
			current.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		} else if line == "bare" {
			current.IsBare = true
		}
	}
	if current.Path != "" {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}
