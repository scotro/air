package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [plans...]",
	Short: "Create worktrees and launch agents",
	Long: `Creates git worktrees for each plan and launches Claude agents in a tmux session.

Use 'air run all' to run all plans, or specify plan names.
With no arguments, shows available plans.`,
	RunE: runRun,
}

var noAutoAccept bool

func init() {
	runCmd.Flags().BoolVar(&noAutoAccept, "no-auto-accept", false, "Disable auto-accept mode (require permission for edits)")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	plansDir := getPlansDir()

	// Get available plans
	available, err := getAvailablePlans(plansDir)
	if err != nil {
		return err
	}

	if len(available) == 0 {
		fmt.Println("No plans found. Run 'air plan' to create some.")
		return nil
	}

	// No args: show available plans
	if len(args) == 0 {
		fmt.Println("Available plans:")
		for _, p := range available {
			fmt.Printf("  %s\n", p)
		}
		fmt.Println("\nUsage: air run <plan1> [plan2] ...")
		fmt.Println("       air run all")
		return nil
	}

	// Handle 'all'
	var plans []string
	if len(args) == 1 && args[0] == "all" {
		plans = available
	} else {
		// Validate plan names
		for _, name := range args {
			if !contains(available, name) {
				return fmt.Errorf("plan '%s' not found", name)
			}
		}
		plans = args
	}

	// Get the absolute path of the project root
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Read context once
	contextContent, err := os.ReadFile(getContextPath())
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Get paths
	worktreesDir := getWorktreesDir()
	agentsDir := getAgentsDir()
	channelsDir := getChannelsDir()

	// Create directories
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}
	if err := os.MkdirAll(channelsDir, 0755); err != nil {
		return fmt.Errorf("failed to create channels directory: %w", err)
	}

	// Permission and allowed tools flags for claude
	permFlag := ""
	if !noAutoAccept {
		permFlag = "--permission-mode acceptEdits"
	}

	// Language-agnostic allowed tools: air commands, read-only git, info gathering
	allowedTools := `--allowedTools "Bash(air:*) Bash(git status:*) Bash(git log:*) Bash(git diff:*) Bash(git branch:*) Bash(ls:*) Bash(find:*) Bash(cat:*) Bash(head:*) Bash(tail:*) Bash(wc:*)"`

	// Create worktrees for each plan
	for _, name := range plans {
		wtPath := filepath.Join(worktreesDir, name)
		branch := "air/" + name

		// Check if worktree already exists
		if _, err := os.Stat(wtPath); err == nil {
			fmt.Printf("Worktree %s already exists\n", name)
		} else {
			// Create worktree
			createCmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch)
			createCmd.Dir = projectRoot
			createCmd.Stdout = os.Stdout
			createCmd.Stderr = os.Stderr
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create worktree for %s: %w", name, err)
			}
			fmt.Printf("Created worktree: %s (branch: %s)\n", wtPath, branch)
		}

		// Read plan content
		planContent, err := os.ReadFile(filepath.Join(plansDir, name+".md"))
		if err != nil {
			return fmt.Errorf("failed to read plan %s: %w", name, err)
		}

		// Build the assignment prompt
		assignment := fmt.Sprintf("Your assignment:\n\n%s\n\nImplement this.", string(planContent))

		// Create agent data directory
		agentDir := filepath.Join(agentsDir, name)
		os.MkdirAll(agentDir, 0755)

		// Write context and assignment files
		if err := os.WriteFile(filepath.Join(agentDir, "context"), contextContent, 0644); err != nil {
			return fmt.Errorf("failed to write context for %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(agentDir, "assignment"), []byte(assignment), 0644); err != nil {
			return fmt.Errorf("failed to write assignment for %s: %w", name, err)
		}

		// Generate launcher script
		launcherScript := fmt.Sprintf(`#!/bin/bash
export AIR_AGENT_ID="%s"
export AIR_WORKTREE="%s"
export AIR_PROJECT_ROOT="%s"
export AIR_CHANNELS_DIR="%s"
cd "$AIR_WORKTREE"
exec claude %s %s --append-system-prompt "$(cat %s/context)" "$(cat %s/assignment)"
`, name, wtPath, projectRoot, channelsDir, permFlag, allowedTools, agentDir, agentDir)

		scriptPath := filepath.Join(agentDir, "launch.sh")
		if err := os.WriteFile(scriptPath, []byte(launcherScript), 0755); err != nil {
			return fmt.Errorf("failed to write launcher script for %s: %w", name, err)
		}
	}

	// Start tmux session
	sessionName := "air"

	// Kill existing session if present
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create new session with first plan
	firstPlan := plans[0]
	firstWtPath := filepath.Join(worktreesDir, firstPlan)
	firstAgentDir := filepath.Join(agentsDir, firstPlan)

	// Create session
	tmuxNew := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", firstPlan, "-c", firstWtPath)
	if err := tmuxNew.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Run launcher script for first plan
	exec.Command("tmux", "send-keys", "-t", sessionName+":"+firstPlan, firstAgentDir+"/launch.sh", "Enter").Run()

	// Create windows for remaining plans
	for _, name := range plans[1:] {
		wtPath := filepath.Join(worktreesDir, name)
		agentDir := filepath.Join(agentsDir, name)

		// Create window
		exec.Command("tmux", "new-window", "-t", sessionName, "-n", name, "-c", wtPath).Run()

		// Run launcher script
		exec.Command("tmux", "send-keys", "-t", sessionName+":"+name, agentDir+"/launch.sh", "Enter").Run()
	}

	// Create dashboard window (before the agent windows so agents are more prominent)
	exec.Command("tmux", "new-window", "-t", sessionName, "-n", "dash", "-c", projectRoot).Run()

	// Select first agent window
	exec.Command("tmux", "select-window", "-t", sessionName+":"+firstPlan).Run()

	fmt.Printf("\nLaunched %d agents in tmux session '%s'\n", len(plans), sessionName)
	fmt.Println("Attach with: tmux attach -t", sessionName)

	// Attach to session
	attachCmd := exec.Command("tmux", "attach", "-t", sessionName)
	attachCmd.Stdin = os.Stdin
	attachCmd.Stdout = os.Stdout
	attachCmd.Stderr = os.Stderr
	return attachCmd.Run()
}

func getAvailablePlans(plansDir string) ([]string, error) {
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plans: %w", err)
	}

	var plans []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			name := strings.TrimSuffix(entry.Name(), ".md")
			plans = append(plans, name)
		}
	}
	return plans, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
