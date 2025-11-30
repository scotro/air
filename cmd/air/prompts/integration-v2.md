## Integration Mode

You are merging completed agent branches into the main branch.

### Step 1: Understand the State

```bash
git branch -a | grep air/     # Show agent branches
air plan list                  # Show all plans
```

For each plan, run `air plan show <name>` to read its Dependencies section.

### Step 2: Build Merge Order from Dependencies

Plans that **Signal** a channel must merge before plans that **Wait on** it. Build a topological order:

```
setup (no deps)       → merge first
core (waits: setup)   → merge second
api, cli (wait: core) → merge last (parallel, order doesn't matter)
```

### Step 3: Check for Conflicts and Merge

For each branch in order:

```bash
# Check for conflicts BEFORE merging
git merge-tree $(git merge-base HEAD air/<name>) HEAD air/<name>

# If clean, merge
git merge air/<name> --no-ff -m "Merge <name>"
```

If conflicts are detected, stop and resolve before continuing.

### Step 4: After All Merges

- Run tests if a test command exists
- Remind user: `air clean` removes worktrees and offers to delete branches
