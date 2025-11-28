package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectMode_SingleRepo(t *testing.T) {
	// Create a temporary directory with a .git folder
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to the temp directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	info, err := detectMode()
	if err != nil {
		t.Fatalf("detectMode() failed: %v", err)
	}

	if info.Mode != ModeSingle {
		t.Errorf("expected ModeSingle, got %v", info.Mode)
	}
	if info.Name != filepath.Base(tmpDir) {
		t.Errorf("expected name %q, got %q", filepath.Base(tmpDir), info.Name)
	}
	// Use EvalSymlinks to handle macOS /var -> /private/var symlink
	expectedRoot, _ := filepath.EvalSymlinks(tmpDir)
	actualRoot, _ := filepath.EvalSymlinks(info.Root)
	if actualRoot != expectedRoot {
		t.Errorf("expected root %q, got %q", expectedRoot, actualRoot)
	}
	if len(info.Repos) != 0 {
		t.Errorf("expected empty repos, got %v", info.Repos)
	}
}

func TestDetectMode_Workspace(t *testing.T) {
	// Create a temporary workspace directory with child repos
	tmpDir := t.TempDir()

	// Create child repos
	repos := []string{"authapi", "schema", "usersvc"}
	for _, repo := range repos {
		repoDir := filepath.Join(tmpDir, repo)
		if err := os.Mkdir(repoDir, 0755); err != nil {
			t.Fatal(err)
		}
		gitDir := filepath.Join(repoDir, ".git")
		if err := os.Mkdir(gitDir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a non-repo directory (should be ignored)
	if err := os.Mkdir(filepath.Join(tmpDir, "docs"), 0755); err != nil {
		t.Fatal(err)
	}

	// Change to the workspace directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	info, err := detectMode()
	if err != nil {
		t.Fatalf("detectMode() failed: %v", err)
	}

	if info.Mode != ModeWorkspace {
		t.Errorf("expected ModeWorkspace, got %v", info.Mode)
	}
	if len(info.Repos) != 3 {
		t.Errorf("expected 3 repos, got %d: %v", len(info.Repos), info.Repos)
	}
	// Repos should be sorted
	expected := []string{"authapi", "schema", "usersvc"}
	for i, r := range expected {
		if info.Repos[i] != r {
			t.Errorf("expected repos[%d] = %q, got %q", i, r, info.Repos[i])
		}
	}
}

func TestDetectMode_NeitherRepoNorWorkspace(t *testing.T) {
	// Create a temporary empty directory
	tmpDir := t.TempDir()

	// Change to the temp directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	_, err := detectMode()
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
}

func TestDetectMode_SkipsHiddenDirs(t *testing.T) {
	// Create a workspace with hidden and visible repos
	tmpDir := t.TempDir()

	// Create visible repo
	visibleRepo := filepath.Join(tmpDir, "myrepo")
	os.Mkdir(visibleRepo, 0755)
	os.Mkdir(filepath.Join(visibleRepo, ".git"), 0755)

	// Create hidden repo (should be ignored)
	hiddenRepo := filepath.Join(tmpDir, ".hidden")
	os.Mkdir(hiddenRepo, 0755)
	os.Mkdir(filepath.Join(hiddenRepo, ".git"), 0755)

	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(tmpDir)

	info, err := detectMode()
	if err != nil {
		t.Fatalf("detectMode() failed: %v", err)
	}

	if len(info.Repos) != 1 {
		t.Errorf("expected 1 repo, got %d: %v", len(info.Repos), info.Repos)
	}
	if info.Repos[0] != "myrepo" {
		t.Errorf("expected 'myrepo', got %q", info.Repos[0])
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
