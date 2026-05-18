package executor

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Args holds the parsed arguments for the aoe add command.
type Args struct {
	Path      string // positional arg, always "." for now
	Title     string // --title
	Cmd       string // --cmd
	Launch    bool   // --launch
	Worktree  string // --worktree (omitted if empty; empty when attaching to existing worktree)
	Sandbox   bool   // --sandbox
	NewBranch bool   // -b (create new branch)
}

// Exec builds and runs the `aoe add` command with the provided arguments.
func Exec(args *Args) error {
	cmdArgs := []string{"add"}

	// When attaching to an existing worktree, pass its path directly and leave Worktree empty.
	// When creating new worktree from existing branch, pass --worktree with branch name (no -b).
	// When creating new branch with worktree, pass --worktree with -b.
	// Otherwise use the current directory (worktree creation is handled by aoe).
	if args.Path != "" && args.Path != "." {
		cmdArgs = append(cmdArgs, args.Path)
	} else {
		cmdArgs = append(cmdArgs, ".")
	}

	cmdArgs = append(cmdArgs, "--title", args.Title)
	cmdArgs = append(cmdArgs, "--cmd", args.Cmd)

	if args.Launch {
		cmdArgs = append(cmdArgs, "--launch")
	}

	if args.Worktree != "" {
		cmdArgs = append(cmdArgs, "--worktree", args.Worktree)
	}

	if args.NewBranch {
		cmdArgs = append(cmdArgs, "-b")
	}

	if args.Sandbox {
		cmdArgs = append(cmdArgs, "--sandbox")
	}

	cmd := exec.Command("aoe", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			stderr := strings.TrimSpace(string(exitErr.Stderr))
			if stderr != "" {
				return fmt.Errorf("aoe exited with code %d: %s", exitErr.ExitCode(), stderr)
			}

			return fmt.Errorf("aoe exited with code %d", exitErr.ExitCode())
		}

		return fmt.Errorf("failed to run aoe: %w", err)
	}

	return nil
}

// String returns the shell-like command string for review display.
func (a *Args) String() string {
	var b strings.Builder

	b.WriteString("aoe add ")

	if a.Path != "" && a.Path != "." {
		b.WriteString(strconv.Quote(a.Path))
		b.WriteByte(' ')
	} else {
		b.WriteString(". ")
	}

	b.WriteString("--title ")
	b.WriteString(strconv.Quote(a.Title))
	b.WriteString(" --cmd ")
	b.WriteString(strconv.Quote(a.Cmd))

	if a.Launch {
		b.WriteString(" --launch")
	}

	if a.Worktree != "" {
		b.WriteString(" --worktree ")
		b.WriteString(strconv.Quote(a.Worktree))
	}

	if a.NewBranch {
		b.WriteString(" -b")
	}

	if a.Sandbox {
		b.WriteString(" --sandbox")
	}

	return b.String()
}
