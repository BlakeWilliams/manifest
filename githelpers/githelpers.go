package githelpers

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var ErrNoPushedBranch = errors.New("no pushed branch exists for current branch")

// UpstreamSha returns the SHA of the most recent commit on the branch pushed to
// origin.
func UpstreamSha() (string, error) {
	branchCmd := exec.Command(gitPath(), "rev-parse", "--abbrev-ref", "HEAD")
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get current branch for UpstreamSHA: %w", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Get the latest pushed SHA for the current branch
	shaCmd := exec.Command(gitPath(), "rev-parse", "origin/"+branch)
	shaOutput, err := shaCmd.Output()
	if err != nil {
		if strings.Contains(string(shaOutput), "unknown revision") {
			return "", ErrNoPushedBranch
		}
		return "", err
	}
	sha := strings.TrimSpace(string(shaOutput))

	return sha, nil
}

// MostRecentSha returns the SHA of the most recent commit on the current
// branch.
func MostRecentSha() (string, error) {
	shaCmd := exec.Command(gitPath(), "rev-parse", "HEAD")
	shaOutput, err := shaCmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get most recent SHA: %w", err)
	}
	sha := strings.TrimSpace(string(shaOutput))

	return sha, nil
}

var originRegexp = regexp.MustCompile(`(?:https?://github\.com/|git@github\.com:)([^/]+)/([^\.]+)`)

// NwoFromOrigin returns the owner and repo of the origin remote.
func NwoFromOrigin() (string, string, error) {
	cmd := exec.Command(gitPath(), "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("could not get origin remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))

	matches := originRegexp.FindStringSubmatch(remoteURL)
	if len(matches) != 3 {
		return "", "", fmt.Errorf("could not parse owner and repo from remote URL")
	}
	owner := matches[1]
	repo := matches[2]

	return owner, repo, nil
}

func CurrentBranch() (string, error) {
	cmd := exec.Command(gitPath(), "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()

	if err != nil {
		return "", fmt.Errorf("could not get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func gitPath() string {
	path, err := exec.LookPath("git")
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	if err != nil {
		panic(err)
	}

	return path
}
