package detector

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDetectGitBranch(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantBranch string
		wantIsRepo bool
		wantErr    bool
	}{
		{
			name: "non-git directory returns empty",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantBranch: "",
			wantIsRepo: false,
			wantErr:    false,
		},
		{
			name: "git directory returns branch",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Initialize git repo with a branch
				if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
					t.Skip("git not available")
				}
				if err := exec.Command("git", "-C", dir, "checkout", "-b", "feature/test").Run(); err != nil {
					t.Skip("git checkout failed")
				}
				return dir
			},
			wantBranch: "feature/test",
			wantIsRepo: true,
			wantErr:    false,
		},
		{
			name: "git directory with default branch after commit",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
					t.Skip("git not available")
				}
				// Create a commit so HEAD exists with default branch (modern git uses "main")
				_ = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
				_ = exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
				_ = exec.Command("touch", filepath.Join(dir, "dummy")).Run()
				_ = exec.Command("git", "-C", dir, "add", ".").Run()
				_ = exec.Command("git", "-C", dir, "commit", "-m", "init").Run()
				return dir
			},
			wantBranch: "main", // modern git default branch after init+commit
			wantIsRepo: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			gotBranch, gotIsRepo, err := detectGitBranch(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("detectGitBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotBranch != tt.wantBranch {
				t.Errorf("detectGitBranch() branch = %q, want %q", gotBranch, tt.wantBranch)
			}
			if gotIsRepo != tt.wantIsRepo {
				t.Errorf("detectGitBranch() isRepo = %v, want %v", gotIsRepo, tt.wantIsRepo)
			}
		})
	}
}

func TestDetectAgents(t *testing.T) {
	// Save and restore PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	tests := []struct {
		name       string
		setupPath  func(t *testing.T) string
		wantAgents []string
		wantErr    bool
	}{
		{
			name: "empty PATH returns empty agents",
			setupPath: func(t *testing.T) string {
				return ""
			},
			wantAgents: []string{},
			wantErr:    false,
		},
		{
			name: "PATH with no known agents returns empty",
			setupPath: func(t *testing.T) string {
				return t.TempDir()
			},
			wantAgents: []string{},
			wantErr:    false,
		},
		{
			name: "PATH with fake codex binary finds it",
			setupPath: func(t *testing.T) string {
				dir := t.TempDir()
				// Create a fake executable named "codex"
				fakeBinary := filepath.Join(dir, "codex")
				if err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
					t.Fatalf("failed to create fake binary: %v", err)
				}
				return dir
			},
			wantAgents: []string{"codex"},
			wantErr:    false,
		},
		{
			name: "PATH with multiple agents finds all in order",
			setupPath: func(t *testing.T) string {
				dir := t.TempDir()
				// Create fake executables for claude and opencode
				for _, name := range []string{"claude", "opencode"} {
					fakeBinary := filepath.Join(dir, name)
					if err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho fake"), 0755); err != nil {
						t.Fatalf("failed to create fake binary: %v", err)
					}
				}
				return dir
			},
			wantAgents: []string{"claude", "opencode"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupPath(t)
			os.Setenv("PATH", path)

			gotAgents, err := detectAgents()
			if (err != nil) != tt.wantErr {
				t.Fatalf("detectAgents() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(gotAgents) != len(tt.wantAgents) {
				t.Errorf("detectAgents() = %v, want %v", gotAgents, tt.wantAgents)
				return
			}
			for i := range gotAgents {
				if gotAgents[i] != tt.wantAgents[i] {
					t.Errorf("detectAgents()[%d] = %q, want %q", i, gotAgents[i], tt.wantAgents[i])
				}
			}
		})
	}
}

func TestDetectAll(t *testing.T) {
	ctx, err := DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error = %v", err)
	}

	if ctx.CWD == "" {
		t.Error("DetectAll() CWD should not be empty")
	}
	if ctx.FolderName == "" {
		t.Error("DetectAll() FolderName should not be empty")
	}
	if ctx.FolderName != filepath.Base(ctx.CWD) {
		t.Errorf("DetectAll() FolderName = %q, want %q", ctx.FolderName, filepath.Base(ctx.CWD))
	}
	if ctx.ParentFolder != filepath.Base(filepath.Dir(ctx.CWD)) {
		t.Errorf("DetectAll() ParentFolder = %q, want %q", ctx.ParentFolder, filepath.Base(filepath.Dir(ctx.CWD)))
	}
	if ctx.IsGitRepo && ctx.GitBranch == "" {
		t.Error("DetectAll() GitBranch should not be empty when IsGitRepo is true")
	}
	if ctx.Agents == nil {
		t.Error("DetectAll() Agents should not be nil")
	}
}

func TestErrAgentLookup(t *testing.T) {
	err := &ErrAgentLookup{Agent: "codex", Cause: os.ErrNotExist}
	want := `failed to look up agent "codex": file does not exist`
	if got := err.Error(); got != want {
		t.Errorf("ErrAgentLookup.Error() = %q, want %q", got, want)
	}
	if !os.IsNotExist(err.Unwrap()) {
		t.Error("ErrAgentLookup.Unwrap() should be os.ErrNotExist")
	}
}
