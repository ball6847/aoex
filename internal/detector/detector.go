package detector

import (
	"fmt"
	"os"
	"path/filepath"
)

// Context holds auto-discovered environment information.
type Context struct {
	CWD          string            // os.Getwd()
	IsGitRepo    bool              // derived from git branch detection
	GitBranch    string            // current branch or ""
	FolderName   string            // filepath.Base(CWD)
	ParentFolder string            // filepath.Base(filepath.Dir(CWD))
	Agents       []string          // detected agent binaries (claude, codex, opencode, etc.)
	Branches     []string          // all git branches (local + remote)
	Worktrees    map[string]string // branch name → worktree path
	Config       *AoeConfig        // parsed aoe config defaults
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

	branches, err := detectBranches(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to detect branches: %w", err)
	}

	worktrees, err := detectWorktrees(cwd)
	if err != nil {
		return nil, fmt.Errorf("failed to detect worktrees: %w", err)
	}

	cfg := readAoeConfig()

	// Merge custom agents that are actually available on PATH.
	customAgents := detectCustomAgents(cfg.CustomAgents)
	agentSet := make(map[string]bool, len(agents)+len(customAgents))
	for _, a := range agents {
		agentSet[a] = true
	}
	for _, ca := range customAgents {
		if !agentSet[ca] {
			agents = append(agents, ca)
			agentSet[ca] = true
		}
	}

	ctx := &Context{
		CWD:          cwd,
		IsGitRepo:    isRepo,
		GitBranch:    branch,
		FolderName:   filepath.Base(cwd),
		ParentFolder: filepath.Base(filepath.Dir(cwd)),
		Agents:       agents,
		Branches:     branches,
		Worktrees:    worktrees,
		Config:       cfg,
	}

	return ctx, nil
}
