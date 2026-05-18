package detector

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// agentEntry mirrors aoe's built-in agent registry (src/agents.rs).
// For most agents, the detection binary matches the name.
// Exceptions (e.g. cursor -> 'agent', kiro -> 'kiro-cli') are explicitly mapped.
var builtInAgents = []struct {
	name   string
	binary string
}{
	{name: "claude", binary: "claude"},
	{name: "opencode", binary: "opencode"},
	{name: "vibe", binary: "vibe"},
	{name: "codex", binary: "codex"},
	{name: "gemini", binary: "gemini"},
	{name: "cursor", binary: "agent"},
	{name: "copilot", binary: "copilot"},
	{name: "pi", binary: "pi"},
	{name: "droid", binary: "droid"},
	{name: "settl", binary: "settl"},
	{name: "hermes", binary: "hermes"},
	{name: "kiro", binary: "kiro-cli"},
	// NOTE: kimi is from a custom fork of aoe and is not yet natively supported upstream.
	{name: "kimi", binary: "kimi"},
	{name: "qwen", binary: "qwen"},
}

// detectAgents searches $PATH for all known agent binaries (matching aoe's registry).
// It returns the agent canonical names (not binary names) in the same order as builtInAgents.
func detectAgents() ([]string, error) {
	if _, ok := os.LookupEnv("PATH"); !ok {
		return []string{}, nil
	}

	found := make([]string, 0, len(builtInAgents))

	for _, a := range builtInAgents {
		if _, err := exec.LookPath(a.binary); err == nil {
			found = append(found, a.name)
		}
	}

	return found, nil
}

// detectCustomAgents filters a map of custom agent name→command by whether the
// command binary exists on PATH. Returns names of available custom agents.
func detectCustomAgents(customAgents map[string]string) []string {
	found := make([]string, 0, len(customAgents))

	for name, cmd := range customAgents {
		binary := cmd
		// If command contains spaces, extract the first token as the binary name.
		if idx := strings.IndexAny(binary, " \t"); idx >= 0 {
			binary = strings.TrimSpace(binary[:idx])
		}

		if binary == "" {
			continue
		}

		if _, err := exec.LookPath(binary); err == nil {
			found = append(found, name)
		}
	}

	return found
}

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
