package detector

import (
	"fmt"
	"os/exec"
	"strings"
)

// WorktreeInfo holds the path and branch for a git worktree.
type WorktreeInfo struct {
	Path   string
	Branch string // just the branch name without refs/heads/ prefix
	Head   string // commit hash
	Bare   bool
}

// detectWorktrees returns a map of branch name → worktree path for the repo at cwd.
// It returns nil if not in a git repo or git is unavailable.
func detectWorktrees(cwd string) (map[string]string, error) {
	cmd := exec.Command("git", "-C", cwd, "worktree", "list", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		// Graceful degradation
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 128 {
				return nil, nil
			}
		}

		if _, ok := err.(*exec.Error); ok {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	result := make(map[string]string)
	var currentPath string
	var currentBranch string

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line = end of worktree block. Save it.
			if currentPath != "" && currentBranch != "" {
				// Strip refs/heads/ prefix if present.
				branch := strings.TrimPrefix(currentBranch, "refs/heads/")
				if branch != "" {
					result[branch] = currentPath
				}
			}

			currentPath = ""
			currentBranch = ""

			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") {
			currentBranch = strings.TrimPrefix(line, "branch ")
		}
	}

	// Handle last block if file doesn't end with blank line.
	if currentPath != "" && currentBranch != "" {
		branch := strings.TrimPrefix(currentBranch, "refs/heads/")
		if branch != "" {
			result[branch] = currentPath
		}
	}

	return result, nil
}
