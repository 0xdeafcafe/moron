package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second

// RunOpts configures a git command execution.
type RunOpts struct {
	Dir   string
	Stdin string
	Args  []string
	Env   []string
}

// RunResult holds the output of a git command.
type RunResult struct {
	Stdout string
	Stderr string
}

// Run executes a git command with the given options.
func Run(opts RunOpts) (RunResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	return RunCtx(ctx, opts)
}

// RunCtx executes a git command with context.
func RunCtx(ctx context.Context, opts RunOpts) (RunResult, error) {
	cmd := exec.CommandContext(ctx, "git", opts.Args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if len(opts.Env) > 0 {
		cmd.Env = append(cmd.Environ(), opts.Env...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if opts.Stdin != "" {
		cmd.Stdin = strings.NewReader(opts.Stdin)
	}

	err := cmd.Run()
	result := RunResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		return result, fmt.Errorf("git %s: %w\n%s", strings.Join(opts.Args, " "), err, result.Stderr)
	}

	return result, nil
}
