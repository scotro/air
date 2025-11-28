# Tutorial: Build an API Gateway with Air

Build a working API gateway with live dashboard in ~30 minutes using concurrent AI agents.

## What You'll Build

A Go API gateway that supports:
- **Response aggregation** - Fan out to multiple backends, merge JSON responses
- **Rate limiting** - Per-client-IP with configurable limits
- **Response caching** - Configurable TTL per endpoint
- **Timeouts** - Per-backend timeout configuration
- **CORS** - Cross-origin resource sharing support
- **Live dashboard** - Real-time request stream, latency stats, rate limit status

**Dashboard preview:**
```
┌─────────────────────────────────────────────────────────┐
│  Gateway Dashboard                    localhost:8080    │
├─────────────────────────────────────────────────────────┤
│  Requests/sec: 47    │  Avg latency: 142ms   │  ✓ 3/3   │
├─────────────────────────────────────────────────────────┤
│  Live Request Stream                                    │
│  12:04:32  GET /demo  → httpbin, users, posts  [147ms]  │
│  12:04:33  GET /demo  → RATE LIMITED (429)              │
│  12:04:33  GET /demo  → httpbin, users, posts  [89ms]   │
├─────────────────────────────────────────────────────────┤
│  Backend Health                                         │
│  ● httpbin.org      23ms    │  ● jsonplaceholder  45ms  │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed
- [Go 1.21+](https://go.dev/doc/install)
- [tmux](https://github.com/tmux/tmux/wiki/Installing) on mac, `brew install tmux`
- git

## 1. Create Your Project

```bash
mkdir air-gateway && cd air-gateway
git init
git commit --allow-empty -m "Initial commit"
```

## 2. Initialize Air

```bash
air init
```

This creates `~/.air/air-gateway/` with context for agent coordination.

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

Claude will ask what you want to build. Use this prompt:

> Build an API gateway in Go with:
> - JSON config file defining routes and backends
> - Response aggregation (fan out to multiple backends, merge JSON responses)
> - Per-client-IP rate limiting with configurable limits
> - Response caching with configurable TTL
> - Timeouts per backend
> - CORS support
> - Live dashboard showing request stream, latency stats, and rate limit status
> - Example config using public APIs (httpbin.org, jsonplaceholder.typicode.com) for zero-setup testing
>
> Dashboard: Single HTML file with embedded JS, SSE for real-time updates, Tailwind CDN. No React, no npm, no build steps.

Claude will decompose this into several parallel plans and write them to `~/.air/air-gateway/plans/`. For a real project, you'd want to spend most of your time here, ensuring high quality plans.

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
- Each agent works in its own branch (e.g. `air/setup`, `air/proxy`, etc.)
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
DONE: Implemented rate limiting and caching middleware. Tests passing.
```

Check status from the `dash` window:

```bash
air status
```

## 6. Integrate the Work

Once agents are done, exit or detach from tmux (`Ctrl+b d`).

You should be on your main, working copy of `air-gateway`. To integrate all of the work that the agents have completed, run:

```bash
air integrate
```

Claude will analyze the plans and suggest a merge strategy.

With a real project, you'll want to ensure you are on an integration or feature branch before running this command.

## 7. Test Your Gateway

Build and run.

> **NOTE:** These examples are for demonstration purposes. Ask claude for to provide some hands-on test examples.

```bash
make build
./gateway -c examples/demo.json
```

Open the dashboard in your browser:

```bash
open http://localhost:8080/dashboard
```

Test aggregation (in another terminal):

```bash
curl http://localhost:8080/demo
```

You should see a merged response from multiple backends:

```json
{
  "httpbin": {"origin": "...", "headers": {...}},
  "user": {"id": 1, "name": "Leanne Graham", ...},
  "posts": [{"id": 1, "title": "...", ...}, ...]
}
```

**Watch the dashboard light up:**

```bash
for i in {1..50}; do curl -s http://localhost:8080/demo > /dev/null; done
```

You should see:
- Live request stream updating in real-time
- Rate limiting kick in after hitting the limit
- Latency statistics across backends

**If something doesn't work:**

Run `claude` in your project directory and describe the issue:

```
> The dashboard isn't showing live updates - investigate and fix
```

This iterative debugging is a normal part of AI-assisted development.

## 8. Clean Up

```bash
air clean --branches
```

Removes worktrees and deletes the `air/*` branches.

## What's Next?

Try extending your gateway:
- Add JWT authentication middleware
- Implement circuit breaker for failing backends
- Add request/response transformation
- Support WebSocket proxying

Each extension is another `air plan` + `air run` cycle.
