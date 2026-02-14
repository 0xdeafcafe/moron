package diff

import (
	"strings"
	"testing"
)

func makeTestFileDiff() FileDiff {
	return FileDiff{
		OldName: "main.go",
		NewName: "main.go",
		Header: `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go`,
		Hunks: []Hunk{
			{
				OldStart: 10,
				OldCount: 5,
				NewStart: 10,
				NewCount: 6,
				Header:   "@@ -10,5 +10,6 @@ func main() {",
				Lines: []Line{
					{Type: LineContext, Content: "    fmt.Println(\"hello\")", OldNum: 10, NewNum: 10},
					{Type: LineRemoved, Content: "    fmt.Println(\"old\")", OldNum: 11},
					{Type: LineAdded, Content: "    fmt.Println(\"new\")", NewNum: 11},
					{Type: LineAdded, Content: "    fmt.Println(\"extra\")", NewNum: 12},
					{Type: LineContext, Content: "    fmt.Println(\"world\")", OldNum: 12, NewNum: 13},
				},
			},
			{
				OldStart: 20,
				OldCount: 3,
				NewStart: 21,
				NewCount: 3,
				Header:   "@@ -20,3 +21,3 @@",
				Lines: []Line{
					{Type: LineContext, Content: "context", OldNum: 20, NewNum: 21},
					{Type: LineRemoved, Content: "old2", OldNum: 21},
					{Type: LineAdded, Content: "new2", NewNum: 22},
				},
			},
		},
	}
}

func TestBuildHunkPatch(t *testing.T) {
	fd := makeTestFileDiff()
	patch := BuildHunkPatch(fd, 0)

	if patch == "" {
		t.Fatal("expected non-empty patch")
	}

	// Should contain the file header
	if !strings.Contains(patch, "diff --git a/main.go b/main.go") {
		t.Error("patch should contain file header")
	}

	// Should contain the hunk header
	if !strings.Contains(patch, "@@ -10,5 +10,6 @@") {
		t.Error("patch should contain hunk header")
	}

	// Should contain diff lines
	if !strings.Contains(patch, "-    fmt.Println(\"old\")") {
		t.Error("patch should contain removed line")
	}
	if !strings.Contains(patch, "+    fmt.Println(\"new\")") {
		t.Error("patch should contain added line")
	}
	if !strings.Contains(patch, "+    fmt.Println(\"extra\")") {
		t.Error("patch should contain extra added line")
	}

	// Should NOT contain second hunk
	if strings.Contains(patch, "old2") {
		t.Error("patch should not contain second hunk")
	}
}

func TestBuildHunkPatchSecondHunk(t *testing.T) {
	fd := makeTestFileDiff()
	patch := BuildHunkPatch(fd, 1)

	if !strings.Contains(patch, "old2") {
		t.Error("patch should contain second hunk content")
	}
	if strings.Contains(patch, "fmt.Println(\"old\")") {
		t.Error("patch should not contain first hunk content")
	}
}

func TestBuildHunkPatchInvalidIndex(t *testing.T) {
	fd := makeTestFileDiff()
	patch := BuildHunkPatch(fd, 5)
	if patch != "" {
		t.Error("expected empty patch for invalid hunk index")
	}
}

func TestBuildSelectedLinesPatchAllSelected(t *testing.T) {
	fd := makeTestFileDiff()
	// Select all non-context lines in hunk 0
	selected := map[int]bool{
		1: true, // removed line
		2: true, // added line
		3: true, // added line
	}

	patch := BuildSelectedLinesPatch(fd, 0, selected)
	if patch == "" {
		t.Fatal("expected non-empty patch")
	}

	lines := strings.Split(patch, "\n")
	var removedCount, addedCount, contextCount int
	for _, l := range lines {
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			removedCount++
		} else if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			addedCount++
		} else if strings.HasPrefix(l, " ") {
			contextCount++
		}
	}

	if removedCount != 1 {
		t.Errorf("expected 1 removed line, got %d", removedCount)
	}
	if addedCount != 2 {
		t.Errorf("expected 2 added lines, got %d", addedCount)
	}
	if contextCount != 2 {
		t.Errorf("expected 2 context lines, got %d", contextCount)
	}
}

