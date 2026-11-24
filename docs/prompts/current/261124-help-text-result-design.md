<!--
生成元: agree-before-implement スキル
文書種別: 最終決定事項 (最終仕様)
最終更新日時: 2026-11-25 06:51:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261124-help-text-result-overview.md` (全体管理文書)、`261124-help-text-result-discussion.md` (検討用 Markdown)、`261124-help-text-result-lightadr.md` (簡易 ADR)。いずれも削除されている場合があります。
下流文書: `261124-help-text-result-backlog.md` (残課題一覧)、`261124-help-text-result-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。上流文書の全体管理文書が存在する場合は、その再開手順に従って作業 WBS の `← **CURRENT**` の項目から再開します。存在しない場合は、この文書を単体の最終仕様として扱います。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。
-->

# `lounge_help` の案内をテキストだけで返す 最終仕様

ファイル名: `261124-help-text-result-design.md`

## 1. 目的と範囲

どの CLI でも、`lounge_help` で使い方の案内の全文を読めるようにします。対策するのは、残課題 BL-049 だけです ([LADR-001](261124-help-text-result-lightadr.md#ladr-001-案件の対象))。

## 2. 現状と原因

- `lounge_help` は、テキストの結果に案内の本文を、構造化された結果 (`structuredContent`) に `{}` (参加の後は `{"unread":N}`) を返します。出力スキーマ (`outputSchema`) も持ちます。
- Claude Code 2.1.331 は、構造化された結果があると、それだけをモデルに渡します。そのため、本文が渡りません。
- MCP を使わない方式の `call` は、構造化された結果が `{}` でなければそれを出します。そのため、参加の後の `lounge_help` では、本文ではなく `{"unread":N}` を出すと考えられます (コードの読み取り)。

## 3. 変更の仕様

### 3.1. `lounge_help` の結果

[LADR-002](261124-help-text-result-lightadr.md#ladr-002-lounge_help-の結果の形) のとおりです。

| 項目 | 変更後 |
| --- | --- |
| `tools/list` の `outputSchema` | 持たない |
| 結果の `structuredContent` | 持たない (参加の前も後も) |
| 結果の `content` | テキスト 1 つ。参加の前は案内の本文。参加の後は、本文の後に、今と同じ未読の件数の行 |
| 停止の処理中 | 今と同じエラー |
| 説明文、注釈 (読み取り専用)、本文、上限 (`maxGuideBytes` = 19,300 バイト) | 変えない |

Table: `lounge_help` の結果の形

他のツールの結果の形は、変えません。

### 3.2. 実装の方針

- `internal/tools/tools.go` に、テキストだけを返すツールの登録の仕組みを加えます。ハンドラーはテキストを返し、登録の仕組みは、出力の型を `any` にして `mcp.AddTool` に渡し、出力を `nil` にします (go-sdk v1.8.0 は、このとき出力スキーマも構造化された結果も作らない)。停止の処理中のエラー、最終利用時刻の更新、削除されたセッションの扱い、参加の後の未読の件数の行は、今の `registerTool` と同じにします (処理を共有する)。
- `internal/tools/help.go` は、この仕組みで `lounge_help` を登録します。`HelpOutput` と `resultTexter` は、使う所がなくなれば消します。注記 (`HelpOutput`、`maxGuideBytes`) は、構造化された結果を持たない理由 (Claude Code は構造化された結果だけを渡し、Copilot CLI は両方を受け取る) に書き換えます。
- `internal/loungecli/cli.go` の `result` は、構造化された結果がなければテキストを出す今の処理のままとし、注記を新しい形に合わせます。

### 3.3. テスト

- `internal/server/help_test.go`: `tools/list` の `lounge_help` に `outputSchema` がないこと。参加の前と後の結果に `structuredContent` がなく、テキストが本文で始まること。参加の後は、最後の行が未読の件数の行であること。
- `internal/server/delivery_test.go` の 281 行: `lounge_help` の構造化された結果の `unread` を見ている確認を、テキストに未読の件数の行がないことの確認に変えます (削除されたセッションの接続は、参加していない扱いになるため)。
- `internal/loungecli/integration_test.go`: 参加の後の `call lounge_help` が、本文 (`# ` で始まる) を出すこと。

## 4. 文書

| ファイル | 変更 |
| --- | --- |
| `docs/ja/reference.md` | 結果の形の説明 (371 行付近): `lounge_help` は構造化された出力を持たず、テキストだけを返す。`lounge_help` の節 (425 行付近): 構造化の出力を持たないこと |
| `docs/ja/user-guide.md` | 検証済みの範囲に、5 つの CLI で `lounge_help` の本文を読めること (参加の前と後) の行を加える (方法と版。確かめられなかった CLI は未検証) |
| `docs/ja/cli-without-mcp.md` | 134 行の「テキストだけを返すツール (`lounge_help`) は、そのテキストを出します」は、変更後も正しいため変えない |

Table: 文書の変更

使い方の案内 (`help.go` の本文) は、変えません。

## 5. 確認の方法

[LADR-003](261124-help-text-result-lightadr.md#ladr-003-確認の方法) のとおりです。

- 自動: `go vet ./...` (Linux と `GOOS=windows`)、`go test -race ./...`、CI (3 つの OS)。
- 実機: この WSL の環境で、作業ブランチからビルドした `linux-amd64` の実行ファイルを、確認用のデータベース (セッションの一時フォルダー) で起動します。5 つの CLI (Claude Code 2.1.331、Codex 0.171.0、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.15) の非対話の実行で、次を確かめます。
    1. 参加の前に `lounge_help` を呼び、本文の最初の見出しと、最後の節 (Snippet for instruction files) の見出しを答える。
    2. 参加した後に `lounge_help` を呼び、同じく答える。
- Claude Code は、加えて、`stream-json` の記録で、モデルに渡った結果が本文であることを確かめます。Copilot CLI は、結果を一時ファイルに保存せずに読んだことを、出力で確かめます。
- 指示には、ツール名 `lounge_help` を含めます (案内を読めるかの確認であり、案内にたどり着けるかの確認ではないため)。
- CLI の利用の上限に達して確かめられない CLI は、未検証として記録します。

## 6. 受入条件

1. `lounge_help` が出力スキーマと構造化された結果を持たず、テキストの結果が 3.1 のとおりであること (自動テスト)。
2. 他のツールの結果の形が変わらないこと (既存の自動テストが通る)。
3. 参加の後の `call lounge_help` が本文を出すこと (自動テスト)。
4. 5 つの CLI で、参加の前と後に、本文を読めること (上限で確かめられない CLI は未検証として記録)。
5. 4 章の文書が、変更後の仕様だけを書いていること。
6. CI が 3 つの OS で成功すること。

## 7. 作業の順と依頼先

[LADR-004](261124-help-text-result-lightadr.md#ladr-004-作業の進め方) のとおりです。

1. 作業ブランチ `fix/help-text-result` を作り、作業前の記録をコミットする (Claude Code)。
2. 3.2 と 3.3 の実装とテスト (Sonnet のサブエージェント。Claude Code が点検)。
3. 4 章の文書のうち、`reference.md` (Claude Code)。
4. 自動の確認と 5 つの CLI での実機の確認、`user-guide.md` の検証済みの範囲、確認の記録 (Claude Code)。
5. 残課題の整理、第 3 ゲート。push と `main` へのマージは、第 3 ゲートで利用者の指示を受けてから行う。

## 8. 第 2 ゲートで確認を受ける細部

決定から一意に決まらない細部です。

- 構造化された結果の `unread` を、`lounge_help` からなくします。未読の件数は、テキストの最後の行で伝わります。
- 実機の確認の指示に、ツール名 `lounge_help` を含めます (5 章)。
- `call` の処理は変えず、注記だけを直します (3.2)。
