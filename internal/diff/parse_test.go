package diff

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	result := Parse("")
	if len(result) != 0 {
		t.Errorf("expected 0 diffs, got %d", len(result))
	}
}

func TestParseSingleFileModified(t *testing.T) {
	raw := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -10,7 +10,8 @@ func main() {
     fmt.Println("hello")
-    fmt.Println("old line")
+    fmt.Println("new line")
+    fmt.Println("extra line")
     fmt.Println("world")
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 file diff, got %d", len(diffs))
	}

	fd := diffs[0]
	if fd.OldName != "main.go" {
		t.Errorf("OldName = %q, want %q", fd.OldName, "main.go")
	}
	if fd.NewName != "main.go" {
		t.Errorf("NewName = %q, want %q", fd.NewName, "main.go")
	}
	if fd.IsBinary {
		t.Error("expected non-binary file")
	}
	if len(fd.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(fd.Hunks))
	}

	h := fd.Hunks[0]
	if h.OldStart != 10 || h.OldCount != 7 {
		t.Errorf("old range = %d,%d, want 10,7", h.OldStart, h.OldCount)
	}
	if h.NewStart != 10 || h.NewCount != 8 {
		t.Errorf("new range = %d,%d, want 10,8", h.NewStart, h.NewCount)
	}
	if len(h.Lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(h.Lines))
	}

	// Verify line types
	expectedTypes := []LineType{LineContext, LineRemoved, LineAdded, LineAdded, LineContext}
	for i, lt := range expectedTypes {
		if h.Lines[i].Type != lt {
			t.Errorf("line %d type = %d, want %d", i, h.Lines[i].Type, lt)
		}
	}
}

func TestParseNewFile(t *testing.T) {
	raw := `diff --git a/new.txt b/new.txt
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/new.txt
@@ -0,0 +1,3 @@
+line 1
+line 2
+line 3
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	fd := diffs[0]
	if fd.OldName != "/dev/null" {
		t.Errorf("OldName = %q, want %q", fd.OldName, "/dev/null")
	}
	if fd.NewName != "new.txt" {
		t.Errorf("NewName = %q, want %q", fd.NewName, "new.txt")
	}
	if len(fd.Hunks) != 1 {
		t.Fatalf("expected 1 hunk, got %d", len(fd.Hunks))
	}
	if len(fd.Hunks[0].Lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(fd.Hunks[0].Lines))
	}
	for _, line := range fd.Hunks[0].Lines {
		if line.Type != LineAdded {
			t.Errorf("expected all lines to be additions, got %d", line.Type)
		}
	}
}

func TestParseDeletedFile(t *testing.T) {
	raw := `diff --git a/old.txt b/old.txt
deleted file mode 100644
index 1234567..0000000
--- a/old.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-line 1
-line 2
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}

	fd := diffs[0]
	if fd.OldName != "old.txt" {
		t.Errorf("OldName = %q, want %q", fd.OldName, "old.txt")
	}
	if fd.NewName != "/dev/null" {
		t.Errorf("NewName = %q, want %q", fd.NewName, "/dev/null")
	}
	if len(fd.Hunks[0].Lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(fd.Hunks[0].Lines))
	}
	for _, line := range fd.Hunks[0].Lines {
		if line.Type != LineRemoved {
			t.Errorf("expected all lines to be removals, got %d", line.Type)
		}
	}
}

func TestParseBinaryFile(t *testing.T) {
	raw := `diff --git a/image.png b/image.png
index abc..def 100644
Binary files a/image.png and b/image.png differ
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if !diffs[0].IsBinary {
		t.Error("expected binary file")
	}
	if len(diffs[0].Hunks) != 0 {
		t.Error("expected no hunks for binary file")
	}
}

func TestParseMultipleFiles(t *testing.T) {
	raw := `diff --git a/file1.go b/file1.go
index abc..def 100644
--- a/file1.go
+++ b/file1.go
@@ -1,3 +1,3 @@
 context
-old
+new
diff --git a/file2.go b/file2.go
index abc..def 100644
--- a/file2.go
+++ b/file2.go
@@ -5,3 +5,4 @@
 context
+added
 more context
`
	diffs := Parse(raw)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d", len(diffs))
	}
	if diffs[0].NewName != "file1.go" {
		t.Errorf("first file = %q, want %q", diffs[0].NewName, "file1.go")
	}
	if diffs[1].NewName != "file2.go" {
		t.Errorf("second file = %q, want %q", diffs[1].NewName, "file2.go")
	}
}

func TestParseMultipleHunks(t *testing.T) {
	raw := `diff --git a/main.go b/main.go
index abc..def 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 first
+added
 second
@@ -10,3 +11,3 @@
 tenth
-old
+new
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	if len(diffs[0].Hunks) != 2 {
		t.Fatalf("expected 2 hunks, got %d", len(diffs[0].Hunks))
	}
	if diffs[0].Hunks[0].OldStart != 1 {
		t.Errorf("hunk 0 OldStart = %d, want 1", diffs[0].Hunks[0].OldStart)
	}
	if diffs[0].Hunks[1].OldStart != 10 {
		t.Errorf("hunk 1 OldStart = %d, want 10", diffs[0].Hunks[1].OldStart)
	}
}

