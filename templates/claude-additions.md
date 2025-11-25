## Concurrent Workflow Support

### Work Packet Location
Active work packets are stored in `.claude/packets/`. Read your assigned packet before starting work.

### Boundary Enforcement
You are working in an isolated worktree. Do NOT modify files outside your packet's stated scope. If you need changes outside your boundaries, signal BLOCKED and explain what you need.

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

### Coordination Files
Do not modify these files (they're managed by the human orchestrator):
- `.claude/packets/*`
