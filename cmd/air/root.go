package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "air",
	Short: "AI Runner - orchestrate concurrent Claude Code agents",
	Long:  `AIR orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.`,
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(cleanCmd)
}
