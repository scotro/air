package main

import (
	"bufio"
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
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
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

const orchestrationContext = `## Orchestration Mode

You are helping plan work for multiple AI agents that will run in parallel. Each agent works in an isolated git worktree on a specific task.

**Think very, very hard about producing detailed, thorough, and holistically consistent plans.** The quality of your plans directly determines whether the agents succeed or fail. Vague plans lead to buggy implementations. Inconsistent plans lead to integration failures. Take your time.

### Your Job

1. **Understand what the user wants to build** - Ask clarifying questions if needed. Understand scope, constraints, and what "done" looks like.

2. **Decompose into parallel work streams** - Identify 2-4 tasks that can run simultaneously with minimal dependencies. Good decomposition:
   - Clear boundaries (each agent knows exactly which files to touch)
   - Minimal overlap (agents won't create merge conflicts)
   - Testable independently (each task has clear acceptance criteria)

   **New/empty projects:** Always create a dedicated "setup" plan that runs first and completes before other agents start. The setup plan should ONLY create scaffolding:
   - ` + "`" + `go.mod` + "`" + ` (or equivalent for other languages)
   - Empty package directories
   - Basic project structure

   All other plans must depend on setup via ` + "`" + `setup-complete` + "`" + ` channel. Do NOT bundle feature work into the setup plan - keep it minimal so it completes quickly. This prevents conflicts from multiple agents trying to create foundational files like go.mod.

3. **Create plans** - Write plan files to ` + "`" + `~/.air/<project>/plans/<name>.md` + "`" + ` for each task (where ` + "`" + `<project>` + "`" + ` is the current directory name).

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

### Acceptance Criteria Guidelines

Acceptance criteria MUST be specific and testable. For each command/feature:
- Include at least one concrete test case with expected input/output
- Specify edge cases (empty input, missing keys, etc.)

**Examples:**
- Bad: ` + "`" + `- [ ] GET command works` + "`" + `
- Good: ` + "`" + `- [ ] GET existing key returns value: GET foo → "bar" after SET foo bar` + "`" + `
- Good: ` + "`" + `- [ ] GET missing key returns nil: GET nonexistent → (nil)` + "`" + `

### Testing Boundaries

**Critical:** Parallel agents must not compete for shared resources.

- Parallel agents should only run **unit tests** (no servers, no ports, no shared state)
- Smoke tests and integration tests require the full system and should happen **after** ` + "`" + `air integrate` + "`" + `
- If a test requires starting a server, binding a port, or accessing shared state - it's NOT safe for parallel execution

**In acceptance criteria, write:**
- Good: ` + "`" + `- [ ] Unit tests pass` + "`" + `
- Bad: ` + "`" + `- [ ] Smoke test with redis-cli works` + "`" + ` (this conflicts across parallel agents!)

### Concurrent Plans (with Dependencies)

When one plan MUST wait for another to complete some work first, add a **Dependencies** section.

**IMPORTANT: Prefer parallel (independent) plans whenever possible.** Only use dependencies when absolutely necessary - each integration point is a potential merge conflict.

` + "```" + `markdown
## Dependencies

**Waits on:**
- ` + "`" + `<channel-name>` + "`" + ` - Description of what must be ready first

**Signals:**
- ` + "`" + `<channel-name>` + "`" + ` - Description of what this plan provides to others

**Sequence:**
1. Run ` + "`" + `air agent wait <channel>` + "`" + ` before starting dependent work
2. Run ` + "`" + `air agent merge <channel>` + "`" + ` to pull in changes
3. Do implementation work
4. Commit changes
5. Run ` + "`" + `air agent signal <channel>` + "`" + ` to notify waiting agents
6. Run ` + "`" + `air agent done` + "`" + ` when complete
` + "```" + `

**Design principles for concurrent plans:**
- **Prefer independent plans** - parallel plans with no dependencies are simpler and safer
- **Complete the chain** - CRITICAL: every channel that appears in "Waits on" MUST have exactly one plan that "Signals" it. If plan B waits on ` + "`" + `setup-complete` + "`" + `, plan A MUST have a Dependencies section that signals ` + "`" + `setup-complete` + "`" + `. Incomplete chains cause agents to wait forever.
- **Minimize integration points** - fewer signals = fewer conflicts
- **Non-overlapping files** - agents consuming the same channel must work on different files
- **Signal late** - only signal after committing stable, tested code
- **Name channels clearly** - use descriptive names like ` + "`" + `core-ready` + "`" + `, ` + "`" + `auth-complete` + "`" + `

**Before finalizing plans, verify the dependency chain is complete:**
1. List all channels that appear in any "Waits on" section
2. For each channel, confirm exactly one plan has it in "Signals"
3. If a channel has no signaler, add a Dependencies section to the appropriate plan

### Integration Plans

**Prefer simple plans that just work after merging.** The best decomposition produces components that work together without additional wiring - merge the branches and you're done.

However, some projects are complex enough that components need to be wired together in code (imports, initialization, main.go). When this is the case, **create a dedicated integration plan** rather than leaving manual work for the user.

**Signs you need an integration plan:**
- Multiple packages that must be imported and initialized together
- A main.go that needs to connect several components
- You're tempted to tell the user "after merging, you'll need to wire X, Y, Z together"

**Integration plan structure:**
- Waits on completion signals from all parallel plans
- Wires components together (imports, initialization, main.go)
- Runs final build to verify everything connects
- Does NOT signal anything (it's the final plan)

**Example dependency graph:**
` + "```" + `
setup → [core, features, dashboard] → integration
` + "```" + `

**Important:** The integration plan is agent work, not git merging. ` + "`" + `air integrate` + "`" + ` handles git merging after all agents (including the integration agent) complete.

### After planning

1. Use the Write tool to create each plan file in ` + "`" + `~/.air/<project>/plans/<name>.md` + "`" + ` (where ` + "`" + `<project>` + "`" + ` is the current directory name)
2. Summarize what each agent will do
3. If plans have dependencies, explain the dependency graph to the user
4. Tell the user: "Exit Claude Code, then run: ` + "`" + `air run <name1> <name2> ...` + "`" + `"
`
