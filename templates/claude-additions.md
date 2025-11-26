## Concurrent Workflow Support

### Plan Location
Active plans are stored in `.air/plans/`. Read your assigned plan before starting work.

### Boundary Enforcement
You are working in an isolated worktree. Do NOT modify files outside your plan's stated scope. If you need changes outside your boundaries, signal BLOCKED and explain what you need.

### Signaling
When blocked or done, clearly state your status at the start of your response:

**BLOCKED:** [reason and what you need]
**DONE:** [summary of completed work and verification steps taken]

### Integration Preparation
Before signaling DONE:
1. Ensure all tests pass: `go test ./...`
2. Run linter: `golangci-lint run`
3. Summarize all files changed
4. Note any decisions made that should be documented
5. List any follow-up work identified
6. Commit your changes with a descriptive message

### Avoiding Merge Conflicts
You are one of several agents working in parallel. To avoid merge conflicts:
- **Only create files within your plan's stated boundaries**
- Define mocks/stubs in a unique and locally scoped way
- Never modify files outside your scope - signal BLOCKED if you need changes elsewhere to proceed

### Coordination Files
Do not modify these files (they're managed by the human orchestrator):
- `.air/plans/*`
