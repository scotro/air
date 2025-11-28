package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ============================================================================
// detectMode tests - use subprocess sandbox pattern for isolation
// ============================================================================

func TestDetectMode_SingleRepo(t *testing.T) {
	t.Parallel()
	env := setupTestRepo(t)
	defer env.cleanup()

	// Run air init which will call detectMode internally
	out, err := env.run(t, nil, "init")
	if err != nil {
		t.Fatalf("air init failed: %v\n%s", err, out)
	}

	// Verify it was detected as single mode (should see project name, not "Workspace:")
	if strings.Contains(out, "Workspace:") {
		t.Error("should not detect as workspace mode for single repo")
	}
}

func TestDetectMode_Workspace(t *testing.T) {
	t.Parallel()
	env := setupTestWorkspace(t)
	defer env.cleanup()

	// Run air init
	out, err := env.run(t, nil, "init")
	if err != nil {
		t.Fatalf("air init failed: %v\n%s", err, out)
	}

	// Verify it was detected as workspace mode
	if !strings.Contains(out, "workspace") {
		t.Errorf("expected workspace mode detection, got: %s", out)
	}

	// Should list the repos
	if !strings.Contains(out, "authapi") || !strings.Contains(out, "schema") || !strings.Contains(out, "usersvc") {
		t.Errorf("expected repos to be listed, got: %s", out)
	}
}

func TestDetectMode_NeitherRepoNorWorkspace(t *testing.T) {
	t.Parallel()
	// Use setupTestDir (no git) instead of setupTestRepo
	env := setupTestDir(t)
	defer env.cleanup()

	_, err := env.run(t, nil, "init")
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestDetectMode_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()
	env := setupTestDir(t)
	defer env.cleanup()

	// Create visible repo
	visibleRepo := filepath.Join(env.dir, "myrepo")
	os.Mkdir(visibleRepo, 0755)
	os.Mkdir(filepath.Join(visibleRepo, ".git"), 0755)

	// Create hidden repo (should be ignored)
	hiddenRepo := filepath.Join(env.dir, ".hidden")
	os.Mkdir(hiddenRepo, 0755)
	os.Mkdir(filepath.Join(hiddenRepo, ".git"), 0755)

	out, err := env.run(t, nil, "init")
	if err != nil {
		t.Fatalf("air init failed: %v\n%s", err, out)
	}

	// Should only see the visible repo
	if strings.Contains(out, ".hidden") {
		t.Error("hidden repo should not be listed")
	}
	if !strings.Contains(out, "myrepo") {
		t.Errorf("visible repo should be listed, got: %s", out)
	}
}

// setupTestWorkspace creates a temp workspace with multiple child repos
func setupTestWorkspace(t *testing.T) *testEnv {
	t.Helper()

	// Create temp workspace directory (NOT a git repo itself)
	tmpDir, err := os.MkdirTemp("", "air-workspace-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create fake HOME directory
	fakeHome, err := os.MkdirTemp("", "air-home-*")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create fake home: %v", err)
	}

	// Create child repos
	repos := []string{"authapi", "schema", "usersvc"}
	for _, repo := range repos {
		repoDir := filepath.Join(tmpDir, repo)
		os.Mkdir(repoDir, 0755)
		os.Mkdir(filepath.Join(repoDir, ".git"), 0755)
	}

	return &testEnv{
		dir:  tmpDir,
		home: fakeHome,
		cleanup: func() {
			os.RemoveAll(tmpDir)
			os.RemoveAll(fakeHome)
		},
	}
}

func TestWorkspaceInfo_GetRepoPath_SingleMode(t *testing.T) {
	info := &WorkspaceInfo{
		Mode: ModeSingle,
		Name: "myproject",
		Root: "/home/user/projects/myproject",
	}

	// Empty name should return root
	path, err := info.getRepoPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != info.Root {
		t.Errorf("expected %q, got %q", info.Root, path)
	}

	// Same name should return root
	path, err = info.getRepoPath("myproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != info.Root {
		t.Errorf("expected %q, got %q", info.Root, path)
	}

	// Different name should error
	_, err = info.getRepoPath("otherproject")
	if err == nil {
		t.Error("expected error for different repo name in single mode")
	}
}

func TestWorkspaceInfo_GetRepoPath_WorkspaceMode(t *testing.T) {
	info := &WorkspaceInfo{
		Mode:  ModeWorkspace,
		Name:  "myteam",
		Root:  "/home/user/myteam",
		Repos: []string{"authapi", "schema", "usersvc"},
	}

	// Valid repo
	path, err := info.getRepoPath("schema")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/home/user/myteam/schema"
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}

	// Invalid repo
	_, err = info.getRepoPath("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent repo")
	}
}

func TestWorkspaceInfo_GetWorktreePath(t *testing.T) {
	// Single mode
	singleInfo := &WorkspaceInfo{
		Mode: ModeSingle,
		Name: "myproject",
		Root: "/home/user/projects/myproject",
	}

	home, _ := os.UserHomeDir()

	path, err := singleInfo.getWorktreePath("", "my-plan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(home, ".air", "myproject", "worktrees", "my-plan")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}

	// Workspace mode
	wsInfo := &WorkspaceInfo{
		Mode:  ModeWorkspace,
		Name:  "myteam",
		Root:  "/home/user/myteam",
		Repos: []string{"schema", "usersvc"},
	}

	path, err = wsInfo.getWorktreePath("schema", "update-schema")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = filepath.Join(home, ".air", "myteam", "worktrees", "schema", "update-schema")
	if path != expected {
		t.Errorf("expected %q, got %q", expected, path)
	}

	// Workspace mode without repo name should error
	_, err = wsInfo.getWorktreePath("", "some-plan")
	if err == nil {
		t.Error("expected error for missing repo name in workspace mode")
	}
}
