package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/scotro/air/cmd/air/prompts"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Start orchestration session to create plans",
	Long:  `Launches Claude with orchestration context to help decompose work into plans.`,
	RunE:  runPlan,
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all plans",
	RunE:  runPlanList,
}

var planShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a specific plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanShow,
}

var planArchiveCmd = &cobra.Command{
	Use:   "archive <name>",
	Short: "Archive a plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanArchive,
}

var planRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore an archived plan",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanRestore,
}

var listArchived bool

func init() {
	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planArchiveCmd)
	planCmd.AddCommand(planRestoreCmd)
	planListCmd.Flags().BoolVar(&listArchived, "archived", false, "Show archived plans")
}

func runPlan(cmd *cobra.Command, args []string) error {
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Detect mode
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
	}

	// Check for existing state
	worktrees := getExistingWorktrees()
	plans := getExistingPlans()

	// Case 1: Worktrees exist - work is in progress
	if len(worktrees) > 0 {
		fmt.Println("Work is already in progress from a previous session.")
		fmt.Println("\nTo continue: use `air status` or `tmux attach -t air`")
		fmt.Println("To start fresh: run `air clean` first")
		return nil
	}

	// Case 2: Plans exist but no worktrees - offer to extend or start fresh
	if len(plans) > 0 {
		fmt.Println("Found existing plans:")
		plansDir := getPlansDir()
		for _, name := range plans {
			// Read objective from plan
			content, _ := os.ReadFile(filepath.Join(plansDir, name+".md"))
			lines := strings.Split(string(content), "\n")
			objective := ""
			for _, line := range lines {
				if strings.HasPrefix(line, "**Objective:**") {
					objective = strings.TrimPrefix(line, "**Objective:**")
					objective = strings.TrimSpace(objective)
					break
				}
			}
			fmt.Printf("  %-15s %s\n", name, objective)
		}

		fmt.Println("\nAre you:")
		fmt.Println("  [e] Extending/modifying these plans")
		fmt.Println("  [c] Starting fresh")
		fmt.Print("\nChoice [e/c]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "c" {
			fmt.Println("Cleaning up...")
			err := cleanWorkspace(plans, cleanOptions{
				deleteBranches: true,
				deletePlans:    true,
				quiet:          true,
				cleanAll:       true,
			})
			if err != nil {
				return fmt.Errorf("failed to clean workspace: %w", err)
			}
			fmt.Println("Done.")
		} else if response != "e" {
			fmt.Println("Cancelled.")
			return nil
		}
		// If "e", just proceed with orchestration
	}

	// Read context
	context, err := os.ReadFile(getContextPath())
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Build orchestration prompt based on mode
	var orchestrationPrompt string
	if info.Mode == ModeWorkspace {
		repoContext := buildWorkspaceRepoContext(info)
		orchestrationPrompt = string(context) + "\n\n" + repoContext + "\n\n" + prompts.OrchestrationWorkspace
	} else {
		orchestrationPrompt = string(context) + "\n\n" + prompts.Orchestration
	}

	// Launch claude with initial prompt
	initialPrompt := "Begin orchestration. Ask me what I want to build."
	if info.Mode == ModeWorkspace {
		initialPrompt = fmt.Sprintf("Begin orchestration for workspace '%s' with %d repositories. Ask me what I want to build.", info.Name, len(info.Repos))
	}

	claudeCmd := exec.Command("claude",
		"--allowedTools", "Bash(air plan:*)",
		"--append-system-prompt", orchestrationPrompt,
		initialPrompt)
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

