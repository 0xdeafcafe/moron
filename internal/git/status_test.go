package git

import (
	"testing"
)

func TestParseStatusEmpty(t *testing.T) {
	files := parseStatus("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParseStatusOrdinaryModified(t *testing.T) {
	// Porcelain v2 format for a modified file (both staged and unstaged)
	output := "1 MM N... 100644 100644 100644 abc123 def456 main.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "M" {
		t.Errorf("Staged = %q, want %q", files[0].Staged, "M")
	}
	if files[0].Unstaged != "M" {
		t.Errorf("Unstaged = %q, want %q", files[0].Unstaged, "M")
	}
	if files[0].Path != "main.go" {
		t.Errorf("Path = %q, want %q", files[0].Path, "main.go")
	}
}

func TestParseStatusStagedOnly(t *testing.T) {
	output := "1 M. N... 100644 100644 100644 abc123 def456 file.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "M" {
		t.Errorf("Staged = %q, want %q", files[0].Staged, "M")
	}
	if files[0].Unstaged != "" {
		t.Errorf("Unstaged = %q, want empty", files[0].Unstaged)
	}
}

func TestParseStatusUnstagedOnly(t *testing.T) {
	output := "1 .M N... 100644 100644 100644 abc123 def456 file.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "" {
		t.Errorf("Staged = %q, want empty", files[0].Staged)
	}
	if files[0].Unstaged != "M" {
		t.Errorf("Unstaged = %q, want %q", files[0].Unstaged, "M")
	}
}

func TestParseStatusAddedFile(t *testing.T) {
	output := "1 A. N... 000000 100644 100644 0000000 abc1234 new.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "A" {
		t.Errorf("Staged = %q, want %q", files[0].Staged, "A")
	}
}

func TestParseStatusDeletedFile(t *testing.T) {
	output := "1 D. N... 100644 000000 100644 abc1234 0000000 removed.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "D" {
		t.Errorf("Staged = %q, want %q", files[0].Staged, "D")
	}
}

func TestParseStatusUntracked(t *testing.T) {
	output := "? newfile.txt"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "" {
		t.Errorf("Staged = %q, want empty", files[0].Staged)
	}
	if files[0].Unstaged != "?" {
		t.Errorf("Unstaged = %q, want %q", files[0].Unstaged, "?")
	}
	if files[0].Path != "newfile.txt" {
		t.Errorf("Path = %q, want %q", files[0].Path, "newfile.txt")
	}
}

func TestParseStatusRename(t *testing.T) {
	output := "2 R. N... 100644 100644 100644 abc123 def456 R100 new.go\told.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Staged != "R" {
		t.Errorf("Staged = %q, want %q", files[0].Staged, "R")
	}
	if files[0].Path != "new.go" {
		t.Errorf("Path = %q, want %q", files[0].Path, "new.go")
	}
	if files[0].OrigPath != "old.go" {
		t.Errorf("OrigPath = %q, want %q", files[0].OrigPath, "old.go")
	}
}

func TestParseStatusMultipleFiles(t *testing.T) {
	output := `1 M. N... 100644 100644 100644 abc def file1.go
1 .M N... 100644 100644 100644 abc def file2.go
? untracked.txt`
	files := parseStatus(output)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}
	if files[0].Path != "file1.go" {
		t.Errorf("files[0].Path = %q, want %q", files[0].Path, "file1.go")
	}
	if files[1].Path != "file2.go" {
		t.Errorf("files[1].Path = %q, want %q", files[1].Path, "file2.go")
	}
	if files[2].Path != "untracked.txt" {
		t.Errorf("files[2].Path = %q, want %q", files[2].Path, "untracked.txt")
	}
}

func TestTranslateStatus(t *testing.T) {
	tests := []struct {
		input byte
		want  string
	}{
		{'.', ""},
		{'M', "M"},
		{'A', "A"},
		{'D', "D"},
		{'R', "R"},
		{'?', "?"},
	}

	for _, tt := range tests {
		got := translateStatus(tt.input)
		if got != tt.want {
			t.Errorf("translateStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseStatusPathWithSpaces(t *testing.T) {
	output := "1 M. N... 100644 100644 100644 abc def my file.go"
	files := parseStatus(output)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "my file.go" {
		t.Errorf("Path = %q, want %q", files[0].Path, "my file.go")
	}
}
