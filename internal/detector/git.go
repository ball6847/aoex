package detector

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// detectGitBranch returns the current git branch and whether the directory is a git repo.
func detectGitBranch(cwd string) (branch string, isRepo bool, err error) {
	cmd := exec.Command("git", "-C", cwd, "branch", "--show-current")

	out, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			// Git exits with 128 when not inside a repository.
			if exitErr.ExitCode() == 128 {
				return "", false, nil
			}
		}
		// Git not installed or other error — graceful degradation.
		error := &exec.Error{}
		if errors.As(err, &error) {
			return "", false, nil
		}

		return "", false, fmt.Errorf("failed to detect git branch: %w", err)
	}

	b := strings.TrimSpace(string(out))
	if b == "" {
		return "", false, nil
	}

	return b, true, nil
}
