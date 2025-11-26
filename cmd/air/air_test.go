package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelper sets up a temp git repo and returns cleanup function
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "air-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create initial commit (needed for worktrees)
	readme := filepath.Join(tmpDir, "README.md")
	os.WriteFile(readme, []byte("# Test Project\n"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "Initial commit").Run()

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// runAir runs the air command in the given directory
func runAir(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Build the binary if needed
	binPath := filepath.Join(os.TempDir(), "air-test-binary")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = filepath.Join(mustGetwd(t), ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build air: %v\n%s", err, out)
	}

	cmd := exec.Command(binPath, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return wd
}

// ============================================================================
// air init tests
// ============================================================================

func TestInit_CreatesAirDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	out, err := runAir(t, tmpDir, "init")
	if err != nil {
		t.Fatalf("air init failed: %v\n%s", err, out)
	}

	// Check .air/ exists
	airDir := filepath.Join(tmpDir, ".air")
	if _, err := os.Stat(airDir); os.IsNotExist(err) {
		t.Error(".air/ directory was not created")
	}

	// Check .air/plans/ exists
	plansDir := filepath.Join(airDir, "plans")
	if _, err := os.Stat(plansDir); os.IsNotExist(err) {
		t.Error(".air/plans/ directory was not created")
	}

	// Check .air/context.md exists
	contextFile := filepath.Join(airDir, "context.md")
	if _, err := os.Stat(contextFile); os.IsNotExist(err) {
		t.Error(".air/context.md was not created")
	}
}

func TestInit_CreatesContextWithExpectedContent(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	content, err := os.ReadFile(filepath.Join(tmpDir, ".air", "context.md"))
	if err != nil {
		t.Fatalf("failed to read context.md: %v", err)
	}

	// Check for key sections
	checks := []string{
		"AI Runner Workflow",
		"CRITICAL: Stay In Your Worktree",
		"NEVER access paths outside",
		"Your Assignment",
		"Boundaries",
		"BLOCKED",
		"DONE",
	}

	for _, check := range checks {
		if !strings.Contains(string(content), check) {
			t.Errorf("context.md missing expected content: %q", check)
		}
	}
}

func TestInit_UpdatesGitignore(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	content, err := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}

	if !strings.Contains(string(content), ".air/") {
		t.Error(".gitignore does not contain .air/")
	}
}

func TestInit_IsIdempotent(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// Run init twice
	runAir(t, tmpDir, "init")
	out, err := runAir(t, tmpDir, "init")
	if err != nil {
		t.Fatalf("second air init failed: %v\n%s", err, out)
	}

	// Should not duplicate .gitignore entries
	content, _ := os.ReadFile(filepath.Join(tmpDir, ".gitignore"))
	count := strings.Count(string(content), ".air/")
	if count > 1 {
		t.Errorf(".gitignore contains .air/ %d times, expected 1", count)
	}
}

func TestInit_FailsOutsideGitRepo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "air-test-nogit-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = runAir(t, tmpDir, "init")
	if err == nil {
		t.Error("expected air init to fail outside git repo")
	}
}

// ============================================================================
// air plan tests
// ============================================================================

