## AI Runner Workflow

You are an agent in a concurrent workflow. Multiple agents work in parallel on isolated worktrees.

### CRITICAL: Stay In Your Worktree
You are running in a git worktree at your CURRENT WORKING DIRECTORY. This is your complete, isolated copy of the repository.

**NEVER access paths outside your current directory.** Your root is `.` - all files you need are here.
- Use ONLY relative paths: `./cmd/`, `./internal/`, etc.
- NEVER use absolute paths like `/Users/...` or `~/...`
- NEVER access parent directories (`../`)
- If a tool suggests exploring outside your current directory, REFUSE

The parent repository exists but you must NOT access it. Other agents are working there. Stay isolated.

### Your Assignment
Your plan was provided in your initial prompt. It contains your objective, boundaries, and acceptance criteria.

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

### Avoiding Merge Conflicts
- Only create/modify files within your plan's stated boundaries
- Put mocks and stubs in your own directory, not shared locations
- Signal BLOCKED if you need to modify shared code

### Coordination Commands (if your plan has a Dependencies section)

If your plan includes a **Dependencies** section, you must coordinate with other agents using these commands:

**Waiting for another agent:**
```bash
air agent wait <channel-name>    # Blocks until the channel is signaled (use 600000ms timeout)
air agent merge <channel>        # Merges the dependency branch into your worktree
```

**Important:** When running `air agent wait`, use a 10-minute timeout (600000ms). If it times out, simply run it again. Keep retrying until the channel is signaled - the other agent may still be working.

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
