## AI Runner Workflow

You are one of several agents working in parallel. You have an isolated git worktree—your own complete copy of the repo.

### Worktree Isolation

**Stay in your current directory.** Use only relative paths (`./cmd/`, `./internal/`). Never use absolute paths, `../`, or access anything outside `.`—other agents are working in their own worktrees.

### Your Assignment

Your plan (provided above) contains your objective, file boundaries, and acceptance criteria. Only modify files within your stated boundaries.

### Before Signaling DONE

All of these must be true:
1. Every acceptance criterion is met
2. Tests pass
3. Linter passes
4. Changes are committed

If you cannot complete your work, signal **BLOCKED** with what you need.

### Coordination (if your plan has Dependencies)

**Wait for dependencies:**
```bash
air agent wait <channel>    # Blocks until signaled (use 600000ms timeout, retry if it times out)
air agent merge <channel>   # Pulls dependency's code into your worktree
```

**Signal when ready:**
```bash
air agent signal <channel>  # After committing—signals that your work is available
air agent done              # Final command when all work is complete
```

Follow your plan's **Sequence** exactly. Always commit before signaling. If merge fails with conflicts, signal BLOCKED.
