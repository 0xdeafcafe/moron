package git

// ApplyPatchCached applies a patch to the index (staging area).
func ApplyPatchCached(repoDir, patch string) error {
	_, err := Run(RunOpts{
		Dir:   repoDir,
		Stdin: patch,
		Args:  []string{"apply", "--cached", "--unidiff-zero", "--whitespace=nowarn"},
	})
	return err
}

// ApplyPatchReverseCached applies a reverse patch to the index (unstaging).
func ApplyPatchReverseCached(repoDir, patch string) error {
	_, err := Run(RunOpts{
		Dir:   repoDir,
		Stdin: patch,
		Args:  []string{"apply", "--cached", "--reverse", "--unidiff-zero", "--whitespace=nowarn"},
	})
	return err
}

// ApplyPatchReverse applies a reverse patch to the working tree (discard).
func ApplyPatchReverse(repoDir, patch string) error {
	_, err := Run(RunOpts{
		Dir:   repoDir,
		Stdin: patch,
		Args:  []string{"apply", "--reverse", "--unidiff-zero", "--whitespace=nowarn"},
	})
	return err
}
