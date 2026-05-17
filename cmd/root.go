package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aoex",
	Short: "agent-of-empires CLI extension",
	Long:  "agent-of-empires CLI extension — an interactive wrapper for the aoe tool",
}

// Execute runs the root command.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
