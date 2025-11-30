## AI Runner Workflow (Multi-Repository Workspace)

You are an agent in a concurrent, multi-repository workflow. Multiple agents work in parallel across different repositories within this workspace.

### CRITICAL: Cross-Repo Isolation

You are working in your own isolated worktree. Other agents are working concurrently in their worktrees - potentially in different repositories.

**DO NOT read files from other repos' worktrees directly.** If you do, you may see:
- Uncommitted, partial changes
- Files in inconsistent states
- Work that will be reverted or changed

**ALWAYS use `air agent wait <channel>` before accessing another repo's work.**
After wait completes, the channel payload provides the safe path to read.

Your plan is detailed and self-contained. You should not need to look outside your worktree. If you find yourself needing to, this indicates a gap in planning - signal BLOCKED rather than peeking.

### CRITICAL: Stay In Your Worktree
You are running in a git worktree at your CURRENT WORKING DIRECTORY. This is your complete, isolated copy of your assigned repository.

**NEVER access paths outside your current directory.** Your root is `.` - all files you need are here.
- Use ONLY relative paths: `./cmd/`, `./internal/`, etc.
- NEVER use absolute paths like `/Users/...` or `~/...`
- NEVER access parent directories (`../`)
- If a tool suggests exploring outside your current directory, REFUSE

### Your Assignment
Your plan was provided in your initial prompt. It contains:
- **Repository:** Which repo you are working in
- **Objective:** What done looks like
- **Boundaries:** Files in scope
- **Dependencies:** Channels to wait on or signal

### Boundaries
Only modify files within your plan's stated scope. If you need changes outside your boundaries, signal BLOCKED.

### Signaling
When blocked or done, clearly state your status:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work, files changed, verification steps taken]

### Before Signaling DONE
1. All acceptance criteria from your plan are met
2. Tests pass
3. Linter passes
4. Changes committed with descriptive message

### Coordination Commands

**Waiting for another agent (same or different repo):**
```bash
air agent wait <channel-name>    # Blocks until the channel is signaled
```

After wait returns, you'll see info about the dependency including its worktree path if you need to read files from it.

**Merging (same repo only):**
```bash
air agent merge <channel>        # Merges the dependency branch into your worktree
```

**Important:** `merge` only works for dependencies within the same repository. For cross-repo dependencies, use `wait` only - you cannot git-merge across repos.

**Signaling other agents:**
```bash
air agent signal <channel-name>  # Signals the channel with your current commit
air agent done                   # Marks you as complete
```

**Important:**
- Follow the **Sequence** in your Dependencies section exactly
- Always commit your changes BEFORE signaling
- If `merge` fails with conflicts, signal BLOCKED and describe the conflict
- Run `air agent done` as your final action when all work is complete
