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

func init() {
	planCmd.AddCommand(planListCmd)
	planCmd.AddCommand(planShowCmd)
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
	packetsDir := filepath.Join(".air", "packets")

	entries, err := os.ReadDir(packetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No packets yet. Run 'air plan' to create some.")
			return nil
		}
		return fmt.Errorf("failed to read packets: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No packets yet. Run 'air plan' to create some.")
		return nil
	}

	fmt.Println("Packets:")
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
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

const orchestrationContext = `## Orchestration Mode

You are helping the user plan concurrent work. Ask:

1. **What do you want to build?**

Then ask:

2. **Should I create the work packets automatically, or do you want to describe them for manual creation?**

If automatic:
- Analyze the goal and identify 2-4 parallelizable work streams
- Create packet files in ` + "`" + `.air/packets/<name>.md` + "`" + ` using this format:

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

After creating packets, output:
` + "```" + `
Run: air run <name1> <name2> ...
` + "```" + `
`
