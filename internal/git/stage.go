package git

// StageFile stages a file.
func StageFile(repoDir, path string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"add", "--", path},
	})
	return err
}

// UnstageFile unstages a file.
func UnstageFile(repoDir, path string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"reset", "HEAD", "--", path},
	})
	return err
}

// StageAll stages all changes.
func StageAll(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"add", "-A"},
	})
	return err
}

// UnstageAll unstages all changes.
func UnstageAll(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"reset", "HEAD"},
	})
	return err
}

// DiscardFile discards changes to a file in the working tree.
func DiscardFile(repoDir, path string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"checkout", "--", path},
	})
	return err
}
