package git

import (
	"os"
	"path/filepath"
	"strings"
)

// AddToGitignore appends a pattern to the repo's .gitignore file, creating it if needed.
func AddToGitignore(repoDir, pattern string) error {
	gitignorePath := filepath.Join(repoDir, ".gitignore")

	existing, _ := os.ReadFile(gitignorePath)
	content := string(existing)

	// Check if pattern already exists
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil // already ignored
		}
	}

	// Ensure file ends with newline before appending
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += pattern + "\n"

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}
