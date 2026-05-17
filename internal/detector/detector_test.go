package detector

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
				err := exec.Command("git", "-C", dir, "init").Run()
				if err != nil {
					t.Skip("git not available")
				}
				err = exec.Command("git", "-C", dir, "checkout", "-b", "feature/test").Run()

				if err != nil {
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
				err := exec.Command("git", "-C", dir, "init").Run()
				if err != nil {
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

func TestDetectBranches(t *testing.T) {
	t.Run("non-git directory returns nil", func(t *testing.T) {
		dir := t.TempDir()

		branches, err := detectBranches(dir)
		if err != nil {
			t.Fatalf("detectBranches() error = %v", err)
		}

		if branches != nil && len(branches) > 0 {
			t.Errorf("detectBranches() = %v, want nil or empty", branches)
		}
	})

	t.Run("git repo returns branches", func(t *testing.T) {
		dir := t.TempDir()
		if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
			t.Skip("git not available")
		}

		_ = exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
		_ = exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()
		_ = exec.Command("touch", filepath.Join(dir, "dummy")).Run()
		_ = exec.Command("git", "-C", dir, "add", ".").Run()
		_ = exec.Command("git", "-C", dir, "commit", "-m", "init").Run()

		branches, err := detectBranches(dir)
		if err != nil {
			t.Fatalf("detectBranches() error = %v", err)
		}

		if len(branches) == 0 {
			t.Error("detectBranches() should return at least one branch")
		}
	})
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
				err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho fake"), 0755)
				if err != nil {
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
					err := os.WriteFile(fakeBinary, []byte("#!/bin/sh\necho fake"), 0755)
					if err != nil {
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

	if ctx.Config == nil {
		t.Error("DetectAll() Config should not be nil")
	}

	if ctx.Branches == nil {
		t.Error("DetectAll() Branches should not be nil")
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

func TestReadAoeConfig(t *testing.T) {
	// Override configDir to use a temp directory.
	t.Run("missing config uses defaults", func(t *testing.T) {
		cfg := defaultAoeConfig()
		if cfg.DefaultTool != "opencode" {
			t.Errorf("default DefaultTool = %q, want opencode", cfg.DefaultTool)
		}

		if cfg.Sandbox != false {
			t.Error("default Sandbox should be false")
		}

		if cfg.Worktree != false {
			t.Error("default Worktree should be false")
		}
	})

	t.Run("parses simple config values", func(t *testing.T) {
		dir := t.TempDir()

		content := `[session]
default_tool = "codex"
yolo_mode_default = true

[sandbox]
enabled_by_default = true
default_image = "custom:latest"

[worktree]
enabled = true
`
		err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644)

		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg := readAoeConfigFromDir(dir)
		if cfg.DefaultTool != "codex" {
			t.Errorf("DefaultTool = %q, want codex", cfg.DefaultTool)
		}

		if !cfg.Sandbox {
			t.Error("Sandbox should be true")
		}

		if cfg.DefaultImage != "custom:latest" {
			t.Errorf("DefaultImage = %q, want custom:latest", cfg.DefaultImage)
		}

		if !cfg.Worktree {
			t.Error("Worktree should be true")
		}
	})

	t.Run("parses custom agents", func(t *testing.T) {
		dir := t.TempDir()

		content := `[session]
custom_agents = { "kimi" = "kimi", "foo" = "bar" }
`
		err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(content), 0644)

		if err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg := readAoeConfigFromDir(dir)
		if len(cfg.CustomAgents) != 2 {
			t.Errorf("CustomAgents len = %d, want 2", len(cfg.CustomAgents))
		}

		if cfg.CustomAgents["kimi"] != "kimi" {
			t.Errorf("CustomAgents[kimi] = %q, want kimi", cfg.CustomAgents["kimi"])
		}

		if cfg.CustomAgents["foo"] != "bar" {
			t.Errorf("CustomAgents[foo] = %q, want bar", cfg.CustomAgents["foo"])
		}
	})

	t.Run("config dir on darwin", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("darwin-specific test")
		}

		dir := configDir()
		if !strings.Contains(dir, ".config") {
			t.Errorf("configDir() = %q, should contain .config on darwin", dir)
		}
	})
}

// readAoeConfigFromDir is a test helper that reads config from a given dir.
func readAoeConfigFromDir(dir string) *AoeConfig {
	cfg := defaultAoeConfig()

	f, err := os.Open(filepath.Join(dir, "config.toml"))

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

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.Trim(line, "[]"))

			continue
		}

		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		val = strings.Trim(val, `"'`)

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
				cfg.CustomAgents = parseInlineTable(val)
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

func TestParseInlineTable(t *testing.T) {
	tests := []struct {
		input string
		want  map[string]string
	}{
		{
			input: `{ "kimi" = "kimi" }`,
			want:  map[string]string{"kimi": "kimi"},
		},
		{
			input: `{ "kimi" = "kimi", "foo" = "bar" }`,
			want:  map[string]string{"kimi": "kimi", "foo": "bar"},
		},
		{
			input: `invalid`,
			want:  map[string]string{},
		},
		{
			input: `{}`,
			want:  map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInlineTable(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseInlineTable(%q) = %v, want %v", tt.input, got, tt.want)

				return
			}

			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseInlineTable(%q)[%q] = %q, want %q", tt.input, k, got[k], v)
				}
			}
		})
	}
}
