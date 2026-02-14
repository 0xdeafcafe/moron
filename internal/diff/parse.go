package diff

import (
	"fmt"
	"strconv"
	"strings"
)

// LineType represents the type of a diff line.
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// Line represents a single line in a diff hunk.
type Line struct {
	Type    LineType
	Content string // line content without +/- prefix
	OldNum  int    // line number in old file (0 for added lines)
	NewNum  int    // line number in new file (0 for removed lines)
}

// Hunk represents a single diff hunk.
type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Header   string // the @@ line
	Lines    []Line
}

// FileDiff represents the diff for a single file.
type FileDiff struct {
	OldName  string
	NewName  string
	Header   string // everything before the first hunk
	Hunks    []Hunk
	IsBinary bool
}

// Parse parses unified diff output into structured FileDiff objects.
func Parse(raw string) []FileDiff {
	var diffs []FileDiff
	lines := strings.Split(raw, "\n")

	i := 0
	for i < len(lines) {
		if !strings.HasPrefix(lines[i], "diff --git ") {
			i++
			continue
		}

		fd := FileDiff{}
		headerStart := i

		// Parse diff --git a/file b/file
		diffLine := lines[i]
		parts := strings.SplitN(diffLine, " b/", 2)
		if len(parts) == 2 {
			fd.NewName = parts[1]
			aParts := strings.SplitN(parts[0], " a/", 2)
			if len(aParts) == 2 {
				fd.OldName = aParts[1]
			}
		}

		i++

		// Skip file metadata lines until we hit @@ or next diff or binary
		for i < len(lines) && !strings.HasPrefix(lines[i], "@@") &&
			!strings.HasPrefix(lines[i], "diff --git ") &&
			!strings.HasPrefix(lines[i], "Binary") {
			if strings.HasPrefix(lines[i], "--- a/") {
				fd.OldName = lines[i][6:]
			} else if strings.HasPrefix(lines[i], "--- /dev/null") {
				fd.OldName = "/dev/null"
			} else if strings.HasPrefix(lines[i], "+++ b/") {
				fd.NewName = lines[i][6:]
			} else if strings.HasPrefix(lines[i], "+++ /dev/null") {
				fd.NewName = "/dev/null"
			}
			i++
		}

		fd.Header = strings.Join(lines[headerStart:i], "\n")

		if i < len(lines) && strings.HasPrefix(lines[i], "Binary") {
			fd.IsBinary = true
			i++
			diffs = append(diffs, fd)
			continue
		}

		// Parse hunks
		for i < len(lines) && strings.HasPrefix(lines[i], "@@") {
			hunk, nextI := parseHunk(lines, i)
			fd.Hunks = append(fd.Hunks, hunk)
			i = nextI
		}

		diffs = append(diffs, fd)
	}

	return diffs
}

func parseHunk(lines []string, start int) (Hunk, int) {
	h := Hunk{Header: lines[start]}
	h.OldStart, h.OldCount, h.NewStart, h.NewCount = parseHunkHeader(lines[start])

	i := start + 1
	oldNum := h.OldStart
	newNum := h.NewStart

	for i < len(lines) {
		line := lines[i]

		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff --git ") {
			break
		}

		// Skip empty lines (e.g., trailing newline from split)
		if line == "" {
			i++
			continue
		}

		if strings.HasPrefix(line, "+") {
			h.Lines = append(h.Lines, Line{
				Type:    LineAdded,
				Content: line[1:],
				NewNum:  newNum,
			})
			newNum++
		} else if strings.HasPrefix(line, "-") {
			h.Lines = append(h.Lines, Line{
				Type:    LineRemoved,
				Content: line[1:],
				OldNum:  oldNum,
			})
			oldNum++
		} else if strings.HasPrefix(line, `\`) {
			// "\ No newline at end of file"
			i++
			continue
		} else {
			content := ""
			if len(line) > 0 {
				content = line[1:]
			}
			h.Lines = append(h.Lines, Line{
				Type:    LineContext,
				Content: content,
				OldNum:  oldNum,
				NewNum:  newNum,
			})
			oldNum++
			newNum++
		}
		i++
	}

	return h, i
}

func parseHunkHeader(header string) (oldStart, oldCount, newStart, newCount int) {
	header = strings.TrimPrefix(header, "@@ ")
	end := strings.Index(header, " @@")
	if end >= 0 {
		header = header[:end]
	}

	parts := strings.Split(header, " ")
	if len(parts) >= 2 {
		oldStart, oldCount = parseRange(parts[0][1:]) // skip -
		newStart, newCount = parseRange(parts[1][1:]) // skip +
	}
	return
}

func parseRange(s string) (start, count int) {
	parts := strings.Split(s, ",")
	start, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		count, _ = strconv.Atoi(parts[1])
	} else {
		count = 1
	}
	return
}

// FormatHunkHeader creates a @@ header line.
func FormatHunkHeader(oldStart, oldCount, newStart, newCount int) string {
	return fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount)
}
