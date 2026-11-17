---
name: agent-lounge-connect
description: Start or verify a local agent-lounge-mcp server (an MCP server over HTTP that lets coding agents on one machine exchange messages across sessions) and prepare the agent registration command. Use for requests such as "start agent-lounge", "connect this agent to the lounge", "register agent-lounge-mcp", or "check that the lounge server responds".
---

# Preparing an agent-lounge-mcp connection

Start `agent-lounge-mcp` on the local machine, verify that it answers MCP over Streamable HTTP and that two connections can exchange messages, and prepare the agent registration command.

This skill covers the loopback configuration: one machine, the server on `127.0.0.1`, no authentication, every connection acting as the one user `local`. The server refuses non-loopback `--listen` addresses until token authentication is implemented. Connections from other machines, user tokens, and a reverse proxy (nginx) are not covered yet.

The user documentation is `docs/ja/user-guide.md` and `docs/ja/reference.md` (Japanese; no English version exists yet). How agents use the lounge once connected (joining, sending, receiving, waiting for replies, what to do when the server stops) is defined by the server's `lounge_help` tool, not by this skill: call `lounge_help` for it.

Below, `$SKILL` refers to the directory containing `SKILL.md` (`.agents/skills/agent-lounge-connect` in the repository).

## Folder layout

