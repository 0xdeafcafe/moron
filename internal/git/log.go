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
	Graph      string   // graph prefix chars (e.g. "* ", "| * ")
	GraphTail  []string // continuation graph lines after this commit
}

// Log returns the commit log for the given ref (branch name, HEAD, etc.).
// Includes graph data for visualizing branch topology.
func Log(repoDir string, limit int, ref ...string) ([]LogEntry, error) {
	args := []string{
		"log",
		"--graph",
		"--format=%x00%H%x00%h%x00%s%x00%an%x00%aI",
		"-n", fmt.Sprintf("%d", limit),
	}
	if len(ref) > 0 && ref[0] != "" {
		args = append(args, ref[0])
	}
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: args,
	})
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}

		// Commit lines contain \x00: <graph>\x00<hash>\x00<short>\x00<subject>\x00<author>\x00<date>
		if idx := strings.Index(line, "\x00"); idx >= 0 {
			graphPrefix := line[:idx]
			rest := line[idx+1:]
			parts := strings.SplitN(rest, "\x00", 5)
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
				Graph:      graphPrefix,
			})
		} else {
			// Graph-only continuation line — attach to the last entry
			if len(entries) > 0 {
				entries[len(entries)-1].GraphTail = append(entries[len(entries)-1].GraphTail, line)
			}
		}
	}

	return entries, nil
}

// CurrentBranch returns the name of the current branch.
// If HEAD is detached, returns the short hash prefixed with "detached@".
func CurrentBranch(repoDir string) (string, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"symbolic-ref", "--short", "HEAD"},
	})
	if err == nil {
		return strings.TrimSpace(result.Stdout), nil
	}
	// Detached HEAD — get the short hash
	result, err = Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"rev-parse", "--short", "HEAD"},
	})
	if err != nil {
		return "", err
	}
	return "detached@" + strings.TrimSpace(result.Stdout), nil
}

// IsDetachedHEAD returns true if the given branch string represents a detached HEAD.
func IsDetachedHEAD(branch string) bool {
	return strings.HasPrefix(branch, "detached@")
}
