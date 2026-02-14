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
		info, err := os.Stat(gitDir)
		if err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (or any parent up to mount point /)")
		}
		dir = parent
	}
}
