<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-23 12:10:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `../past/261121-branch-scope-backlog.md`、`../past/261122-cli-without-mcp-backlog.md` (対策元の残課題一覧。全件を `261123-backlog-batch-backlog.md` に転記済み)。移動または削除されている場合があります。
下流文書: `261123-backlog-batch-discussion.md` (検討用 Markdown)、`261123-backlog-batch-lightadr.md` (簡易 ADR)、`261123-backlog-batch-design.md` (最終仕様)、`261123-backlog-batch-backlog.md` (残課題一覧)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# 残課題の一括対策 (WSL で進められる項目) 全体管理

ファイル名: `261123-backlog-batch-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261123-backlog-batch-discussion.md`) に、決定事項は簡易 ADR (`261123-backlog-batch-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: backlog-batch

検討用 Markdown の版: 3

状態: 作業中 (第 2 ゲート通過)

## 作業 WBS と進捗管理

この案件の作業 WBS です。対策元の 2 つの残課題一覧 (18 件。この案件の残課題一覧 `261123-backlog-batch-backlog.md` に全件を転記済み) のうち、WSL の環境だけで進められる項目を一括で対策します ([LADR-001](261123-backlog-batch-lightadr.md#ladr-001-案件の対象と進め方))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261123-backlog-batch-backlog.md`) に追記します。対策元の残課題一覧の状態と経緯は、進行に合わせて随時更新します。第 4 章の項目は、第 1 ターンの回答を受けて詳細にします。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 依頼とスキル (agree-before-implement。`/home/user/c-modernization-kit/app/general/.agents/skills/agree-before-implement`) の確認
    - ✅ 1.2. 対話セッションでの事前の確認 (今回の範囲、保留する項目、取り下げの決め方、スキルの場所。[LADR-001](261123-backlog-batch-lightadr.md#ladr-001-案件の対象と進め方)、[LADR-002](261123-backlog-batch-lightadr.md#ladr-002-今回扱わない項目))
    - ✅ 1.3. 前提条件の調査 (2 つの残課題一覧、`internal/profile/profile.go`、`internal/store/` の `memory.go`・`retention.go`・`admin.go`、`internal/config/config.go`、`internal/tools/` の `lounge.go`・`presence.go`・`help.go`、`cmd/agent-lounge-mcp/` の `main.go`・`admin.go`、`scripts/interactive/harness/agents.py`、host-shell-mcp の `docs/ja/cli-without-mcp.md`)。論点の洗い出しは、利用者の指示に従い、Codex (gpt-6.2-sol、reasoning effort low、読み取り専用) で行い、このセッションが要点をコードで確かめた
    - ✅ 1.4. 対策元の残課題一覧 (`261121-branch-scope-backlog.md`、`261122-cli-without-mcp-backlog.md`) のメタ情報と状態の更新 (対象の項目を「検討中」にし、この案件の開始を記録)
    - ✅ 1.5. 確認項目 ([Q-001](261123-backlog-batch-discussion.md#q-001-残課題の引き継ぎ方) から [Q-036](261123-backlog-batch-discussion.md#q-036-cli-の方式の実機確認の対象)) の作成、簡易 ADR への最初の決定 (LADR-001、LADR-002) の記録
    - ✅ 1.6. 第 1 ターンの回答 (Q-001 から Q-036) の受付と記録 (2026-11-23 09:11 JST、Tetsuo Honda)
    - ✅ 1.7. 第 1 ターンの回答の検証と、利用者の指示による推奨の再点検 (企画提案書 `docs/agent-lounge-mcp-proposal.md` の目的と要件に照らして、変更を小さくする理由で推奨していないかを確かめた)。Q-008、Q-018、Q-019、Q-021、Q-027 を未解決とし、追加の確認項目 ([Q-037](261123-backlog-batch-discussion.md#q-037-受信者ごとの送達状態の表示) から [Q-046](261123-backlog-batch-discussion.md#q-046-cli-の方式の使い方の案内の置き場所)) を作成。RFC 8265 の規則の動作を、一時フォルダーで確認
    - ✅ 1.8. 解決済みの Q の、簡易 ADR への反映 (LADR-003 から LADR-010)
    - ✅ 1.9. 第 2 ターンの回答 (Q-037 から Q-046) の受付と記録 (2026-11-23 09:32 JST、Tetsuo Honda)
    - ✅ 1.10. 第 2 ターンの回答の検証、簡易 ADR への決定の反映 (LADR-006、LADR-008、LADR-009 の更新、LADR-011 の追加)。Q-008、Q-018、Q-019、Q-021、Q-027 を解決済みにした。Q-040 (B: ユーザー名も広げる) から決まらない点を、[Q-047](261123-backlog-batch-discussion.md#q-047-ユーザー名の大文字と小文字) から [Q-049](261123-backlog-batch-discussion.md#q-049-書き出しのファイル名での-unicode-の文字) として追加
    - ✅ 1.11. 残課題の全量の引き継ぎ ([LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方)): `261123-backlog-batch-backlog.md` を作成 (18 件)。対策元の 2 つの残課題一覧は全件を「次フェーズへ転記」にし、`branch-scope` の記録 (6 つ) と `cli-without-mcp` の残課題一覧を `docs/prompts/past/` へ移動 (`git mv`。コミットは、作業ブランチの最初のコミットで行う)
    - ✅ 1.12. 第 3 ターンの回答 (Q-047 から Q-049) の受付と記録 (2026-11-23 09:41 JST、Tetsuo Honda)
    - ✅ 1.13. 第 3 ターンの回答の検証、簡易 ADR への決定の反映 (LADR-011)。RFC 8265 の UsernameCaseMapped の動作を、一時フォルダーで確認
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認 (Q-001 から Q-049 が解決済み)
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の LADR-001 から LADR-011 と、根拠となった解決済み Q の回答および利用者の明示的な指示との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261123-backlog-batch-design.md` の作成 (知見の反映計画を含む)。残課題一覧の、対策する項目の状態を「対応中」に更新
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認 (確認項目への回答から一意に決まらない細部は、最終仕様の 15 章にまとめ、第 2 ゲートで確認を受ける)
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)。2026-11-23 10:30 JST に承認受領。あわせて、難易度の低い作業を Grok に依頼する指示を受けた ([LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方))
- 🔄 4. 承認された最終仕様に従った作業の実施 (順は、最終仕様の 12.1。実装の依頼先は、難易度の低い BL-046、BL-045、BL-011 を Grok、BL-014、BL-034、BL-047、BL-035 の (17) を Codex とする。難易度の区分は、Claude Code の判断)
    - ✅ 4.1. 作業ブランチ `feature/backlog-batch` の作成と、作業前の記録の最初のコミット (`56fa246`)
    - ✅ 4.2. (コミット `3e31fe9`。Grok が実装、Claude Code が点検) BL-046 (最終仕様の 2 章。`internal/profile/` とテスト)
    - ✅ 4.3. (コミット `f3e5425`。Grok が実装、Claude Code が点検) BL-045 (3 章。`internal/store/` の定期の処理と `user remove`、テスト)
    - ✅ 4.4. (コミット `d424768`。Codex が実装。点検で、空の名前の扱い、宛先のエラーの案内文、書き出しのファイル名の長さの上限を Claude Code が直した) BL-014 (4 章。名前の検査と正規化、通す箇所、`export.SafeName`、管理コマンドの `--`、`internal/tools/help.go` の名前の説明、テスト)
    - ⏸️ 4.5. (未着手。依頼文は用意済みで、Grok に依頼する。利用量の上限で中断) ← **CURRENT** BL-011 (5 章。`message list`、`message show`、絞り込み、`memory list --scope-id`、テスト)
    - ⏸️ 4.6. BL-034 (6 章。起動オプション、種類ごとの適用、スキーマの版 8、`memory list`、`lounge_help` と `lounge_info`、テスト)
    - ⏸️ 4.7. BL-047 (7 章。`connection`、`tools`、`call`、常駐プロセス、プロフィールの収集、案内の文章、テスト)
    - ✅ 4.8. (コミット `ace7065`。Codex が実装。点検で、静止の判定を英数字の出力に戻し、再応答の条件に英数字の出力を加えた。最終仕様の 8 章に反映) BL-035 の (17) (8 章。`scripts/interactive/harness/agents.py` とテスト)
    - ⏸️ 4.9. 各課題の点検 (差分、`go vet`、`go test -race`、仕様との照合) と、GitHub への push、CI の 3 つの OS の結果の確認
    - ⏸️ 4.10. 実機の確認 (9.2 の a から h。WSL)
    - ⏸️ 4.11. 利用者向けの文書の更新 (11 章)、取り下げの反映 (10 章)、スキルの文書
    - ⏸️ 4.12. 正本ドキュメントおよびコード コメントへの永続的知見の反映 (14 章)、確認の記録 (`261123-backlog-batch-validation.md`) の作成、文書の「検証済みの範囲」への反映
    - ⏸️ 4.13. 全体のレビュー、受入条件 (13 章) への適合確認、簡易 ADR の最終確認
    - ⏸️ 4.14. 残課題一覧 (`261123-backlog-batch-backlog.md`) の再構築と整理
- ⏸️ 5. 成果物の承認の受付 (第 3 ゲート)
    - ⏸️ 5.1. 成果物・検証結果・残課題の提示
    - ⏸️ 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了)

## 対象と参照資料

対象成果物: サーバーと管理コマンドのコード (`cmd/agent-lounge-mcp/`、`internal/`)、検証の基盤 (`scripts/interactive/`)、利用者向けの文書 (`docs/ja/`、`README.md`、`docs/README.md`、`.agents/skills/` の `SKILL.md`)。

今回の対象範囲:

| 区分 | 課題 |
| --- | --- |
| 実装する | BL-046、BL-045、BL-011、BL-035 の (17)、BL-034、BL-014、BL-047 |
| WSL で実機確認する | BL-044 の WSL の側 (最終仕様 `261121-branch-scope-design.md` の 5.2 の a から c)、BL-040 (agy 1.5.14 での確かめ直し)、BL-014 と BL-047 の実機の確認 |
| 取り下げるかを検討用 Markdown で決める | BL-033 の (2)、BL-031 の (4) と (5)、BL-030 のうちサーバー自身の HTTPS |
| 今回扱わない (保留) | BL-004、BL-009、BL-033 の (1)、BL-031 の (2)(3)(6)、BL-013、BL-028、BL-030 の別のマシンでの検証、BL-036、BL-042、BL-044 の Windows の側、BL-035 の (17) 以外 |

書き込み可能な検討文書: この文書、検討用 Markdown (`261123-backlog-batch-discussion.md`)、簡易 ADR (`261123-backlog-batch-lightadr.md`)。残課題一覧 (`261123-backlog-batch-backlog.md` と、対策元の 2 つ) は、どの段階でも更新できます。

参照資料と版:

- `main` の `5088ad0` (2026-11-23) のコードと文書。
- 対策元の残課題一覧: `261121-branch-scope-backlog.md` (17 件)、`261122-cli-without-mcp-backlog.md` (1 件)。
- host-shell-mcp の `docs/ja/cli-without-mcp.md` (BL-047 の先行する実装の文書)。
- この環境の版: Go 1.27.3 (linux/amd64)、Go SDK `github.com/modelcontextprotocol/go-sdk` v1.8.0 (公開されている最新は v1.9.0-pre.1 で、正式版の v1.9.0 は出ていません)、Claude Code 2.1.330、Codex CLI 0.171.0、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.14、Git 2.56.0。

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 利用者の依頼と対話での回答 ([LADR-001](261123-backlog-batch-lightadr.md#ladr-001-案件の対象と進め方))。各課題の目的は、対策元の残課題一覧の「概要と問題点」 |
| 範囲・境界 | 確認済み | LADR-001、LADR-002、LADR-010 (取り下げ)、LADR-011 (ユーザー名を対象に加えた)。最終仕様の 1.2 |
| 入力・根拠 | 確認済み | 「参照資料と版」のとおり |
| 整合性・依存関係 | 確認済み | 企画提案書の目的と要件に照らした再点検を行い、Q-037 から Q-049 で解決した。`AGENTS.md` の規則 (SQL の置き場所、スキーマは新しいスクリプト、使い方の変更は `help.go` が先、文書は `docs/ja/` が正本) は、最終仕様の 6.3 と 11 章に反映 |
| 実現性・例外 | 確認済み | BL-047 の構成、切断と参加のし直しは LADR-009、BL-034 の設定の変更と記録は LADR-008 |
| 受入条件・検証 | 確認済み | 最終仕様の 9 章 (確認) と 13 章 (受入条件) |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。Q-ID には検討用 Markdown の該当見出しへのリンクを設けます。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261123-backlog-batch-discussion.md#q-001-残課題の引き継ぎ方) | 残課題の引き継ぎ方 | 解決済み | なし | [LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方) |
| [Q-002](261123-backlog-batch-discussion.md#q-002-作業ブランチとマージ) | 作業ブランチとマージ | 解決済み | なし | [LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方) |
| [Q-003](261123-backlog-batch-discussion.md#q-003-実装と審査の担当) | 実装と審査の担当 | 解決済み | なし | [LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方) |
| [Q-004](261123-backlog-batch-discussion.md#q-004-ブランチの名前の前後の空白) | BL-046: ブランチの名前の前後の空白 | 解決済み | Q-005 | [LADR-004](261123-backlog-batch-lightadr.md#ladr-004-ブランチの名前の検査-bl-046) |
| [Q-005](261123-backlog-batch-discussion.md#q-005-head-という名前の拒否の範囲) | BL-046: HEAD という名前の拒否の範囲 | 解決済み | Q-004 | [LADR-004](261123-backlog-batch-lightadr.md#ladr-004-ブランチの名前の検査-bl-046) |
| [Q-006](261123-backlog-batch-discussion.md#q-006-git-の構文に合わせる範囲) | BL-046: Git の構文に合わせる範囲 | 解決済み | Q-004、Q-005 | [LADR-004](261123-backlog-batch-lightadr.md#ladr-004-ブランチの名前の検査-bl-046) |
| [Q-007](261123-backlog-batch-discussion.md#q-007-メモリのない活動行の保存期間) | BL-045: メモリのない活動行の保存期間 | 解決済み | Q-015、Q-017 | [LADR-005](261123-backlog-batch-lightadr.md#ladr-005-メモリのない活動行の削除-bl-045) |
| [Q-008](261123-backlog-batch-discussion.md#q-008-メッセージの一覧の表示の粒度) | BL-011: メッセージの一覧の表示の粒度 | 解決済み | Q-037、Q-038 | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-009](261123-backlog-batch-discussion.md#q-009-メッセージの一覧の既定の件数) | BL-011: メッセージの一覧の既定の件数 | 解決済み | Q-008 | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-010](261123-backlog-batch-discussion.md#q-010-ユーザーでの絞り込みの意味) | BL-011: ユーザーでの絞り込みの意味 | 解決済み | なし | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-011](261123-backlog-batch-discussion.md#q-011-スコープでの絞り込みの単位) | BL-011: スコープでの絞り込みの単位 | 解決済み | なし | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-012](261123-backlog-batch-discussion.md#q-012-期間での絞り込み) | BL-011: 期間での絞り込み | 解決済み | なし | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-013](261123-backlog-batch-discussion.md#q-013-絞り込みを一覧にも付けるか) | BL-011: 絞り込みを一覧にも付けるか | 解決済み | Q-009 から Q-012 | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-014](261123-backlog-batch-discussion.md#q-014-同じ許可の画面に再び応答する条件) | BL-035 の (17): 同じ許可の画面に再び応答する条件 | 解決済み | なし | [LADR-007](261123-backlog-batch-lightadr.md#ladr-007-許可の画面への再応答の条件-bl-035-の-17) |
| [Q-015](261123-backlog-batch-discussion.md#q-015-忘却の設定を変える単位) | BL-034: 忘却の設定を変える単位 | 解決済み | なし | [LADR-008](261123-backlog-batch-lightadr.md#ladr-008-空間の種類ごとの忘却の設定-bl-034) |
| [Q-016](261123-backlog-batch-discussion.md#q-016-空間ごとの起動オプションの書式) | BL-034: 空間ごとの起動オプションの書式 | 解決済み | Q-015 | [LADR-008](261123-backlog-batch-lightadr.md#ladr-008-空間の種類ごとの忘却の設定-bl-034) |
| [Q-017](261123-backlog-batch-discussion.md#q-017-半減期を変えたときの蓄積済みの割引) | BL-034: 半減期を変えたときの蓄積済みの割引 | 解決済み | Q-015 | [LADR-008](261123-backlog-batch-lightadr.md#ladr-008-空間の種類ごとの忘却の設定-bl-034) |
| [Q-018](261123-backlog-batch-discussion.md#q-018-セッション名に使える文字の範囲) | BL-014: セッション名に使える文字の範囲 | 解決済み | Q-039 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-019](261123-backlog-batch-discussion.md#q-019-正規化と名前の同一性) | BL-014: 正規化と名前の同一性 | 解決済み | Q-018、Q-039 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-020](261123-backlog-batch-discussion.md#q-020-常駐プロセスのコマンドの名前) | BL-047: 常駐プロセスのコマンドの名前 | 解決済み | なし | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-021](261123-backlog-batch-discussion.md#q-021-手元の接続の名前と参加の名前の関係) | BL-047: 手元の接続の名前と参加の名前の関係 | 解決済み | Q-020、Q-018、Q-042 から Q-044 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-022](261123-backlog-batch-discussion.md#q-022-起動時に参加まで行うか) | BL-047: 起動時に参加まで行うか | 解決済み | Q-021、Q-027 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-023](261123-backlog-batch-discussion.md#q-023-トークンの渡し方) | BL-047: トークンの渡し方 | 解決済み | Q-033、Q-034 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-024](261123-backlog-batch-discussion.md#q-024-接続を張り直す時点) | BL-047: 接続を張り直す時点 | 解決済み | Q-025、Q-026 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-025](261123-backlog-batch-discussion.md#q-025-参加を自動でし直す条件) | BL-047: 参加を自動でし直す条件 | 解決済み | Q-022、Q-024、Q-027 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-026](261123-backlog-batch-discussion.md#q-026-切断に遭った呼び出しの再送) | BL-047: 切断に遭った呼び出しの再送 | 解決済み | Q-024、Q-025 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-027](261123-backlog-batch-discussion.md#q-027-参加時のプロフィールの取得元) | BL-047: 参加時のプロフィールの取得元 | 解決済み | Q-022、Q-025、Q-045 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-028](261123-backlog-batch-discussion.md#q-028-受信の待ち時間とシェルの時間の制限) | BL-047: 受信の待ち時間とシェルの時間の制限 | 解決済み | Q-026 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-029](261123-backlog-batch-discussion.md#q-029-再発行されたトークンの反映) | BL-047: 再発行されたトークンの反映 | 解決済み | Q-023、Q-025 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-030](261123-backlog-batch-discussion.md#q-030-常駐プロセスのアイドル終了) | BL-047: 常駐プロセスのアイドル終了 | 解決済み | Q-024 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-031](261123-backlog-batch-discussion.md#q-031-cli-の方式の対応-os) | BL-047: CLI の方式の対応 OS | 解決済み | なし | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-032](261123-backlog-batch-discussion.md#q-032-受信と同時の完了の記録の取り下げ) | BL-033 の (2) の取り下げ | 解決済み | なし | [LADR-010](261123-backlog-batch-lightadr.md#ladr-010-残課題の取り下げ) |
| [Q-033](261123-backlog-batch-discussion.md#q-033-設定ファイルの平文のトークンの取り下げ) | BL-031 の (4) の取り下げ | 解決済み | Q-023 | [LADR-010](261123-backlog-batch-lightadr.md#ladr-010-残課題の取り下げ) |
| [Q-034](261123-backlog-batch-discussion.md#q-034-履歴に残るトークンの取り下げ) | BL-031 の (5) の取り下げ | 解決済み | Q-023 | [LADR-010](261123-backlog-batch-lightadr.md#ladr-010-残課題の取り下げ) |
| [Q-035](261123-backlog-batch-discussion.md#q-035-サーバー自身の-https-の取り下げ) | BL-030 のサーバー自身の HTTPS の取り下げ | 解決済み | なし | [LADR-010](261123-backlog-batch-lightadr.md#ladr-010-残課題の取り下げ) |
| [Q-036](261123-backlog-batch-discussion.md#q-036-cli-の方式の実機確認の対象) | BL-047: 実機確認の対象 | 解決済み | Q-031 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-037](261123-backlog-batch-discussion.md#q-037-受信者ごとの送達状態の表示) | BL-011: 受信者ごとの送達状態の表示 | 解決済み | Q-008、Q-038 | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-038](261123-backlog-batch-discussion.md#q-038-送達状態での絞り込み) | BL-011: 送達状態での絞り込み | 解決済み | Q-037、Q-013 | [LADR-006](261123-backlog-batch-lightadr.md#ladr-006-メッセージの一覧と書き出しの絞り込み-bl-011) |
| [Q-039](261123-backlog-batch-discussion.md#q-039-セッション名の規則の基準) | BL-014: セッション名の規則の基準 | 解決済み | Q-018、Q-019、Q-040 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-040](261123-backlog-batch-discussion.md#q-040-ユーザー名の文字の範囲) | BL-014: ユーザー名の文字の範囲 | 解決済み | Q-039、Q-047、Q-048、Q-049 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-041](261123-backlog-batch-discussion.md#q-041-忘却の設定を管理コマンドへ伝える方法) | BL-034: 忘却の設定を管理コマンドへ伝える方法 | 解決済み | Q-016 | [LADR-008](261123-backlog-batch-lightadr.md#ladr-008-空間の種類ごとの忘却の設定-bl-034) |
| [Q-042](261123-backlog-batch-discussion.md#q-042-常駐プロセスと参加者の対応) | BL-047: 常駐プロセスと参加者の対応 | 解決済み | Q-020、Q-021、Q-043、Q-044 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-043](261123-backlog-batch-discussion.md#q-043-参加者の名前の省略) | BL-047: 参加者の名前の省略 | 解決済み | Q-042 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-044](261123-backlog-batch-discussion.md#q-044-参加者の名前を参加の名前の既定値にするか) | BL-047: 参加者の名前を参加の名前の既定値にするか | 解決済み | Q-021、Q-042、Q-045 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-045](261123-backlog-batch-discussion.md#q-045-cli-によるプロフィールの収集) | BL-047: CLI によるプロフィールの収集 | 解決済み | Q-027、Q-025、Q-044 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-046](261123-backlog-batch-discussion.md#q-046-cli-の方式の使い方の案内の置き場所) | BL-047: CLI の方式の使い方の案内の置き場所 | 解決済み | Q-020、Q-042 | [LADR-009](261123-backlog-batch-lightadr.md#ladr-009-mcp-を使わない-cli-の方式-bl-047) |
| [Q-047](261123-backlog-batch-discussion.md#q-047-ユーザー名の大文字と小文字) | BL-014: ユーザー名の大文字と小文字 | 解決済み | Q-040、Q-039 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-048](261123-backlog-batch-discussion.md#q-048-ユーザー名の長さと記号の制限) | BL-014: ユーザー名の長さと記号の制限 | 解決済み | Q-040、Q-047 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |
| [Q-049](261123-backlog-batch-discussion.md#q-049-書き出しのファイル名での-unicode-の文字) | BL-014: 書き出しのファイル名での Unicode の文字 | 解決済み | Q-039、Q-040 | [LADR-011](261123-backlog-batch-lightadr.md#ladr-011-セッション名とユーザー名の規則-bl-014) |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261123-backlog-batch-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 版 3 (Q-001 から Q-049。2026-11-23 09:41 JST の回答まで)

最終仕様 (design.md) の合意版 (第 2 ゲート): 2026-11-23 10:20 JST の版 (2026-11-23 10:30 JST に承認)

更新予定の成果物: 最終仕様の 2 章から 8 章のコードとテスト、11 章の文書

受入条件: 最終仕様の 13 章

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-23 08:10 | 計画の検討 | 利用者の依頼 (残件の対策、計画の検討に Codex を使う) を受け、Codex (gpt-6.2-sol、low) で 18 件の分類と順序を検討し、利用者に提示 |
| 2026-11-23 08:25 | 範囲の指示 | 利用者が、WSL だけで進められる項目の一括の対処と、仕様の問い合わせによる複数の残件の消化を指示。対話で範囲を確認 ([LADR-001](261123-backlog-batch-lightadr.md#ladr-001-案件の対象と進め方)、[LADR-002](261123-backlog-batch-lightadr.md#ladr-002-今回扱わない項目)) |
| 2026-11-23 09:00 | 案件の開始 | 全体管理文書、検討用 Markdown (版 1、Q-001 から Q-036)、簡易 ADR を作成。対策元の残課題一覧の対象の項目を「検討中」に更新 |
| 2026-11-23 09:12 | 第 1 ターンの回答と、再点検の指示 | 利用者が Q-001 から Q-036 に回答 (Q-003 は B、ほかは推奨の案)。あわせて、推奨が小さな変更に寄っていないかを、大局の目的に照らして再点検するよう指示 |
| 2026-11-23 09:40 | 再点検と第 2 ターンの質問 | 31 件を解決済みにして簡易 ADR に反映 (LADR-003 から LADR-010)。5 件 (Q-008、Q-018、Q-019、Q-021、Q-027) を未解決とし、Q-037 から Q-046 を追加 (版 2) |
| 2026-11-23 09:32 | 第 2 ターンの回答 | 利用者が Q-037 から Q-046 に回答 (Q-040 は B、ほかは推奨の案) |
| 2026-11-23 09:55 | 第 3 ターンの質問と、残課題の引き継ぎ | Q-037 から Q-039、Q-041 から Q-046 と、未解決だった 5 件を解決済みにして簡易 ADR に反映。Q-040 を未解決とし、Q-047 から Q-049 を追加 (版 3)。残課題の全量をこの案件の残課題一覧に引き継ぎ、前の記録を `docs/prompts/past/` へ移動 |
| 2026-11-23 09:41 | 第 3 ターンの回答 | 利用者が Q-047 から Q-049 に回答 (Q-048 は B、ほかは推奨の案) |
| 2026-11-23 10:20 | 第 1 ゲートの通過と、最終仕様の作成 | 全 49 件が解決済み。最終仕様 `261123-backlog-batch-design.md` を作成し、承認を依頼 |
| 2026-11-23 10:30 | 第 2 ゲートの通過 | 利用者が最終仕様を承認。難易度の低い作業は Grok (`grok --always-approve`、grok 4.7、High) に依頼するよう指示 (Codex のトークンを温存するため) |
| 2026-11-23 12:10 | 作業の中断 | BL-046、BL-045、BL-014、BL-035 の (17) をコミット済み。利用量の上限で中断。残りは BL-011、BL-034、BL-047、push と CI、実機の確認、文書、記録 |
| 2026-11-23 12:20 | 作業の再開 | BL-011 を Grok に依頼。作業ブランチを GitHub に push |
| 2026-11-23 12:30 | 担当の指示 | 利用者が、文章の推敲と編集を agy (gemini-4-hydrogen-high) に依頼するよう指示 ([LADR-003](261123-backlog-batch-lightadr.md#ladr-003-記録の扱いと作業の進め方)) |

## 反映と検証の記録

未着手。作業後の対応箇所、検証結果、未完了項目を記録します。
