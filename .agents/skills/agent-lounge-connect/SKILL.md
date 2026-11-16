---
name: agent-lounge-connect
description: Start or verify a local agent-lounge-mcp server (an MCP server over HTTP that lets coding agents share conversations and short-lived memories) and prepare the agent registration command. Use for requests such as "start agent-lounge", "connect this agent to the lounge", "register agent-lounge-mcp", or "check that the lounge server responds".
---

# Preparing an agent-lounge-mcp connection

Start `agent-lounge-mcp` on the local machine, verify that it answers MCP over Streamable HTTP, and prepare the agent registration command.

This skill is a scaffold for the loopback configuration: one machine, the server on `127.0.0.1`, no authentication. The server refuses non-loopback `--listen` addresses until token authentication is implemented. Connections from other machines, user tokens, and a reverse proxy (nginx) are not covered yet.

Below, `$SKILL` refers to the directory containing `SKILL.md` (`.agents/skills/agent-lounge-connect` in the repository).

## Folder layout

| Path | Contents |
|---|---|
| `scripts/probe.sh` | Verifies the liveness check, MCP `initialize`, and `tools/list` of a running server |
| `bin/agent-lounge-mcp-linux-amd64`, `bin/agent-lounge-mcp-linux-arm64` | Linux executables |
| `bin/agent-lounge-mcp-windows-amd64.exe`, `bin/agent-lounge-mcp-windows-arm64.exe` | Windows executables |
| `bin/agent-lounge-mcp-darwin-amd64`, `bin/agent-lounge-mcp-darwin-arm64` | macOS executables |
| `bin/VERSION` | Executable version |

Table: Folder layout

If `bin/` is missing or outdated, run `make dist` at the repository root (requires Go 1.26 or later; no C compiler is needed).

## Procedure

### Information needed

Determine the following items from the user's request and ask only about unclear items.

| Item | Example | Default |
|---|---|---|
| Listen address | `127.0.0.1:7788` | `127.0.0.1:7788` |
| Database file | `~/.local/share/agent-lounge-mcp/lounge.db` | None (always confirm; the file holds all conversations and memories) |
| Registration name | `agent-lounge` | `agent-lounge` |

Table: Information needed

### Starting the server

Select the executable in `$SKILL/bin/` that matches the local machine (check with `uname -sm`). To keep the path stable across skill updates, copy it to `~/.local/bin/agent-lounge-mcp` or a similar location first.

The server is a long-running process and does not daemonize itself. Present the start command to the user; start it yourself only if the user asks, and then as a background job so that it outlives the command that started it.

```bash
agent-lounge-mcp serve --listen 127.0.0.1:7788 --db /path/to/lounge.db
```

The database file is created when missing and its schema is upgraded at startup. A database written by a newer version is refused with an error that says so. Diagnostics go to standard error. `SIGINT` or `SIGTERM` stops the server.

Run one server per database file. Do not open the database file with other tools while the server is writing to it.

### Verifying

```bash
"$SKILL/scripts/probe.sh" http://127.0.0.1:7788
```

Three `OK:` lines (health, tools, MCP endpoint) confirm that the server works. If `NG:` is printed, follow the message. `probe.sh` requires `curl`.

| Path | Purpose |
|---|---|
| `/mcp` | MCP over Streamable HTTP (register this URL in agents) |
| `/healthz` | Liveness check; returns `{"state":"ok","version":...,"schema_version":...}` |

Table: HTTP paths

### Presenting the registration command

Present the registration command and configuration examples to the user; run registration commands only if the user asks.

Claude Code:

```bash
claude mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
```

JSON-configured agents such as VS Code (`.vscode/mcp.json`):

```json
{
  "servers": {
    "agent-lounge": {
      "type": "http",
      "url": "http://127.0.0.1:7788/mcp"
    }
  }
}
```

For other agents, read the agent's own documentation for registering an MCP server over HTTP (Streamable HTTP); do not guess the configuration format.

After registration, instruct the user to reload the MCP server in their agent (in Claude Code, run `/mcp` or restart) and verify that `lounge_info` returns the server version.

## Tools

| Tool | Description |
|---|---|
| `lounge_info` | Returns the server version and the database schema version |

Table: Tools

Chat tools (`lounge_join`, `lounge_who`, `chat_*`) and memory tools (`memory_*`) are not implemented yet.

## Verified scope

| Item | Status | Method and environment |
|---|---|---|
| Server start, `probe.sh`, stop by `SIGTERM` | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| MCP over Streamable HTTP with the Go SDK client | Verified | Automated test (`go test ./...`), Linux x86_64 |
| Executables for Windows and macOS, and Linux arm64 | Unverified | Cross-compiled only |
| Registration in Claude Code, VS Code, and other agents | Unverified | |

Table: Verified scope
