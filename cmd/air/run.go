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
var dryRun bool

func init() {
	runCmd.Flags().BoolVar(&noAutoAccept, "no-auto-accept", false, "Disable auto-accept mode (require permission for edits)")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate plans and show what would run, without launching")
}

func runRun(cmd *cobra.Command, args []string) error {
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Detect mode
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
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
	var planNames []string
	if len(args) == 1 && args[0] == "all" {
		planNames = available
	} else {
		// Validate plan names
		for _, name := range args {
			if !contains(available, name) {
				return fmt.Errorf("plan '%s' not found", name)
			}
		}
		planNames = args
	}

	// Validate dependency graph before launching (with mode awareness)
	planDeps, validationErrs := ValidatePlansWithMode(info)
	if len(validationErrs) > 0 {
		fmt.Println("Dependency validation failed:")
		for _, err := range validationErrs {
			fmt.Printf("  ✗ %s\n", err)
		}
		fmt.Println("\nRun 'air plan validate' for details, or fix plans before running.")
		return fmt.Errorf("invalid dependency graph")
	}

	// Build a map of plan name -> PlanDependencies for repo lookup
	planInfoMap := make(map[string]PlanDependencies)
	for _, pd := range planDeps {
		planInfoMap[pd.Name] = pd
	}

	// Dry run: show what would happen and exit
	if dryRun {
		fmt.Println("Validation passed. Would launch agents for:")
		for _, name := range planNames {
			pd := planInfoMap[name]
			if info.Mode == ModeWorkspace && pd.Repository != "" {
				fmt.Printf("  %s [repo: %s] (branch: air/%s)\n", name, pd.Repository, name)
			} else {
				fmt.Printf("  %s (branch: air/%s)\n", name, name)
			}
		}
		fmt.Printf("\nRun without --dry-run to launch %d agents.\n", len(planNames))
		return nil
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
	allowedTools := `--allowedTools "Bash(air:*) Bash(git status:*) Bash(git log:*) Bash(git diff:*) Bash(git branch:*) Bash(git merge-tree:*) Bash(mkdir:*) Bash(ls:*) Bash(find:*) Bash(cat:*) Bash(head:*) Bash(tail:*) Bash(wc:*)"`

	// Settings: disable co-authored-by to keep commits clean
	settings := `--settings '{"includeCoAuthoredBy": false}'`

	// Track worktree paths for tmux
	type agentInfo struct {
		name       string
		wtPath     string
		agentDir   string
		repoName   string
		repoPath   string
	}
	var agents []agentInfo

	// Create worktrees for each plan
	for _, name := range planNames {
		pd := planInfoMap[name]

		// Determine target repo and paths based on mode
		var repoName, repoPath, wtPath string
		if info.Mode == ModeWorkspace {
			repoName = pd.Repository
			repoPath = filepath.Join(info.Root, repoName)
			// In workspace mode: worktrees/<repo>/<plan>
			repoWorktreeDir := filepath.Join(worktreesDir, repoName)
			os.MkdirAll(repoWorktreeDir, 0755)
			wtPath = filepath.Join(repoWorktreeDir, name)
		} else {
			repoName = ""
			repoPath = info.Root
			// In single mode: worktrees/<plan>
			wtPath = filepath.Join(worktreesDir, name)
		}

		branch := "air/" + name

		// Check if worktree already exists
		if _, err := os.Stat(wtPath); err == nil {
			fmt.Printf("Worktree %s already exists\n", name)
		} else {
			// Create worktree in the target repo
			createCmd := exec.Command("git", "worktree", "add", wtPath, "-b", branch)
			createCmd.Dir = repoPath
			createCmd.Stdout = os.Stdout
			createCmd.Stderr = os.Stderr
			if err := createCmd.Run(); err != nil {
				return fmt.Errorf("failed to create worktree for %s: %w", name, err)
			}
			if info.Mode == ModeWorkspace {
				fmt.Printf("Created worktree: %s [repo: %s] (branch: %s)\n", name, repoName, branch)
			} else {
				fmt.Printf("Created worktree: %s (branch: %s)\n", wtPath, branch)
			}
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

		// Generate launcher script with workspace-aware environment variables
		sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
		sshExport := ""
		if sshAuthSock != "" {
			sshExport = fmt.Sprintf("export SSH_AUTH_SOCK=\"%s\"\n", sshAuthSock)
		}

		// Workspace-specific env vars
		workspaceEnv := ""
		if info.Mode == ModeWorkspace {
			workspaceEnv = fmt.Sprintf(`export AIR_REPO="%s"
export AIR_WORKSPACE="%s"
export AIR_WORKSPACE_ROOT="%s"
`, repoName, info.Name, info.Root)
		}

		launcherScript := fmt.Sprintf(`#!/bin/bash
%s%sexport AIR_AGENT_ID="%s"
export AIR_WORKTREE="%s"
export AIR_PROJECT_ROOT="%s"
export AIR_CHANNELS_DIR="%s"
cd "$AIR_WORKTREE"
exec claude %s %s %s --append-system-prompt "$(cat %s/context)" "$(cat %s/assignment)"
`, sshExport, workspaceEnv, name, wtPath, repoPath, channelsDir, permFlag, allowedTools, settings, agentDir, agentDir)

		scriptPath := filepath.Join(agentDir, "launch.sh")
		if err := os.WriteFile(scriptPath, []byte(launcherScript), 0755); err != nil {
			return fmt.Errorf("failed to write launcher script for %s: %w", name, err)
		}

		agents = append(agents, agentInfo{
			name:     name,
			wtPath:   wtPath,
			agentDir: agentDir,
			repoName: repoName,
			repoPath: repoPath,
		})
	}

	// Start tmux session
	sessionName := "air"

	// Kill existing session if present
	exec.Command("tmux", "kill-session", "-t", sessionName).Run()

	// Create new session with first agent
	firstAgent := agents[0]

	// Create session
	tmuxNew := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-n", firstAgent.name, "-c", firstAgent.wtPath)
	if err := tmuxNew.Run(); err != nil {
		return fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Run launcher script for first agent
	exec.Command("tmux", "send-keys", "-t", sessionName+":"+firstAgent.name, firstAgent.agentDir+"/launch.sh", "Enter").Run()

	// Create windows for remaining agents
	for _, agent := range agents[1:] {
		// Create window
		exec.Command("tmux", "new-window", "-t", sessionName, "-n", agent.name, "-c", agent.wtPath).Run()

		// Run launcher script
		exec.Command("tmux", "send-keys", "-t", sessionName+":"+agent.name, agent.agentDir+"/launch.sh", "Enter").Run()
	}

	// Create dashboard window
	dashDir := info.Root
	exec.Command("tmux", "new-window", "-t", sessionName, "-n", "dash", "-c", dashDir).Run()

	// Select first agent window
	exec.Command("tmux", "select-window", "-t", sessionName+":"+firstAgent.name).Run()

	fmt.Printf("\nLaunched %d agents in tmux session '%s'\n", len(agents), sessionName)
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
