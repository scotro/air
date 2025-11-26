package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	// Check .air/ exists
	if _, err := os.Stat(".air"); os.IsNotExist(err) {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Read context
	contextPath := filepath.Join(".air", "context.md")
	context, err := os.ReadFile(contextPath)
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Build orchestration prompt
	orchestrationPrompt := string(context) + "\n\n" + orchestrationContext

	// Launch claude with initial prompt
	claudeCmd := exec.Command("claude",
		"--append-system-prompt", orchestrationPrompt,
		"Begin orchestration. Ask me what I want to build.")
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

func runPlanList(cmd *cobra.Command, args []string) error {
	var plansDir string
	var label string

	if listArchived {
		plansDir = filepath.Join(".air", "plans", "archive")
		label = "Archived Plans:"
	} else {
		plansDir = filepath.Join(".air", "plans")
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
	planPath := filepath.Join(".air", "plans", name+".md")

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
	srcPath := filepath.Join(".air", "plans", name+".md")
	archiveDir := filepath.Join(".air", "plans", "archive")
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
	srcPath := filepath.Join(".air", "plans", "archive", name+".md")
	dstPath := filepath.Join(".air", "plans", name+".md")

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

const orchestrationContext = `## Orchestration Mode

You are helping plan work for multiple AI agents that will run in parallel. Each agent works in an isolated git worktree on a specific task.

### Your Job

1. **Understand what the user wants to build** - Ask clarifying questions if needed. Understand scope, constraints, and what "done" looks like.

2. **Decompose into parallel work streams** - Identify 2-4 tasks that can run simultaneously with minimal dependencies. Good decomposition:
   - Clear boundaries (each agent knows exactly which files to touch)
   - Minimal overlap (agents won't create merge conflicts)
   - Testable independently (each task has clear acceptance criteria)

   **New projects:** If the project lacks foundational setup (e.g., no main.go, no package structure), identify this as a prerequisite. Either:
   - Tell the user to create minimal boilerplate first, OR
   - Create a single "setup" plan that must run before the parallel plans

3. **Create plans** - Write plan files to ` + "`" + `.air/plans/<name>.md` + "`" + ` for each task.

4. **Provide launch command** - Tell the user exactly how to start the agents.

### Start by asking:

"What would you like to build? Describe the feature, task, or goal - I'll help break it down into parallel work streams for multiple agents."

### Plan format:

` + "```" + `markdown
# Plan: <name>

**Objective:** [One sentence describing what "done" looks like]

## Boundaries

**In scope:**
- [files/directories this agent should touch]

**Out of scope:**
- [what this agent should NOT modify]

## Acceptance Criteria

- [ ] [Specific, verifiable condition]
- [ ] Tests pass
- [ ] No lint errors

## Notes

[Any additional context]
` + "```" + `

### After planning

1. Use the Write tool to create each plan file in ` + "`" + `.air/plans/<name>.md` + "`" + `
2. Summarize what each agent will do
3. Tell the user to run: ` + "`" + `air run <name1> <name2> ...` + "`" + `
`
