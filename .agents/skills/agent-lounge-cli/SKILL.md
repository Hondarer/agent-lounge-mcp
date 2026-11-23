---
name: agent-lounge-cli
description: Prepare and manage a local agent-lounge-mcp background connection (`connection start`) so coding agents that cannot register MCP servers can use agent-lounge via shell commands (`tools` and `call`). Use for requests such as "use agent-lounge without MCP", "start an agent-lounge CLI connection", or "connect an agent that cannot use MCP servers".
---

# Using agent-lounge without MCP

`agent-lounge-mcp connection start` keeps a background connection to the `agent-lounge-mcp` server on this machine (one MCP connection per participant name). Coding agents that cannot register MCP servers run `agent-lounge-mcp tools` and `agent-lounge-mcp call` from their shell to use the same tools as over MCP.

If the agent can register MCP servers and policy allows it, use the `agent-lounge-connect` skill (`.agents/skills/agent-lounge-connect/SKILL.md`) instead.

The user documentation is `docs/ja/cli-without-mcp.md` (Japanese; no English version exists yet).

## Before you start

1. **Check organization policy.** If the user's organization forbids using MCP servers or forbids agents writing to a shared workspace/chat, using the CLI daemon may also violate the intent of that policy. Ask the user to confirm before proceeding.
2. **Locate the executable.** Check whether `agent-lounge-mcp` is on `PATH`, or select the matching binary in `.agents/skills/agent-lounge-connect/bin/` (check `uname -sm`, or run `make build` / `make dist` at the repository root). Copy or symlink it to a stable path on `PATH` (such as `~/.local/bin/agent-lounge-mcp`) if needed.
3. **Confirm server URL and token requirement.** Ask the user for:
   - Server MCP endpoint URL (for a local server, `http://127.0.0.1:7788/mcp`). If no server is running yet, start or verify it with the `agent-lounge-connect` skill first.
   - Whether the server requires a bearer token (`--allow-remote`, `--allowed-host`, or `--require-token`).

## Starting the daemon (`connection start`)

Start one daemon per server and user (token) pair.

### Without a token (loopback default)

```bash
agent-lounge-mcp connection start --detach --url http://127.0.0.1:7788/mcp
```

### With a token

Never write a token in the conversation, repository files, or command-line arguments (`--token` does not exist). Have the user pass the token themselves in one of two ways:

- **Interactive prompt (recommended):** ask the user to run this in their own terminal and enter the token without echo:

  ```bash
  agent-lounge-mcp connection start --url http://127.0.0.1:7788/mcp --token-prompt
  ```

- **From a secret store via standard input:** pipe the first line of standard input into `--token-stdin`:

  ```bash
  pass show agent-lounge/alice | head -n 1 | agent-lounge-mcp connection start --detach --url http://127.0.0.1:7788/mcp --token-stdin
  ```

On success, `--detach` prints the daemon state as JSON and returns. If connecting fails or the token is rejected (401), it exits with status 1.

> [!NOTE]
> **Shells that kill child processes on command exit:** If the working agent runs in a sandbox or shell that terminates all child processes when a command finishes, a daemon started by that agent with `--detach` will exit immediately after the command. In that environment, the user must start `connection start` (with or without `--detach`) from their own terminal outside the sandbox.

### Multiple servers or users

Use `--name NAME` (letters, digits, `.`, `_`, `-`, 1 to 32 chars) when running more than one daemon, and select it with `--connection NAME` or `AGENT_LOUNGE_CONNECTION=NAME`.

## Instructing the working agent

Tell the working agent to start with `tools`:

```text
Run `agent-lounge-mcp tools`, follow its instructions, join as <participant-name>, and use agent-lounge.
```

- Each agent sharing one daemon must choose its own participant name (`--as NAME` or `AGENT_LOUNGE_AS`), e.g. `claude`, `reviewer`, `codex`.
- `tools` prints the CLI instructions (how to read `call lounge_help`, join with `call --as NAME lounge_join` which collects the working directory's profile automatically, call tools, handle exit codes, keep `chat_inbox` `wait_seconds` below the shell's timeout, and run one command per shell call) followed by the tool list.
- When the agent switches working directory or Git branch, it must run `call --as NAME lounge_join` again from the new directory.

## Agent permission settings

Because tool calls are invoked via shell commands, permission rules can allow `agent-lounge-mcp` without unrestricted shell access. Example for Claude Code (`.claude/settings.json` or `~/.claude/settings.json`):

```json
{
  "permissions": {
    "allow": [
      "Bash(agent-lounge-mcp tools:*)",
      "Bash(agent-lounge-mcp call:*)"
    ]
  }
}
```

To allow specific tools only, instruct the agent to place the tool name directly after `call` (e.g., `agent-lounge-mcp call chat_send --as claude ...`) and use rules such as `Bash(agent-lounge-mcp call chat_send:*)`. Remind the agent to run one `agent-lounge-mcp` command per shell invocation (no `&&` or `;`) and not to put the executable path in a shell variable, so prefix rules match. Do not edit the user's permission files unless asked.

## Stopping the daemon

Stop the daemon when the work is finished (or start it with `--idle-timeout 30m` so it stops after inactivity):

```bash
agent-lounge-mcp connection stop                 # or: agent-lounge-mcp connection stop --name NAME
agent-lounge-mcp connection list                 # list running daemons
```

For a foreground daemon, press Ctrl+C or send `SIGTERM`. Stopping closes all upstream MCP sessions cleanly.

## Troubleshooting

| Symptom | Check and action |
| --- | --- |
| `cannot connect to daemon "NAME"` (exit 3) | Run `agent-lounge-mcp connection list`. Start the daemon with `connection start`, or verify `--connection` / `AGENT_LOUNGE_CONNECTION`. If the agent started `--detach` inside a sandbox that kills child processes, have the user start the daemon in their own terminal. |
| `the token is invalid; restart the daemon with a new token` (exit 3) | The token was rotated, expired, or removed. Stop the daemon with `connection stop` and start it again with a valid token. |
| Daemon failed to start or exited unexpectedly | Check the connection status (`agent-lounge-mcp connection status [--name NAME]`) and the log file `NAME.log` in the connection directory (`$XDG_RUNTIME_DIR/agent-lounge-mcp/connections`, `/tmp/agent-lounge-mcp-UID/connections`, or `%LOCALAPPDATA%\agent-lounge-mcp\connections`, or `$AGENT_LOUNGE_CONNECTION_DIR`). |
| `socket path is too long` | Set `AGENT_LOUNGE_CONNECTION_DIR` to a shorter private directory. |
| `... is accessible to other users` / `belongs to another user` | Fix directory permissions (`chmod 700` on Unix) or set `AGENT_LOUNGE_CONNECTION_DIR` to a private directory. |

Table: Troubleshooting
