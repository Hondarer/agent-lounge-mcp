<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-22 21:55:00 JST
最終更新エージェント: agy / セッション ID: cf8c6742-ed01-40ae-981f-5fef110f9399
稼働 OS: Linux (WSL)
上流文書: `../past/261120-expiry-sandbox-verification-backlog.md` (転記元の残課題一覧)。移動または削除されている場合があります。
下流文書: `261121-branch-scope-discussion.md` (検討用 Markdown)、`261121-branch-scope-lightadr.md` (簡易 ADR)、`261121-branch-scope-design.md` (最終仕様)、`261121-branch-scope-backlog.md` (残課題一覧)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# ブランチのスコープ 全体管理

ファイル名: `261121-branch-scope-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261121-branch-scope-discussion.md`) に、決定事項は簡易 ADR (`261121-branch-scope-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: branch-scope

検討用 Markdown の版: 2

状態: 作業実施中 (第 2 ゲート通過済み)

## 作業 WBS と進捗管理

この案件の作業 WBS です。リポジトリよりも詳細な単位として、ブランチを設ける計画を検討します ([LADR-001](261121-branch-scope-lightadr.md#ladr-001-案件の対象と進め方))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261121-branch-scope-backlog.md`) に追記します。残課題は全量をこの案件の残課題一覧に引き継いであり、転記元の残課題一覧 (`docs/prompts/past/261120-expiry-sandbox-verification-backlog.md`) は、すべての項目を「次フェーズへ転記」にして `docs/prompts/past/` へ移しました ([LADR-002](261121-branch-scope-lightadr.md#ladr-002-残課題の全量の引き継ぎ))。以後の各課題の状態と経緯は、この案件の残課題一覧で更新します。第 4 章の項目は、最終仕様 (`261121-branch-scope-design.md`) の 7.3 の手順に合わせています。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 依頼とスキル (agree-before-implement) の確認
    - ✅ 1.2. 前提条件の調査 (`internal/store/scope.go`、`internal/store/chat.go`、`internal/store/memory.go`、`internal/store/schema/` の 0002 と 0005、`internal/profile/profile.go`、`internal/tools/lounge.go`、`internal/tools/help.go`、`internal/tools/chat.go`、`cmd/agent-lounge-mcp/admin.go`、`internal/export/export.go`、`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/agent-lounge-mcp-proposal.md`、これまでの案件の記録)
    - ✅ 1.3. 一時フォルダーでの事前の確認 (リポジトリは変えない): ブランチを調べる Git のコマンドの出力 ([Q-004](261121-branch-scope-discussion.md#q-004-ブランチの名前の調べ方))、ブランチの名前に使える文字 ([Q-006](261121-branch-scope-discussion.md#q-006-ブランチのスコープの識別子の形))、`lounge_help` の本文の今の大きさ ([Q-013](261121-branch-scope-discussion.md#q-013-使い方の案内文の大きさ))
    - ✅ 1.4. 残課題の全量の引き継ぎ (`261121-branch-scope-backlog.md` の作成、転記元 `261120-expiry-sandbox-verification-backlog.md` の状態とメタ情報の更新、前の案件の記録の `docs/prompts/past/` への移動)
    - ✅ 1.5. 確認項目 ([Q-001](261121-branch-scope-discussion.md#q-001-ブランチを設ける形) から [Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て)) の作成、簡易 ADR への最初の決定 (LADR-001、LADR-002) の記録
    - ✅ 1.6. 第 1 ターンの回答 (Q-001 から Q-016) の受付と記録 (JST 日時、Git ユーザー名)
    - ✅ 1.7. 回答の検証 (追加の確認項目は、なし。[Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て) の補足にない 2 点は、最終仕様の 7.2 に書いて、第 2 ゲートで確認を受ける)
    - ✅ 1.8. 簡易 ADR への決定の反映 (LADR-003 から LADR-009)、作業 WBS の見直し。残課題一覧は、追記する事項がなく、変更なし
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の決定事項と、根拠となった解決済み Q の回答または利用者の明示的な指示との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261121-branch-scope-design.md` の作成 (知見の反映計画を含む)
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)。2026-11-22 21:54 JST に承認受領
    - ✅ 3.4. セッション用の一時フォルダーの複製での試作 (リポジトリは変えない)。Grok 4.7 (high) による試作 ([Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て) の「10:50 JST まで Grok を優先」に合わせ、承認を待つ間に行う)。コードとテストの試作は完了し、このセッションが点検した (`go vet`、`go test -race` が通る。案内文は 18,968 バイト。版 6 と版 7 のテーブルと索引の違いは、意図した制約と `sessions.branch` だけ)。文書の試作は、Grok の利用量が 10:46 に尽きて、変更のないまま終了した。結果をリポジトリに取り込むのは、第 2 ゲートの承認の後
- 🔄 4. 承認された最終仕様に従った作業の実施
    - 🔄 4.1. 作業ブランチ `feature/branch-scope` の作成と、作業前の記録の最初のコミット ([Q-015](261121-branch-scope-discussion.md#q-015-作業ブランチとマージ)) ← **CURRENT**
    - ⏸️ 4.2. スキーマの版 7 (`internal/store/schema/0007_branch.sql`)、`internal/store/` (スコープの規則、セッションのブランチ、`ErrNoBranch`、セッションの一覧)、`internal/profile/` (`Branch`) とテスト
    - ⏸️ 4.3. `internal/tools/` (`lounge_join` の引数と出力と `hint`、`lounge_who` の出力、`chat.go`、`delivery.go`、`memory.go` の `scope` の説明文とエラーの文、`help.go` の本文と上限) とテスト
    - ⏸️ 4.4. 管理コマンドと書き出し (`cmd/agent-lounge-mcp/admin.go`、`internal/export/`) とテスト
    - ⏸️ 4.5. 機械的検証 (`go vet ./...`、`go test -race ./...`) と、実装のレビュー
    - ⏸️ 4.6. 文書 (`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/agent-lounge-mcp-proposal.md`。該当する記述があれば `README.md`、`docs/README.md`、`docs/ja/interactive-verification.md`、`.agents/skills/` の `SKILL.md`) の更新
    - ⏸️ 4.7. 正本ドキュメントおよびコード コメントへの永続的知見の反映 (最終仕様の 8 章)
    - ⏸️ 4.8. 実機の確認 (WSL の側。最終仕様の 5.2 の a から c)
    - ⏸️ 4.9. 実機の確認 (Windows の側のセッションに委譲。最終仕様の 5.3 の d から f)
    - ⏸️ 4.10. 確認の記録 (`261121-branch-scope-validation.md`) の作成と、文書の「検証済みの範囲」への反映
    - ⏸️ 4.11. 受入条件への適合確認 (全作業の完了確認)
    - ⏸️ 4.12. 簡易 ADR の最終確認 (各決定が案件終了時点の決定を示し、経緯が時系列にそろっていることの確認)
    - ⏸️ 4.13. 残課題一覧の再構築と整理
- ⏸️ 5. 成果物の承認の受付 (第 3 ゲート)
    - ⏸️ 5.1. 成果物・検証結果・残課題の提示
    - ⏸️ 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了)
    - ⏸️ 5.3. 承認の後の、`main` へのマージ、push、作業ブランチの削除 ([Q-015](261121-branch-scope-discussion.md#q-015-作業ブランチとマージ))

## 対象と参照資料

対象成果物: 最終仕様 (`261121-branch-scope-design.md`) の 3 章の成果物 (`internal/store/`、`internal/profile/`、`internal/tools/`、`internal/export/`、`cmd/agent-lounge-mcp/`、`docs/ja/`、`docs/agent-lounge-mcp-proposal.md` ほか)。

今回の対象範囲: スコープの種類に `branch` を足すこと ([LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲))。チャットとメモリの両方で使えます。引き継いだ残課題 14 件は、この案件の対象にしていません。

書き込み可能な検討文書: この文書、検討用 Markdown (`261121-branch-scope-discussion.md`)、簡易 ADR (`261121-branch-scope-lightadr.md`)、残課題一覧 (`261121-branch-scope-backlog.md`)。第 1 ゲートの後は、最終仕様 (`261121-branch-scope-design.md`)

参照資料と版:

- 利用者の依頼: 2026-11-21 の対話セッション (本文は、簡易 ADR の LADR-001 の経緯)
- スキル: agree-before-implement (リポジトリ `c-modernization-kit` の `app/general/.agents/skills/agree-before-implement/SKILL.md`。2026-11-21 07:43 JST に更新された版)
- サーバー: コミット `9b83815` (`main`)。スキーマの版は 6
- 転記元の残課題一覧: `docs/prompts/past/261120-expiry-sandbox-verification-backlog.md`
- Git: 2.56.0 (Linux x86_64、WSL 2)
- Go: モジュールのキャッシュにある 1.27.3 の道具 (この環境では、`go` が PATH にありません)
- `lounge_help` の本文: 既定の設定で 17,907 バイト (上限は 18,000 バイト)
- Grok の CLI: モデルの一覧に `grok-4.7`。agy の CLI: モデルの一覧に `claude-sonnet-5-5-low`、`-medium`、`-high` (2026-11-22 に確認)
- 各 CLI の版と利用量の残り、Windows の側のセッションの状態: 未確認 (実機の確認の前に確かめます)

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 設ける形 ([Q-001](261121-branch-scope-discussion.md#q-001-ブランチを設ける形))、利用場面 ([Q-002](261121-branch-scope-discussion.md#q-002-想定する利用場面)) |
| 範囲・境界 | 確認済み | 使える機能 ([Q-003](261121-branch-scope-discussion.md#q-003-ブランチのスコープを使える機能))、他のブランチの内容 ([Q-010](261121-branch-scope-discussion.md#q-010-他のブランチの内容を読む手段))、一覧での表示 ([Q-011](261121-branch-scope-discussion.md#q-011-相手の一覧でのブランチの表示))、既存の規則の当てはめ ([Q-012](261121-branch-scope-discussion.md#q-012-既存の規則をそのまま当てはめる事項))。引き継いだ残課題は対象外 ([LADR-002](261121-branch-scope-lightadr.md#ladr-002-残課題の全量の引き継ぎ)) |
| 入力・根拠 | 確認済み | 参照資料と版を上に記載。Git のコマンドの出力は、事前の確認で取得。Windows での、ブランチの名前の大文字と小文字の動作は、確認していないまま方針を確定し ([Q-007](261121-branch-scope-discussion.md#q-007-ブランチの名前の大文字と小文字))、実機の確認で確かめる ([Q-014](261121-branch-scope-discussion.md#q-014-確認の範囲)) |
| 整合性・依存関係 | 確認済み | 識別子の形 ([Q-006](261121-branch-scope-discussion.md#q-006-ブランチのスコープの識別子の形))、置き換えの規則 ([Q-008](261121-branch-scope-discussion.md#q-008-申告したブランチの置き換えの規則))、今の規則の当てはめ ([Q-012](261121-branch-scope-discussion.md#q-012-既存の規則をそのまま当てはめる事項))、本文の上限 ([Q-013](261121-branch-scope-discussion.md#q-013-使い方の案内文の大きさ))、作業ブランチ ([Q-015](261121-branch-scope-discussion.md#q-015-作業ブランチとマージ))。利用場面に C がなく、メモリでも使えること ([Q-002](261121-branch-scope-discussion.md#q-002-想定する利用場面)、[Q-003](261121-branch-scope-discussion.md#q-003-ブランチのスコープを使える機能)) は、矛盾しない (Q-002 の Assessment) |
| 実現性・例外 | 確認済み | 調べ方 ([Q-004](261121-branch-scope-discussion.md#q-004-ブランチの名前の調べ方))、ブランチがない場合 ([Q-005](261121-branch-scope-discussion.md#q-005-ブランチがないセッションの扱い))、大文字と小文字 ([Q-007](261121-branch-scope-discussion.md#q-007-ブランチの名前の大文字と小文字))、切り替えへの追従 ([Q-009](261121-branch-scope-discussion.md#q-009-ブランチの切り替えへの追従))、作業の担当 ([Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て)。補足にない 2 点は、最終仕様の 7.2 に書いて第 2 ゲートで確認) |
| 受入条件・検証 | 確認済み | 確認の範囲 ([Q-014](261121-branch-scope-discussion.md#q-014-確認の範囲))。受入条件は、最終仕様の 4 章 |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261121-branch-scope-discussion.md#q-001-ブランチを設ける形) | ブランチを設ける形 | 解決済み | Q-002 から Q-013 (A が前提) | [LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲) |
| [Q-002](261121-branch-scope-discussion.md#q-002-想定する利用場面) | 想定する利用場面 | 解決済み | Q-001、Q-003 | [LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲) |
| [Q-003](261121-branch-scope-discussion.md#q-003-ブランチのスコープを使える機能) | ブランチのスコープを使える機能 | 解決済み | Q-002、Q-010、Q-012 | [LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲) |
| [Q-004](261121-branch-scope-discussion.md#q-004-ブランチの名前の調べ方) | ブランチの名前の調べ方 | 解決済み | Q-005、Q-007 | [LADR-004](261121-branch-scope-lightadr.md#ladr-004-ブランチの申告と追従) |
| [Q-005](261121-branch-scope-discussion.md#q-005-ブランチがないセッションの扱い) | ブランチがないセッションの扱い | 解決済み | Q-004 | [LADR-004](261121-branch-scope-lightadr.md#ladr-004-ブランチの申告と追従) |
| [Q-006](261121-branch-scope-discussion.md#q-006-ブランチのスコープの識別子の形) | ブランチのスコープの識別子の形 | 解決済み | Q-007、Q-012 | [LADR-005](261121-branch-scope-lightadr.md#ladr-005-識別子と一覧での表示) |
| [Q-007](261121-branch-scope-discussion.md#q-007-ブランチの名前の大文字と小文字) | ブランチの名前の大文字と小文字 | 解決済み | Q-004、Q-006、Q-014 | [LADR-004](261121-branch-scope-lightadr.md#ladr-004-ブランチの申告と追従) |
| [Q-008](261121-branch-scope-discussion.md#q-008-申告したブランチの置き換えの規則) | 申告したブランチの置き換えの規則 | 解決済み | Q-009 | [LADR-004](261121-branch-scope-lightadr.md#ladr-004-ブランチの申告と追従) |
| [Q-009](261121-branch-scope-discussion.md#q-009-ブランチの切り替えへの追従) | ブランチの切り替えへの追従 | 解決済み | Q-008、Q-014 | [LADR-004](261121-branch-scope-lightadr.md#ladr-004-ブランチの申告と追従) |
| [Q-010](261121-branch-scope-discussion.md#q-010-他のブランチの内容を読む手段) | 他のブランチの内容を読む手段 | 解決済み | Q-003 | [LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲) |
| [Q-011](261121-branch-scope-discussion.md#q-011-相手の一覧でのブランチの表示) | 相手の一覧でのブランチの表示 | 解決済み | Q-012 | [LADR-005](261121-branch-scope-lightadr.md#ladr-005-識別子と一覧での表示) |
| [Q-012](261121-branch-scope-discussion.md#q-012-既存の規則をそのまま当てはめる事項) | 既存の規則をそのまま当てはめる事項 | 解決済み | Q-003、Q-006、Q-011 | [LADR-003](261121-branch-scope-lightadr.md#ladr-003-ブランチのスコープの形と範囲) |
| [Q-013](261121-branch-scope-discussion.md#q-013-使い方の案内文の大きさ) | 使い方の案内文の大きさ | 解決済み | Q-004、Q-009 | [LADR-006](261121-branch-scope-lightadr.md#ladr-006-使い方の案内文の上限) |
| [Q-014](261121-branch-scope-discussion.md#q-014-確認の範囲) | 確認の範囲 | 解決済み | Q-007、Q-009、Q-016 | [LADR-007](261121-branch-scope-lightadr.md#ladr-007-確認の範囲) |
| [Q-015](261121-branch-scope-discussion.md#q-015-作業ブランチとマージ) | 作業ブランチとマージ | 解決済み | Q-014、Q-016 | [LADR-008](261121-branch-scope-lightadr.md#ladr-008-作業ブランチとマージ) |
| [Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て) | モデルごとの作業の割り当て | 解決済み | Q-014、Q-015 | [LADR-009](261121-branch-scope-lightadr.md#ladr-009-作業の割り当て) |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261121-branch-scope-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 版 2 (2026-11-22。Q-001 から Q-016 がすべて解決済み)

最終仕様 (design.md) の合意版 (第 2 ゲート): 承認済み (2026-11-22 21:54 JST。担当変更および実機確認の運用方針を反映)

更新予定の成果物: 最終仕様の 3 章

受入条件: 最終仕様の 4 章 (11 項目)

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-21 23:00 ごろ | 案件の開始 | 利用者の依頼 ([LADR-001](261121-branch-scope-lightadr.md#ladr-001-案件の対象と進め方))。依頼に書かれたスキルのパスは、このリポジトリにはなく、リポジトリ `c-modernization-kit` の同じパスのスキルを使用 |
| 2026-11-21 23:16 | 残課題の全量の引き継ぎ | 前の案件の未対処の 14 件を `261121-branch-scope-backlog.md` へ転記し、前の案件の記録を `docs/prompts/past/` へ移動 ([LADR-002](261121-branch-scope-lightadr.md#ladr-002-残課題の全量の引き継ぎ)) |
| 2026-11-21 23:21 | 確認項目の提示 | 全体管理文書、検討用 Markdown (版 1、Q-001 から Q-016)、簡易 ADR を作成。記録は、`main` の作業ツリーにあり、コミットしていない |
| 2026-11-22 09:46 | 第 1 ターンの回答の受付 | 利用者が、検討用 Markdown に Q-001 から Q-016 の回答を記入して保存。推奨と違う回答は、[Q-013](261121-branch-scope-discussion.md#q-013-使い方の案内文の大きさ) (B: 上限を上げる) と [Q-016](261121-branch-scope-discussion.md#q-016-モデルごとの作業の割り当て) (その他: Grok、agy、サブエージェントの順) |
| 2026-11-22 09:49 | 第 1 ゲートの通過 | 16 件をすべて解決済みにし、決定を簡易 ADR (LADR-003 から LADR-009) に記録 (記録は 09:53)。追加の確認項目は、なし。検討用 Markdown の版を 2 に更新 |
| 2026-11-22 09:52 | 一時フォルダーでの試作の開始 | セッション用の一時フォルダーに、リポジトリの複製 (コミット `9b83815`、リモートなし) を作り、Grok 4.7 (high) に、最終仕様の 2 章と 3 章のコードとテストの試作を指示。リポジトリは変えていない |
| 2026-11-22 09:55 | 最終仕様の提示 | `261121-branch-scope-design.md` (09:51 に作成) の承認を依頼 (第 2 ゲート) |
| 2026-11-22 10:33 | 一時フォルダーでの試作 (コードとテスト) の完了 | Grok の試作が終わり、このセッションが複製の中で点検。`go vet ./...` と `go test -race ./...` が通り、案内文は 18,968 バイト (上限 19,000)。取り込みの前に直す点: 案内文の「After a merge」の文が 2 か所にある。10:34 から、同じ複製で文書の試作を Grok に指示 (10:50 までで打ち切り) |
| 2026-11-22 10:46 | Grok の利用量の上限 | 文書の試作の途中で、Grok が「usage balance exhausted」(402) で終了。文書の変更はなし。以後の実装の担当は、agy (Sonnet 5.5)。承認の前の試作は、ここまでとする |
| 2026-11-22 21:54 | 第 2 ゲートの通過と引き継ぎ | agy セッションへ引き継ぎ、最終仕様の承認を受領。本セッションが推進を担当し、実機確認は必要最小限として未実施分は残課題一覧 (`261121-branch-scope-backlog.md`) へ回す方針を確定 ([LADR-007](261121-branch-scope-lightadr.md#ladr-007-確認の範囲)、[LADR-009](261121-branch-scope-lightadr.md#ladr-009-作業の割り当て)) |

## 反映と検証の記録

未着手。作業後の対応箇所、検証結果、未完了項目を記録します。
