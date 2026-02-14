package git

// DiffOptions configures how to run a diff.
type DiffOptions struct {
	Cached bool
	File   string
}

// Diff runs git diff and returns the raw output.
func Diff(repoDir string, opts DiffOptions) (string, error) {
	args := []string{"diff", "--no-color"}
	if opts.Cached {
		args = append(args, "--cached")
	}
	if opts.File != "" {
		args = append(args, "--", opts.File)
	}

	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: args,
	})
	if err != nil {
		return "", err
	}

	return result.Stdout, nil
}

// DiffUntracked returns the content of an untracked file formatted as a diff.
func DiffUntracked(repoDir string, file string) (string, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"diff", "--no-color", "--no-index", "--", "/dev/null", file},
	})
	// git diff --no-index returns exit code 1 when files differ, which is expected
	if err != nil && result.Stdout == "" {
		return "", err
	}

	return result.Stdout, nil
}
