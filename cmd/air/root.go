package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "air",
	Short: "AI Runner - orchestrate concurrent Claude Code agents",
	Long:  `AIR orchestrates multiple Claude Code agents working in parallel on decomposed tasks using git worktrees.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("air v%s\n", version)
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
