package git

// Rebase rebases the current branch onto the given target.
func Rebase(repoDir, onto string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"rebase", onto},
	})
	return err
}

// RebaseAbort aborts an in-progress rebase.
func RebaseAbort(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"rebase", "--abort"},
	})
	return err
}

// RebaseContinue continues an in-progress rebase.
func RebaseContinue(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"rebase", "--continue"},
	})
	return err
}
