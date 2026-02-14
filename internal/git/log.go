package git

import (
	"fmt"
	"strings"
	"time"
)

// LogEntry represents a single commit in the log.
type LogEntry struct {
	Hash       string
	ShortHash  string
	Subject    string
	Author     string
	AuthorDate time.Time
}

// Log returns the commit log for the current branch.
func Log(repoDir string, limit int) ([]LogEntry, error) {
	result, err := Run(RunOpts{
		Dir: repoDir,
		Args: []string{
			"log",
			"--format=%H\x00%h\x00%s\x00%an\x00%aI",
			"-n", fmt.Sprintf("%d", limit),
		},
	})
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 5)
		if len(parts) < 5 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, parts[4])
		entries = append(entries, LogEntry{
			Hash:       parts[0],
			ShortHash:  parts[1],
			Subject:    parts[2],
			Author:     parts[3],
			AuthorDate: date,
		})
	}

	return entries, nil
}

// CurrentBranch returns the name of the current branch.
// Works even on repos with no commits by falling back to symbolic-ref.
func CurrentBranch(repoDir string) (string, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"symbolic-ref", "--short", "HEAD"},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}
