package diff

import (
	"fmt"
	"strings"
)

// BuildHunkPatch constructs a patch for a single hunk.
func BuildHunkPatch(fd FileDiff, hunkIdx int) string {
	if hunkIdx >= len(fd.Hunks) {
		return ""
	}

	hunk := fd.Hunks[hunkIdx]
	var b strings.Builder

	b.WriteString(fd.Header)
	b.WriteString("\n")
	b.WriteString(hunk.Header)
	b.WriteString("\n")

	for _, line := range hunk.Lines {
		switch line.Type {
		case LineContext:
			b.WriteString(" ")
			b.WriteString(line.Content)
			b.WriteString("\n")
		case LineAdded:
			b.WriteString("+")
			b.WriteString(line.Content)
			b.WriteString("\n")
		case LineRemoved:
			b.WriteString("-")
			b.WriteString(line.Content)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// BuildSelectedLinesPatch constructs a patch from selected lines within a hunk.
// selectedLines is a set of line indices (within the hunk) that are selected.
func BuildSelectedLinesPatch(fd FileDiff, hunkIdx int, selectedLines map[int]bool) string {
	if hunkIdx >= len(fd.Hunks) {
		return ""
	}

	hunk := fd.Hunks[hunkIdx]
	var patchLines []string

	oldCount := 0
	newCount := 0

	for i, line := range hunk.Lines {
		selected := selectedLines[i]

		switch line.Type {
		case LineContext:
			patchLines = append(patchLines, " "+line.Content)
			oldCount++
			newCount++
		case LineRemoved:
			if selected {
				patchLines = append(patchLines, "-"+line.Content)
				oldCount++
			} else {
				// Unselected deletion becomes context
				patchLines = append(patchLines, " "+line.Content)
				oldCount++
				newCount++
			}
		case LineAdded:
			if selected {
				patchLines = append(patchLines, "+"+line.Content)
				newCount++
			}
			// Unselected additions are omitted
		}
	}

	var b strings.Builder
	b.WriteString(fd.Header)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", hunk.OldStart, oldCount, hunk.OldStart, newCount))
	for _, pl := range patchLines {
		b.WriteString(pl)
		b.WriteString("\n")
	}

	return b.String()
}
