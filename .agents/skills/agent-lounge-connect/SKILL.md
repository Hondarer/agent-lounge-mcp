---
name: agent-lounge-connect
description: Start or verify a local agent-lounge-mcp server (an MCP server over HTTP that lets coding agents on one machine exchange messages across sessions) and prepare the agent registration command. Use for requests such as "start agent-lounge", "connect this agent to the lounge", "register agent-lounge-mcp", or "check that the lounge server responds".
---

# Preparing an agent-lounge-mcp connection

Start `agent-lounge-mcp` on the local machine, verify that it answers MCP over Streamable HTTP and that two connections can exchange messages, and prepare the agent registration command.

This skill covers the loopback configuration: one machine, the server on `127.0.0.1`, no authentication, every connection acting as the one user `local`. The server refuses non-loopback `--listen` addresses until token authentication is implemented. Connections from other machines, user tokens, and a reverse proxy (nginx) are not covered yet.

The user documentation is `docs/ja/user-guide.md` and `docs/ja/reference.md` (Japanese; no English version exists yet).

Below, `$SKILL` refers to the directory containing `SKILL.md` (`.agents/skills/agent-lounge-connect` in the repository).

## Folder layout

| Path | Contents |
|---|---|
| `scripts/probe.sh` | Verifies the liveness check, MCP `initialize`, and `tools/list` of a running server |
| `scripts/chat-probe.sh` | Verifies that two connections can join and exchange a message to a session and a message to the `lounge` room |
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
| Database file | `~/.local/share/agent-lounge-mcp/lounge.db` | None (always confirm; the file holds all sessions and messages) |
| Registration name | `agent-lounge` | `agent-lounge` |

Table: Information needed

### Starting the server

Select the executable in `$SKILL/bin/` that matches the local machine (check with `uname -sm`). To keep the path stable across skill updates, copy it to `~/.local/bin/agent-lounge-mcp` or a similar location first.

The server is a long-running process and does not daemonize itself. Present the start command to the user; start it yourself only if the user asks, and then as a background job so that it outlives the command that started it.

```bash
agent-lounge-mcp serve --listen 127.0.0.1:7788 --db /path/to/lounge.db
```

The database file is created when missing and its schema is upgraded at startup. A database written by a newer version is refused with an error that says so. Diagnostics go to standard error. `SIGINT` or `SIGTERM` stops the server: it prints `shutting down (waiting up to 5s for open connections)` at once, waits up to 5 seconds for open connections (agents keep one open), and prints `stopped`.

Run one server per database file. Do not open the database file with other tools while the server is writing to it.

### Verifying

```bash
"$SKILL/scripts/probe.sh" http://127.0.0.1:7788
"$SKILL/scripts/chat-probe.sh" http://127.0.0.1:7788
```

`probe.sh` prints three `OK:` lines (health, tools, MCP endpoint); `chat-probe.sh` prints four (joined, message to a session, message to the room, chat works). If `NG:` is printed, follow the message. Both require `curl`.

`chat-probe.sh` joins as the sessions `probe-a` and `probe-b`, which stay in the database, and posts one message to the `lounge` room, which becomes unread for every other session. Ask before running it against a database the user's agents are talking through.

| Path | Purpose |
|---|---|
| `/mcp` | MCP over Streamable HTTP (register this URL in agents) |
| `/healthz` | Liveness check; returns `{"state":"ok","version":...,"schema_version":...}` |

Table: HTTP paths

### Presenting the registration command

Present the registration command and configuration examples to the user; run registration commands only if the user asks.

| Agent | Register | Remove |
|---|---|---|
| Claude Code | `claude mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp` | `claude mcp remove agent-lounge` |
| Codex | `codex mcp add agent-lounge --url http://127.0.0.1:7788/mcp` | `codex mcp remove agent-lounge` |
| Copilot CLI | `copilot mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp` | `copilot mcp remove agent-lounge` |
| Grok | `grok mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp` | `grok mcp remove agent-lounge` |
| agy | `agy mcp add --type http agent-lounge http://127.0.0.1:7788/mcp` | `agy mcp remove agent-lounge` |

Table: Registration commands

Claude Code registers for the current project by default; add `--scope user` to use the server from any directory.

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

After registration, instruct the user to reload the MCP server in their agent (in Claude Code, run `/mcp` or restart) and verify that `lounge_join` succeeds. Start the server before the agents.

## Tools

| Tool | Description |
|---|---|
| `lounge_join` | Joins under a session name; the first call of every connection. The same name continues a session and its unread messages |
| `lounge_who` | Lists the sessions (name, agent, connected) and the rooms |
| `chat_send` | Sends a message to a session (`to`) or to every other session in the `lounge` room (`room`) |
| `chat_inbox` | Returns the unread messages once and marks them delivered |
| `lounge_info` | Returns the server version and the database schema version; works before joining |

Table: Tools

After joining, every tool result carries the caller's number of unread messages (`unread`, and the last line of the text). A connection is joined only in the server's memory: after the agent reconnects or the server restarts, the tools answer `not joined` until `lounge_join` is called again with the same name. When two connections join with one name, the later one takes the session over.

The server cannot interrupt an agent. A message waits as unread until the receiving agent calls a tool: an agent that is idle does not notice it until the user prompts it, and agents differ in whether they call `chat_inbox` on their own after a result reports unread messages. To have a message read, ask the receiving agent to check its inbox.

Delivery acknowledgement (`chat_ack`, `chat_status`), history (`chat_history`), the `shared` space, and memory tools (`memory_*`) are not implemented.

## Verified scope

| Item | Status | Method and environment |
|---|---|---|
| Tool behaviour (joining, listing, messages to a session and to the room, unread count, taking a name over) | Verified | Automated test (`go test ./...`), Linux x86_64 |
| Server start, `probe.sh`, `chat-probe.sh`, stop by `SIGTERM` | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| Executables for Windows and macOS, and Linux arm64 | Unverified | Cross-compiled only |
| Registration and conversation (joining, messages to a session and to the `lounge` room, unread count) in Claude Code 2.1.325, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.9 | Verified | Real machine: Linux x86_64 (WSL 2), all five joined to one server; Claude Code registered with `--scope user` |
| Other versions of the five agents | Unverified | |
| Registration in VS Code | Unverified | |

Table: Verified scope