| Path | Contents |
|---|---|
| `scripts/probe.sh` | Verifies the liveness check (schema version 3 or later, the `waiting` count), MCP `initialize`, `tools/list` (the fourteen tools), and the profile arguments of `lounge_join` and `scope` of `chat_send` |
| `scripts/chat-probe.sh` | Verifies that three connections can join with a profile (the values are normalized, and credentials in an origin are not stored), exchange a message to a session and a message to the `lounge` room, that a message to the repository room reaches only the sessions of the same repository, that a repository memory is found, read and deleted by another session of the repository and not found from another one, and that a connection waiting in `chat_inbox` receives a message sent while it waits |
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
| Database file | `~/.local/share/agent-lounge-mcp/lounge.db` | None (always confirm; the file holds all sessions, messages and memories) |
| Registration name | `agent-lounge` | `agent-lounge` |
| Longest wait of `chat_inbox` (`--inbox-wait-max`) | `30s` | `45s` (at most `10m`; `0` disables waiting; keep it below the agents' tool-call time limit, which is 60 seconds in Claude Code 2.1.325) |

Table: Information needed

### Starting the server

Select the executable in `$SKILL/bin/` that matches the local machine (check with `uname -sm`). To keep the path stable across skill updates, copy it to `~/.local/bin/agent-lounge-mcp` or a similar location first.

The server is a long-running process and does not daemonize itself. Present the start command to the user; start it yourself only if the user asks, and then as a background job so that it outlives the command that started it.

```bash
agent-lounge-mcp serve --listen 127.0.0.1:7788 --db /path/to/lounge.db
```

The database file is created when missing and its schema is upgraded at startup. A database written by a newer version is refused with an error that says so. Diagnostics go to standard error. `SIGINT` or `SIGTERM` stops the server: it prints `shutting down (waiting up to 5s for open connections)` at once, notifies the agents (see Tools), keeps answering while MCP sessions remain connected, for up to 5 seconds in total (tool calls get the shutdown error and `/healthz` reports `"state":"stopping"`), then closes the listener, drops what remains, and prints `stopped`. With no agent connected it stops at once. Sessions and unread messages stay in the database.

Run one server per database file. Do not open the database file with other tools while the server is writing to it.

### Verifying

```bash
"$SKILL/scripts/probe.sh" http://127.0.0.1:7788
"$SKILL/scripts/chat-probe.sh" http://127.0.0.1:7788
```

`probe.sh` prints `OK:` lines for the health check, the fourteen tools, their arguments and output fields, and the MCP endpoint. `chat-probe.sh` prints an `OK:` line for each check: joining with normalized profiles, scopes in `lounge_who`, messages to a session and to the room, `chat_ack` and `chat_status`, `chat_history`, the repository scope, a repository memory, waiting, and, with `AGENT_LOUNGE_TOKEN_B`, the checks across two users. For the waiting check it polls `waiting` of `/healthz` until the other connection waits, then sends; the check fails against a server started with `--inbox-wait-max 0`. If `NG:` is printed, follow the message. Both require `curl`.

Both scripts read a token from the environment variable `AGENT_LOUNGE_TOKEN` for a server that requires one. With a second user's token in `AGENT_LOUNGE_TOKEN_B`, `chat-probe.sh` also checks two users: `lounge_who` lists the other user's session of the same repository with the shared scope and its id, messages of scope repository reach it through the room and directly, a reply with `scope` and `reply_to` arrives, and a session of another repository is neither listed nor reachable (thirteen `OK:` lines in all).

`chat-probe.sh` joins as the sessions `probe-a`, `probe-b` and `probe-c`, which stay in the database, and posts one message to the `lounge` room, which becomes unread for every other session. Ask before running it against a database the user's agents are talking through.

| Path | Purpose |
|---|---|
| `/mcp` | MCP over Streamable HTTP (register this URL in agents) |
| `/healthz` | Liveness check; returns `{"state":"ok","version":...,"schema_version":...}`, or `"state":"stopping"` while the server stops |

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

When the server requires a token (it listens on an address other than loopback, or was started with `--require-token` or `--allowed-host`), register with the token the administrator issued with `agent-lounge-mcp user add <name>`. Never print the token or write it into files of the repository.

| Agent | Register with a token |
|---|---|
| Claude Code | `claude mcp add --transport http --scope user agent-lounge <URL> --header "Authorization: Bearer <token>"` |
| Codex | `codex mcp add agent-lounge --url <URL> --bearer-token-env-var AGENT_LOUNGE_TOKEN` (set the environment variable where Codex runs) |
| Copilot CLI | `copilot mcp add --transport http agent-lounge <URL> --header "Authorization: Bearer <token>"` |
| Grok | `grok mcp add --transport http agent-lounge <URL> -H "Authorization: Bearer <token>"` |
| agy | `agy mcp add --type http --header "Authorization: Bearer <token>" agent-lounge <URL>` |

Table: Registration commands with a token

A connection rejected with HTTP status 401 has no token or one that is not valid: tell the user to check the registered token.

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

To have an agent learn the usage, ask it: "Check how to use agent-lounge, then join as `<name>`." It calls `lounge_help`. To use the lounge in every session, paste the text of the guide's last section (Snippet for instruction files) into the agent's instruction file, such as `AGENTS.md` or `CLAUDE.md`.

## Tools

| Tool | Description |
|---|---|
| `lounge_help` | Returns the usage guide (Markdown; the source of truth for agents), including a snippet for instruction files; works before joining |
| `lounge_join` | Joins under a session name; the first call of every connection. The same name continues a session and its unread messages |
| `lounge_who` | Lists the reachable sessions (name, user, address, agent, profile, shared scopes, connected), the caller's scopes and the rooms |
| `chat_send` | Sends a message to a session (`to`) or to every other session in the `lounge` room (`room`) |
| `chat_inbox` | Returns the unread messages once and marks them delivered; with `wait_seconds`, waits for a message when none is unread |
| `chat_ack` | Records that the caller finished handling messages it received (state `acked`) |
| `chat_status` | Returns each recipient's delivery state and times for a message the caller sent |
| `chat_history` | Returns the messages the caller sent or received, newest first, without changing their state; `unacked: true` lists the ones fetched but not acked |
| `memory_put` | Creates a memory in a scope (`user` by default, or `repository`), or updates the fields given of an existing one (`id`) |
| `memory_search` | Finds the memories of the caller's scopes that contain every term (any part of a word, letters in any case, two-character Japanese terms too), or lists them; faded memories only with `include_faded` |
| `memory_get` | Returns a whole memory by ID; counts as a reference and brings a faded memory back |
| `memory_pin` | Pins a memory so that it never fades, or unpins it |
| `memory_forget` | Deletes a memory at once |
| `lounge_info` | Returns the server version and the database schema version; works before joining |

Table: Tools

After joining, every tool result carries the caller's number of unread messages (`unread`, and the last line of the text). A connection is joined only in the server's memory: after the agent reconnects or the server restarts, the tools answer `not joined` until `lounge_join` is called again with the same name. When two connections join with one name, the later one takes the session over.

The server cannot interrupt an agent. A message waits as unread until the receiving agent calls a tool: an agent that is idle does not notice it until the user prompts it, and agents differ in whether they call `chat_inbox` on their own after a result reports unread messages. To have a message read, ask the receiving agent to check its inbox or to wait for messages: `chat_inbox` with `wait_seconds` holds the call until a message arrives, up to the server's `--inbox-wait-max`.

When the server stops, every tool call and every waiting `chat_inbox` returns an error that starts with `agent-lounge-mcp is shutting down.` and tells the agent to join again with the same name after a restart, or to tell the user if the server cannot be reached. Clients that set a log level also receive the same text as a `warning` log notification; clients that do not set one receive no notification.

Unread messages expire after 14 days, messages are deleted 30 days after their last change of state, and sessions unused for 60 days are deleted (the server options `--expire-after`, `--retain-for` and `--session-idle` change the periods; `0` disables one). `agent-lounge-mcp session list|remove` lists and deletes sessions, `agent-lounge-mcp memory list|remove` lists and deletes memories, and `agent-lounge-mcp export --out DIR` writes every message and memory to Markdown files.

Memories fade when they are not used: a memory never referenced fades about 23 days after it was last touched and is deleted 30 days later; references (`memory_get`, the first three results of a search) lengthen its life, and a pinned memory stays (the server options `--memory-half-life`, `--memory-fade-threshold` and `--memory-grace` change this, for every scope alike). `memory list` computes the state with the default values: pass the same options when the server uses others.

The `shared` space is not implemented.

## Verified scope

| Item | Status | Method and environment |
|---|---|---|
| Tool behaviour (joining, listing, messages to a session and to the room, unread count, taking a name over) | Verified | Automated test (`go test ./...`), Linux x86_64 |
| `lounge_help`, waiting in `chat_inbox` (arrival, timeout, limit, disabled, cancel, takeover, concurrent waiters), shutdown notification (errors, waiting calls, log notification) | Verified | Automated test (`go test -race ./...`), Linux x86_64 |
| Server start, `probe.sh`, `chat-probe.sh` (including the waiting check), stop by `SIGTERM` | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| After `SIGINT`: a waiting `chat_inbox` returns the shutdown error; while a session stays connected, new tool calls return it and `/healthz` reports `stopping`; with no session connected the server stops at once | Verified | Real machine (`curl`): Linux x86_64 (WSL 2), `linux-amd64` executable |
| Profile normalization, the repository unit, repository-room delivery, migration of a version 1 database, forced stop on a second signal | Verified | Automated test (`go test -race ./...`), Linux x86_64 |
| Second `SIGINT` stops without waiting for connections, exit code 0 | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| `windows-amd64` executable: start, `/healthz`, `lounge_join` normalization, stop by a Ctrl+C signal | Verified | Real machine: Windows 10.0.26300 |
| Token authentication (issue, rotate, remove, the per-request decision, the three conditions that require a token, binding a connection to its user), the Host check, scopes and targets with two users, the limits, migration of a version 2 database | Verified | Automated test (`go test -race ./...`), Linux x86_64 |
| Listening on an address other than loopback with `--allow-remote`; 401 without a token and with a wrong one; `user rotate` and `user remove` effective at the next request; `probe.sh` with a token and `chat-probe.sh` with two users | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| Two users, ten agents (five on WSL registered with one user's token on loopback, five on Windows registered with the other user's token on the WSL address): joining as the token's user, `lounge_who` scopes with ids, messages of scope repository to the room and to one session, replies with `scope` and `reply_to`, a session of another repository neither listed nor reachable | Verified | Real machine: Windows 10.0.26300 and WSL 2 on one machine; Claude Code 2.1.326 / 2.1.325, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.10 / 1.5.9; on WSL, Codex and agy with their sandboxes off |
| A physically separate machine; a reverse proxy with `--allowed-host` | Unverified | The Host check is covered by automated tests |
| Delivery states, `chat_ack`, `chat_status`, `chat_history`, the sender's values at sending time, expiry and deletion (with an advanced clock), the error for a deleted session, the period and limit options, `session list`, `session remove`, `export`, migration of a version 3 database | Verified | Automated test (`go test -race ./...`), Linux x86_64 |
| Expiry, message and session deletion with short periods; `session list`, `session remove` and `export` while the server runs | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| In the five agents (Claude Code 2.1.326, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.10): replying to a question and calling `chat_ack` from the guidance alone; in a run without context, finding a fetched but unhandled request with `chat_history` (`unacked: true`) and handling it; summarizing past messages; checking a sent message with `chat_status` | Verified | Real machine: Linux x86_64 (WSL 2), non-interactive runs allowing only the agent-lounge tools |
| `chat_ack`, `chat_status` and `chat_history` in agents on Windows | Unverified | |
| Memories: saving, reading, updating, pinning, deleting, searching (two-character Japanese terms, letter case, several terms, symbols, tags, scopes, order, limits), scopes across two users, forgetting with an advanced clock (fading, leaving searches, coming back when read, deletion after the grace period, pinning, unpinning), reference counting, deletion of sessions and users, limits, migration of a version 4 database, `memory list`, `memory remove`, memory export | Verified | Automated test (`go test -race ./...`), Linux x86_64 |
| `probe.sh` and `chat-probe.sh` with the memory check | Verified | Real machine: Linux x86_64 (WSL 2), `linux-amd64` executable |
| Memory tools in agents on Windows | Unverified | |
| Executables for Windows arm64, macOS, and Linux arm64 | Unverified | Cross-compiled only |
| In the five agents (Claude Code 2.1.326, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.10): declaring the profile from the server's guidance alone, the repository room with and without an origin (a WSL folder and a folder on a Windows drive) | Verified | Real machine: Linux x86_64 (WSL 2), non-interactive runs; Codex and agy with their sandboxes off (inside them they cannot read the Windows host values) |
| The five agents on Windows (Claude Code 2.1.325, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.9): declaring the profile, and a repository-room round trip with a WSL session in the same folder without an origin | Verified | Real machine: Windows 10.0.26300 (PowerShell 7.7.0) and WSL 2 on one machine, non-interactive runs allowing only the agent-lounge tools and read-only commands (Codex with `-s read-only`) |
| Registration and conversation (joining, messages to a session and to the `lounge` room, unread count) in Claude Code 2.1.325, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.9 | Verified | Real machine: Linux x86_64 (WSL 2), all five joined to one server; Claude Code registered with `--scope user` |
| In the five agents (Claude Code 2.1.325, Codex 0.169.0, Copilot CLI 1.0.112, Grok 1.0.77, agy 1.5.9): learning the usage and joining in a folder without skills or instruction files, waiting for a reply and a round trip without the user relaying (five pairs), completing a wait at the 45-second limit, reporting the shutdown error while waiting | Verified | Real machine, each CLI run non-interactively: Linux x86_64 (WSL 2) |
| The shutdown log notification in the five agents | Unverified | It does not appear in the non-interactive output of any of the five; interactive screens were not checked |
| What an idle agent in an interactive screen shows when a message or the shutdown notification arrives | Unverified | |
| Other versions of the five agents | Unverified | |
| Registration in VS Code | Unverified | |

Table: Verified scope
