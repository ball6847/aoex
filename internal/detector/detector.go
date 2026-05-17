package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

// Context holds auto-discovered environment information.
type Context struct {
	CWD          string   // os.Getwd()
	IsGitRepo    bool     // derived from git branch detection
	GitBranch    string   // current branch or ""
	FolderName   string   // filepath.Base(CWD)
	ParentFolder string   // filepath.Base(filepath.Dir(CWD))
	Agents       []string // detected agent binaries (claude, codex, opencode, etc.)
}

// DetectAll gathers context by composing git, cwd, and agent discovery.
func DetectAll() (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	branch, isRepo, err := detectGitBranch(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to detect git branch: %w", err)
	}

	agents, err := detectAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to detect agents: %w", err)
	}

	ctx := &Context{
		CWD:          cwd,
		IsGitRepo:    isRepo,
		GitBranch:    branch,
		FolderName:   filepath.Base(cwd),
		ParentFolder: filepath.Base(filepath.Dir(cwd)),
		Agents:       agents,
	}

	return ctx, nil
}
