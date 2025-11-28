package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate",
	Short: "Start integration session to merge completed work",
	RunE:  runIntegrate,
}

func runIntegrate(cmd *cobra.Command, args []string) error {
	// Check initialization
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Detect mode
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
	}

	// Read context
	context, err := os.ReadFile(getContextPath())
	if err != nil {
		return fmt.Errorf("failed to read context: %w", err)
	}

	// Build integration prompt based on mode
	var integrationPrompt string
	if info.Mode == ModeWorkspace {
		integrationPrompt = string(context) + "\n\n" + buildWorkspaceIntegrationContext(info)
	} else {
		integrationPrompt = string(context) + "\n\n" + integrationContext
	}

	// Launch claude with initial prompt
	claudeCmd := buildIntegrateCommand(integrationPrompt, info)
	claudeCmd.Stdin = os.Stdin
	claudeCmd.Stdout = os.Stdout
	claudeCmd.Stderr = os.Stderr

	return claudeCmd.Run()
}

// buildIntegrateCommand constructs the claude command for integration mode.
// Extracted for testability - allows verifying command args are correctly structured.
func buildIntegrateCommand(integrationPrompt string, info *WorkspaceInfo) *exec.Cmd {
	// Allowed tools for integration: read-only git commands, air commands, and file inspection
	allowedTools := `Bash(git worktree:*) Bash(git branch:*) Bash(git log:*) Bash(git diff:*) Bash(git merge-tree:*) Bash(git merge-base:*) Bash(air plan:*) Bash(cat:*) Bash(ls:*)`

	initialPrompt := "Begin integration. Show me the status of agent branches and guide me through merging."
	if info.Mode == ModeWorkspace {
		initialPrompt = "Begin integration. Show me the status of agent branches across all repositories and guide me through merging."
	}

	return exec.Command("claude",
		"--allowedTools", allowedTools,
		"--append-system-prompt", integrationPrompt,
		initialPrompt)
}

// buildWorkspaceIntegrationContext generates integration instructions for workspace mode
func buildWorkspaceIntegrationContext(info *WorkspaceInfo) string {
	var sb strings.Builder

	sb.WriteString(`## Workspace Integration Mode

You are helping integrate completed agent work across multiple repositories in a workspace.

**Workspace:** `)
	sb.WriteString(info.Name)
	sb.WriteString("\n**Repositories:** ")
	sb.WriteString(strings.Join(info.Repos, ", "))
	sb.WriteString("\n**Root:** ")
	sb.WriteString(info.Root)
	sb.WriteString(`

### Step 1: Assess the situation

For each repository, check the agent branches:
`)

	for _, repo := range info.Repos {
		sb.WriteString("- `cd ")
		sb.WriteString(info.Root)
		sb.WriteString("/")
		sb.WriteString(repo)
		sb.WriteString(" && git branch | grep air/`\n")
	}

	sb.WriteString(`
Also check active worktrees: ` + "`air status`" + `

### Step 2: Determine merge order from dependencies

Use ` + "`air plan list`" + ` to see all plans, then ` + "`air plan show <name>`" + ` to read each one. Look for:
- **Repository:** field - which repo each plan targets
- **Dependencies** section - plans that "Signal" must be merged before plans that "Wait on" that channel

For cross-repo dependencies:
- Upstream repos (e.g., schema, shared libraries) should be integrated first
- Downstream repos that depend on them can be integrated after

Build a topological merge order that respects both channel dependencies AND repo dependencies.

### Step 3: Present the merge strategy

Group branches by repository and show:
1. The recommended order (repo by repo, respecting dependencies)
2. For each branch, preview changes: ` + "`git log --oneline HEAD..air/<name>`" + `
3. Conflict check: ` + "`git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>`" + `

Then ask: **"Would you like me to handle the merging for you?"**

### Step 4a: If user wants you to handle it

For each repository in order:
1. ` + "`cd <repo-path>`" + `
2. For each branch targeting this repo (in dependency order):
   - Check for conflicts with merge-tree
   - If clean: ` + "`git merge air/<name> --no-ff -m \"Merge <name>\"`" + `
   - If conflicts: STOP and help resolve before continuing
3. Move to next repo

After all merges complete:
- Summarize what was merged per repo
- Remind user to run tests in each repo
- Remind user: ` + "`air clean`" + ` removes worktrees

### Step 4b: If user wants to do it themselves

Provide commands grouped by repository:
`)

	for _, repo := range info.Repos {
		sb.WriteString("\n**")
		sb.WriteString(repo)
		sb.WriteString(":**\n```\ncd ")
		sb.WriteString(info.Root)
		sb.WriteString("/")
		sb.WriteString(repo)
		sb.WriteString("\ngit merge air/<plan-name> --no-ff -m \"Merge <plan-name>\"\n```\n")
	}

	sb.WriteString(`
### Handling conflicts

If a merge has conflicts:
1. Show which files conflict
2. Help resolve them interactively
3. After resolution: ` + "`git add <files>`" + ` then ` + "`git commit`" + `
4. Continue with remaining branches in that repo, then move to next repo
`)

	return sb.String()
}

const integrationContext = `## Integration Mode

You are helping integrate completed agent work into the main branch.

### Step 1: Assess the situation

Run these commands to understand the current state:
- ` + "`" + `git branch -a | grep air/` + "`" + ` - Show agent branches
- ` + "`" + `git worktree list` + "`" + ` - Show active worktrees

### Step 2: Determine merge order from dependencies

Use ` + "`" + `air plan list` + "`" + ` to see all plans, then ` + "`" + `air plan show <name>` + "`" + ` to read each one. Look for the **Dependencies** sections - plans that "Signal" a channel must be merged before plans that "Wait on" that channel.

Build a topological merge order. For example:
- setup (no dependencies) → merge first
- core (waits on setup) → merge second
- strings, hashes, ttl (wait on core) → merge last (order among these doesn't matter)

### Step 3: Present the merge strategy

Show the user:
1. The recommended merge order with rationale
2. A preview of what each branch changes: ` + "`" + `git log --oneline HEAD..air/<name>` + "`" + `
3. Conflict check for the first branch: ` + "`" + `git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>` + "`" + `

Then ask: **"Would you like me to handle the merging for you?"**

### Step 4a: If user wants you to handle it

For each branch in order:
1. Check for conflicts: ` + "`" + `git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>` + "`" + `
2. If clean, execute: ` + "`" + `git merge air/<name> --no-ff -m "Merge <name>"` + "`" + `
3. If conflicts detected, STOP and help resolve before continuing
4. After each successful merge, briefly confirm and move to the next

After all merges complete:
- Summarize what was merged
- Offer to run tests if a test command exists (check for Makefile, go.mod, package.json)
- Remind user: ` + "`" + `air clean` + "`" + ` removes worktrees and will ask about deleting branches

### Step 4b: If user wants to do it themselves

Provide the merge commands in the correct order. For each branch show:
` + "`" + `git merge air/<name> --no-ff -m "Merge <name>"` + "`" + `

### Handling conflicts

If a merge has conflicts:
1. Show which files conflict
2. Help resolve them interactively
3. After resolution: ` + "`" + `git add <files>` + "`" + ` then ` + "`" + `git commit` + "`" + `
4. Continue with remaining branches
`
