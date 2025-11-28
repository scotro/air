package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var planValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate plan dependency graph",
	Long: `Validates that all plan dependencies are satisfiable:
- Every channel waited on has exactly one plan that signals it
- No cycles exist in the dependency graph
- No channel is signaled by multiple plans`,
	RunE: runPlanValidate,
}

func init() {
	planCmd.AddCommand(planValidateCmd)
}

// PlanDependencies represents the dependency information extracted from a plan
type PlanDependencies struct {
	Name       string
	Repository string   // Target repository (required in workspace mode)
	WaitsOn    []string
	Signals    []string
}

// channelRegex matches backtick-wrapped channel names like `setup-complete`
var channelRegex = regexp.MustCompile("`([^`]+)`")

// repositoryRegex matches **Repository:** field value
var repositoryRegex = regexp.MustCompile(`^\*\*Repository:\*\*\s*(.+)$`)

// parsePlanDependencies extracts dependency information from plan markdown content
func parsePlanDependencies(name, content string) PlanDependencies {
	deps := PlanDependencies{Name: name}

	lines := strings.Split(content, "\n")
	var currentSection string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for Repository field
		if matches := repositoryRegex.FindStringSubmatch(trimmed); len(matches) >= 2 {
			deps.Repository = strings.TrimSpace(matches[1])
			continue
		}

		// Detect section headers
		if strings.HasPrefix(trimmed, "**Waits on:**") {
			currentSection = "waits"
			continue
		}
		if strings.HasPrefix(trimmed, "**Signals:**") {
			currentSection = "signals"
			continue
		}

		// End section on other bold headers or section headers
		if strings.HasPrefix(trimmed, "**") || strings.HasPrefix(trimmed, "##") {
			currentSection = ""
			continue
		}

		// Parse list items in current section
		if currentSection != "" && strings.HasPrefix(trimmed, "- ") {
			matches := channelRegex.FindStringSubmatch(trimmed)
			if len(matches) >= 2 {
				channel := matches[1]
				if currentSection == "waits" {
					deps.WaitsOn = append(deps.WaitsOn, channel)
				} else if currentSection == "signals" {
					deps.Signals = append(deps.Signals, channel)
				}
			}
		}
	}

	return deps
}

// ValidationError represents a single validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// validateDependencyGraph checks that all dependencies are satisfiable
func validateDependencyGraph(plans []PlanDependencies) []error {
	var errs []error

	// Track which plan signals which channel
	signaled := make(map[string]string) // channel -> signaling plan
	// Track which plans wait on which channel
	waited := make(map[string][]string) // channel -> waiting plans

	// First pass: collect all signals and waits
	for _, p := range plans {
		for _, ch := range p.Signals {
			if existing, ok := signaled[ch]; ok {
				errs = append(errs, ValidationError{
					Message: fmt.Sprintf("channel '%s' is signaled by both '%s' and '%s'", ch, existing, p.Name),
				})
			}
			signaled[ch] = p.Name
		}
		for _, ch := range p.WaitsOn {
			waited[ch] = append(waited[ch], p.Name)
		}
	}

	// Check every waited channel has a signaler
	for ch, waiters := range waited {
		if _, ok := signaled[ch]; !ok {
			errs = append(errs, ValidationError{
				Message: fmt.Sprintf("channel '%s' is waited on by [%s] but no plan signals it", ch, strings.Join(waiters, ", ")),
			})
		}
	}

	// Check for cycles using topological sort (Kahn's algorithm)
	cycleErrs := detectCycles(plans, signaled)
	errs = append(errs, cycleErrs...)

	return errs
}

