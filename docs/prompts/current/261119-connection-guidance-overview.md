<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-20 04:50:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53SsJ6j1XoXzWU5edaQ
稼働 OS: Linux (WSL)
上流文書: `../past/261118-interactive-verification-backlog.md` (対策元の残課題一覧)。移動または削除されている場合があります。
下流文書: `261119-connection-guidance-discussion.md` (検討用 Markdown)、`261119-connection-guidance-lightadr.md` (簡易 ADR)、`261119-connection-guidance-design.md` (最終仕様)、`261119-connection-guidance-backlog.md` (残課題一覧)、`261119-connection-guidance-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# 接続と受信の改善 全体管理

ファイル名: `261119-connection-guidance-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261119-connection-guidance-discussion.md`) に、決定事項は簡易 ADR (`261119-connection-guidance-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: connection-guidance

検討用 Markdown の版: 3

状態: 成果物の承認待ち

## 作業 WBS と進捗管理

この案件の作業 WBS です。前の案件 `interactive-verification` の残課題のうち、BL-036 と BL-037 を対策します ([LADR-001](261119-connection-guidance-lightadr.md#ladr-001-今回の実施範囲))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261119-connection-guidance-backlog.md`) に追記します。残課題は全量をこの案件の残課題一覧に引き継いであり、対策元の残課題一覧 (`docs/prompts/past/261118-interactive-verification-backlog.md`) は、すべての項目を「次フェーズへ転記」にして `docs/prompts/past/` へ移しました。以後の各課題の状態と経緯は、この案件の残課題一覧で更新します。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 前提条件と要求事項の調査 (残課題 17 件、Go SDK v1.8.0 と v1.9.0-pre.1 のプロトコルの版の扱い、`server/discover` への応答、確認用のフォルダーでの Claude Code の接続、`lounge_help` とツールの一覧の大きさ、前の案件のシナリオ 4 と 5 の結果)
    - ✅ 1.2. 残課題の全量の引き継ぎ (`261119-connection-guidance-backlog.md` の作成と BL-039 の追加、対策元 `261118-interactive-verification-backlog.md` の状態とメタ情報の更新、前の案件の記録の `docs/prompts/past/` への移動)
    - ✅ 1.3. 実施範囲と記録の形の確認 ([Q-001](261119-connection-guidance-discussion.md#q-001-今回の実施範囲)、[Q-002](261119-connection-guidance-discussion.md#q-002-記録の形)) と、`AGENTS.md` の記録の種類の更新
    - ✅ 1.4. 確認項目 ([Q-003](261119-connection-guidance-discussion.md#q-003-プロトコルの版-2026-07-28-への対応の進め方) から [Q-013](261119-connection-guidance-discussion.md#q-013-作業ブランチと実装の担当)) の作成
    - ✅ 1.5. 第 1 ターンの回答 (Q-003 から Q-013) の受付と記録
    - ✅ 1.6. 回答の検証、簡易 ADR への決定の反映 (LADR-003 から LADR-010)、残課題一覧の状態の更新
    - ✅ 1.7. 一時フォルダーでの事前の確認 (リポジトリは変えない): 使われない接続を閉じる設定と、`lounge_help` の本文だけを返す形を、環境変数で切り替える確認用のサーバーを作成。5 つの CLI の、閉じられた接続の後の動作 ([Q-005](261119-connection-guidance-discussion.md#q-005-使われなくなった接続の扱い))。Copilot CLI の `lounge_help` の読み方と、一度に読める大きさ ([Q-009](261119-connection-guidance-discussion.md#q-009-lounge_help-の大きさ))。BL-040 を残課題一覧に追加
    - ✅ 1.8. 第 2 ターンの確認項目 ([Q-014](261119-connection-guidance-discussion.md#q-014-windows-の側の確認を---で表す範囲) から [Q-019](261119-connection-guidance-discussion.md#q-019-lounge_help-の本文の大きさ)) の作成
    - ✅ 1.9. 第 2 ターンの回答 (Q-010、Q-014 から Q-019) の受付と記録
    - ✅ 1.10. 回答の検証、簡易 ADR への決定の反映 (LADR-005、LADR-008 の更新と LADR-011 の追加)、WBS の見直し、残課題一覧の状態の更新
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の決定事項と、根拠となった解決済み Q の回答または利用者の明示的な指示との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261119-connection-guidance-design.md` の作成 (残課題一覧の対象課題の状態と経緯を更新)
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)
- ✅ 4. 承認された最終仕様に従った作業の実施
    - ✅ 4.1. 作業ブランチ `feature/connection-guidance` の作成と、作業前の記録の最初のコミット ([LADR-010](261119-connection-guidance-lightadr.md#ladr-010-作業ブランチと実装の担当))
    - ✅ 4.2. 検証の基盤のシナリオ 5 の変更 (`scripts/interactive/harness/scenarios/guidance.py`: 作業の途中のメッセージと、その受信の判定) と、基盤の自己試験 ([LADR-006](261119-connection-guidance-lightadr.md#ladr-006-受信の改善の評価))
    - ✅ 4.3. 改善の前の確認: コミット `a24aee1` のサーバーで、5 つの CLI のシナリオ 5 を 5 回ずつ実行し、モデルの名前を記録 ([LADR-009](261119-connection-guidance-lightadr.md#ladr-009-確認の回数とモデル))
    - ✅ 4.4. 停止のログ通知の除去 (`internal/server/server.go`、`internal/server/connlog.go`、`logging` の capability、テスト、検証の基盤のシナリオ 1 と `connections()`) ([LADR-004](261119-connection-guidance-lightadr.md#ladr-004-停止の通知の手段))
    - ✅ 4.5. プロトコルの版の扱いのテスト (`server/discover` への応答と、2026-07-28 の他の要求への `-32022`) ([LADR-003](261119-connection-guidance-lightadr.md#ladr-003-プロトコルの版-2026-07-28-への対応))
    - ✅ 4.6. 使われない接続の扱い (`internal/server/server.go`、`cmd/agent-lounge-mcp/main.go`、`internal/config/`、テスト。`--connection-idle`、既定 1 時間) ([LADR-005](261119-connection-guidance-lightadr.md#ladr-005-使われなくなった接続の扱い))
    - ✅ 4.7. `lounge_help` の出力 (`internal/tools/help.go`: 構造化の出力からの本文の除去と、本文を 18,000 バイト以下にすることと、大きさのテスト) ([LADR-008](261119-connection-guidance-lightadr.md#ladr-008-lounge_help-の出力))
    - ✅ 4.8. ツールの説明文と貼る文面 (`internal/tools/lounge.go`、`internal/tools/chat.go`、`internal/tools/help.go`) ([LADR-007](261119-connection-guidance-lightadr.md#ladr-007-受信の案内の改善))
    - ✅ 4.9. 改善の後の確認: 5 つの CLI のシナリオ 5 を 5 回ずつ実行。Copilot CLI の `lounge_help` の読み方。5 つの CLI の閉じられた接続の後の動作 (`--connection-idle 30s`)
    - ✅ 4.10. 文書 (`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/ja/interactive-verification.md`) と `AGENTS.md` の更新 (Windows の側の `-` の表し方を含む。[LADR-011](261119-connection-guidance-lightadr.md#ladr-011-確認する-os-と-windows-の側の表し方))
    - ✅ 4.11. 正本ドキュメントおよびコード コメントへの永続的知見の反映
    - ✅ 4.12. 機械的検証 (`go vet ./...`、`go test ./...`、`python3 -m unittest discover -s scripts/interactive/tests`)
    - ✅ 4.13. 実装のレビュー (サブエージェント)
    - ✅ 4.14. 確認の記録 (`261119-connection-guidance-validation.md`) の作成
    - ✅ 4.15. 受入条件への適合確認 (全作業の完了確認)
    - ✅ 4.16. 簡易 ADR の最終確認 (各決定が案件終了時点の決定を示し、経緯が時系列にそろっていることの確認)
    - ✅ 4.17. 残課題一覧の再構築と整理 (当該作業中に解消・容認した項目の除外、起票できる詳細度への書き直し)
- 🔄 5. 成果物の承認の受付 (第 3 ゲート)
    - ✅ 5.1. 成果物・検証結果・残課題の提示
    - 🔄 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了) ← **CURRENT**

## 対象と参照資料

対象成果物: サーバー (`internal/server/`、`internal/tools/`)、検証の基盤 (`scripts/interactive/`)、文書 (`docs/ja/`)。範囲は、最終仕様 (`261119-connection-guidance-design.md`) の第 1 章のとおりです。

今回の対象範囲: BL-036 (MCP のプロトコルの版 2026-07-28 への対応と、停止の通知の手段)、BL-037 (指示ファイルに貼る文面と、接続時の案内文の届き方の改善)。前提調査で判明した BL-039 (使われなくなった MCP の接続が残り続ける) も扱います ([LADR-005](261119-connection-guidance-lightadr.md#ladr-005-使われなくなった接続の扱い))。

書き込み可能な検討文書: この文書、検討用 Markdown (`261119-connection-guidance-discussion.md`)、簡易 ADR (`261119-connection-guidance-lightadr.md`)、残課題一覧 (`261119-connection-guidance-backlog.md`)

参照資料と版:

- 対策元の残課題一覧: `docs/prompts/past/261118-interactive-verification-backlog.md`
- 前の案件の確認の記録: `docs/prompts/past/261118-interactive-verification-validation.md` (シナリオ 1、4、5)
- 企画提案書: `docs/agent-lounge-mcp-proposal.md` (停止の要件、受信の制約と対策)
- Go SDK: v1.8.0 (`go.mod`)。比較の対象として v1.9.0-pre.1 (先行版) の `docs/protocol.md` と `docs/mcpgodebug.md`
- CLI (前提調査と事前の確認で使用、WSL の側): Claude Code 2.1.328、Codex 0.169.0、Copilot CLI 1.0.114、Grok 1.0.89、agy 1.5.12
- 使い方の案内: `internal/tools/help.go`、`internal/tools/tools.go` (コミット `a24aee1`)

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 主題は BL-036 と BL-037 ([Q-001](261119-connection-guidance-discussion.md#q-001-今回の実施範囲))。受信の改善の目的は、作業の途中に届いたメッセージを、作業を終えるまでに読むこと ([Q-006](261119-connection-guidance-discussion.md#q-006-受信の改善の測り方)) |
| 範囲・境界 | 確認済み | 2026-07-28 には対応しない ([Q-003](261119-connection-guidance-discussion.md#q-003-プロトコルの版-2026-07-28-への対応の進め方))、ログ通知をやめる ([Q-004](261119-connection-guidance-discussion.md#q-004-停止のログ通知の扱い))、BL-039 を扱い、接続を 1 時間で閉じ、`--connection-idle` で変えられる ([Q-005](261119-connection-guidance-discussion.md#q-005-使われなくなった接続の扱い)、[Q-016](261119-connection-guidance-discussion.md#q-016-agy-の接続し直しの結果の扱い)、[Q-017](261119-connection-guidance-discussion.md#q-017-使われない接続を閉じるまでの時間)、[Q-018](261119-connection-guidance-discussion.md#q-018-閉じるまでの時間の起動オプション))、説明文と文面 ([Q-007](261119-connection-guidance-discussion.md#q-007-案内文を渡さない-cli-への対策)、[Q-008](261119-connection-guidance-discussion.md#q-008-貼る文面の直し方))、`lounge_help` の出力と大きさ ([Q-009](261119-connection-guidance-discussion.md#q-009-lounge_help-の大きさ)、[Q-019](261119-connection-guidance-discussion.md#q-019-lounge_help-の本文の大きさ)) |
| 入力・根拠 | 確認済み | 参照資料と版を上に記載。`server/discover` への応答、5 つの CLI の閉じられた接続の後の動作、Copilot CLI の一度に読める大きさ (19,500 バイトは直接読み、20,600 バイトは保存) は、一時フォルダーで確かめた |
| 整合性・依存関係 | 確認済み | `-` の表し方は、`AGENTS.md` の規則を改めて整合させる ([Q-014](261119-connection-guidance-discussion.md#q-014-windows-の側の確認を---で表す範囲)、[Q-015](261119-connection-guidance-discussion.md#q-015-文書での---の表し方))。企画提案書の停止の要件とログ通知は、[Q-004](261119-connection-guidance-discussion.md#q-004-停止のログ通知の扱い) で整合。`--connection-idle` は、`--session-idle` と名前で区別する |
| 実現性・例外 | 確認済み | agy の 1 回目の失敗は、制限として記録し、残課題 BL-040 とする ([Q-016](261119-connection-guidance-discussion.md#q-016-agy-の接続し直しの結果の扱い))。本文を 18,000 バイト以下にできるかは、実装で確かめ、テストで守る ([Q-019](261119-connection-guidance-discussion.md#q-019-lounge_help-の本文の大きさ))。Codex の週の利用量の残り (事前の確認の時点で 19%) は、改善の前の実行の前に確かめ、足りなければ利用者に確認する (最終仕様の第 8 章)。ビルドとテストは、モジュールのキャッシュにある Go 1.27.3 の道具 (`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.3.linux-amd64/bin/go`、`GOTOOLCHAIN=local`) で行える |
| 受入条件・検証 | 確認済み | 評価の指標 ([Q-006](261119-connection-guidance-discussion.md#q-006-受信の改善の測り方))、回数 ([Q-011](261119-connection-guidance-discussion.md#q-011-確認の回数))、モデル ([Q-012](261119-connection-guidance-discussion.md#q-012-確認で使うモデル))、OS と Windows の側の表し方 ([Q-010](261119-connection-guidance-discussion.md#q-010-確認する-os)、[Q-014](261119-connection-guidance-discussion.md#q-014-windows-の側の確認を---で表す範囲)、[Q-015](261119-connection-guidance-discussion.md#q-015-文書での---の表し方))。受入条件は最終仕様の第 10 章 |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261119-connection-guidance-discussion.md#q-001-今回の実施範囲) | 今回の実施範囲 | 解決済み | Q-003、Q-006 | LADR-001 |
| [Q-002](261119-connection-guidance-discussion.md#q-002-記録の形) | 記録の形 | 解決済み | なし | LADR-002 |
| [Q-003](261119-connection-guidance-discussion.md#q-003-プロトコルの版-2026-07-28-への対応の進め方) | プロトコルの版 2026-07-28 への対応の進め方 | 解決済み | Q-001、Q-004 | LADR-003 |
| [Q-004](261119-connection-guidance-discussion.md#q-004-停止のログ通知の扱い) | 停止のログ通知の扱い | 解決済み | Q-003 | LADR-004 |
| [Q-005](261119-connection-guidance-discussion.md#q-005-使われなくなった接続の扱い) | 使われなくなった接続の扱い | 解決済み | Q-003、Q-010、Q-016 | LADR-005 |
| [Q-006](261119-connection-guidance-discussion.md#q-006-受信の改善の測り方) | 受信の改善の測り方 | 解決済み | Q-001、Q-008、Q-011 | LADR-006 |
| [Q-007](261119-connection-guidance-discussion.md#q-007-案内文を渡さない-cli-への対策) | 案内文を渡さない CLI への対策 | 解決済み | Q-008 | LADR-007 |
| [Q-008](261119-connection-guidance-discussion.md#q-008-貼る文面の直し方) | 貼る文面の直し方 | 解決済み | Q-006、Q-007 | LADR-007 |
| [Q-009](261119-connection-guidance-discussion.md#q-009-lounge_help-の大きさ) | lounge_help の大きさ | 解決済み | Q-019 | LADR-008 |
| [Q-010](261119-connection-guidance-discussion.md#q-010-確認する-os) | 確認する OS | 解決済み | Q-005、Q-011、Q-014、Q-015 | LADR-011 |
| [Q-011](261119-connection-guidance-discussion.md#q-011-確認の回数) | 確認の回数 | 解決済み | Q-006、Q-010 | LADR-009 |
| [Q-012](261119-connection-guidance-discussion.md#q-012-確認で使うモデル) | 確認で使うモデル | 解決済み | なし | LADR-009 |
| [Q-013](261119-connection-guidance-discussion.md#q-013-作業ブランチと実装の担当) | 作業ブランチと実装の担当 | 解決済み | なし | LADR-010 |
| [Q-014](261119-connection-guidance-discussion.md#q-014-windows-の側の確認を---で表す範囲) | Windows の側の確認を `-` で表す範囲 | 解決済み | Q-010、Q-015 | LADR-011 |
| [Q-015](261119-connection-guidance-discussion.md#q-015-文書での---の表し方) | 文書での `-` の表し方 | 解決済み | Q-010、Q-014 | LADR-011 |
| [Q-016](261119-connection-guidance-discussion.md#q-016-agy-の接続し直しの結果の扱い) | agy の接続し直しの結果の扱い | 解決済み | Q-005、Q-017、Q-018 | LADR-005 |
| [Q-017](261119-connection-guidance-discussion.md#q-017-使われない接続を閉じるまでの時間) | 使われない接続を閉じるまでの時間 | 解決済み | Q-016、Q-018 | LADR-005 |
| [Q-018](261119-connection-guidance-discussion.md#q-018-閉じるまでの時間の起動オプション) | 閉じるまでの時間の起動オプション | 解決済み | Q-016、Q-017 | LADR-005 |
| [Q-019](261119-connection-guidance-discussion.md#q-019-lounge_help-の本文の大きさ) | lounge_help の本文の大きさ | 解決済み | Q-004、Q-008、Q-009 | LADR-008 |
| [Q-020](261119-connection-guidance-discussion.md#q-020-作業の途中のメッセージを送る時点) | 作業の途中のメッセージを送る時点 | 解決済み | Q-006、Q-011 | LADR-006 |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261119-connection-guidance-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 2

最終仕様 (design.md) の合意版 (第 2 ゲート): 承認済み (2026-11-19 23:00 の版。第 7.1 節は 2026-11-20 03:38 に Q-020 で改めた)

更新予定の成果物: `internal/server/`、`internal/tools/`、`internal/config/`、`cmd/agent-lounge-mcp/`、`scripts/interactive/`、`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/ja/interactive-verification.md`、`AGENTS.md` (最終仕様の各章)

受入条件: 最終仕様の第 10 章 (12 項目)

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-19 20:36 より前 | 案件の開始 | 利用者の依頼 (「docs/prompts/current の残件を進める。」)。対話セッションで、実施範囲 ([LADR-001](261119-connection-guidance-lightadr.md#ladr-001-今回の実施範囲)) と記録の形 ([LADR-002](261119-connection-guidance-lightadr.md#ladr-002-案件の記録の形)) を確認 |
| 2026-11-19 20:50 | 記録の作成 | `AGENTS.md` の記録の種類を更新。全体管理文書、検討用 Markdown (版 1)、簡易 ADR、残課題一覧を作成。前の案件の記録を `docs/prompts/past/` へ移動 |
| 2026-11-19 22:13 | 第 1 ターンの回答の受領 | Q-003 から Q-013 に回答 (すべて A。Q-010 に補足)。Q-010 以外を解決済みにし、簡易 ADR に LADR-003 から LADR-010 を追加 |
| 2026-11-19 22:16 から 22:43 | 事前の確認 | 一時フォルダーの確認用のサーバーで、5 つの CLI の閉じられた接続の後の動作 (agy を 4 回、他を 1 回ずつ) と、Copilot CLI の `lounge_help` の読み方と一度に読める大きさを確認。BL-040 を残課題一覧に追加 |
| 2026-11-19 22:46 | 第 2 ターンの確認項目の提示 | 検討用 Markdown の版 2。Q-014 から Q-019 を追加 |
| 2026-11-19 22:54 | 第 2 ターンの回答の受領 | Q-014 から Q-019 に回答 (すべて A、補足なし)。Q-010 と Q-014 から Q-019 を解決済みにし、LADR-005 と LADR-008 を更新、LADR-011 を追加 |
| 2026-11-19 23:00 | 第 1 ゲートの通過と最終仕様の作成 | 未解決の Q と観点の未確認がないことを確認 (合意版 2)。`261119-connection-guidance-design.md` を作成し、第 2 ゲートの承認を依頼 |
| 2026-11-20 03:09 | 第 2 ゲートの通過 | 利用者が最終仕様を承認 (「承認します」)。作業の実施に着手 |
| 2026-11-20 03:11 | 作業ブランチの作成 | `feature/connection-guidance` を作成し、作業前の記録を最初のコミットにした |
| 2026-11-20 03:12 | 改善の前の確認の開始 | コミット `a24aee1` から一時フォルダーにビルドしたサーバーで、シナリオ 5 を実行。Codex の週の利用量の残りは 18% (前の実行の記録から確認。1 回の実行で 1% 未満) で、足りると判断。Claude Code は、確認用のフォルダーが Claude Code の作業用の一時フォルダーの下にあり `CLAUDE.md` を読まなかったため、`/tmp` の直下でやり直す (前の案件の既知の制限) |
| 2026-11-20 03:20 より前 | 利用者の指示 | 「codexが10%を切ったら、OpusやSonnetを活用してください。」Codex の週の利用量の残りが 10% を切ったら、残りの確認を Opus または Sonnet (Claude Code) で行う |
| 2026-11-20 03:34 | 改善の前の確認の中断 | 改善の前の実行で、Claude Code と agy が参加の前に `notes.txt` を作り、作業の途中のメッセージを送れなかった。Codex の週の利用量の残りは 18% |
| 2026-11-20 03:38 | 追加の確認と最終仕様の改訂 | [Q-020](261119-connection-guidance-discussion.md#q-020-作業の途中のメッセージを送る時点) を対話セッションで確認 (A)。最終仕様の第 7.1 節と LADR-006 を改め、検証の基盤を直して、改善の前の実行を 5 つの CLI ともやり直す |
| 2026-11-20 04:20 | 改善の後の確認 | シナリオ 5 (5 つの CLI × 5 回)、Copilot CLI の `lounge_help` の読み方、閉じられた接続の後の動作 (5 つの CLI) を確認。実装のレビュー (サブエージェント) の指摘に対応 |
| 2026-11-20 04:50 | 第 3 ゲートの承認の依頼 | 文書の更新、確認の記録の作成、簡易 ADR の最終確認 (LADR-005 を更新)、残課題一覧の整理 (BL-037 と BL-039 を除き、BL-036、BL-028、BL-040 を更新) を終え、成果物の承認を依頼 |

## 反映と検証の記録

| 項目 | 内容 |
| --- | --- |
| コミット (作業ブランチ `feature/connection-guidance`) | `fe83cca` (作業前の記録)、`833040b` (検証の基盤のシナリオ 5)、`0b8a73d` (サーバー)、`120fa1f` (Q-020 の基盤の修正と記録)、`ef1f072` (レビューの指摘への対応)、`403dfe8` (文書)。確認の記録と、この文書の整理は、未コミット |
| 対象成果物 | `internal/server/` (`server.go`、`connlog.go`、テスト)、`internal/tools/` (`help.go`、`tools.go`、`lounge.go`、`chat.go`、`notify.go`、テスト)、`internal/config/`、`cmd/agent-lounge-mcp/`、`scripts/interactive/`、`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/ja/interactive-verification.md`、`AGENTS.md`、`.agents/skills/agent-lounge-connect/SKILL.md` |
| 機械的検証 | `go vet ./...`、`go test -race ./...`、基盤の自己試験 (18 件)。すべて成功 |
| 確認の結果 | `261119-connection-guidance-validation.md`。作業の途中の知らせを読んだ回数は、改善の前の 25 回中 16 回から、改善の後の 25 回中 25 回になった。agy は、閉じられた接続の後にツールを呼べない (BL-040) |
| 受入条件 (最終仕様の第 10 章) | 1 から 12 を満たすことを確認した。第 9 章の文書に加え、SKILL.md を更新した (確認の記録の「最終仕様との相違」) |
| 未完了 | なし。`main` へのマージは、成果物の承認の後に、利用者の指示で行う |
