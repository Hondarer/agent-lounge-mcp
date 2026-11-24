<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-24 08:05:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261123-backlog-batch-backlog.md` (対策元の残課題一覧。BL-004、BL-009、BL-033 の (1)。BL-031 の (6) は容認して除く)。移動または削除されている場合があります。
下流文書: `261124-shared-space-rooms-discussion.md` (検討用 Markdown)、`261124-shared-space-rooms-lightadr.md` (簡易 ADR)、`261124-shared-space-rooms-design.md` (最終仕様)、`261124-shared-space-rooms-backlog.md` (残課題一覧)、`261124-shared-space-rooms-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# `shared` の空間とルームの作成・参加 全体管理

ファイル名: `261124-shared-space-rooms-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261124-shared-space-rooms-discussion.md`) に、決定事項は簡易 ADR (`261124-shared-space-rooms-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: shared-space-rooms

検討用 Markdown の版: 3

状態: 完了 (第 3 ゲート通過。CI の成功をもって承認)

## 作業 WBS と進捗管理

この案件の作業 WBS です。対策元の残課題一覧 `261123-backlog-batch-backlog.md` の BL-004 (`shared` の空間と `scope` 引数) と BL-009 (ルームの作成と参加の操作)、および BL-033 の (1) (参加の前のルームのやりとり) を対策します ([LADR-001](261124-shared-space-rooms-lightadr.md#ladr-001-案件の対象))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261124-shared-space-rooms-backlog.md`) に追記します。対策元の残課題一覧の状態と経緯は、進行に合わせて随時更新します。第 4 章の項目は、回答を受けて詳細にします。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 依頼とスキル (agree-before-implement。`/home/user/c-modernization-kit/app/general/.agents/skills/agree-before-implement`) の確認
    - ✅ 1.2. 前提条件の調査 (対策元の残課題一覧の BL-004、BL-009 と関連の BL-031、BL-033、前の案件の簡易 ADR の LADR-002、企画提案書の「空間の分け方」「宛先」「ツール」、`internal/store/scope.go`、`internal/store/chat.go`、`internal/store/schema/` の 0001・0002・0003・0007、`internal/tools/chat.go`・`address.go`・`delivery.go`・`help.go`、`docs/ja/reference.md` の「スコープと対象」「lounge_who」「メモリの項目と空間」「制限事項」、初回の検討 `../past/261117-chat-connectivity-discussion.md` の Q-003・Q-005)
    - ✅ 1.3. 対策元の残課題一覧 (`261123-backlog-batch-backlog.md`) のメタ情報と状態の更新 (BL-004、BL-009 を「検討中」にし、この案件の開始を記録)
    - ✅ 1.4. 確認項目 ([Q-001](261124-shared-space-rooms-discussion.md#q-001-2-つの課題の進め方) から [Q-012](261124-shared-space-rooms-discussion.md#q-012-企画提案書の更新)) の作成、簡易 ADR への最初の決定 (LADR-001) の記録
    - ✅ 1.5. 第 1 ターンの回答 (Q-001 から Q-012) の受付と記録 (2026-11-24 06:38 JST、Tetsuo Honda。あわせて対話での発言「MCP が管理する情報は共有目的であり、接続と情報の共有は別に考えるべき。情報は広く共有可能であるべき」)
    - ✅ 1.6. 第 1 ターンの回答の検証、簡易 ADR への反映 (LADR-001 の更新、LADR-002 から LADR-006 の追加)、WBS の見直し (BL-033 の (1) を追加)、対策元の残課題一覧への方針の反映 (BL-031 の (6) を容認して除き、BL-033 を「検討中」に)。Q-001 から Q-012 を解決済みにした
    - ✅ 1.7. 第 2 ターンの確認項目 ([Q-013](261124-shared-space-rooms-discussion.md#q-013-参加の前のやりとりを読めるルーム) から [Q-025](261124-shared-space-rooms-discussion.md#q-025-確認の方法)) の作成 (参加の前のやりとり、`shared` のメモリの細部、ルームの細部、確認の方法)
    - ✅ 1.8. 第 2 ターンの回答 (Q-013 から Q-025) の受付と記録 (2026-11-24 06:47 JST、Tetsuo Honda)
    - ✅ 1.9. 第 2 ターンの回答の検証、簡易 ADR への反映 (LADR-003 から LADR-005 の更新、LADR-007 の追加)、WBS の見直し (実機の確認を Claude Code の代表に)、残課題一覧 `261124-shared-space-rooms-backlog.md` の作成 (BL-048)。Q-013 から Q-025 を解決済みにした
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認 (Q-001 から Q-025 が解決済み)
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の LADR-001 から LADR-007 と、根拠となった解決済み Q の回答および対話での発言との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261124-shared-space-rooms-design.md` の作成 (知見の反映計画を含む)。対策元の残課題一覧の BL-004、BL-009、BL-033 を「対応中」に更新
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認 (決定から一意に決まらない細部は、最終仕様の 13 章にまとめ、第 2 ゲートで確認を受ける)
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)。2026-11-24 06:55 JST に承認受領。あわせて、難易度の低い実装を Sonnet のサブエージェントに依頼する指示を受けた ([LADR-008](261124-shared-space-rooms-lightadr.md#ladr-008-作業の進め方))
- ✅ 4. 承認された最終仕様に従った作業の実施 (順は、最終仕様の 12 章。依頼先は、難易度の低い 4.2 を Sonnet のサブエージェント、4.3 と 4.4 と文書を Claude Code とする。難易度の区分は、Claude Code の判断)
    - ✅ 4.1. 作業ブランチ `feature/shared-space-rooms` の作成と、作業前の記録の最初のコミット (`40f3549`)
    - ✅ 4.2. (コミット `ab7f967`。Sonnet のサブエージェントが実装し、Claude Code が点検。`memory_retention` の CHECK に `shared` を加えるスキーマ版 9 を追加) BL-004: `shared` のスコープ (`internal/store/scope.go`、`internal/store/` の関連箇所、`internal/tools/`、`internal/config/`、`cmd/agent-lounge-mcp/`、テスト)。完了時に対策元の残課題一覧を更新
    - ✅ 4.3. (コミット `6d38c0c`。Claude Code が実装。スキーマ版 10。使い方の案内も、このコミットで最終の内容に更新) BL-009: ルームの作成と参加 (`internal/store/schema/` の新しいスクリプト、`internal/store/`、`internal/tools/` の `chat_room`、テスト)。完了時に対策元の残課題一覧を更新
    - ✅ 4.4. (コミット `e91f7da`。Claude Code が実装) BL-033 の (1): 参加の前のルームのやりとりの読み返し (`internal/store/delivery.go`、`internal/tools/delivery.go`、テスト)。完了時に対策元の残課題一覧を更新
    - ✅ 4.5. (コミット `6d38c0c` の `help.go`、`0c8894d` の文書。文書の本文は Sonnet のサブエージェントが書き、Claude Code が点検。`probe.sh` のツールの一覧も更新) 使い方の案内 (`internal/tools/help.go`) と正本の文書 (`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/agent-lounge-mcp-proposal.md`、`.agents/skills/` の `SKILL.md`) の更新
    - ✅ 4.6. 機械的検証 (`go vet ./...`、`go test ./...`) と、Claude Code での実機の確認 (最終仕様の 9.2)、確認の記録 `261124-shared-space-rooms-validation.md` の作成。手元の `go vet ./...` (Linux、`GOOS=windows`)、`go test -race ./...`、Claude Code 2.1.331 での実機の確認は完了。CI (3 つの OS) は、run 45284144295 (`c54c91d`) で成功
    - ✅ 4.7. 受入条件への適合確認 (全作業の完了確認)。受入条件の 1 から 5 に適合
    - ✅ 4.8. 簡易 ADR の最終確認 (各決定が案件終了時点の決定を示し、経緯が時系列にそろっていることの確認)
    - ✅ 4.9. 残課題一覧 (`261124-shared-space-rooms-backlog.md`) の再構築と整理 (対策元の未対処の項目の全量の転記を含む)、および対策元の残課題一覧の最終状態・転記先の確定。13 件 (転記 11 件、BL-048、BL-049)。対策元の BL-004、BL-009、BL-033 を対応済み、BL-031 の (6) を容認、他の 11 件を「次フェーズへ転記」に更新
- ✅ 5. 成果物の承認の受付 (第 3 ゲート)
    - ✅ 5.1. 成果物・検証結果・残課題の提示
    - ✅ 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了)。利用者の「CI成功を以て承認します。」により、CI の成功 (2026-11-24 08:02 JST に確認) をもって承認

## 対象と参照資料

対象成果物: 最終仕様の 11 章と 12 章の各ファイル

今回の対象範囲: 対策元の残課題一覧の BL-004、BL-009、BL-033 の (1) ([LADR-001](261124-shared-space-rooms-lightadr.md#ladr-001-案件の対象))。BL-031 の (6) は、仕様として容認して除く ([LADR-002](261124-shared-space-rooms-lightadr.md#ladr-002-情報の共有の方針))。BL-031 の (2)(3) とその他の項目は対象外

書き込み可能な検討文書: この文書、検討用 Markdown (`261124-shared-space-rooms-discussion.md`)、簡易 ADR (`261124-shared-space-rooms-lightadr.md`)。残課題一覧 (`261124-shared-space-rooms-backlog.md`、対策元の `261123-backlog-batch-backlog.md`) は、どの段階でも更新できる

参照資料と版:

- 対策元の残課題一覧: `261123-backlog-batch-backlog.md` (整理済み。BL-004、BL-009、BL-031、BL-033)
- 前の案件の簡易 ADR: `261123-backlog-batch-lightadr.md` の LADR-002
- 企画提案書: `docs/agent-lounge-mcp-proposal.md` (コミット `055a085` の時点)
- 正本の文書: `docs/ja/reference.md`、`docs/ja/user-guide.md` (同上)
- ソース コード: `main` のコミット `055a085`

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 情報の共有の方針 ([LADR-002](261124-shared-space-rooms-lightadr.md#ladr-002-情報の共有の方針))。`shared` はすべてのユーザーが使える ([Q-004](261124-shared-space-rooms-discussion.md#q-004-shared-を使えるユーザー)) |
| 範囲・境界 | 確認済み | [Q-001](261124-shared-space-rooms-discussion.md#q-001-2-つの課題の進め方)、[Q-002](261124-shared-space-rooms-discussion.md#q-002-bl-031-の-6-を合わせて扱うか)、[Q-003](261124-shared-space-rooms-discussion.md#q-003-bl-033-の-1-を合わせて扱うか)、[Q-006](261124-shared-space-rooms-discussion.md#q-006-メモリの-shared)、[Q-010](261124-shared-space-rooms-discussion.md#q-010-ルームを作れるスコープ)、[Q-012](261124-shared-space-rooms-discussion.md#q-012-企画提案書の更新)、[Q-013](261124-shared-space-rooms-discussion.md#q-013-参加の前のやりとりを読めるルーム)、[Q-014](261124-shared-space-rooms-discussion.md#q-014-他のセッションの間のセッション宛てのやりとり) |
| 入力・根拠 | 確認済み | 企画提案書の空間とツールの定義、残課題一覧、`internal/store/scope.go`、`internal/store/cleanup.go` (保存期限) |
| 整合性・依存関係 | 確認済み | 共有の方針と Q-004、Q-005、Q-013、BL-031 の (6) の容認が整合。`shared` のメモリの規則は `repository` とそろえる ([Q-015](261124-shared-space-rooms-discussion.md#q-015-shared-のメモリの更新と削除)、[Q-016](261124-shared-space-rooms-discussion.md#q-016-ユーザーの削除と-shared-のメモリ)) |
| 実現性・例外 | 確認済み | ルームの細部 ([Q-019](261124-shared-space-rooms-discussion.md#q-019-ルームの名前の規則) から [Q-024](261124-shared-space-rooms-discussion.md#q-024-スコープを出たセッションの参加の記録))、メモリの上限と忘却 ([Q-017](261124-shared-space-rooms-discussion.md#q-017-shared-のメモリの件数の上限)、[Q-018](261124-shared-space-rooms-discussion.md#q-018-shared-のメモリの忘却の既定値))。一意に決まらない細部は最終仕様の 13 章で第 2 ゲートの確認を受ける |
| 受入条件・検証 | 確認済み | [Q-025](261124-shared-space-rooms-discussion.md#q-025-確認の方法)。最終仕様の 9 章と 10 章 |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261124-shared-space-rooms-discussion.md#q-001-2-つの課題の進め方) | 2 つの課題の進め方 | 解決済み | Q-010 | LADR-001 |
| [Q-002](261124-shared-space-rooms-discussion.md#q-002-bl-031-の-6-を合わせて扱うか) | BL-031 の (6) を合わせて扱うか | 解決済み | Q-003、Q-004、Q-013 | LADR-001、LADR-002 |
| [Q-003](261124-shared-space-rooms-discussion.md#q-003-bl-033-の-1-を合わせて扱うか) | BL-033 の (1) を合わせて扱うか | 解決済み | Q-002、Q-008、Q-013、Q-014 | LADR-001、LADR-005 |
| [Q-004](261124-shared-space-rooms-discussion.md#q-004-shared-を使えるユーザー) | `shared` を使えるユーザー | 解決済み | Q-002、Q-005 | LADR-003 |
| [Q-005](261124-shared-space-rooms-discussion.md#q-005-lounge_who-の一覧の範囲) | `lounge_who` の一覧の範囲 | 解決済み | Q-004 | LADR-003 |
| [Q-006](261124-shared-space-rooms-discussion.md#q-006-メモリの-shared) | メモリの `shared` | 解決済み | Q-004、Q-015 から Q-018 | LADR-003 |
| [Q-007](261124-shared-space-rooms-discussion.md#q-007-shared-の識別子の表し方) | `shared` の識別子の表し方 | 解決済み | なし | LADR-003 |
| [Q-008](261124-shared-space-rooms-discussion.md#q-008-ルームの届く範囲の決め方) | ルームの届く範囲の決め方 | 解決済み | Q-003、Q-009 から Q-011、Q-013、Q-019 から Q-024 | LADR-004 |
| [Q-009](261124-shared-space-rooms-discussion.md#q-009-ルームの操作のツールの形) | ルームの操作のツールの形 | 解決済み | Q-008、Q-012 | LADR-004 |
| [Q-010](261124-shared-space-rooms-discussion.md#q-010-ルームを作れるスコープ) | ルームを作れるスコープ | 解決済み | Q-001、Q-004、Q-008 | LADR-004 |
| [Q-011](261124-shared-space-rooms-discussion.md#q-011-lounge-ルームの扱い) | `lounge` ルームの扱い | 解決済み | Q-008 | LADR-004 |
| [Q-012](261124-shared-space-rooms-discussion.md#q-012-企画提案書の更新) | 企画提案書の更新 | 解決済み | Q-009 | LADR-006 |
| [Q-013](261124-shared-space-rooms-discussion.md#q-013-参加の前のやりとりを読めるルーム) | 参加の前のやりとりを読めるルーム | 解決済み | Q-003、Q-008、Q-014 | LADR-005 |
| [Q-014](261124-shared-space-rooms-discussion.md#q-014-他のセッションの間のセッション宛てのやりとり) | 他のセッションの間のセッション宛てのやりとり | 解決済み | Q-003、Q-013 | LADR-005 |
| [Q-015](261124-shared-space-rooms-discussion.md#q-015-shared-のメモリの更新と削除) | `shared` のメモリの更新と削除 | 解決済み | Q-006、Q-016 | LADR-003 |
| [Q-016](261124-shared-space-rooms-discussion.md#q-016-ユーザーの削除と-shared-のメモリ) | ユーザーの削除と `shared` のメモリ | 解決済み | Q-015 | LADR-003 |
| [Q-017](261124-shared-space-rooms-discussion.md#q-017-shared-のメモリの件数の上限) | `shared` のメモリの件数の上限 | 解決済み | Q-006 | LADR-003 |
| [Q-018](261124-shared-space-rooms-discussion.md#q-018-shared-のメモリの忘却の既定値) | `shared` のメモリの忘却の既定値 | 解決済み | Q-006 | LADR-003 |
| [Q-019](261124-shared-space-rooms-discussion.md#q-019-ルームの名前の規則) | ルームの名前の規則 | 解決済み | Q-008、Q-021 | LADR-004 |
| [Q-020](261124-shared-space-rooms-discussion.md#q-020-ルームの数の上限) | ルームの数の上限 | 解決済み | Q-021 | LADR-004 |
| [Q-021](261124-shared-space-rooms-discussion.md#q-021-ルームの削除) | ルームの削除 | 解決済み | Q-020 | LADR-004 |
| [Q-022](261124-shared-space-rooms-discussion.md#q-022-作成したセッションの参加) | 作成したセッションの参加 | 解決済み | Q-009 | LADR-004 |
| [Q-023](261124-shared-space-rooms-discussion.md#q-023-参加していないルームへの投稿) | 参加していないルームへの投稿 | 解決済み | Q-008、Q-022 | LADR-004 |
| [Q-024](261124-shared-space-rooms-discussion.md#q-024-スコープを出たセッションの参加の記録) | スコープを出たセッションの参加の記録 | 解決済み | Q-008 | LADR-004 |
| [Q-025](261124-shared-space-rooms-discussion.md#q-025-確認の方法) | 確認の方法 | 解決済み | なし | LADR-007 |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261124-shared-space-rooms-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 版 3 (2026-11-24 06:50 JST)

最終仕様 (design.md) の合意版 (第 2 ゲート): 2026-11-24 06:53 JST の版 (2026-11-24 06:55 JST に承認)

更新予定の成果物: 最終仕様の 11 章 (知見の反映計画) と 12 章 (実装の順) のとおり

受入条件: 最終仕様の 10 章のとおり

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-24 06:30 | 案件の開始 | 利用者の指示「BL-004, BL-009 を agree-before-implement で進行したい」。全体管理文書、検討用 Markdown (版 1、Q-001 から Q-012)、簡易 ADR (LADR-001) を作成。対策元の残課題一覧の BL-004、BL-009 を「検討中」に更新 |
| 2026-11-24 06:39 | 第 1 ターンの回答の受領 | Q-001 から Q-012 (検討用 Markdown の版 1)。対話での発言 (情報の共有の方針) |
| 2026-11-24 06:42 | 方針の変更と WBS の見直し | 範囲に BL-033 の (1) を追加し、BL-031 の (6) を容認して除いた ([LADR-001](261124-shared-space-rooms-lightadr.md#ladr-001-案件の対象)、[LADR-002](261124-shared-space-rooms-lightadr.md#ladr-002-情報の共有の方針))。検討用 Markdown を版 2 にし、Q-013 から Q-025 を追加 |
| 2026-11-24 06:47 | 第 2 ターンの回答の受領 | Q-013 から Q-025 (検討用 Markdown の版 2) |
| 2026-11-24 06:50 | 第 1 ゲートの通過 | 全 Q が解決済み。検討用 Markdown の版 3 を合意版とする。Q-025 の補足により、実機の CLI の確認を Claude Code の代表に変更 ([LADR-007](261124-shared-space-rooms-lightadr.md#ladr-007-確認の方法)) |
| 2026-11-24 06:53 | 最終仕様の作成 | `261124-shared-space-rooms-design.md`。第 2 ゲートの承認を待つ |
| 2026-11-24 06:55 | 第 2 ゲートの通過 | 利用者の「承認します。難易度の低い実装はSonnetをサブエージェントで利用してください。」([LADR-008](261124-shared-space-rooms-lightadr.md#ladr-008-作業の進め方)) |
| 2026-11-24 07:12 | 実装の完了 (4.2 から 4.4) | 実装中に、`memory_retention` の CHECK 制約が `shared` を受け付けないことが判明し、スキーマ版 9 を追加した。参加の記録のスクリプトは版 10 になった。最終仕様の 6.9 をこれに合わせて直した。`go vet ./...`、`go test -race ./...`、`GOOS=windows go vet ./...` が通過 |
| 2026-11-24 07:19 | 文書、実機の確認、残課題の整理の完了 | 実機の確認で、Claude Code 2.1.331 が `lounge_help` の案内を空として扱う問題を見つけ、範囲外として残課題 BL-049 に記録した。成果物の承認 (第 3 ゲート) を待つ |
| 2026-11-24 07:48 | 作業ブランチの push と CI の起動 | 利用者の許可「pushを許可する。CI成功を以て承認します。」。案内の上限は 19,300 バイトのまま ([LADR-009](261124-shared-space-rooms-lightadr.md#ladr-009-使い方の案内の大きさの上限)) |
| 2026-11-24 07:54 | 注記の追加 | `maxGuideBytes` に、上限を広げる前に Copilot CLI で測り直す旨を記載 (`c54c91d`) |
| 2026-11-24 08:02 | 第 3 ゲートの通過 | CI (run 45284144295、`c54c91d`) が 3 つの OS で成功し、利用者の指示により承認とした。前のコミットの run 45283192071 も成功 |
| 2026-11-24 08:05 | 対策元の記録の移動 | 案件 `backlog-batch` の残課題が全件「対応済み」または「次フェーズへ転記」になったため、その記録 (6 つ) を `docs/prompts/past/` へ移動した。この案件の記録は、残課題一覧に未対処の項目があるため `docs/prompts/current/` に置く |

## 反映と検証の記録

| 課題 | コミット | 対応箇所 |
| --- | --- | --- |
| BL-004 (`shared`) | `ab7f967` | `internal/store/scope.go`、`internal/store/schema/0009_shared_scope.sql`、`internal/tools/` のスコープの説明とエラー文、`internal/export/export.go`、`cmd/agent-lounge-mcp/admin.go`、テスト |
| BL-009 (ルーム) | `6d38c0c` | `internal/store/rooms.go`、`internal/store/chat.go`、`internal/store/cleanup.go`、`internal/store/schema/0010_room_members.sql`、`internal/names/names.go`、`internal/tools/room.go`、`internal/tools/lounge.go`、`internal/tools/help.go`、`internal/config/config.go`、`cmd/agent-lounge-mcp/main.go`・`cleanup.go`、テスト |
| BL-033 の (1) (読み返し) | `e91f7da` | `internal/store/delivery.go`、`internal/tools/delivery.go`、テスト |
| 文書 | `0c8894d` | `docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/agent-lounge-mcp-proposal.md`、`README.md`、`.agents/skills/agent-lounge-connect/` (`SKILL.md`、`scripts/probe.sh`) |

Table: 対応箇所

検証の結果は、確認の記録 `261124-shared-space-rooms-validation.md` のとおりです。手元の自動テスト、CI (3 つの OS。run 45284144295)、Claude Code 2.1.331 での実機の確認 (最終仕様の 9.2 の 4 項目) は、すべて通りました。

未完了の項目:

- Codex、Copilot CLI、Grok、agy での実機の確認は、未検証として残課題 BL-048 にしました。
- 確認の中で見つけた、Claude Code 2.1.331 で `lounge_help` の案内が空として扱われる問題は、範囲外として残課題 BL-049 にしました。