func TestPlanList_ShowsPlans(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plans
	plansDir := filepath.Join(tmpDir, ".air", "plans")
	os.WriteFile(filepath.Join(plansDir, "auth.md"), []byte("# Auth plan\n**Objective:** Test"), 0644)
	os.WriteFile(filepath.Join(plansDir, "api.md"), []byte("# API plan\n**Objective:** Test"), 0644)

	out, err := runAir(t, tmpDir, "plan", "list")
	if err != nil {
		t.Fatalf("air plan list failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "auth") {
		t.Error("plan list output missing 'auth' plan")
	}
	if !strings.Contains(out, "api") {
		t.Error("plan list output missing 'api' plan")
	}
}

func TestPlanList_EmptyMessage(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	out, err := runAir(t, tmpDir, "plan", "list")
	if err != nil {
		t.Fatalf("air plan list failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "No plans") {
		t.Error("expected 'No plans' message for empty plans dir")
	}
}

func TestPlanShow_DisplaysPlan(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan
	content := "# Test Plan\n\n**Objective:** Do the thing\n\n## Details\nMore info here."
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte(content), 0644)

	out, err := runAir(t, tmpDir, "plan", "show", "test")
	if err != nil {
		t.Fatalf("air plan show failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Test Plan") {
		t.Error("plan show output missing plan content")
	}
	if !strings.Contains(out, "Do the thing") {
		t.Error("plan show output missing objective")
	}
}

func TestPlanShow_FailsForMissingPlan(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	_, err := runAir(t, tmpDir, "plan", "show", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestPlanArchiveAndRestore(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan
	plansDir := filepath.Join(tmpDir, ".air", "plans")
	planPath := filepath.Join(plansDir, "test.md")
	os.WriteFile(planPath, []byte("# Test"), 0644)

	// Archive it
	out, err := runAir(t, tmpDir, "plan", "archive", "test")
	if err != nil {
		t.Fatalf("air plan archive failed: %v\n%s", err, out)
	}

	// Original should be gone
	if _, err := os.Stat(planPath); !os.IsNotExist(err) {
		t.Error("plan should be removed after archive")
	}

	// Should be in archive
	archivedPath := filepath.Join(plansDir, "archive", "test.md")
	if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
		t.Error("plan should exist in archive")
	}

	// Restore it
	out, err = runAir(t, tmpDir, "plan", "restore", "test")
	if err != nil {
		t.Fatalf("air plan restore failed: %v\n%s", err, out)
	}

	// Should be back
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Error("plan should exist after restore")
	}

	// Should be gone from archive
	if _, err := os.Stat(archivedPath); !os.IsNotExist(err) {
		t.Error("plan should be removed from archive after restore")
	}
}

// ============================================================================
// air run tests
// ============================================================================

func TestRun_FailsIfNotInitialized(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	_, err := runAir(t, tmpDir, "run", "test")
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestRun_ShowsPlansWithNoArgs(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte("# Test"), 0644)

	out, err := runAir(t, tmpDir, "run")
	if err != nil {
		t.Fatalf("air run failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "Available plans") {
		t.Error("expected available plans list")
	}
	if !strings.Contains(out, "test") {
		t.Error("expected 'test' plan in list")
	}
}

func TestRun_FailsForMissingPlan(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create one plan so we get past the "no plans" check
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "exists.md"), []byte("# Exists"), 0644)

	_, err := runAir(t, tmpDir, "run", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent plan")
	}
}

func TestRun_CreatesWorktreeDirectory(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte("# Test\n**Objective:** Test"), 0644)

	// Note: This will fail to actually run claude/tmux, but should create the worktree
	runAir(t, tmpDir, "run", "test")

	// Check worktree was created
	wtPath := filepath.Join(tmpDir, ".air", "worktrees", "test")
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Error("worktree directory was not created")
	}
}

func TestRun_GeneratesLaunchScript(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte("# Test\n**Objective:** Test"), 0644)

	runAir(t, tmpDir, "run", "test")

	// Check launch script exists
	scriptPath := filepath.Join(tmpDir, ".air", "worktrees", "test", ".air", "launch.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Error("launch.sh was not created")
	}

	// Check it's executable
	info, _ := os.Stat(scriptPath)
	if info.Mode()&0111 == 0 {
		t.Error("launch.sh is not executable")
	}

	// Check content includes claude command
	content, _ := os.ReadFile(scriptPath)
	if !strings.Contains(string(content), "claude") {
		t.Error("launch.sh missing claude command")
	}
	if !strings.Contains(string(content), "--append-system-prompt") {
		t.Error("launch.sh missing --append-system-prompt")
	}
}

