# Concurrent AI Agent Workflow

A systematic approach to managing multiple AI coding agents for professional software development.

## Core Principles

**Your attention is the bottleneck, not AI capacity.** This workflow optimizes for human cognitive bandwidth by batching supervisory work into discrete rounds rather than continuous context-switching.

**Agents are fast but context-limited.** Treat them like capable junior engineers who need clear direction, explicit boundaries, and regular check-ins.

**Isolation prevents interference.** Each agent operates in its own environment. Coordination happens through you, not between agents.

---

## The Rounds Pattern

Work flows through three types of rounds, cycling as needed:

### Setup Round (Serial, Human-Heavy)

**Duration:** 15-45 minutes depending on complexity  
**Goal:** Decompose work into parallelizable packets

1. Define the overall objective for this work session
2. Identify natural decomposition points (by feature, layer, or domain)
3. Map dependencies between packets (independent, sequential, soft)
4. Write work packets using the template in `TEMPLATES.md`
5. Prepare isolated environments (git worktrees)
6. Dispatch agents with initial context

**Exit criteria:** All agents have clear direction and are running.

### Execution Rounds (Parallel, Agent-Heavy)

**Duration:** 20-30 minutes between rounds  
**Goal:** Unblock agents with minimal intervention

For each active agent:
1. Check status: Running / Blocked / Done / Drifting
2. If blocked: Provide the minimum input to unblock
3. If drifting: Redirect with clarification
4. If done: Mark for integration round
5. Update the tracking dashboard

**Techniques for non-blocking supervision:**
- Use `/status` or ask "summarize progress and blockers" 
- Scan recent file changes rather than reading full conversation
- Provide decisions as constraints, not discussions
- Defer non-critical questions to integration round

**Exit criteria:** All agents unblocked and working toward objectives.

### Integration Round (Serial, Human-Heavy)

**Duration:** 30-60 minutes depending on scope  
**Goal:** Collect, verify, and merge completed work

1. Review completed work packets against acceptance criteria
2. Run integration tests across merged work
3. Identify gaps, regressions, or coordination issues
4. Document decisions made during execution
5. Update project documentation if needed
6. Define new work packets for next cycle

**Exit criteria:** Main branch updated, next cycle planned or session complete.

---

## Dependency Management

Before spawning agents, classify each work packet:

### Independent
No dependencies on other packets. Fully parallelizable.
```
[Auth Service] ←──────→ [Email Templates]
     ↓                        ↓
  (no connection, run simultaneously)
```

### Sequential  
Output of A required before B can start.
```
[Database Schema] → [Repository Layer] → [API Endpoints]
```
Strategy: Complete earlier packets before spawning dependent agents.

### Soft Dependencies
Can run in parallel with assumptions, reconcile later.
```
[API Design] ~~~~ [Frontend Components]
     ↓                    ↓
  (both proceed with agreed interface contract)
```
Strategy: Establish interface contract upfront, run in parallel, reconcile in integration round.

---

## Environment Setup

### Git Worktree Structure

```bash
project/
├── .git/                    # Shared git state
├── main/                    # Main working directory
│   ├── CLAUDE.md
│   └── ...
├── worktrees/
│   ├── agent-auth/          # Agent 1: Authentication work
│   ├── agent-api/           # Agent 2: API development
│   └── agent-tests/         # Agent 3: Test coverage
```

### Creating Worktrees

```bash
# From project root
mkdir -p worktrees
git worktree add worktrees/agent-auth -b feature/auth
git worktree add worktrees/agent-api -b feature/api

# Launch agents
cd worktrees/agent-auth && claude
cd worktrees/agent-api && claude
```

### Cleanup

```bash
# After merging
git worktree remove worktrees/agent-auth
git branch -d feature/auth
```

---

## Session Workflow

### Starting a Session

1. Review project state and decide on session goals
2. Pull latest changes to main
3. Complete Setup Round (see above)
4. Create/update `SESSION.md` with today's objectives

### During a Session

1. Run Execution Rounds on your chosen cadence (20-30 min)
2. Use the tracking dashboard to maintain awareness
3. Take breaks between rounds (agents continue working)

### Ending a Session

1. Complete Integration Round
2. Clean up worktrees for completed work
3. Document any work-in-progress for next session
4. Update `SESSION.md` with outcomes and next steps

---

## Multi-Project Workflow

When working across multiple projects, add time-boxing:

### Option A: Time Blocks
```
09:00-12:00  Project A agents (morning rounds)
13:00-17:00  Project B agents (afternoon rounds)
19:00-21:00  Personal project agents (evening rounds)
```

### Option B: Interleaved Rounds
```
Round 1: Check Project A agents
Round 2: Check Project B agents  
Round 3: Check Project A agents
...
```

Use Option A when projects require deep focus. Use Option B when work is more routine and context-switching cost is low.

---

## Risk Calibration

Adjust supervision intensity based on risk:

### Autonomous Mode (Low Risk)
- Documentation updates
- Test coverage expansion
- Linting and formatting fixes
- Dependency updates

Use `--autoyes` flag, check once per hour.

### Standard Mode (Medium Risk)  
- New feature implementation
- Refactoring existing code
- Bug fixes in non-critical paths

Check every 20-30 minutes, review before merge.

### High-Touch Mode (High Risk)
- Security-sensitive code
- Core business logic
- Database migrations
- Public API changes

Check every 10-15 minutes, pair-review before merge.

---

## Anti-Patterns to Avoid

**Hovering:** Checking agents every 5 minutes breaks your focus and doesn't help them.

**Under-specifying:** Vague work packets cause drift. Invest time in setup.

**Skipping integration:** Merging without verification compounds errors.

**Too many agents:** Start with 2-3. Add more only when you can maintain quality.

**Ignoring drift:** An agent building the wrong thing wastes both your time and tokens.

**No boundaries:** Agents that can touch anything will touch everything.
