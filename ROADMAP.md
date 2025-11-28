# Air Roadmap: Production Readiness

Suggestions for maturing Air into a production-ready tool.

## Reducing Prompt Engineering Fragility

### 1. Schema-enforced plans

Instead of regex parsing markdown, consider a structured format:

```yaml
# ~/.air/<project>/plans/core.yaml
objective: "Implement RESP parser"
boundaries:
  in_scope: ["pkg/resp/"]
  out_of_scope: ["pkg/storage/", "cmd/"]
acceptance:
  - "Parser handles bulk strings"
  - "Unit tests pass"
dependencies:
  waits_on: ["setup-complete"]
  signals: ["core-complete"]
```

Benefits:
- `air run` can validate the schema before launch
- Agents receive structured data rather than prose they have to parse
- Prompts become shorter: "Your task is defined in the structured plan below. Follow it exactly."
- Easier to build tooling around (IDE extensions, validation, visualization)

### 2. Agent contracts via tooling

Extend agent self-reporting beyond `air agent done`:

```bash
air agent checkpoint "finished parser, starting tests"
air agent progress 60  # percentage
air agent blockedBy "need clarification on X"
```

These write to `~/.air/<project>/agents/<id>/state.json`. Benefits:
- `air status` has real data to show
- Can detect stuck agents programmatically
- Creates an audit trail of agent work

### 3. Pre-flight validation

Before `air run` launches agents, verify:
- All files in `in_scope` exist (or are expected to be created)
- No two plans have overlapping `in_scope` paths
- The dependency DAG is executable
- Required tools are available

This catches plan bugs before agents waste time.

## Production Robustness

### 4. Agent health monitoring

```go
// In a goroutine during air run
for {
    for _, agent := range agents {
        if time.Since(agent.LastCheckpoint) > 10*time.Minute {
            warn("Agent %s may be stuck", agent.ID)
        }
    }
    time.Sleep(1 * time.Minute)
}
```

NOTE: Assess the feasibility of this feature.. it depends on hooking into claude somehow

Could also expose via `air status --watch` for real-time monitoring.

### 5. Graceful degradation

```bash
air run --no-tmux  # fallback to sequential execution
air run --resume   # restart failed agents from last checkpoint
```

Handle edge cases:
- tmux not installed
- Agent crashes mid-execution
- Disk full scenarios
- Network issues during git operations

### 6. Conflict prediction

Before integration, run `git merge-tree` on all branch pairs to predict conflicts. Surface this in `air status`:

```
Branches ready to merge:
  air/setup    ✓ clean
  air/core     ✓ clean
  air/strings  ⚠ conflicts with air/core in pkg/commands/strings.go
```

This lets users address conflicts proactively rather than discovering them during integration.

## Quick Wins for Adoption

### 7. `air doctor`

Diagnose the environment:

```
$ air doctor
✓ git 2.40.0
✓ tmux 3.3a
✓ claude cli 1.0.0
✗ SSH agent not running (git push may fail)
```

Low effort, high value for debugging user issues.

### 8. `air logs <agent>`

Tail the agent's conversation history or activity log. Useful for:
- Debugging stuck agents
- Understanding what an agent did
- Post-mortem analysis

Before building this... consider, is this necessary? Users will probably go directly to the claude code session to troubleshoot.

### 9. `--verbose` flag

Show what commands are being constructed. Useful for:
- Debugging prompt issues
- Understanding the tool's behavior
- Filing bug reports with full context

## Prompt Architecture

Instead of monolithic prompt strings, consider modular composition:

```go
// Instead of one massive string
var orchestrationPrompt = buildPrompt(
    sections.Role,
    sections.Decomposition,
    sections.PlanFormat,
    sections.DependencyRules,
    sections.ValidationReminder,
)
```

Benefits:
- A/B test individual sections
- Version control prompt changes granularly
- Let users override specific parts via `~/.air/<project>/prompts/`
- Easier to maintain and iterate

## Testing Gaps to Address

Current tests cover happy paths well. Add coverage for:
- Merge conflicts during `air agent merge`
- Missing dependencies (tmux not installed)
- Agent crashes mid-execution
- Disk full scenarios
- Malformed plan files
- Concurrent access to channel files
- Git operations failing (network, permissions)
