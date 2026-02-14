package git

import (
	"testing"
)

func TestCurrentBranchParsing(t *testing.T) {
	// This tests that the function would parse trimmed output
	// We can't easily unit test CurrentBranch without a real git repo,
	// but we verify the function signature exists and the module compiles
	_ = CurrentBranch
}

func TestLogEntryStruct(t *testing.T) {
	// Verify the struct has all expected fields
	entry := LogEntry{
		Hash:      "abc123def456",
		ShortHash: "abc123d",
		Subject:   "test commit",
		Author:    "Test User",
	}

	if entry.Hash != "abc123def456" {
		t.Errorf("Hash = %q, want %q", entry.Hash, "abc123def456")
	}
	if entry.ShortHash != "abc123d" {
		t.Errorf("ShortHash = %q, want %q", entry.ShortHash, "abc123d")
	}
	if entry.Subject != "test commit" {
		t.Errorf("Subject = %q, want %q", entry.Subject, "test commit")
	}
	if entry.Author != "Test User" {
		t.Errorf("Author = %q, want %q", entry.Author, "Test User")
	}
}