func TestParseLineNumbers(t *testing.T) {
	raw := `diff --git a/test.go b/test.go
index abc..def 100644
--- a/test.go
+++ b/test.go
@@ -5,5 +5,5 @@
 context1
-removed
+added
 context2
 context3
`
	diffs := Parse(raw)
	h := diffs[0].Hunks[0]

	// context1: old=5, new=5
	if h.Lines[0].OldNum != 5 || h.Lines[0].NewNum != 5 {
		t.Errorf("context1: old=%d new=%d, want old=5 new=5", h.Lines[0].OldNum, h.Lines[0].NewNum)
	}
	// removed: old=6, new=0
	if h.Lines[1].OldNum != 6 || h.Lines[1].NewNum != 0 {
		t.Errorf("removed: old=%d new=%d, want old=6 new=0", h.Lines[1].OldNum, h.Lines[1].NewNum)
	}
	// added: old=0, new=6
	if h.Lines[2].OldNum != 0 || h.Lines[2].NewNum != 6 {
		t.Errorf("added: old=%d new=%d, want old=0 new=6", h.Lines[2].OldNum, h.Lines[2].NewNum)
	}
	// context2: old=7, new=7
	if h.Lines[3].OldNum != 7 || h.Lines[3].NewNum != 7 {
		t.Errorf("context2: old=%d new=%d, want old=7 new=7", h.Lines[3].OldNum, h.Lines[3].NewNum)
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		header   string
		oldStart int
		oldCount int
		newStart int
		newCount int
	}{
		{"@@ -1,3 +1,4 @@", 1, 3, 1, 4},
		{"@@ -10,7 +10,8 @@ func main() {", 10, 7, 10, 8},
		{"@@ -0,0 +1,5 @@", 0, 0, 1, 5},
		{"@@ -1 +1 @@", 1, 1, 1, 1},
		{"@@ -100,200 +150,300 @@", 100, 200, 150, 300},
	}

	for _, tt := range tests {
		os, oc, ns, nc := parseHunkHeader(tt.header)
		if os != tt.oldStart || oc != tt.oldCount || ns != tt.newStart || nc != tt.newCount {
			t.Errorf("parseHunkHeader(%q) = %d,%d,%d,%d, want %d,%d,%d,%d",
				tt.header, os, oc, ns, nc, tt.oldStart, tt.oldCount, tt.newStart, tt.newCount)
		}
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		input string
		start int
		count int
	}{
		{"1,3", 1, 3},
		{"10,7", 10, 7},
		{"0,0", 0, 0},
		{"5", 5, 1},
	}

	for _, tt := range tests {
		s, c := parseRange(tt.input)
		if s != tt.start || c != tt.count {
			t.Errorf("parseRange(%q) = %d,%d, want %d,%d", tt.input, s, c, tt.start, tt.count)
		}
	}
}

func TestFormatHunkHeader(t *testing.T) {
	got := FormatHunkHeader(10, 7, 10, 8)
	want := "@@ -10,7 +10,8 @@"
	if got != want {
		t.Errorf("FormatHunkHeader = %q, want %q", got, want)
	}
}

func TestParseNoNewlineAtEnd(t *testing.T) {
	raw := `diff --git a/test.go b/test.go
index abc..def 100644
--- a/test.go
+++ b/test.go
@@ -1,2 +1,2 @@
-old line
\ No newline at end of file
+new line
\ No newline at end of file
`
	diffs := Parse(raw)
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	h := diffs[0].Hunks[0]
	if len(h.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(h.Lines))
	}
	if h.Lines[0].Type != LineRemoved {
		t.Errorf("line 0 type = %d, want LineRemoved", h.Lines[0].Type)
	}
	if h.Lines[1].Type != LineAdded {
		t.Errorf("line 1 type = %d, want LineAdded", h.Lines[1].Type)
	}
}
