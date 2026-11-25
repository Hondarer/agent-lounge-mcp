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

## 2. 検証の基盤の Codex の更新の確認 (BL-052)

`scripts/interactive/harness/agents.py` の Codex の定義の `interactive` に `-c check_for_update_on_startup=false` を足しました (`4e8b0ec`)。実装は Sonnet のサブエージェントが書き、Claude Code が点検しました。

| 項目 | 結果 | 環境 |
| --- | --- | --- |
| 自己試験 (`python3 -m unittest discover -s scripts/interactive/tests`。定義の引数の試験を 1 件追加) | 51 件が通過 | Linux x86_64 (WSL 2)、Python 3.12 |
| 同じ (Windows 担当) | 50 件が通過、1 件は省略 (`PWD` の試験)。`ResourceWarning` の行は 0 件 | Windows 10.0.26300、Python 3.14.3 |
| Codex のシナリオ 1 (`verify.py run --agents codex --scenarios 1`) | `done` (起動して指示を受け付けた) | WSL: Codex 0.172.0 (50.7 秒)。Windows (Windows 担当): Codex 0.172.0 (56.2 秒) |
| 新しい版があっても更新の案内が出ないこと | 未検証 | 0.172.0 が最新で、案内が出る状況がなかった |

Table: BL-052 の確認

## 3. Windows の Codex のプロフィールの申告 (BL-051)

### 3.1. 5 回の確認

Windows 担当が、Windows のループバックのサーバー (トークンなし) と、`%TEMP%\alm-wv\repo` (origin あり、ブランチ `verify/wv-branch`) で、Codex 0.172.0 を `codex exec --skip-git-repo-check -s read-only` と agent-lounge のツールの承認で 5 回実行しました。

- 5 回とも、`repository` と `branch` が空でした (サーバーの `session list` でも `-`)。
- 5 回とも、Codex は `pwsh.exe` (`WindowsApps` の下) で、`hostname`、`$env:USERNAME`、`git remote get-url origin`、`git rev-parse --show-toplevel`、`git branch --show-current` を実行しようとし、すべて `Failed to create unified exec process: CreateProcessAsUserW failed: 5 (アクセスが拒否されました。)` でした。代わりのシェルは試さず、値なしで `lounge_join` を呼びました。
- 事前の確認: `cmd.exe /c hostname` などを直接実行させても、Codex は `pwsh.exe -Command 'cmd.exe /c hostname'` の形で起動し、同じエラーで失敗しました。

### 3.2. 原因の調査

| 確認 | 結果 |
| --- | --- |
| シェルを選ぶ設定 | `codex --help`、`codex exec --help`、パッケージの README、`codex features list` からは、シェルのパスを選ぶ設定の項目は見つからなかった。`--enable powershell_shell_version` は効果なし |
| `-s workspace-write` | `read-only` と同じく、3 つのコマンドとも失敗 |
| `pwsh` の実体 | `WindowsApps` の下の実行エイリアス (リパース ポイント `IO_REPARSE_TAG_APPEXECLINK`)。実体は Microsoft Store 版の PowerShell 7.7.0。インストーラー版の PowerShell 7 はない |
| その実行だけ `PATH` から `WindowsApps` を外す (`-s read-only`) | Codex は `C:\WINDOWS\System32\WindowsPowerShell\v1.0\powershell.exe` (5.1) を使い、プロセスの作成に成功。`git branch --show-current` は成功。`hostname` と `cmd.exe /c hostname` は、`PATH` で先に見つかる Git for Windows の `usr\bin\hostname.exe` (MSYS2) が、サンドボックスの中で共有メモリを作れず失敗 |
| 同じ構成で 5 回と同じ指示を 3 回 | 3 回とも `repository` が `github.com/hondarer/agent-lounge-mcp`、`branch` が `verify/wv-branch` (サーバーの記録) |

Table: 原因の調査 (Windows 担当の報告)

原因は、Codex がシェルのコマンドを包むために使う Microsoft Store 版の `pwsh` (実行エイリアス) を、サンドボックスの中から起動できないことです。利用者の判断 (検討用 Markdown の Q-007 の A) で、使い方の案内は変えず、利用ガイドに注意と回避の手順を書きました。Codex の設定ファイル (`~/.codex/config.toml`) は、書き換えていません。

## 4. 後片付け

- WSL の側: 確認用のサーバーは止めました。5 つの CLI の `mcp list` に agent-lounge はありません。
- Windows の側 (Windows 担当の報告): Codex の登録は外し、サーバーは止め、`alm.exe` のプロセスは残っていません。確認用のデータとログは `%TEMP%\alm-wv` にあります。
