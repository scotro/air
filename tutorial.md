# Tutorial: Build a Redis Clone with Air

Build a working toy Redis server in ~1 hour using parallel AI agents.

## What You'll Build

A Go implementation of Redis that supports:
- `SET key value` / `GET key`
- `LPUSH`, `RPUSH`, `LRANGE` for lists
- `HSET`, `HGET`, `HGETALL` for hashes
- `EXPIRE`, `TTL` for key expiration

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed
- [Go 1.21+](https://go.dev/doc/install)
- [tmux](https://github.com/tmux/tmux/wiki/Installing) on mac, `brew install tmux`
- git

## 1. Create Your Project

```bash
mkdir mini-redis && cd mini-redis
go mod init mini-redis
git init
```

Before parallelizing work, establish your project foundation. Create a minimal `main.go`:

```go
// main.go
package main

func main() {
	// TODO: start server
}
```

Commit the foundation:

```bash
git add .
git commit -m "Initial project structure"
```

## 2. Initialize Air

```bash
air init
```

This creates `~/.air/mini-redis/` with context for agent coordination.

## 3. Plan the Work

```bash
air plan
```

Claude will ask what you want to build. Describe the Redis clone:

> I want to build a minimal Redis clone in Go. It should:
> - Listen on port 6379 using the RESP protocol
> - Support string commands: SET, GET, DEL
> - Support list commands: LPUSH, RPUSH, LRANGE
> - Support hash commands: HSET, HGET, HGETALL
> - Support TTL: EXPIRE, TTL commands with background expiration
> - Be safe for concurrent access

Claude will decompose this into 3-4 parallel plans and write them to `~/.air/mini-redis/plans/`. Typical decomposition:

| Plan | Scope |
|------|-------|
| `core` | RESP parser, TCP server, main.go |
| `strings` | GET/SET/DEL with thread-safe storage |
| `collections` | List and hash data structures |
| `ttl` | EXPIRE/TTL commands, background reaper |

Review the plans:

```bash
air plan list
air plan show core
```

## 4. Launch the Agents

```bash
air run all
```

This creates isolated git worktrees and launches Claude agents in tmux:
- Each agent works in its own branch (`air/core`, `air/strings`, etc.)
- Agents auto-accept file edits (use `--no-auto-accept` for manual approval)
- A `dash` window is available for running commands yourself

**tmux basics:**
- `Ctrl+b n` - next window
- `Ctrl+b p` - previous window
- `Ctrl+b d` - detach (agents keep running)
- `tmux attach -t air` - reattach

## 5. Monitor Progress

Watch the agents work. They'll signal when done:

```
DONE: Implemented RESP parser and TCP server. Tests passing.
```

If an agent signals `BLOCKED`, provide guidance in that window.

Check status from the `dash` window:

```bash
air status
```

## 6. Integrate the Work

Once agents are done, detach from tmux (`Ctrl+b d`) and run:

```bash
air integrate
```

Claude helps merge each branch:

```bash
# Typical flow for each branch:
git merge air/core --no-ff -m "Merge core: RESP parser and server"
git merge air/strings --no-ff -m "Merge strings: GET/SET/DEL commands"
git merge air/collections --no-ff -m "Merge collections: lists and hashes"
git merge air/ttl --no-ff -m "Merge ttl: expiration support"
```

Run tests after each merge to catch conflicts early.

## 7. Test Your Redis

Build and run:

```bash
go build -o mini-redis .
./mini-redis
```

In another terminal, use `redis-cli` (or `nc`):

```bash
redis-cli -p 6379
> SET hello world
OK
> GET hello
"world"
> EXPIRE hello 10
(integer) 1
> TTL hello
(integer) 9
> LPUSH mylist a b c
(integer) 3
> LRANGE mylist 0 -1
1) "c"
2) "b"
3) "a"
```

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
- Add `INCR`/`DECR` commands
- Implement pub/sub with `SUBSCRIBE`/`PUBLISH`
- Add persistence with RDB snapshots

Each extension is another `air plan` + `air run` cycle.
