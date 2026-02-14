package git

import (
	"strings"
)

// FileStatus represents the status of a single file.
type FileStatus struct {
	Staged   string // Status in index (M, A, D, R, ?)
	Unstaged string // Status in worktree
	Path     string
	OrigPath string // For renames
}

// Status returns the current working tree status.
func Status(repoDir string) ([]FileStatus, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"status", "--porcelain=v2", "--untracked-files=all"},
	})
	if err != nil {
		return nil, err
	}

	return parseStatus(result.Stdout), nil
}

func parseStatus(output string) []FileStatus {
	var files []FileStatus
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "1 "):
			// Ordinary changed entry: 1 XY sub mH mI mW hH hI path
			fields := strings.SplitN(line, " ", 9)
			if len(fields) < 9 {
				continue
			}
			xy := fields[1]
			path := fields[8]
			files = append(files, FileStatus{
				Staged:   translateStatus(xy[0]),
				Unstaged: translateStatus(xy[1]),
				Path:     path,
			})

		case strings.HasPrefix(line, "2 "):
			// Rename/copy entry: 2 XY sub mH mI mW hH hI Xscore path\torigPath
			fields := strings.SplitN(line, " ", 10)
			if len(fields) < 10 {
				continue
			}
			xy := fields[1]
			paths := strings.SplitN(fields[9], "\t", 2)
			path := paths[0]
			origPath := ""
			if len(paths) > 1 {
				origPath = paths[1]
			}
			files = append(files, FileStatus{
				Staged:   translateStatus(xy[0]),
				Unstaged: translateStatus(xy[1]),
				Path:     path,
				OrigPath: origPath,
			})

		case strings.HasPrefix(line, "? "):
			// Untracked: ? path
			path := line[2:]
			files = append(files, FileStatus{
				Staged:   "",
				Unstaged: "?",
				Path:     path,
			})

		case strings.HasPrefix(line, "u "):
			// Unmerged entry
			fields := strings.SplitN(line, " ", 11)
			if len(fields) < 11 {
				continue
			}
			xy := fields[1]
			path := fields[10]
			files = append(files, FileStatus{
				Staged:   translateStatus(xy[0]),
				Unstaged: translateStatus(xy[1]),
				Path:     path,
			})
		}
	}
	return files
}

func translateStatus(b byte) string {
	switch b {
	case '.':
		return ""
	default:
		return string(b)
	}
}
