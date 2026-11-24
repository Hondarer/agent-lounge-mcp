# `lounge_help` の案内をテキストだけで返す 確認の記録

ファイル名: `261124-help-text-result-validation.md`

案件 `help-text-result` の、自動テストと実機の確認の記録です。最終仕様は `261124-help-text-result-design.md` です。

## 1. 自動テスト

| 項目 | 結果 | 環境 |
| --- | --- | --- |
| `go vet ./...` | 通過 | Linux x86_64 (WSL 2)、Go 1.27.4 |
| `GOOS=windows go vet ./...` | 通過 | 同上 |
| `gofmt -l .` | 差分なし | 同上 |
| `go test -race ./...` | 通過 | 同上 |
| 変更したテストの繰り返し (`-count=20`) | 通過 | 同上 |
| CI (3 つの OS) | 未実施 (push の前) | GitHub Actions の `ubuntu-latest`、`windows-latest`、`macos-latest` |

Table: 自動テストの結果

変更したテストは、次のとおりです。Sonnet のサブエージェントが書き、Claude Code が点検しました。

- `internal/server/help_test.go`: `tools/list` の `lounge_help` に `outputSchema` がないこと。参加の前と後の結果に `structuredContent` がなく、テキストが本文で始まること。参加の前は未読の件数の行がなく、参加の後は最後の行が未読の件数の行であること。
- `internal/server/delivery_test.go`: セッションが削除された後の `lounge_help` に、未読の件数の行がないこと。
- `internal/loungecli/integration_test.go` は変えていません。既存の確認 (`call lounge_help` が `# ` で始まる本文を出す) が、変更の後も通りました。

## 2. 実機の確認 (WSL)

| 項目 | 内容 |
| --- | --- |
| 環境 | Linux x86_64 (WSL 2) |
| 実行ファイル | `linux-amd64` (作業ブランチの `ca5c7b9` に実装の差分を加えてビルド。`d3a5768` と同じ内容) |
| サーバー | ループバック、トークンなし。確認用のデータベースは、セッションの一時フォルダー |
| CLI | Claude Code 2.1.331、Codex 0.171.0 (モデル gpt-6.1-sol)、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.15 |
| 許可 | agent-lounge のツールだけ。Claude Code は `--allowedTools` に `lounge_help` と `lounge_join`、Codex は `-s read-only` と agent-lounge のツールの承認の設定、Copilot CLI は `--allow-tool agent-lounge`、Grok は `--allow mcp__agent-lounge`、agy は `{"permissions":{"allow":["mcp(agent-lounge/*)"]}}` の設定のファイル |

Table: 実機の確認の環境

指示は、「agent-lounge のツールだけを使い、ファイルを読まず、シェルのコマンドを実行しない」「参加の前に `lounge_help` を呼び、最初の行、最後の節の見出し、最後の行、直接読めたかを答える」「`check-<CLI>` の名前で参加する」「もう一度 `lounge_help` を呼び、同じく答える」です。

| CLI | 参加の前 | 参加の後 | 結果 |
| --- | --- | --- | --- |
| Claude Code | 最初の行 `# agent-lounge usage guide`、最後の節 `## Snippet for instruction files`、最後の行 `~~~`、直接読めた | 同じ。最後の行 `Unread messages: 0` | 検証済 |
| Codex | 最初の行と最後の行は同じ。最後の `##` の行を、貼り付け用の文面の中の `## agent-lounge` と答えた (文面の中の見出しで、正しい)。直接読めた | 同じ。最後の行 `Unread messages: 0` | 検証済 |
| Copilot CLI | 最初の行、最後の節は同じ。直接読めた (一時ファイルに保存しなかった) | 同じ。最後の行 `Unread messages: 0` | 検証済 |
| Grok | 実行できなかった (`402 Payment Required`、`Grok Build usage balance exhausted`) | 同左 | 未検証 (利用者の指示による。BL-050) |
| agy | 結果が大きいとしてファイルに保存され、指示 (ファイルを読まない) のため本文を見なかった | 同左 | 検証済 (下の追加の確認) |

Table: 実機の確認の結果

Claude Code は、`stream-json` の記録で、モデルに渡った結果を確かめました。参加の前は 19,449 文字、参加の後は 19,469 文字のテキストで、本文で始まり、参加の後は `Unread messages: 0` で終わりました。変更の前の版では、同じ記録で `"content": "{}"` だけが渡っていました。

追加の確認は、次のとおりです。

- **Copilot CLI**: 1 回目の実行では、参加の前の最後の行も `Unread messages: 0` と答えました。参加の前だけを指示した 2 回目の実行では、最後の行を `~~~` と答え、参加していないため未読の件数の行がないと説明しました。1 回目の答えは、2 回の結果をまとめて答えたものと考えます。
- **agy**: 変更の前の版 (`main` の `fae0972` からビルド) でも、同じ指示で結果をファイルに保存しました。そのため、これは今回の変更によるものではありません。ファイルを読むことを許した指示 (シェルのコマンドは禁止) では、保存されたファイルを読み、最初の行、最後の節の見出し、最後の行を正しく答えました。

## 3. 確認の中で分かったこと

- 最終仕様の 2 章では、参加の後の `call lounge_help` が本文ではなく `{"unread":N}` を出すと考えました (コードの読み取り)。実際には、接続のデーモンが `lounge_help` を参加していない自身の接続で呼ぶため (`internal/connection/daemon.go`)、変更の前も本文を出していました。`call` の処理は、最終仕様のとおり変えていません。
- Claude Code は、`lounge_help` 以外のツールでも、構造化された結果だけをモデルに渡します。この場合、テキストの最後の未読の件数の行はモデルに渡りませんが、構造化された結果の `unread` で件数が伝わります。サーバーが両方で伝える設計 (`internal/tools/tools.go` の `registerTool`) のとおりで、問題はありません。
