package main

import (
	"fmt"
	"os"

	"github.com/scotro/air/cmd/air/prompts"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project for Air workflow",
	Long: `Creates ~/.air/<project>/ directory with context and plans subdirectories.

Supports two modes:
  - Single-repo mode: Run in a git repository
  - Workspace mode: Run in a directory containing multiple git repos`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Detect mode based on directory structure
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("cannot initialize Air here: %w", err)
	}

	// Get air directory path
	airDir, err := info.getAirDirForWorkspace()
	if err != nil {
		return fmt.Errorf("failed to determine air directory: %w", err)
	}

	// Check for collision (directory already exists for different project)
	if _, err := os.Stat(airDir); err == nil {
		fmt.Printf("Air directory already exists: %s\n", airDir)
	}

	// Create directories
	plansDir := getPlansDir()
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	// Create context.md with appropriate template
	contextPath := getContextPath()
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		template := prompts.AgentContext
		if info.Mode == ModeWorkspace {
			template = prompts.AgentContextWorkspace
		}
		if err := os.WriteFile(contextPath, []byte(template), 0644); err != nil {
			return fmt.Errorf("failed to create context.md: %w", err)
		}
		fmt.Printf("Created %s\n", contextPath)
	} else {
		fmt.Printf("context.md already exists at %s\n", contextPath)
	}

	// Print initialization summary
	if info.Mode == ModeWorkspace {
		fmt.Printf("\nInitialized Air workspace '%s' with %d repositories:\n", info.Name, len(info.Repos))
		for _, repo := range info.Repos {
			fmt.Printf("  - %s\n", repo)
		}
	} else {
		fmt.Printf("\nInitialized Air workflow for '%s'.\n", info.Name)
	}

	fmt.Printf("Air directory: %s\n", airDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  air plan              # Start planning session")
	fmt.Println("  air plan list         # View plans")
	if info.Mode == ModeWorkspace {
		fmt.Println("  air run               # Launch agents across repos")
	} else {
		fmt.Println("  air run <names...>    # Launch agents")
	}

	return nil
}
