package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version info - set via ldflags at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "air",
	Short: "AI Runner - orchestrate concurrent Claude Code agents",
	Long:  `Air orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("air v%s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	// Disable alphabetical sorting to show commands in workflow order
	cobra.EnableCommandSorting = false

	// Hide the auto-generated completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	// Add commands in workflow order
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(cleanCmd)

	// Utility commands
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(versionCmd)

	// Agent commands (used during execution, not by users)
	rootCmd.AddCommand(agentCmd)
}
