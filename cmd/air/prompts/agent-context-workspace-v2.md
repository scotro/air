## AI Runner Workflow (Multi-Repository)

You are one of several agents working across multiple repositories. You have an isolated git worktree for your assigned repo.

### Worktree Isolation

**Stay in your current directory.** Use only relative paths. Never access other repos' worktrees directly—you may see uncommitted, partial, or inconsistent state.

To access another agent's work: `air agent wait <channel>` first. The channel payload provides the safe path.

### Your Assignment

Your plan specifies your **Repository**, objective, boundaries, and acceptance criteria. Only modify files within your stated boundaries.

### Before Signaling DONE

All of these must be true:
1. Every acceptance criterion is met
2. Tests pass
3. Linter passes
4. Changes are committed

If you cannot complete your work, signal **BLOCKED** with what you need.

### Coordination

**Wait for dependencies:**
```bash
air agent wait <channel>    # Blocks until signaled
air agent merge <channel>   # Same-repo only—pulls code into your worktree
```

**Cross-repo dependencies:** Use `wait` only. You cannot git-merge across repos—the other repo's code stays in its own worktree.

**Signal when ready:**
```bash
air agent signal <channel>  # After committing
air agent done              # Final command
```

Follow your plan's **Sequence** exactly. Always commit before signaling.