// buildWorkspaceRepoContext builds context about each repo in the workspace
func buildWorkspaceRepoContext(info *WorkspaceInfo) string {
	var sb strings.Builder
	sb.WriteString("## Workspace Repositories\n\n")
	sb.WriteString(fmt.Sprintf("This is a multi-repo workspace '%s' containing %d repositories:\n\n", info.Name, len(info.Repos)))

	for _, repo := range info.Repos {
		repoPath := filepath.Join(info.Root, repo)
		sb.WriteString(fmt.Sprintf("### %s\n\n", repo))

		// Try to read CLAUDE.md
		claudeMdPath := filepath.Join(repoPath, "CLAUDE.md")
		if content, err := os.ReadFile(claudeMdPath); err == nil {
			sb.WriteString("**CLAUDE.md:**\n```\n")
			// Truncate if too long
			text := string(content)
			if len(text) > 2000 {
				text = text[:2000] + "\n...(truncated)"
			}
			sb.WriteString(text)
			sb.WriteString("\n```\n\n")
		}

		// Try to read README.md if no CLAUDE.md
		if _, err := os.Stat(claudeMdPath); os.IsNotExist(err) {
			readmePath := filepath.Join(repoPath, "README.md")
			if content, err := os.ReadFile(readmePath); err == nil {
				sb.WriteString("**README.md:**\n```\n")
				text := string(content)
				if len(text) > 1000 {
					text = text[:1000] + "\n...(truncated)"
				}
				sb.WriteString(text)
				sb.WriteString("\n```\n\n")
			}
		}

		// Detect project type
		projectType := detectProjectType(repoPath)
		if projectType != "" {
			sb.WriteString(fmt.Sprintf("**Project type:** %s\n\n", projectType))
		}
	}

	return sb.String()
}

// detectProjectType tries to identify the project type from files
func detectProjectType(repoPath string) string {
	types := []struct {
		file string
		name string
	}{
		{"go.mod", "Go"},
		{"package.json", "Node.js/TypeScript"},
		{"Cargo.toml", "Rust"},
		{"pyproject.toml", "Python"},
		{"requirements.txt", "Python"},
		{"pom.xml", "Java/Maven"},
		{"build.gradle", "Java/Gradle"},
	}

	for _, t := range types {
		if _, err := os.Stat(filepath.Join(repoPath, t.file)); err == nil {
			return t.name
		}
	}
	return ""
}

func runPlanList(cmd *cobra.Command, args []string) error {
	var plansDir string
	var label string

	basePlansDir := getPlansDir()
	if listArchived {
		plansDir = filepath.Join(basePlansDir, "archive")
		label = "Archived Plans:"
	} else {
		plansDir = basePlansDir
		label = "Plans:"
	}

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			if listArchived {
				fmt.Println("No archived plans.")
			} else {
				fmt.Println("No plans yet. Run 'air plan' to create some.")
			}
			return nil
		}
		return fmt.Errorf("failed to read plans: %w", err)
	}

	// Filter to only .md files (exclude archive directory)
	var plans []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			plans = append(plans, entry)
		}
	}

	if len(plans) == 0 {
		if listArchived {
			fmt.Println("No archived plans.")
		} else {
			fmt.Println("No plans yet. Run 'air plan' to create some.")
		}
		return nil
	}

	fmt.Println(label)
	for _, entry := range plans {
		name := strings.TrimSuffix(entry.Name(), ".md")

		// Read first line for objective
		content, _ := os.ReadFile(filepath.Join(plansDir, entry.Name()))
		lines := strings.Split(string(content), "\n")
		objective := ""
		for _, line := range lines {
			if strings.HasPrefix(line, "**Objective:**") {
				objective = strings.TrimPrefix(line, "**Objective:**")
				objective = strings.TrimSpace(objective)
				break
			}
		}

		fmt.Printf("  %-15s %s\n", name, objective)
	}

	return nil
}

func runPlanShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	planPath := filepath.Join(getPlansDir(), name+".md")

	content, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("plan '%s' not found", name)
		}
		return fmt.Errorf("failed to read plan: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

func runPlanArchive(cmd *cobra.Command, args []string) error {
	name := args[0]
	plansDir := getPlansDir()
	srcPath := filepath.Join(plansDir, name+".md")
	archiveDir := filepath.Join(plansDir, "archive")
	dstPath := filepath.Join(archiveDir, name+".md")

	// Check source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("plan '%s' not found", name)
	}

	// Create archive directory
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to archive plan: %w", err)
	}

	fmt.Printf("Archived: %s\n", name)
	return nil
}

func runPlanRestore(cmd *cobra.Command, args []string) error {
	name := args[0]
	plansDir := getPlansDir()
	srcPath := filepath.Join(plansDir, "archive", name+".md")
	dstPath := filepath.Join(plansDir, name+".md")

	// Check source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("archived plan '%s' not found", name)
	}

	// Check destination doesn't exist
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("plan '%s' already exists (not archived)", name)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to restore plan: %w", err)
	}

	fmt.Printf("Restored: %s\n", name)
	return nil
}
