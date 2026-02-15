package git

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchGitDir watches the .git directory for changes and calls onChange
// when relevant files are modified. It debounces rapid changes.
// The caller should run this in a goroutine. It blocks until the done channel is closed.
func WatchGitDir(repoDir string, onChange func(), done <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	gitDir := resolveGitDir(repoDir)
	if gitDir == "" {
		return
	}

	// Watch .git root (catches HEAD, index, packed-refs, FETCH_HEAD, etc.)
	_ = watcher.Add(gitDir)

	// Recursively watch refs/ and all subdirectories
	refsDir := filepath.Join(gitDir, "refs")
	_ = filepath.Walk(refsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			_ = watcher.Add(path)
		}
		return nil
	})

	// Debounce: coalesce rapid events into a single callback
	var timer *time.Timer
	debounce := 50 * time.Millisecond

	for {
		select {
		case <-done:
			if timer != nil {
				timer.Stop()
			}
			return
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Skip lock files — git writes these temporarily
			if strings.HasSuffix(ev.Name, ".lock") {
				continue
			}

			// If a new directory was created under refs/, watch it too
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = watcher.Add(ev.Name)
				}
			}

			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(debounce, onChange)
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func resolveGitDir(repoDir string) string {
	gitDir := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return gitDir
	}

	// .git is a file (submodule/worktree) — read the gitdir pointer
	data, err := os.ReadFile(gitDir)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return ""
	}
	target := strings.TrimPrefix(line, "gitdir: ")
	if !filepath.IsAbs(target) {
		target = filepath.Join(repoDir, target)
	}
	return filepath.Clean(target)
}
