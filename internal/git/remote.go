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

// PushTag pushes a single tag to the given remote.
func PushTag(repoDir, remote, tag string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"push", remote, "refs/tags/" + tag},
	})
	return err
}

// DeleteRemoteTag deletes a tag from the given remote.
func DeleteRemoteTag(repoDir, remote, tag string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"push", remote, "--delete", "refs/tags/" + tag},
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

// FetchAll fetches from all remotes.
func FetchAll(repoDir string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"fetch", "--all"},
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