// detectCycles finds cycles in the dependency graph
func detectCycles(plans []PlanDependencies, signaled map[string]string) []error {
	// Build adjacency list: plan -> plans it depends on
	dependsOn := make(map[string][]string)
	planNames := make(map[string]bool)

	for _, p := range plans {
		planNames[p.Name] = true
		for _, ch := range p.WaitsOn {
			if signalerPlan, ok := signaled[ch]; ok {
				dependsOn[p.Name] = append(dependsOn[p.Name], signalerPlan)
			}
		}
	}

	// Calculate in-degrees (number of dependencies)
	inDegree := make(map[string]int)
	for name := range planNames {
		inDegree[name] = 0
	}
	for _, deps := range dependsOn {
		for _, dep := range deps {
			inDegree[dep]++ // dep has one more dependent
		}
	}

	// Actually we need reverse: dependents, not dependencies
	// Let's redo: edge from A to B means "A must complete before B"
	// So if B waits on channel C, and A signals C, then A -> B
	dependents := make(map[string][]string) // plan -> plans that depend on it
	for _, p := range plans {
		for _, ch := range p.WaitsOn {
			if signalerPlan, ok := signaled[ch]; ok {
				dependents[signalerPlan] = append(dependents[signalerPlan], p.Name)
			}
		}
	}

	// Recalculate in-degrees correctly
	// in-degree of X = number of plans X waits on
	for name := range planNames {
		inDegree[name] = len(dependsOn[name])
	}

	// Kahn's algorithm
	var queue []string
	for name := range planNames {
		if inDegree[name] == 0 {
			queue = append(queue, name)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, dependent := range dependents[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if visited != len(planNames) {
		// There's a cycle - find which plans are involved
		var cyclePlans []string
		for name := range planNames {
			if inDegree[name] > 0 {
				cyclePlans = append(cyclePlans, name)
			}
		}
		return []error{ValidationError{
			Message: fmt.Sprintf("dependency cycle detected involving plans: [%s]", strings.Join(cyclePlans, ", ")),
		}}
	}

	return nil
}

// loadAllPlanDependencies reads all plans and extracts their dependencies
func loadAllPlanDependencies() ([]PlanDependencies, error) {
	plansDir := getPlansDir()

	entries, err := os.ReadDir(plansDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var plans []PlanDependencies
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		content, err := os.ReadFile(filepath.Join(plansDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read plan %s: %w", name, err)
		}

		deps := parsePlanDependencies(name, string(content))
		plans = append(plans, deps)
	}

	return plans, nil
}

// ValidatePlans loads all plans and validates their dependency graph
func ValidatePlans() ([]PlanDependencies, []error) {
	return ValidatePlansWithMode(nil)
}

// ValidatePlansWithMode loads all plans and validates them with mode awareness
func ValidatePlansWithMode(info *WorkspaceInfo) ([]PlanDependencies, []error) {
	plans, err := loadAllPlanDependencies()
	if err != nil {
		return nil, []error{err}
	}

	if len(plans) == 0 {
		return nil, nil
	}

	var errs []error

	// If workspace info provided, validate repository references
	if info != nil && info.Mode == ModeWorkspace {
		repoErrs := validateRepositoryReferences(plans, info)
		errs = append(errs, repoErrs...)
	}

	// Validate dependency graph
	graphErrs := validateDependencyGraph(plans)
	errs = append(errs, graphErrs...)

	return plans, errs
}

// validateRepositoryReferences checks that all plans have valid repository references
func validateRepositoryReferences(plans []PlanDependencies, info *WorkspaceInfo) []error {
	var errs []error

	// Build set of valid repos
	validRepos := make(map[string]bool)
	for _, r := range info.Repos {
		validRepos[r] = true
	}

	for _, p := range plans {
		// In workspace mode, Repository field is required
		if p.Repository == "" {
			errs = append(errs, ValidationError{
				Message: fmt.Sprintf("plan '%s' is missing required **Repository:** field (workspace mode)", p.Name),
			})
			continue
		}

		// Validate repo exists
		if !validRepos[p.Repository] {
			errs = append(errs, ValidationError{
				Message: fmt.Sprintf("plan '%s' references unknown repository '%s' (available: %v)", p.Name, p.Repository, info.Repos),
			})
		}
	}

	return errs
}

func runPlanValidate(cmd *cobra.Command, args []string) error {
	if !isInitialized() {
		return fmt.Errorf("not initialized (run 'air init' first)")
	}

	// Detect mode for workspace-aware validation
	info, err := detectMode()
	if err != nil {
		return fmt.Errorf("failed to detect mode: %w", err)
	}

	plans, errs := ValidatePlansWithMode(info)

	if len(plans) == 0 {
		fmt.Println("No plans found.")
		return nil
	}

	// Print mode info
	if info.Mode == ModeWorkspace {
		fmt.Printf("Workspace: %s (%d repos)\n\n", info.Name, len(info.Repos))
	}

	// Print dependency summary
	fmt.Println("Plans:")
	for _, p := range plans {
		if info.Mode == ModeWorkspace && p.Repository != "" {
			fmt.Printf("  %s [repo: %s]\n", p.Name, p.Repository)
		} else {
			fmt.Printf("  %s\n", p.Name)
		}
		if len(p.WaitsOn) > 0 {
			fmt.Printf("    waits on: %s\n", strings.Join(p.WaitsOn, ", "))
		}
		if len(p.Signals) > 0 {
			fmt.Printf("    signals:  %s\n", strings.Join(p.Signals, ", "))
		}
	}

	if len(errs) > 0 {
		fmt.Println("\nValidation errors:")
		for _, err := range errs {
			fmt.Printf("  ✗ %s\n", err)
		}
		return fmt.Errorf("validation failed with %d error(s)", len(errs))
	}

	fmt.Println("\n✓ All dependencies valid")
	return nil
}
