package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ball6847/aoex/internal/detector"
	"github.com/ball6847/aoex/internal/executor"
	"github.com/ball6847/aoex/internal/prompts"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Interactively add a new agent session",
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Pre-flight: verify aoe is on PATH before starting any prompts.
	if _, err := exec.LookPath("aoe"); err != nil {
		return fmt.Errorf("aoe binary not found in PATH. Please install agent-of-empires first")
	}

	ctx, err := detector.DetectAll()
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}

	execArgs, err := prompts.RunWizard(ctx)
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}

	// If user cancelled (nil args), exit cleanly.
	if execArgs == nil {
		fmt.Fprintln(os.Stderr, "Cancelled. No command executed.")
		return nil
	}

	if err := executor.Exec(execArgs); err != nil {
		return fmt.Errorf("add: %w", err)
	}

	return nil
}
