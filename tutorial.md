# Tutorial: Build a Redis Clone with Air

Build a working toy Redis server in 30 minutes using concurrent AI agents.

## What You'll Build

A Go implementation of Redis that supports:
- `SET key value` / `GET key` / `DEL key`
- `INCR`, `DECR` for counters
- `HSET`, `HGET`, `HGETALL` for hashes
- `EXPIRE`, `TTL` for key expiration

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed
- [Go 1.21+](https://go.dev/doc/install)
- [tmux](https://github.com/tmux/tmux/wiki/Installing) on mac, `brew install tmux`
- git

## 1. Create Your Project

```bash
mkdir air-tutorial && cd air-tutorial
git init
git commit --allow-empty -m "Initial commit"
```

## 2. Initialize Air

```bash
air init
```

This creates `~/.air/air-tutorial/` with context for agent coordination.

**Optional but recommended:** Add permissions for Go tooling so agents can build and test without prompts:

```bash
mkdir -p .claude
cat > .claude/settings.json << 'EOF'
{
  "permissions": {
    "allow": [
      "Bash(go build:*)",
      "Bash(go test:*)",
      "Bash(go vet:*)",
      "Bash(go fmt:*)",
      "Bash(go mod:*)",
      "Bash(go run:*)"
    ]
  }
}
EOF
git add .claude && git commit -m "Add Claude permissions for Go tooling"
```

## 3. Plan the Work

```bash
air plan
```

Claude will ask what you want to build. Describe the Redis clone:

> I want to build a minimal Redis clone in Go. It should:
> - Listen on port 6379 using the RESP protocol
> - Support string commands: SET, GET, DEL
> - Support counter commands: INCR, DECR
> - Support hash commands: HSET, HGET, HGETALL
> - Support TTL: EXPIRE, TTL commands with background expiration
> - Support safe high-throughput concurrency
> - Include a Makefile for easy build and run

Claude will decompose this into several parallel plans and write them to `~/.air/air-tutorial/plans/`. For a real project, you'd want to spend most of your time here, ensuring high quality plans.

Review the plans:

```bash
air plan list
air plan show <name>
```

## 4. Launch the Agents

```bash
air run all
```

This creates isolated git worktrees and launches Claude agents in tmux:
- Each agent works in its own branch (e.g. `air/core`, `air/strings`, etc.)
- Agents auto-accept file edits (use `--no-auto-accept` for manual approval)
- A `dash` window is available for running commands yourself

**tmux basics:**
- `Ctrl+b w` - view all windows
- `Ctrl+b n` - next window
- `Ctrl+b p` - previous window
- `Ctrl+b 0-9` - go to a specific window
- `Ctrl+b d` - detach (agents keep running)
- `tmux attach -t air` - reattach

**First launch:** Each agent window will prompt for initial approval. Use `Ctrl+b n` to cycle through windows and approve each agent to start working. This only happens once per session.

## 5. Monitor Progress

Watch the agents work. You will need to monitor agents for permission requests. To streamline things, consider adding project-level permissions for safe tool calls.

Agents will signal when done:

```
DONE: Implemented RESP parser and TCP server. Tests passing.
```

If an agent signals `BLOCKED`, provide guidance in that window.

Check status from the `dash` window:

```bash
air status
```

## 6. Integrate the Work

Once agents are done, exit or detach from tmux (`Ctrl+b d`).

You should be on your main, working copy of `air-tutorial`. To integrate all of the work that the agents have completed, run:

```bash
air integrate
```

Claude helps merge each branch.

With a real project, you'll want to ensure you are on an integration or feature branch before running this command.

## 7. Test and Iterate

Test your implementation.

HINT: Ask Claude to give you some commands to test.

Build and run:

```bash
make build
make run
```

In another terminal, test with `redis-cli`:

```bash
redis-cli -p 6379
> SET hello world
OK
> GET hello
"world"
> SET counter 10
OK
> INCR counter
(integer) 11
> DECR counter
(integer) 10
> HSET user:1 name Alice age 30
(integer) 2
> HGETALL user:1
1) "name"
2) "Alice"
3) "age"
4) "30"
```

**If something doesn't work:**

Run `claude` in your project directory and describe the issue:

```
> INCR on a non-numeric key is crashing - investigate and fix
```

This iterative debugging is a normal part of AI-assisted development.

## 8. Clean Up

```bash
air clean --branches
```

Removes worktrees and deletes the `air/*` branches.

## Tips

- **Start small**: 3-4 parallel agents is ideal. More creates coordination overhead.
- **Clear boundaries**: Good plans have explicit file/package ownership.
- **Check early**: Glance at agents every 10-15 minutes to catch drift.
- **Test incrementally**: Merge and test one branch at a time.

## What's Next?

Try extending your Redis clone:
- Implement pub/sub with `SUBSCRIBE`/`PUBLISH`
- Add persistence with RDB snapshots

Each extension is another `air plan` + `air run` cycle.
