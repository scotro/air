package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// parsePlanDependencies tests
// ============================================================================

func TestParsePlanDependencies_NoDependencies(t *testing.T) {
	t.Parallel()

	content := `# Plan: simple

**Objective:** Do something simple

## Boundaries

**In scope:**
- src/simple.go
`

	deps := parsePlanDependencies("simple", content)

	if deps.Name != "simple" {
		t.Errorf("expected name 'simple', got %q", deps.Name)
	}
	if len(deps.WaitsOn) != 0 {
		t.Errorf("expected no WaitsOn, got %v", deps.WaitsOn)
	}
	if len(deps.Signals) != 0 {
		t.Errorf("expected no Signals, got %v", deps.Signals)
	}
}

func TestParsePlanDependencies_WithRepository(t *testing.T) {
	t.Parallel()

	content := `# Plan: schema-update

**Repository:** schema

**Objective:** Update schema definitions

## Boundaries

**In scope:**
- protos/
`

	deps := parsePlanDependencies("schema-update", content)

	if deps.Name != "schema-update" {
		t.Errorf("expected name 'schema-update', got %q", deps.Name)
	}
	if deps.Repository != "schema" {
		t.Errorf("expected repository 'schema', got %q", deps.Repository)
	}
}

func TestParsePlanDependencies_RepositoryWithWhitespace(t *testing.T) {
	t.Parallel()

	content := `# Plan: usersvc-feature

**Repository:**   usersvc

**Objective:** Add feature
`

	deps := parsePlanDependencies("usersvc-feature", content)

	if deps.Repository != "usersvc" {
		t.Errorf("expected repository 'usersvc', got %q", deps.Repository)
	}
}

func TestParsePlanDependencies_WithDependencies(t *testing.T) {
	t.Parallel()

	content := `# Plan: core

**Objective:** Build core functionality

## Dependencies

**Waits on:**
- ` + "`setup-complete`" + ` - Need project scaffolding first

**Signals:**
- ` + "`core-ready`" + ` - Core module is ready for dependents

## Boundaries

**In scope:**
- src/core/
`

	deps := parsePlanDependencies("core", content)

	if deps.Name != "core" {
		t.Errorf("expected name 'core', got %q", deps.Name)
	}
	if len(deps.WaitsOn) != 1 || deps.WaitsOn[0] != "setup-complete" {
		t.Errorf("expected WaitsOn ['setup-complete'], got %v", deps.WaitsOn)
	}
	if len(deps.Signals) != 1 || deps.Signals[0] != "core-ready" {
		t.Errorf("expected Signals ['core-ready'], got %v", deps.Signals)
	}
}

func TestParsePlanDependencies_MultipleChannels(t *testing.T) {
	t.Parallel()

	content := `# Plan: integration

## Dependencies

**Waits on:**
- ` + "`core-ready`" + ` - Core module
- ` + "`auth-ready`" + ` - Auth module
- ` + "`api-ready`" + ` - API module

**Signals:**
- ` + "`integration-complete`" + ` - All wired together
`

	deps := parsePlanDependencies("integration", content)

	if len(deps.WaitsOn) != 3 {
		t.Errorf("expected 3 WaitsOn, got %v", deps.WaitsOn)
	}
	expected := []string{"core-ready", "auth-ready", "api-ready"}
	for i, ch := range expected {
		if deps.WaitsOn[i] != ch {
			t.Errorf("expected WaitsOn[%d] = %q, got %q", i, ch, deps.WaitsOn[i])
		}
	}

	if len(deps.Signals) != 1 || deps.Signals[0] != "integration-complete" {
		t.Errorf("expected Signals ['integration-complete'], got %v", deps.Signals)
	}
}

// ============================================================================
// validateDependencyGraph tests
// ============================================================================

