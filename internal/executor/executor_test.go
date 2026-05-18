package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArgsString(t *testing.T) {
	tests := []struct {
		name string
		args Args
		want string
	}{
		{
			name: "all defaults except title and cmd",
			args: Args{
				Path:  ".",
				Title: "myproject",
				Cmd:   "codex",
			},
			want: `aoe add . --title "myproject" --cmd "codex"`,
		},
		{
			name: "with custom path",
			args: Args{
				Path:  "/home/user/myproject",
				Title: "myproject",
				Cmd:   "codex",
			},
			want: `aoe add "/home/user/myproject" --title "myproject" --cmd "codex"`,
		},
		{
			name: "with launch flag",
			args: Args{
				Path:   ".",
				Title:  "myproject",
				Cmd:    "codex",
				Launch: true,
			},
			want: `aoe add . --title "myproject" --cmd "codex" --launch`,
		},
		{
			name: "with worktree",
			args: Args{
				Path:     ".",
				Title:    "myproject",
				Cmd:      "codex",
				Worktree: "feature/hello",
			},
			want: `aoe add . --title "myproject" --cmd "codex" --worktree "feature/hello"`,
		},
		{
			name: "with sandbox",
			args: Args{
				Path:    ".",
				Title:   "myproject",
				Cmd:     "codex",
				Sandbox: true,
			},
			want: `aoe add . --title "myproject" --cmd "codex" --sandbox`,
		},
		{
			name: "with new branch",
			args: Args{
				Path:      ".",
				Title:     "myproject",
				Cmd:       "codex",
				NewBranch: true,
			},
			want: `aoe add . --title "myproject" --cmd "codex" -b`,
		},
		{
			name: "all flags enabled",
			args: Args{
				Path:      ".",
				Title:     "myproject",
				Cmd:       "codex",
				Launch:    true,
				Worktree:  "feature/hello",
				Sandbox:   true,
				NewBranch: true,
			},
			want: `aoe add . --title "myproject" --cmd "codex" --launch --worktree "feature/hello" -b --sandbox`,
		},
		{
			name: "attach to existing worktree (no worktree flag)",
			args: Args{
				Path:      "/path/to/worktree",
				Title:     "feature/existing",
				Cmd:       "codex",
				Worktree:  "", // Empty when attaching to existing worktree
				NewBranch: false,
			},
			want: `aoe add "/path/to/worktree" --title "feature/existing" --cmd "codex"`,
		},
		{
			name: "create worktree from existing branch (no -b flag)",
			args: Args{
				Path:      ".",
				Title:     "feature/no-wt",
				Cmd:       "codex",
				Worktree:  "feature/no-wt",
				NewBranch: false,
			},
			want: `aoe add . --title "feature/no-wt" --cmd "codex" --worktree "feature/no-wt"`,
		},
		{
			name: "create new branch with worktree (-b flag)",
			args: Args{
				Path:      ".",
				Title:     "feature/new",
				Cmd:       "codex",
				Worktree:  "feature/new",
				NewBranch: true,
			},
			want: `aoe add . --title "feature/new" --cmd "codex" --worktree "feature/new" -b`,
		},
		{
			name: "empty path defaults to dot",
			args: Args{
				Path:  "",
				Title: "myproject",
				Cmd:   "codex",
			},
			want: `aoe add . --title "myproject" --cmd "codex"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.String()
			if got != tt.want {
				t.Errorf("Args.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExec(t *testing.T) {
	// Save and restore PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Test 1: aoe not found when PATH is empty
	t.Run("aoe_not_found_returns_error", func(t *testing.T) {
		os.Setenv("PATH", "")

		args := &Args{
			Path:  ".",
			Title: "myproject",
			Cmd:   "codex",
		}

		err := Exec(args)
		if err == nil {
			t.Fatal("Exec() should return error when aoe is not found")
		}

		if !strings.Contains(err.Error(), "aoe") {
			t.Errorf("Exec() error = %q, should mention 'aoe'", err.Error())
		}
	})

	// Test 2: aoe found but exits with error
	t.Run("aoe_exit_error", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a fake "aoe" that exits with code 1
		fakeAoe := filepath.Join(tmpDir, "aoe")

		script := `#!/bin/sh
exit 1`

		if err := os.WriteFile(fakeAoe, []byte(script), 0755); err != nil {
			t.Fatalf("failed to create fake aoe: %v", err)
		}

		os.Setenv("PATH", tmpDir)

		args := &Args{
			Path:  ".",
			Title: "myproject",
			Cmd:   "codex",
		}

		err := Exec(args)
		if err == nil {
			t.Fatal("Exec() should return error when aoe exits with code 1")
		}
		// Should contain exit code info
		errStr := err.Error()
		if !strings.Contains(errStr, "exited with code 1") {
			t.Errorf("Exec() error = %q, should contain 'exited with code 1'", errStr)
		}
	})

	// Test 3: aoe exits successfully
	t.Run("aoe_success", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a fake "aoe" that exits with code 0
		fakeAoe := filepath.Join(tmpDir, "aoe")

		script := `#!/bin/sh
exit 0`

		if err := os.WriteFile(fakeAoe, []byte(script), 0755); err != nil {
			t.Fatalf("failed to create fake aoe: %v", err)
		}

		os.Setenv("PATH", tmpDir)

		args := &Args{
			Path:  ".",
			Title: "myproject",
			Cmd:   "codex",
		}

		err := Exec(args)
		if err != nil {
			t.Fatalf("Exec() unexpected error = %v", err)
		}
	})

	// Test 4: verify command arguments passed correctly
	t.Run("args_passed_correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a fake "aoe" that dumps its args to a file
		markerFile := filepath.Join(tmpDir, "marker")
		fakeAoe := filepath.Join(tmpDir, "aoe")

		script := fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
exit 0`, markerFile)

		if err := os.WriteFile(fakeAoe, []byte(script), 0755); err != nil {
			t.Fatalf("failed to create fake aoe: %v", err)
		}

		os.Setenv("PATH", tmpDir)

		args := &Args{
			Path:      ".",
			Title:     "myproject",
			Cmd:       "codex",
			Launch:    true,
			Worktree:  "feature/hello",
			Sandbox:   true,
			NewBranch: true,
		}
		if err := Exec(args); err != nil {
			t.Fatalf("Exec() unexpected error = %v", err)
		}

		// Read what arguments were passed
		gotArgs, err := os.ReadFile(markerFile)
		if err != nil {
			t.Fatalf("failed to read marker file: %v", err)
		}

		// Verify all expected args are in the output
		for _, expected := range []string{"add", ".", "--title", "myproject", "--cmd", "codex", "--launch", "--worktree", "feature/hello", "-b", "--sandbox"} {
			if !strings.Contains(string(gotArgs), expected) {
				t.Errorf("command args missing %q in output: %q", expected, string(gotArgs))
			}
		}
	})

	// Test 5: custom path is passed correctly
	t.Run("custom_path", func(t *testing.T) {
		tmpDir := t.TempDir()
		markerFile := filepath.Join(tmpDir, "marker")
		fakeAoe := filepath.Join(tmpDir, "aoe")

		script := fmt.Sprintf(`#!/bin/sh
echo "$@" > %q
exit 0`, markerFile)

		if err := os.WriteFile(fakeAoe, []byte(script), 0755); err != nil {
			t.Fatalf("failed to create fake aoe: %v", err)
		}

		os.Setenv("PATH", tmpDir)

		args := &Args{
			Path:  "/home/user/myproject",
			Title: "myproject",
			Cmd:   "codex",
		}
		if err := Exec(args); err != nil {
			t.Fatalf("Exec() unexpected error = %v", err)
		}

		gotArgs, err := os.ReadFile(markerFile)
		if err != nil {
			t.Fatalf("failed to read marker file: %v", err)
		}

		if !strings.Contains(string(gotArgs), "/home/user/myproject") {
			t.Errorf("custom path not found in args: %q", string(gotArgs))
		}
	})
}

func TestExecWithKillSignal(t *testing.T) {
	// Test that kill signal is handled properly
	tmpDir := t.TempDir()
	fakeAoe := filepath.Join(tmpDir, "aoe")
	// Create a script that kills itself with SIGTERM
	script := `#!/bin/sh
kill $$`
	if err := os.WriteFile(fakeAoe, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake aoe: %v", err)
	}

	os.Setenv("PATH", tmpDir)

	args := &Args{
		Path:  ".",
		Title: "myproject",
		Cmd:   "codex",
	}

	err := Exec(args)
	if err == nil {
		t.Fatal("Exec() should return error when process is killed")
	}
	// Error should mention the exit
	if !strings.Contains(err.Error(), "aoe") {
		t.Errorf("Exec() error = %q, want error mentioning 'aoe'", err.Error())
	}
}
