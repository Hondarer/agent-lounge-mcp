# 残課題の継続対策 確認の記録

ファイル名: `261125-backlog-followup-validation.md`

案件 `backlog-followup` の、確認の記録です。最終仕様は `261125-backlog-followup-design.md` です。Windows の側の結果は、Windows 担当 (Windows の側の Claude Code のセッション) の報告によります。

## 1. agy の開発元への報告の文面 (BL-040)

利用者が送るための文面です (最終仕様の 4 章)。エージェントは送っていません。利用者の環境の固有の情報 (ユーザー名、コンピューター名、パス、トークン) は含めていません。

````markdown
### Summary

After an MCP server closes an idle Streamable HTTP session (or after the server restarts), the first instruction in an already running agy session cannot call MCP tools: the model's tool call fails with `unknown tool` (the `call_mcp_tool` function is missing from the declared tools). A second instruction often works, but not always.

### Versions and platforms

- agy 1.5.12, 1.5.13, 1.5.14 and 1.5.17 (all reproduce)
- Linux x86_64 (WSL 2) and Windows 11 (10.0.26300)
- MCP server: agent-lounge-mcp (Go, github.com/modelcontextprotocol/go-sdk v1.8.0, Streamable HTTP, stateful sessions). Protocol version negotiated: 2025-11-25.

### Steps to reproduce

1. Start an MCP server that closes sessions without requests after 30 seconds. With agent-lounge-mcp: `agent-lounge-mcp serve --db test.db --listen 127.0.0.1:7793 --connection-idle 30s`
2. Register it: `agy mcp add --type http agent-lounge http://127.0.0.1:7793/mcp`
3. Start agy interactively and ask it to call a tool of the server (for example: "Use the agent-lounge MCP server: call lounge_join with the name test-agy"). This works.
4. Wait 50 seconds without using the tools. The server closes the session; requests on it get HTTP 404.
5. Ask agy to call a tool again (for example: "read your inbox with chat_inbox").

Variant: instead of step 4, restart the MCP server.

### Expected

agy opens a new session (it does: the server sees `initialize` and `tools/list` from agy on a new session) and the model calls the tool.

### Actual

The first instruction after step 4 fails: the model reports that the tool call failed with `unknown tool` / that `call_mcp_tool` is not among the declared tools. Counts with agy 1.5.17 (3 trials per scenario on each platform):

| Platform | Scenario | 1st instruction | 2nd instruction |
| --- | --- | --- | --- |
| Linux (WSL 2) | idle session closed | 0 of 3 succeeded | 2 of 3 succeeded |
| Linux (WSL 2) | server restarted | 0 of 3 succeeded | 2 of 3 succeeded |
| Windows 11 | idle session closed | 0 of 3 succeeded | 2 of 3 succeeded |
| Windows 11 | server restarted | 0 of 3 succeeded | 3 of 3 succeeded |

Earlier versions (1.5.12 to 1.5.14) showed the same pattern. Restarting agy always recovers. Other MCP clients tested with the same server (Claude Code, Codex CLI, Copilot CLI, Grok CLI) re-initialize and call the tool on the first instruction.
````
