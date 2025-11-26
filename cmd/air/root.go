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
	Long:  `AIR orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("air v%s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(versionCmd)
}
