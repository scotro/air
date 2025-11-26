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
	Short: "Start orchestration session to create work packets",
	Long:  `Launches Claude with orchestration context to help decompose work into packets.`,
	RunE:  runPlan,
}

var planListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all work packets",
	RunE:  runPlanList,
}

var planShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a specific work packet",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanShow,
}

var planArchiveCmd = &cobra.Command{
	Use:   "archive <name>",
	Short: "Archive a work packet",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanArchive,
}

var planRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore an archived packet",
	Args:  cobra.ExactArgs(1),
	RunE:  runPlanRestore,
}

var listArchived bool

func init() {
	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planShowCmd)
	planCmd.AddCommand(planArchiveCmd)
	planCmd.AddCommand(planRestoreCmd)
	planListCmd.Flags().BoolVar(&listArchived, "archived", false, "Show archived packets")
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
	var packetsDir string
	var label string

	if listArchived {
		packetsDir = filepath.Join(".air", "packets", "archive")
		label = "Archived Packets:"
	} else {
		packetsDir = filepath.Join(".air", "packets")
		label = "Packets:"
	}

	entries, err := os.ReadDir(packetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			if listArchived {
				fmt.Println("No archived packets.")
			} else {
				fmt.Println("No packets yet. Run 'air plan' to create some.")
			}
			return nil
		}
		return fmt.Errorf("failed to read packets: %w", err)
	}

	// Filter to only .md files (exclude archive directory)
	var packets []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			packets = append(packets, entry)
		}
	}

	if len(packets) == 0 {
		if listArchived {
			fmt.Println("No archived packets.")
		} else {
			fmt.Println("No packets yet. Run 'air plan' to create some.")
		}
		return nil
	}

	fmt.Println(label)
	for _, entry := range packets {
		name := strings.TrimSuffix(entry.Name(), ".md")

		// Read first line for objective
		content, _ := os.ReadFile(filepath.Join(packetsDir, entry.Name()))
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
	packetPath := filepath.Join(".air", "packets", name+".md")

	content, err := os.ReadFile(packetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("packet '%s' not found", name)
		}
		return fmt.Errorf("failed to read packet: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

func runPlanArchive(cmd *cobra.Command, args []string) error {
	name := args[0]
	srcPath := filepath.Join(".air", "packets", name+".md")
	archiveDir := filepath.Join(".air", "packets", "archive")
	dstPath := filepath.Join(archiveDir, name+".md")

	// Check source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("packet '%s' not found", name)
	}

	// Create archive directory
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to archive packet: %w", err)
	}

	fmt.Printf("Archived: %s\n", name)
	return nil
}

func runPlanRestore(cmd *cobra.Command, args []string) error {
	name := args[0]
	srcPath := filepath.Join(".air", "packets", "archive", name+".md")
	dstPath := filepath.Join(".air", "packets", name+".md")

	// Check source exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("archived packet '%s' not found", name)
	}

	// Check destination doesn't exist
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("packet '%s' already exists (not archived)", name)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to restore packet: %w", err)
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

3. **Create work packets** - Write packet files to ` + "`" + `.air/packets/<name>.md` + "`" + ` for each task.

4. **Provide launch command** - Tell the user exactly how to start the agents.

### Start by asking:

"What would you like to build? Describe the feature, task, or goal - I'll help break it down into parallel work streams for multiple agents."

### Packet format:

` + "```" + `markdown
# Packet: <name>

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

1. Use the Write tool to create each packet file in ` + "`" + `.air/packets/<name>.md` + "`" + `
2. Summarize what each agent will do
3. Tell the user to run: ` + "`" + `air run <name1> <name2> ...` + "`" + `
`
