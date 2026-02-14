package git

import (
	"sort"
	"strconv"
	"strings"
)

// Branch represents a git branch.
type Branch struct {
	Name      string
	IsCurrent bool
	IsRemote  bool
	Upstream  string
	Ahead     int
	Behind    int
}

// Tag represents a git tag.
type Tag struct {
	Name string
}

// Remote represents a git remote.
type Remote struct {
	Name string
	URL  string
}

// ListBranches returns all local branches with ahead/behind counts.
func ListBranches(repoDir string) ([]Branch, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"branch", "--format=%(HEAD)%(refname:short)\t%(upstream:short)\t%(upstream:track)"},
	})
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		isCurrent := strings.HasPrefix(line, "*")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimPrefix(line, " ")

		parts := strings.SplitN(line, "\t", 3)
		name := parts[0]
		upstream := ""
		if len(parts) > 1 {
			upstream = parts[1]
		}
		var ahead, behind int
		if len(parts) > 2 {
			ahead, behind = parseTrackInfo(parts[2])
		}

		branches = append(branches, Branch{
			Name:      name,
			IsCurrent: isCurrent,
			Upstream:  upstream,
			Ahead:     ahead,
			Behind:    behind,
		})
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

// parseTrackInfo parses git's %(upstream:track) output like "[ahead 3, behind 2]".
func parseTrackInfo(track string) (ahead, behind int) {
	track = strings.TrimSpace(track)
	if track == "" || track == "[gone]" {
		return 0, 0
	}
	track = strings.TrimPrefix(track, "[")
	track = strings.TrimSuffix(track, "]")
	for _, part := range strings.Split(track, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ahead ") {
			ahead, _ = strconv.Atoi(strings.TrimPrefix(part, "ahead "))
		} else if strings.HasPrefix(part, "behind ") {
			behind, _ = strconv.Atoi(strings.TrimPrefix(part, "behind "))
		}
	}
	return
}

// ListRemoteBranches returns all remote branches.
func ListRemoteBranches(repoDir string) ([]Branch, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"branch", "-r", "--format=%(refname:short)"},
	})
	if err != nil {
		return nil, err
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" || strings.HasSuffix(line, "/HEAD") {
			continue
		}
		branches = append(branches, Branch{
			Name:     line,
			IsRemote: true,
		})
	}

	return branches, nil
}

// ListTags returns all tags sorted by version.
func ListTags(repoDir string) ([]Tag, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"tag", "--sort=-version:refname"},
	})
	if err != nil {
		return nil, err
	}

	var tags []Tag
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		tags = append(tags, Tag{Name: line})
	}

	return tags, nil
}

// CreateTag creates a lightweight tag at the given ref (or HEAD if empty).
func CreateTag(repoDir, name, ref string) error {
	args := []string{"tag", name}
	if ref != "" {
		args = append(args, ref)
	}
	_, err := Run(RunOpts{Dir: repoDir, Args: args})
	return err
}

// DeleteTag deletes a tag.
func DeleteTag(repoDir, name string) error {
	_, err := Run(RunOpts{Dir: repoDir, Args: []string{"tag", "-d", name}})
	return err
}

// ListRemotes returns all configured remotes.
func ListRemotes(repoDir string) ([]Remote, error) {
	result, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"remote", "-v"},
	})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var remotes []Remote
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		remotes = append(remotes, Remote{
			Name: name,
			URL:  parts[1],
		})
	}

	return remotes, nil
}

// AddRemote adds a new remote with the given name and URL.
func AddRemote(repoDir, name, url string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"remote", "add", name, url},
	})
	return err
}

// Checkout switches to the given branch.
func Checkout(repoDir, branch string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"checkout", branch},
	})
	return err
}

// CreateBranch creates a new branch and checks it out.
func CreateBranch(repoDir, name string) error {
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"checkout", "-b", name},
	})
	return err
}

// DeleteBranch deletes a local branch.
func DeleteBranch(repoDir, name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := Run(RunOpts{
		Dir:  repoDir,
		Args: []string{"branch", flag, name},
	})
	return err
}