func TestBuildSelectedLinesPatchPartialSelection(t *testing.T) {
	fd := makeTestFileDiff()
	// Only select the first added line, not the removal or second addition
	selected := map[int]bool{
		2: true, // only the first added line
	}

	patch := BuildSelectedLinesPatch(fd, 0, selected)
	if patch == "" {
		t.Fatal("expected non-empty patch")
	}

	lines := strings.Split(patch, "\n")

	// The removed line (unselected) should become context
	// The first added line (selected) should remain as +
	// The second added line (unselected) should be omitted
	var removedCount, addedCount, contextCount int
	for _, l := range lines {
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			removedCount++
		} else if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			addedCount++
		} else if strings.HasPrefix(l, " ") {
			contextCount++
		}
	}

	// Unselected removal becomes context (1 original context + 1 converted = 3 context + omitted),
	// wait let's be precise:
	// line 0: context (original) -> " "
	// line 1: removed (unselected) -> " " (becomes context)
	// line 2: added (selected) -> "+"
	// line 3: added (unselected) -> omitted
	// line 4: context (original) -> " "
	if removedCount != 0 {
		t.Errorf("expected 0 removed lines (unselected become context), got %d", removedCount)
	}
	if addedCount != 1 {
		t.Errorf("expected 1 added line, got %d", addedCount)
	}
	if contextCount != 3 {
		t.Errorf("expected 3 context lines, got %d", contextCount)
	}
}

func TestBuildSelectedLinesPatchOnlyRemoval(t *testing.T) {
	fd := makeTestFileDiff()
	// Only select the removal line
	selected := map[int]bool{
		1: true, // only the removed line
	}

	patch := BuildSelectedLinesPatch(fd, 0, selected)
	if patch == "" {
		t.Fatal("expected non-empty patch")
	}

	lines := strings.Split(patch, "\n")
	var removedCount, addedCount int
	for _, l := range lines {
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			removedCount++
		} else if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			addedCount++
		}
	}

	// Removal selected -> stays as "-"
	// Additions unselected -> omitted
	if removedCount != 1 {
		t.Errorf("expected 1 removed line, got %d", removedCount)
	}
	if addedCount != 0 {
		t.Errorf("expected 0 added lines, got %d", addedCount)
	}
}

func TestBuildSelectedLinesPatchEmptySelection(t *testing.T) {
	fd := makeTestFileDiff()
	patch := BuildSelectedLinesPatch(fd, 0, map[int]bool{})

	// With no lines selected, removals become context and additions are omitted
	// This effectively creates a no-op patch (all context)
	if patch == "" {
		t.Fatal("expected non-empty patch (even if no-op)")
	}

	lines := strings.Split(patch, "\n")
	var removedCount, addedCount int
	for _, l := range lines {
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			removedCount++
		} else if strings.HasPrefix(l, "+") && !strings.HasPrefix(l, "+++") {
			addedCount++
		}
	}

	if removedCount != 0 {
		t.Errorf("expected 0 removed lines, got %d", removedCount)
	}
	if addedCount != 0 {
		t.Errorf("expected 0 added lines, got %d", addedCount)
	}
}

func TestBuildSelectedLinesPatchHunkHeaderCounts(t *testing.T) {
	fd := makeTestFileDiff()
	// Select only the removal (line index 1)
	selected := map[int]bool{
		1: true,
	}

	patch := BuildSelectedLinesPatch(fd, 0, selected)
	lines := strings.Split(patch, "\n")

	// Find the hunk header
	var hunkHeader string
	for _, l := range lines {
		if strings.HasPrefix(l, "@@") {
			hunkHeader = l
			break
		}
	}

	if hunkHeader == "" {
		t.Fatal("no hunk header found")
	}

	// With only the removal selected and additions omitted:
	// context(hello) + removal(old) + context(world) = old: 3, new: 2
	oldStart, oldCount, newStart, newCount := parseHunkHeader(hunkHeader)
	if oldStart != 10 {
		t.Errorf("oldStart = %d, want 10", oldStart)
	}
	// old count = 2 context + 1 removal = 3
	if oldCount != 3 {
		t.Errorf("oldCount = %d, want 3", oldCount)
	}
	if newStart != 10 {
		t.Errorf("newStart = %d, want 10", newStart)
	}
	// new count = 2 context + 0 additions = 2
	if newCount != 2 {
		t.Errorf("newCount = %d, want 2", newCount)
	}
}

func TestBuildSelectedLinesPatchInvalidHunk(t *testing.T) {
	fd := makeTestFileDiff()
	patch := BuildSelectedLinesPatch(fd, 99, map[int]bool{0: true})
	if patch != "" {
		t.Error("expected empty patch for invalid hunk index")
	}
}