func TestRun_LaunchScriptContainsPlanContent(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create test plan with unique content
	planContent := "**Objective:** Implement the FOOBAR_UNIQUE_STRING feature"
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte(planContent), 0644)

	runAir(t, tmpDir, "run", "test")

	// Check assignment file contains plan content
	assignmentPath := filepath.Join(tmpDir, ".air", "worktrees", "test", ".air", ".assignment")
	content, err := os.ReadFile(assignmentPath)
	if err != nil {
		t.Fatalf("failed to read .assignment: %v", err)
	}

	if !strings.Contains(string(content), "FOOBAR_UNIQUE_STRING") {
		t.Error(".assignment missing plan content")
	}
}

// ============================================================================
// air clean tests
// ============================================================================

func TestClean_RemovesSpecificWorktree(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create two plans and run them
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "keep.md"), []byte("# Keep"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "remove.md"), []byte("# Remove"), 0644)

	runAir(t, tmpDir, "run", "keep", "remove")

	// Clean only 'remove'
	runAir(t, tmpDir, "clean", "remove", "--branches")

	// 'keep' should still exist
	keepPath := filepath.Join(tmpDir, ".air", "worktrees", "keep")
	if _, err := os.Stat(keepPath); os.IsNotExist(err) {
		t.Error("'keep' worktree should still exist")
	}

	// 'remove' should be gone
	removePath := filepath.Join(tmpDir, ".air", "worktrees", "remove")
	if _, err := os.Stat(removePath); !os.IsNotExist(err) {
		t.Error("'remove' worktree should be removed")
	}
}

func TestClean_FailsForNonexistentWorktree(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	runAir(t, tmpDir, "init")

	// Create and run a plan to have at least one worktree
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "test.md"), []byte("# Test"), 0644)
	runAir(t, tmpDir, "run", "test")

	_, err := runAir(t, tmpDir, "clean", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent worktree")
	}
}

// ============================================================================
// air version test
// ============================================================================

func TestVersion_ShowsVersion(t *testing.T) {
	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	out, err := runAir(t, tmpDir, "version")
	if err != nil {
		t.Fatalf("air version failed: %v\n%s", err, out)
	}

	if !strings.Contains(out, "air v") {
		t.Error("version output missing 'air v'")
	}
}

// ============================================================================
// Integration test
// ============================================================================

func TestFullWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	tmpDir, cleanup := setupTestRepo(t)
	defer cleanup()

	// 1. Initialize
	out, err := runAir(t, tmpDir, "init")
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	// 2. Create a plan manually (simulating what air plan would do)
	plan := `# Plan: feature

**Objective:** Add a new feature

## Boundaries

**In scope:**
- cmd/feature/

**Out of scope:**
- Everything else

## Acceptance Criteria

- [ ] Feature works
- [ ] Tests pass
`
	os.WriteFile(filepath.Join(tmpDir, ".air", "plans", "feature.md"), []byte(plan), 0644)

	// 3. List plans
	out, err = runAir(t, tmpDir, "plan", "list")
	if err != nil {
		t.Fatalf("plan list failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "feature") {
		t.Error("plan list missing 'feature'")
	}

	// 4. Show plan
	out, err = runAir(t, tmpDir, "plan", "show", "feature")
	if err != nil {
		t.Fatalf("plan show failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Add a new feature") {
		t.Error("plan show missing objective")
	}

	// 5. Run (will create worktree but fail on tmux - that's ok)
	runAir(t, tmpDir, "run", "feature")

	// 6. Verify worktree structure
	wtPath := filepath.Join(tmpDir, ".air", "worktrees", "feature")
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree not created")
	}

	launchScript := filepath.Join(wtPath, ".air", "launch.sh")
	if _, err := os.Stat(launchScript); os.IsNotExist(err) {
		t.Fatal("launch.sh not created")
	}

	// 7. Clean up
	out, err = runAir(t, tmpDir, "clean", "feature", "--branches")
	if err != nil {
		t.Fatalf("clean failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Error("worktree not removed after clean")
	}
}
