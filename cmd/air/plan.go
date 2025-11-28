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
		orchestrationPrompt = string(context) + "\n\n" + repoContext + "\n\n" + workspaceOrchestrationContext
	} else {
		orchestrationPrompt = string(context) + "\n\n" + orchestrationContext
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

**CRITICAL: Agent Isolation Model**

Each agent runs in a completely isolated git worktree. Agents CANNOT see each other's work - the ONLY way to access another agent's code is through channel signals:
1. Agent A signals ` + "`" + `channelX` + "`" + ` after committing
2. Agent B runs ` + "`" + `air agent wait channelX` + "`" + ` then ` + "`" + `air agent merge channelX` + "`" + `
3. Now Agent B has Agent A's code in its worktree

There is NO other way. Agents cannot check the filesystem to see if other agents are "done" - they will only see their own isolated worktree.

**Integration plan requirements:**

1. **Every parallel plan MUST signal a completion channel.** If you have plans ` + "`" + `core` + "`" + `, ` + "`" + `middleware` + "`" + `, ` + "`" + `dashboard` + "`" + ` running in parallel, each MUST have:
   ` + "```" + `markdown
   **Signals:**
   - ` + "`" + `core-complete` + "`" + `       # (or middleware-complete, dashboard-complete)
   ` + "```" + `

2. **The integration plan MUST wait on ALL parallel plans:**
   ` + "```" + `markdown
   **Waits on:**
   - ` + "`" + `setup-complete` + "`" + `
   - ` + "`" + `core-complete` + "`" + `
   - ` + "`" + `middleware-complete` + "`" + `
   - ` + "`" + `dashboard-complete` + "`" + `
   ` + "```" + `

3. **The integration plan does NOT signal anything** (it's the final plan)

**Example with integration plan:**
` + "```" + `
Plan: setup
  Waits on: (none)
  Signals: setup-complete

Plan: feature-a
  Waits on: setup-complete
  Signals: feature-a-complete

Plan: feature-b
  Waits on: setup-complete
  Signals: feature-b-complete

Plan: integration
  Waits on: setup-complete, feature-a-complete, feature-b-complete
  Signals: (none - final plan)
` + "```" + `

**Important:** The integration plan is agent work, not git merging. ` + "`" + `air integrate` + "`" + ` handles git merging after all agents (including the integration agent) complete.

### After planning

1. Use the Write tool to create each plan file in ` + "`" + `~/.air/<project>/plans/<name>.md` + "`" + ` (where ` + "`" + `<project>` + "`" + ` is the current directory name)
2. **Run ` + "`" + `air plan validate` + "`" + `** to verify the dependency graph is valid. This checks:
   - Every channel waited on has exactly one plan that signals it
   - No cycles exist in the dependency graph
   - No channel is signaled by multiple plans
   If validation fails, fix the plans before proceeding.
3. Summarize what each agent will do
4. If plans have dependencies, explain the dependency graph to the user
5. Tell the user: "Exit Claude Code, then run: ` + "`" + `air run <name1> <name2> ...` + "`" + `"
`

const workspaceOrchestrationContext = `## Orchestration Mode (Multi-Repository Workspace)

You are helping plan work for multiple AI agents that will run in parallel across MULTIPLE repositories. Each agent works in an isolated git worktree on a specific task in a specific repository.

**Think very, very hard about producing detailed, thorough, and holistically consistent plans.** Multi-repo work is more complex - agents in different repos cannot git-merge each other's code, only coordinate via channels.

### Your Job

1. **Understand what the user wants to build** - This spans multiple repos. Understand which repos are affected and how they relate.

2. **Decompose into plans per repo** - Each plan targets ONE repository:
   - Clear boundaries (each agent knows exactly which repo and files)
   - Explicit repository in every plan
   - Dependencies between repos use channels (wait/signal)
   - Dependencies WITHIN a repo can use merge

3. **Create plans** - Write plan files to ` + "`" + `~/.air/<workspace>/plans/<name>.md` + "`" + `

4. **Provide launch command** - Tell the user how to start the agents.

### Plan format (Workspace Mode):

` + "```" + `markdown
# Plan: <name>

**Repository:** <repo-name>

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

## Dependencies (if needed)

**Waits on:**
- ` + "`" + `<channel-name>` + "`" + ` - Description

**Signals:**
- ` + "`" + `<channel-name>` + "`" + ` - Description

**Sequence:**
1. Run ` + "`" + `air agent wait <channel>` + "`" + ` for each dependency
2. Do implementation work
3. Commit changes
4. Run ` + "`" + `air agent signal <channel>` + "`" + ` for each output
5. Run ` + "`" + `air agent done` + "`" + ` when complete
` + "```" + `

### CRITICAL: Repository Field

Every plan MUST have a **Repository:** field specifying which repo in the workspace it targets.

### CRITICAL: Cross-Repo Dependencies

Unlike single-repo mode, agents in DIFFERENT repos cannot git-merge each other's code. They can only coordinate via channels:

- ` + "`" + `air agent wait <channel>` + "`" + ` - Blocks until the channel is signaled
- ` + "`" + `air agent merge <channel>` + "`" + ` - **ONLY works within the same repo**

For cross-repo dependencies:
1. Agent A (repo: schema) signals ` + "`" + `schema-ready` + "`" + `
2. Agent B (repo: usersvc) waits on ` + "`" + `schema-ready` + "`" + `
3. Agent B then proceeds - it knows schema is done
4. Agent B may need to update its dependency (e.g., ` + "`" + `go get schema@latest` + "`" + `)

### Common Multi-Repo Patterns

**Pattern 1: Schema First**
` + "```" + `
schema-update (repo: schema)
  Signals: schema-ready

usersvc-feature (repo: usersvc)
  Waits on: schema-ready

sdk-regen (repo: platform-sdk)
  Waits on: schema-ready
` + "```" + `

**Pattern 2: Generated Code**
` + "```" + `
api-spec-update (repo: api-spec)
  Signals: spec-ready

client-regen (repo: api-client)
  Waits on: spec-ready
  (reads spec from api-spec worktree, runs generator)
  Signals: client-ready

mobile-update (repo: mobile-app)
  Waits on: client-ready
` + "```" + `

**Pattern 3: Parallel Independent Work**
` + "```" + `
frontend-rebrand (repo: web-frontend)
mobile-rebrand (repo: mobile-app)
admin-rebrand (repo: admin-dashboard)
(All run in parallel, no dependencies)
` + "```" + `

### After planning

1. Create each plan file in ` + "`" + `~/.air/<workspace>/plans/<name>.md` + "`" + `
2. **Run ` + "`" + `air plan validate` + "`" + `** to verify:
   - Every plan has a valid **Repository:** field
   - All dependency chains are complete
   - No cycles exist
3. Summarize the plan structure and cross-repo dependencies
4. Tell the user: "Exit Claude Code, then run: ` + "`" + `air run` + "`" + `"
`
