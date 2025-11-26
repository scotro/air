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
	// Check .air/ exists
	if _, err := os.Stat(".air"); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Check if .gitignore has uncommitted changes containing .air/
	// Worktrees are created from committed state, so uncommitted .gitignore
	// means agents will see .air/ as untracked files
	if err := checkGitignoreCommitted(); err != nil {
		return err
	}

	plansDir := filepath.Join(".air", "plans")

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

	// Read context once from main repo
	contextContent, err := os.ReadFile(filepath.Join(".air", "context.md"))
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Create worktrees directory
	worktreesDir := filepath.Join(".air", "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Permission flag for claude
	permFlag := ""
	if !noAutoAccept {
		permFlag = "--permission-mode acceptEdits"
	}

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
			createCmd.Stdout = os.Stdout
			createCmd.Stderr = os.Stderr
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create worktree for %s: %w", name, err)
			}
			fmt.Printf("Created worktree: %s (branch: %s)\n", wtPath, branch)
		}

		// Read plan content from main repo
		planContent, err := os.ReadFile(filepath.Join(".air", "plans", name+".md"))
		if err != nil {
			return fmt.Errorf("failed to read plan %s: %w", name, err)
		}

		// Build the assignment prompt
		assignment := fmt.Sprintf("Your assignment:\n\n%s\n\nImplement this.", string(planContent))

		// Write content files to worktree (avoids shell escaping issues)
		wtAirDir := filepath.Join(wtPath, ".air")
		os.MkdirAll(wtAirDir, 0755)

		if err := os.WriteFile(filepath.Join(wtAirDir, ".context"), contextContent, 0644); err != nil {
			return fmt.Errorf("failed to write context for %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(wtAirDir, ".assignment"), []byte(assignment), 0644); err != nil {
			return fmt.Errorf("failed to write assignment for %s: %w", name, err)
		}

		// Generate launcher script that reads from files
		launcherScript := fmt.Sprintf("#!/bin/bash\nexec claude %s --append-system-prompt \"$(cat .air/.context)\" \"$(cat .air/.assignment)\"\n", permFlag)

		scriptPath := filepath.Join(wtAirDir, "launch.sh")
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
	firstWtPath := filepath.Join(projectRoot, ".air", "worktrees", firstPlan)

	// Create session
	tmuxNew := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", firstPlan, "-c", firstWtPath)
	if err := tmuxNew.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Run launcher script for first plan
	exec.Command("tmux", "send-keys", "-t", sessionName+":"+firstPlan, ".air/launch.sh", "Enter").Run()

	// Create windows for remaining plans
	for _, name := range plans[1:] {
		wtPath := filepath.Join(projectRoot, ".air", "worktrees", name)

		// Create window
		exec.Command("tmux", "new-window", "-t", sessionName, "-n", name, "-c", wtPath).Run()

		// Run launcher script
		exec.Command("tmux", "send-keys", "-t", sessionName+":"+name, ".air/launch.sh", "Enter").Run()
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

func checkGitignoreCommitted() error {
	// Check if .gitignore has uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain", ".gitignore")
	output, err := statusCmd.Output()
	if err != nil {
		// If git status fails, skip this check
		return nil
	}

	status := strings.TrimSpace(string(output))
	if status == "" {
		// .gitignore is clean
		return nil
	}

	// Check if the uncommitted .gitignore contains .air/
	content, err := os.ReadFile(".gitignore")
	if err != nil {
		return nil
	}

	if strings.Contains(string(content), ".air/") {
		return fmt.Errorf(".gitignore has uncommitted changes containing '.air/'\n\nWorktrees are created from committed state, so agents will see .air/ as untracked files.\nCommit .gitignore first:\n  git add .gitignore && git commit -m \"Add .air/ to gitignore\"")
	}

	return nil
}
