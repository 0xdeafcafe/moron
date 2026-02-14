package git

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindRepo walks up from the given directory to find a .git directory.
func FindRepo(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		_, err := os.Stat(gitDir)
		if err == nil {
			// .git can be a directory (normal repo) or a file (submodule/worktree)
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (or any parent up to mount point /)")
		}
		dir = parent
	}
}
