package git

import "strings"

// Commit creates a new commit with the given message.
func Commit(repoDir, message string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"commit", "-m", message},
	})
	return err
}

// CommitAmend amends the last commit with the given message.
func CommitAmend(repoDir, message string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"commit", "--amend", "-m", message},
	})
	return err
}

// LastCommitMessage returns the subject and body of the last commit.
func LastCommitMessage(repoDir string) (string, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"log", "-1", "--format=%B"},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

// CurrentAuthor returns the configured user name and email.
func CurrentAuthor(repoDir string) (string, error) {
	name, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"config", "user.name"},
	})
	if err != nil {
		return "", err
	}

	email, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"config", "user.email"},
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(name.Stdout) + " <" + strings.TrimSpace(email.Stdout) + ">", nil
}
