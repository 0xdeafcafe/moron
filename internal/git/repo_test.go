package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRepoCurrentDir(t *testing.T) {
	// Create a temp directory with a .git directory
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := FindRepo(tmp)
	if err != nil {
		t.Fatalf("FindRepo error: %v", err)
	}
	if result != tmp {
		t.Errorf("FindRepo = %q, want %q", result, tmp)
	}
}

func TestFindRepoParentDir(t *testing.T) {
	// Create a temp directory structure: root/.git, root/subdir
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(tmp, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := FindRepo(subDir)
	if err != nil {
		t.Fatalf("FindRepo error: %v", err)
	}
	if result != tmp {
		t.Errorf("FindRepo = %q, want %q", result, tmp)
	}
}

func TestFindRepoDeepNesting(t *testing.T) {
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	if err := os.Mkdir(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	deep := filepath.Join(tmp, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := FindRepo(deep)
	if err != nil {
		t.Fatalf("FindRepo error: %v", err)
	}
	if result != tmp {
		t.Errorf("FindRepo = %q, want %q", result, tmp)
	}
}

func TestFindRepoNotFound(t *testing.T) {
	tmp := t.TempDir()
	// No .git directory created

	_, err := FindRepo(tmp)
	if err == nil {
		t.Fatal("expected error for missing git repo")
	}
}
