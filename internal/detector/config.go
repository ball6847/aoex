package detector

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// AoeConfig holds the subset of aoe's config.toml that aoex needs.
type AoeConfig struct {
	DefaultTool  string
	Sandbox      bool
	Worktree     bool
	CustomAgents map[string]string
	DefaultImage string
}

// defaultAoeConfig returns hardcoded defaults matching aoe's defaults.
func defaultAoeConfig() *AoeConfig {
	return &AoeConfig{
		DefaultTool:  "opencode",
		Sandbox:      false,
		Worktree:     false,
		CustomAgents: map[string]string{},
		DefaultImage: "ghcr.io/njbrake/aoe-sandbox:latest",
	}
}

// configDir returns the aoe config directory for the current OS.
func configDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), ".config", "agent-of-empires")
	case "windows":
		// Windows: %APPDATA%\agent-of-empires or %LOCALAPPDATA%\agent-of-empires
		if appData := os.Getenv("LOCALAPPDATA"); appData != "" {
			return filepath.Join(appData, "agent-of-empires")
		}

		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "agent-of-empires")
		}

		return ""
	default:
		// Linux and others: respect XDG_CONFIG_HOME, fall back to ~/.config
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "agent-of-empires")
		}

		return filepath.Join(os.Getenv("HOME"), ".config", "agent-of-empires")
	}
}

// readAoeConfig reads the subset of aoe's config.toml that aoex cares about.
// It performs a simple line-by-line scan to avoid a toml parser dependency.
func readAoeConfig() *AoeConfig {
	cfg := defaultAoeConfig()

	dir := configDir()
	if dir == "" {
		return cfg
	}

	path := filepath.Join(dir, "config.toml")

	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	var section string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Track section headers like [session] or [sandbox]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))

			continue
		}

		// Parse key = value lines
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])
		// Remove quotes
		val = strings.Trim(val, `"'`)
		// Remove inline comments
		if hashIdx := strings.Index(val, "#"); hashIdx >= 0 {
			val = strings.TrimSpace(val[:hashIdx])
			val = strings.Trim(val, `"'`)
		}

		switch section {
		case "session":
			switch key {
			case "default_tool":
				cfg.DefaultTool = val
			case "custom_agents":
				// custom_agents = { "kimi" = "kimi" }
				cfg.CustomAgents = parseInlineTable(val)
			case "yolo_mode_default":
				// Not used by aoex
			}
		case "sandbox":
			switch key {
			case "enabled_by_default":
				cfg.Sandbox = strings.ToLower(val) == "true"
			case "default_image":
				cfg.DefaultImage = val
			}
		case "worktree":
			switch key {
			case "enabled":
				cfg.Worktree = strings.ToLower(val) == "true"
			}
		}
	}

	return cfg
}

// parseInlineTable parses a minimal inline table like { "kimi" = "kimi", "foo" = "bar" }.
func parseInlineTable(s string) map[string]string {
	result := make(map[string]string)

	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return result
	}

	s = strings.Trim(s, "{}")

	pairs := splitTopLevelCommas(s)
	for _, pair := range pairs {
		eqIdx := strings.Index(pair, "=")
		if eqIdx < 0 {
			continue
		}

		k := strings.TrimSpace(pair[:eqIdx])
		v := strings.TrimSpace(pair[eqIdx+1:])
		k = strings.Trim(k, `"'`)

		v = strings.Trim(v, `"'`)
		if k != "" {
			result[k] = v
		}
	}

	return result
}

// splitTopLevelCommas splits a string by commas not inside braces or quotes.
func splitTopLevelCommas(s string) []string {
	var (
		parts   []string
		current strings.Builder
	)

	inQuotes := false
	quoteChar := rune(0)
	depth := 0

	for _, r := range s {
		switch r {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if quoteChar == r {
				inQuotes = false
			}

			current.WriteRune(r)
		case '{':
			if !inQuotes {
				depth++
			}

			current.WriteRune(r)
		case '}':
			if !inQuotes {
				depth--
			}

			current.WriteRune(r)
		case ',':
			if !inQuotes && depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// detectBranches returns a list of branches from the git repo at cwd.
// It returns local branches first, then remote-tracking branches.
func detectBranches(cwd string) ([]string, error) {
	cmd := exec.Command("git", "-C", cwd, "branch", "-a", "--format=%(refname:short)")

	out, err := cmd.Output()
	if err != nil {
		// Graceful degradation if not a repo or git not installed
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			return nil, nil
		}

		error := &exec.Error{}
		if errors.As(err, &error) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var branches []string

	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip detached HEAD entries
		if strings.Contains(line, "HEAD") {
			continue
		}
		// Normalize remote branches: origin/main -> origin/main (keep as-is)
		// But deduplicate if local branch exists
		if seen[line] {
			continue
		}

		seen[line] = true
		branches = append(branches, line)
	}

	return branches, nil
}
