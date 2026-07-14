package diff

import (
	"testing"
)

func TestRenderFileDiffBinary(t *testing.T) {
	fd := FileDiff{
		OldName:  "image.png",
		NewName:  "image.png",
		IsBinary: true,
	}

	styles := DefaultRenderStyles()
	rendered := RenderFileDiff(fd, styles)

	if len(rendered) != 1 {
		t.Fatalf("expected 1 rendered line for binary, got %d", len(rendered))
	}
	if rendered[0].LineIdx != -1 {
		t.Error("binary file line should have LineIdx -1")
	}
}

func TestRenderFileDiffHunks(t *testing.T) {
	fd := FileDiff{
		OldName: "test.go",
		NewName: "test.go",
		Hunks: []Hunk{
			{
				OldStart: 1,
				OldCount: 3,
				NewStart: 1,
				NewCount: 3,
				Header:   "@@ -1,3 +1,3 @@",
				Lines: []Line{
					{Type: LineContext, Content: "ctx", OldNum: 1, NewNum: 1},
					{Type: LineRemoved, Content: "old", OldNum: 2},
					{Type: LineAdded, Content: "new", NewNum: 2},
				},
			},
		},
	}

	styles := DefaultRenderStyles()
	rendered := RenderFileDiff(fd, styles)

	// 1 hunk header + 3 lines = 4
	if len(rendered) != 4 {
		t.Fatalf("expected 4 rendered lines, got %d", len(rendered))
	}

	// First line should be hunk header
	if !rendered[0].IsHunkHead {
		t.Error("first rendered line should be hunk header")
	}
	if rendered[0].HunkIdx != 0 {
		t.Errorf("hunk header HunkIdx = %d, want 0", rendered[0].HunkIdx)
	}
	if rendered[0].LineIdx != -1 {
		t.Errorf("hunk header LineIdx = %d, want -1", rendered[0].LineIdx)
	}

	// Content lines should have correct indices
	for i := 1; i <= 3; i++ {
		if rendered[i].HunkIdx != 0 {
			t.Errorf("line %d HunkIdx = %d, want 0", i, rendered[i].HunkIdx)
		}
		if rendered[i].LineIdx != i-1 {
			t.Errorf("line %d LineIdx = %d, want %d", i, rendered[i].LineIdx, i-1)
		}
	}
}

func TestRenderFileDiffMultipleHunks(t *testing.T) {
	fd := FileDiff{
		OldName: "test.go",
		NewName: "test.go",
		Hunks: []Hunk{
			{
				Header: "@@ -1,2 +1,2 @@",
				Lines: []Line{
					{Type: LineRemoved, Content: "a", OldNum: 1},
					{Type: LineAdded, Content: "b", NewNum: 1},
				},
			},
			{
				Header: "@@ -10,2 +10,2 @@",
				Lines: []Line{
					{Type: LineRemoved, Content: "c", OldNum: 10},
					{Type: LineAdded, Content: "d", NewNum: 10},
				},
			},
		},
	}

	styles := DefaultRenderStyles()
	rendered := RenderFileDiff(fd, styles)

	// 2 hunk headers + 4 lines + 1 separator between hunks = 7
	if len(rendered) != 7 {
		t.Fatalf("expected 7 rendered lines, got %d", len(rendered))
	}

	// Separator sits between the hunks and is not a header
	if rendered[3].IsHunkHead || rendered[3].LineIdx != -1 {
		t.Error("line 3 should be the hunk separator")
	}

	// Check second hunk header
	if !rendered[4].IsHunkHead {
		t.Error("line 4 should be second hunk header")
	}
	if rendered[4].HunkIdx != 1 {
		t.Errorf("second hunk header HunkIdx = %d, want 1", rendered[4].HunkIdx)
	}
}

func TestRenderFileDiffPlain(t *testing.T) {
	fd := FileDiff{
		Hunks: []Hunk{
			{
				Header: "@@ -1,3 +1,3 @@",
				Lines: []Line{
					{Type: LineContext, Content: "ctx"},
					{Type: LineRemoved, Content: "old"},
					{Type: LineAdded, Content: "new"},
				},
			},
		},
	}

	plain := RenderFileDiffPlain(fd)
	if plain == "" {
		t.Fatal("expected non-empty plain output")
	}

	lines := []string{
		"@@ -1,3 +1,3 @@",
		" ctx",
		"-old",
		"+new",
	}
	for _, want := range lines {
		if !contains(plain, want) {
			t.Errorf("plain output missing %q", want)
		}
	}
}

func TestDefaultRenderStyles(t *testing.T) {
	styles := DefaultRenderStyles()
	// Just verify it doesn't panic and returns non-zero value
	_ = styles.Added.Render("test")
	_ = styles.Removed.Render("test")
	_ = styles.Context.Render("test")
	_ = styles.HunkHeader.Render("test")
	_ = styles.LineNum.Render("test")
	_ = styles.Selected.Render("test")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
