<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-23 07:53:00 JST
最終更新エージェント: agy / セッション ID: 56a179ca-dec9-4b75-bd9f-e73a4ce45814
稼働 OS: Linux (WSL)
上流文書: `261121-branch-scope-overview.md` (全体管理文書)、`261121-branch-scope-design.md` (最終仕様)、`261121-branch-scope-lightadr.md` (簡易 ADR)。
-->

# ブランチのスコープ 確認の記録

ファイル名: `261121-branch-scope-validation.md`

## 環境

- OS: Linux x86_64 (WSL 2)
- Go: 1.27.3 (`/home/user/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.3.linux-amd64/bin/go`)
- 対象ブランチ: `feature/branch-scope`

## 1. 自動テスト

- コマンド: `go vet ./...`、`go test -v ./internal/tools -run TestGuideSize`、`go test -race ./...`
- 結果: 検証済 (全パッケージ PASS)
- `lounge_help` の本文 (既定の設定): 19,026 バイト (上限 19,300 バイト以下)
- 追加した主な自動テスト:
  - `internal/profile/profile_test.go` (`TestBranch`): 2.2 の規則全項目、Git が許す名前 (`feature/日本語`、`a@b`、`a#b`)、禁止文字、長さ上限、`head` 拒否
  - `internal/store/branch_test.go`: スコープ規則 (`ScopeKinds`、`ScopesOf`、`SharedScopes`)、`JoinSession` の置き換え規則 4 パターン、`branch` ルーム送信と切り替え後の届き先、ブランチスコープのメモリと上限、版 6 から版 7 への移行 (最大 ID 削除済みおよび全件削除済みでの `sqlite_sequence` 引き継ぎ)
  - `internal/store/forgetting_pace_test.go` (BL-043): `g(q)`・有効経過の単体テスト、使われているスコープ vs 静かなスコープの日数、100 日静かな後の空白クローズ連続性 (38 日)、閉じた後の毎日アクセス、スコープ間の独立性 (`user` / `repository` / `branch`)、定期削除で最終アクセスが進まないこと、版 6 → 版 7 移行での `memory_scope_activity` 初期行作成 (5 日前と 30 日前の保持スコア検証)
  - `internal/tools/catalog_test.go` (`TestGuideBranch`、`TestGuideMemory`、`TestGuideSize`): 案内文の上限とブランチ・忘却ペースの案内記載
  - `internal/server/branch_test.go`、`repository_test.go`: `lounge_join` の引数・出力・`hint` (`wslHint` と `branchHint` の併用を含む)、`lounge_who` の最上位と各セッションの `branch`、`chat_send`/`memory_put`/`memory_get` の `branch` スコープとエラー案内
  - `cmd/agent-lounge-mcp/admin_test.go`: `session list` の `BRANCH` 列、`memory list --scope branch`
  - `internal/export/export_test.go` (`TestWriteMarkdownBranchScope`): `branch-<識別子>` の出力

## 2. 実機確認 (WSL の側・実行ファイル)

- ビルド: `make build` (`dist/agent-lounge-mcp`)
- 起動: 一時フォルダーの DB と Git リポジトリ (`origin: https://github.com/example/branch-verify.git`、ブランチ `main` と `Feature/Check`) を用意し、ループバック (`127.0.0.1:17789`) で起動。

| 確認項目 | 状況 | 結果 |
| --- | --- | --- |
| `/healthz` のスキーマの版 | 検証済 | `schema_version: 7` を確認 |
| `probe.sh`、`chat-probe.sh` | 検証済 | すべて `OK:` で終了 |
| `git branch --show-current` (`Feature/Check`) の申告と小文字化 | 検証済 | `lounge_join` の出力が `"branch":"feature/check"` で記録 |
| ブランチのルームでの送信と別ブランチへの分離 | 検証済 | 同じブランチ (`bob`) のみに配信され、別ブランチ (`carol`、`main`) の未読は 0 |
| ブランチのスコープのメモリ作成・検索・分離 | 検証済 | 同じブランチ (`bob`) から検索可能、別ブランチ (`carol`) からは 0 件 |
| `git switch main` 後の `lounge_join` 呼び直し | 検証済 | 記録の `branch` が `"main"` に更新 |
| サーバー起動中の `session list`、`memory list --scope branch`、`export` | 検証済 | `BRANCH` 列の表示、`--scope branch` の抽出、`memory/branch-github.com_example_branch-verify_feature_check-d80efbfe.md` の出力を確認 |
| BL-043: 短い半減期 (`--memory-half-life 1s --memory-grace 2s`) での起動ログと `memory list` | 検証済 | 起動ログに `scopes with no memory access for 1s forget at 1/3 the pace (up to about 3x longer)` を出力。4 秒後 (通常なら 3.32 秒で薄れる) に `active 0.24`、9 秒後 (静かなスコープは 7.96 秒で薄れる) に `faded` になることを確認 |

## 3. 未実施・残課題へ回した項目

第 2 ゲート承認時の利用者方針 (「実機確認は必要最小限で処理の改善をできる限り実施し、実施できなかったものは backlog に回していい」、[LADR-007](261121-branch-scope-lightadr.md#ladr-007-確認の範囲)) に従い、以下の項目は未検証として文書に明記し、残課題一覧 (`261121-branch-scope-backlog.md`) の `BL-044` へ記録しました。

| 項目 | 状況 | 備考 |
| --- | --- | --- |
| WSL の側の 5 つの CLI (Claude Code、Codex、Copilot CLI、Grok、agy) の非対話実行による、案内からのブランチ申告・1 往復・切り替え追従 (最終仕様 5.2 の a〜c) | 未検証 | `BL-044` へ記録 |
| Windows の側のセッションでの PowerShell 申告・大文字と小文字の動作 (`Main` / `main`)・WSL とのブランチのルーム 1 往復 (最終仕様 5.3 の d〜f) | 未検証 | `BL-044` へ記録 |

## 4. 成果物の審査

Claude Code (Opus 5.5) のセッションが、作業ブランチの差分を審査しました。

- `go vet ./...` と `go test -race ./...`: 検証済 (通る)。
- `lounge_help` の本文: 19,026 バイト (上限 19,300 バイト以下) を確認。
- コードの指摘 3 点:
  - `memory_scope_activity` の行が削除されない: `BL-045` へ起票。
  - ブランチの名前の検査が Git の規則より厳しい (全角空白、`head`): `BL-046` へ起票。
  - 薄れる判定の境界が `now.After(fadesAt)` (より大きい) から「有効経過 >= 薄れるまでの時間」(以上) に変わった。最終仕様 2.8 の式どおりで、差は 1 ミリ秒以下のため、変更は不要とした。