func TestValidateDependencyGraph_Valid(t *testing.T) {
	t.Parallel()

	plans := []PlanDependencies{
		{Name: "setup", Signals: []string{"setup-complete"}},
		{Name: "core", WaitsOn: []string{"setup-complete"}, Signals: []string{"core-ready"}},
		{Name: "features", WaitsOn: []string{"setup-complete"}},
		{Name: "integration", WaitsOn: []string{"core-ready"}},
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

// ============================================================================
// validateRepositoryReferences tests
// ============================================================================

func TestValidateRepositoryReferences_Valid(t *testing.T) {
	t.Parallel()

	info := &WorkspaceInfo{
		Mode:  ModeWorkspace,
		Name:  "myteam",
		Repos: []string{"authapi", "schema", "usersvc"},
	}

	plans := []PlanDependencies{
		{Name: "schema-update", Repository: "schema"},
		{Name: "usersvc-feature", Repository: "usersvc"},
		{Name: "auth-fix", Repository: "authapi"},
	}

	errs := validateRepositoryReferences(plans, info)

	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateRepositoryReferences_MissingRepository(t *testing.T) {
	t.Parallel()

	info := &WorkspaceInfo{
		Mode:  ModeWorkspace,
		Name:  "myteam",
		Repos: []string{"authapi", "schema", "usersvc"},
	}

	plans := []PlanDependencies{
		{Name: "schema-update", Repository: "schema"},
		{Name: "usersvc-feature", Repository: ""}, // Missing!
	}

	errs := validateRepositoryReferences(plans, info)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "missing required") {
		t.Errorf("error should mention 'missing required': %v", errs[0])
	}
}

func TestValidateRepositoryReferences_UnknownRepository(t *testing.T) {
	t.Parallel()

	info := &WorkspaceInfo{
		Mode:  ModeWorkspace,
		Name:  "myteam",
		Repos: []string{"authapi", "schema", "usersvc"},
	}

	plans := []PlanDependencies{
		{Name: "schema-update", Repository: "schema"},
		{Name: "billing-feature", Repository: "billing"}, // Doesn't exist!
	}

	errs := validateRepositoryReferences(plans, info)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "unknown repository") {
		t.Errorf("error should mention 'unknown repository': %v", errs[0])
	}
	if !strings.Contains(errs[0].Error(), "billing") {
		t.Errorf("error should mention 'billing': %v", errs[0])
	}
}

func TestValidateDependencyGraph_MissingSignaler(t *testing.T) {
	t.Parallel()

	plans := []PlanDependencies{
		{Name: "core", WaitsOn: []string{"setup-complete"}}, // nobody signals setup-complete!
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "setup-complete") {
		t.Errorf("error should mention 'setup-complete': %v", errs[0])
	}
	if !strings.Contains(errs[0].Error(), "no plan signals it") {
		t.Errorf("error should mention 'no plan signals it': %v", errs[0])
	}
}

func TestValidateDependencyGraph_DuplicateSignaler(t *testing.T) {
	t.Parallel()

	plans := []PlanDependencies{
		{Name: "setup1", Signals: []string{"setup-complete"}},
		{Name: "setup2", Signals: []string{"setup-complete"}}, // duplicate!
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "signaled by both") {
		t.Errorf("error should mention 'signaled by both': %v", errs[0])
	}
}

func TestValidateDependencyGraph_Cycle(t *testing.T) {
	t.Parallel()

	plans := []PlanDependencies{
		{Name: "a", WaitsOn: []string{"b-ready"}, Signals: []string{"a-ready"}},
		{Name: "b", WaitsOn: []string{"a-ready"}, Signals: []string{"b-ready"}},
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "cycle") {
		t.Errorf("error should mention 'cycle': %v", errs[0])
	}
}

func TestValidateDependencyGraph_IndependentPlans(t *testing.T) {
	t.Parallel()

	// Plans with no dependencies at all - should be valid
	plans := []PlanDependencies{
		{Name: "feature1"},
		{Name: "feature2"},
		{Name: "feature3"},
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 0 {
		t.Errorf("expected no errors for independent plans, got %v", errs)
	}
}

func TestValidateDependencyGraph_ComplexValidGraph(t *testing.T) {
	t.Parallel()

	// Diamond dependency: setup -> [core, auth] -> integration
	plans := []PlanDependencies{
		{Name: "setup", Signals: []string{"setup-complete"}},
		{Name: "core", WaitsOn: []string{"setup-complete"}, Signals: []string{"core-ready"}},
		{Name: "auth", WaitsOn: []string{"setup-complete"}, Signals: []string{"auth-ready"}},
		{Name: "integration", WaitsOn: []string{"core-ready", "auth-ready"}},
	}

	errs := validateDependencyGraph(plans)

	if len(errs) != 0 {
		t.Errorf("expected no errors for valid diamond, got %v", errs)
	}
}

// ============================================================================
// air plan validate tests
// ============================================================================

func TestPlanValidate_ValidGraph(t *testing.T) {
	t.Parallel()
	env := setupTestRepo(t)
	defer env.cleanup()

	// Initialize via air init
	env.run(t, nil, "init")

	// Get the air directory (in fake HOME)
	airDir := env.airDir()
	plansDir := filepath.Join(airDir, "plans")

	// Create valid plans
	setupPlan := `# Plan: setup

**Objective:** Setup project

## Dependencies

**Signals:**
- ` + "`setup-complete`" + ` - Project scaffolding ready
`
	corePlan := `# Plan: core

**Objective:** Build core

## Dependencies

**Waits on:**
- ` + "`setup-complete`" + ` - Need scaffolding
`
	os.WriteFile(filepath.Join(plansDir, "setup.md"), []byte(setupPlan), 0644)
	os.WriteFile(filepath.Join(plansDir, "core.md"), []byte(corePlan), 0644)

	// Run validate - should succeed
	out, err := env.run(t, nil, "plan", "validate")
	if err != nil {
		t.Fatalf("validate failed for valid plans: %v\n%s", err, out)
	}
	if !strings.Contains(out, "All dependencies valid") {
		t.Errorf("expected success message, got: %s", out)
	}
}

func TestPlanValidate_InvalidGraph(t *testing.T) {
	t.Parallel()
	env := setupTestRepo(t)
	defer env.cleanup()

	// Initialize via air init
	env.run(t, nil, "init")

	// Get the air directory (in fake HOME)
	airDir := env.airDir()
	plansDir := filepath.Join(airDir, "plans")

	// Create invalid plan - waits on channel that nobody signals
	corePlan := `# Plan: core

**Objective:** Build core

## Dependencies

**Waits on:**
- ` + "`setup-complete`" + ` - Need scaffolding (but no setup plan!)
`
	os.WriteFile(filepath.Join(plansDir, "core.md"), []byte(corePlan), 0644)

	// Run validate - should fail
	out, err := env.run(t, nil, "plan", "validate")
	if err == nil {
		t.Fatal("validate should have failed for invalid plans")
	}
	if !strings.Contains(out, "setup-complete") {
		t.Errorf("error should mention missing channel, got: %s", out)
	}
}

