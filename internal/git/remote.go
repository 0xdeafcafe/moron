package git

// Push pushes the current branch to the given remote.
func Push(repoDir, remote, branch string, force bool) error {
	args := []string{"push", remote, branch}
	if force {
		args = []string{"push", "--force-with-lease", remote, branch}
	}

	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: args,
	})
	return err
}

// Fetch fetches from the given remote.
func Fetch(repoDir, remote string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"fetch", remote},
	})
	return err
}

// Pull pulls from the given remote.
func Pull(repoDir, remote, branch string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"pull", remote, branch},
	})
	return err
}
