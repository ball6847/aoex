package detector

import (
	"fmt"
	"os"
	"os/exec"
)

var knownAgents = []string{"claude", "codex", "opencode"}

// detectAgents searches $PATH for known agent binaries.
func detectAgents() ([]string, error) {
	if _, ok := os.LookupEnv("PATH"); !ok {
		return []string{}, nil
	}

	var found []string
	for _, name := range knownAgents {
		if _, err := exec.LookPath(name); err == nil {
			found = append(found, name)
		}
	}

	return found, nil
}

// ErrAgentLookup wraps errors from agent detection.
type ErrAgentLookup struct {
	Agent string
	Cause error
}

func (e *ErrAgentLookup) Error() string {
	return fmt.Sprintf("failed to look up agent %q: %v", e.Agent, e.Cause)
}

func (e *ErrAgentLookup) Unwrap() error {
	return e.Cause
}
