<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-20 13:25:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53SsJ6j1XoXzWU5edaQ
稼働 OS: Linux (WSL)
上流文書: `../past/261119-connection-guidance-backlog.md` (対策元の残課題一覧)。移動または削除されている場合があります。
下流文書: `261120-small-fixes-discussion.md` (検討用 Markdown)、`261120-small-fixes-lightadr.md` (簡易 ADR)、`261120-small-fixes-design.md` (最終仕様)、`261120-small-fixes-backlog.md` (残課題一覧)、`261120-small-fixes-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# 残課題の小さな直し 全体管理

ファイル名: `261120-small-fixes-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261120-small-fixes-discussion.md`) に、決定事項は簡易 ADR (`261120-small-fixes-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: small-fixes

検討用 Markdown の版: 2

状態: 成果物の承認待ち (第 3 ゲート)

## 作業 WBS と進捗管理

この案件の作業 WBS です。前の案件 `connection-guidance` の残課題のうち、小さな問題をまとめて対処します ([LADR-001](261120-small-fixes-lightadr.md#ladr-001-今回の実施範囲))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261120-small-fixes-backlog.md`) に追記します。残課題は全量をこの案件の残課題一覧に引き継いであり、対策元の残課題一覧 (`docs/prompts/past/261119-connection-guidance-backlog.md`) は、すべての項目を「次フェーズへ転記」にして `docs/prompts/past/` へ移しました。以後の各課題の状態と経緯は、この案件の残課題一覧で更新します。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 実施範囲の確認 ([Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲)。対話セッションの選択式の質問)
    - ✅ 1.2. 前提条件の調査 (`internal/profile/profile.go` の origin とフォルダーの正規化、`internal/tools/delivery.go` の `chat_history`、`internal/tools/tools.go` の存在の確認、`internal/server/auth.go` の認証、`scripts/interactive/harness/` の該当箇所、手元の CLI のプロセス、WSL から Windows の側の Python の起動)
    - ✅ 1.3. 一時フォルダーでの事前の確認 (リポジトリは変えない): コミット `b021887` のサーバーで、WSL の中のフォルダーでの 5 つの CLI の申告 ([Q-003](261120-small-fixes-discussion.md#q-003-grok-の申告の差の扱い))
    - ✅ 1.4. 残課題の全量の引き継ぎ (`261120-small-fixes-backlog.md` の作成、対策元 `261119-connection-guidance-backlog.md` の状態とメタ情報の更新、前の案件の記録の `docs/prompts/past/` への移動)
    - ✅ 1.5. 確認項目 ([Q-002](261120-small-fixes-discussion.md#q-002-origin-の相対パスの扱い) から [Q-009](261120-small-fixes-discussion.md#q-009-作業ブランチと実装の担当)) の作成
    - ✅ 1.6. 第 1 ターンの回答 (Q-002 から Q-009) の受付と記録
    - ✅ 1.7. 回答の検証と、一時フォルダーでの追加の確認 (リポジトリは変えない): `lounge_help` の Profile の案内の確認、5 つの CLI の許可の画面、入力欄を消すキー、作業中の表示。Windows の側のセッション (`z8g4-base-polished-glade`) との疎通
    - ✅ 1.8. 追加の確認項目 ([Q-010](261120-small-fixes-discussion.md#q-010-許可の画面への応答) から [Q-012](261120-small-fixes-discussion.md#q-012-基盤の直しの実機での確認)。対話セッションの選択式の質問) の回答の受付と記録
    - ✅ 1.9. 簡易 ADR への決定の反映 (LADR-002 から LADR-008)、残課題一覧の状態の更新
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の決定事項と、根拠となった解決済み Q の回答または利用者の明示的な指示との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261120-small-fixes-design.md` の作成 (残課題一覧の対象課題の状態と経緯を更新)
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)
- ✅ 4. 承認された最終仕様に従った作業の実施
    - ✅ 4.1. 作業ブランチの作成と、作業前の記録の最初のコミット ([Q-009](261120-small-fixes-discussion.md#q-009-作業ブランチと実装の担当))
    - ✅ 4.2. プロフィールの正規化 (`internal/profile/`: BL-029 の 2 番と 3 番) とテスト
    - ✅ 4.3. `chat_history` の存在しないルーム名 (`internal/tools/delivery.go`、`internal/store/`: BL-033 の 3 番) とテスト
    - ✅ 4.4. 待ち受けの打ち切り (BL-032 の 1 番) とテスト ([Q-005](261120-small-fixes-discussion.md#q-005-無効化の後の待ち受けの打ち切り方))
    - ✅ 4.5. 存在の確認の問い合わせ (`internal/tools/tools.go`、`internal/store/`: BL-033 の 4 番) とテスト ([Q-006](261120-small-fixes-discussion.md#q-006-呼び出しごとの存在の問い合わせの扱い))
    - ✅ 4.6. 検証の基盤 (`scripts/interactive/`: BL-035 の (1)、(2)、(4)、(5)、(6)、(8)) と自己試験 ([LADR-004](261120-small-fixes-lightadr.md#ladr-004-検証の基盤で直す範囲))
    - ✅ 4.6a. 作業ブランチの GitHub への push と、実機での確認 (WSL はこのセッション、Windows は Windows の側のセッションに委譲。[LADR-007](261120-small-fixes-lightadr.md#ladr-007-確認の範囲))
    - ✅ 4.7. 文書 (`docs/ja/reference.md`、`docs/ja/interactive-verification.md` ほか) の更新
    - ✅ 4.8. 正本ドキュメントおよびコード コメントへの永続的知見の反映
    - ✅ 4.9. 機械的検証 (`go vet ./...`、`go test ./...`、基盤の自己試験。確認する OS は [Q-007](261120-small-fixes-discussion.md#q-007-確認する-os))
    - ✅ 4.10. 実装のレビュー (サブエージェント)
    - ✅ 4.11. 確認の記録 (`261120-small-fixes-validation.md`) の作成
    - ✅ 4.12. 受入条件への適合確認 (全作業の完了確認)
    - ✅ 4.13. 簡易 ADR の最終確認
    - ✅ 4.14. 残課題一覧の再構築と整理
- 🔄 5. 成果物の承認の受付 (第 3 ゲート)
    - ✅ 5.1. 成果物・検証結果・残課題の提示
    - ⏸️ 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了。承認の後に、`main` へのマージと push を行う) ← **CURRENT**

## 対象と参照資料

対象成果物: サーバー (`internal/profile/`、`internal/tools/`、`internal/store/`、`internal/server/`)、検証の基盤 (`scripts/interactive/`)、文書 (`docs/ja/`)。範囲は、最終仕様 (`261120-small-fixes-design.md`) の第 1 章のとおりです。

今回の対象範囲: BL-029 の 2 番から 4 番、BL-033 の 3 番と 4 番、BL-032 の 1 番、BL-035 の一部 ([Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲))。

書き込み可能な検討文書: この文書、検討用 Markdown (`261120-small-fixes-discussion.md`)、簡易 ADR (`261120-small-fixes-lightadr.md`)、残課題一覧 (`261120-small-fixes-backlog.md`)

参照資料と版:

- 対策元の残課題一覧: `docs/prompts/past/261119-connection-guidance-backlog.md`
- Grok の申告の差を見つけた確認の記録: `docs/prompts/past/261118-chat-completion-validation.md`
- サーバー: コミット `b021887` (`main`)
- CLI (事前の確認で使用、WSL の側): Claude Code 2.1.328、Codex 0.169.0、Copilot CLI 1.0.114、Grok 1.0.89、agy 1.5.12
- Windows の側のセッション: `z8g4-base-polished-glade` (作業フォルダー `D:\Users\tetsuo\Local\repos\agent-lounge-mcp`、Windows 11 10.0.26300、Go 1.27.4、Python 3.14.3)
- Windows の側の Python: 3.14.3

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 小さな問題をまとめて対処する (利用者の依頼と [Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲)) |
| 範囲・境界 | 確認済み | 対象の番号は [Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲)。各番号の扱いは [Q-002](261120-small-fixes-discussion.md#q-002-origin-の相対パスの扱い) から [Q-006](261120-small-fixes-discussion.md#q-006-呼び出しごとの存在の問い合わせの扱い) |
| 入力・根拠 | 確認済み | 参照資料と版を上に記載。Grok の申告は、事前の確認で再現しなかった |
| 整合性・依存関係 | 確認済み | origin の正規化済みの形 ([Q-002](261120-small-fixes-discussion.md#q-002-origin-の相対パスの扱い))、最終利用時刻の更新の時点 ([Q-006](261120-small-fixes-discussion.md#q-006-呼び出しごとの存在の問い合わせの扱い)) |
| 実現性・例外 | 確認済み | Windows の側の Python を WSL から起動できる ([Q-007](261120-small-fixes-discussion.md#q-007-確認する-os))。ビルドとテストは、モジュールのキャッシュにある Go 1.27.3 の道具 (`GOTOOLCHAIN=local`) で行える |
| 受入条件・検証 | 確認済み | [Q-007](261120-small-fixes-discussion.md#q-007-確認する-os)、[Q-008](261120-small-fixes-discussion.md#q-008-実機の-cli-での確認) |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲) | 今回の実施範囲 | 解決済み | Q-002 から Q-006 | LADR-001 |
| [Q-002](261120-small-fixes-discussion.md#q-002-origin-の相対パスの扱い) | origin の相対パスの扱い | 解決済み | Q-003 | LADR-002 |
| [Q-003](261120-small-fixes-discussion.md#q-003-grok-の申告の差の扱い) | Grok の申告の差の扱い | 解決済み | Q-002、Q-008 | LADR-003 |
| [Q-004](261120-small-fixes-discussion.md#q-004-検証の基盤で直す番号) | 検証の基盤で直す番号 | 解決済み | Q-007 | LADR-004 |
| [Q-005](261120-small-fixes-discussion.md#q-005-無効化の後の待ち受けの打ち切り方) | 無効化の後の待ち受けの打ち切り方 | 解決済み | Q-006 | LADR-005 |
| [Q-006](261120-small-fixes-discussion.md#q-006-呼び出しごとの存在の問い合わせの扱い) | 呼び出しごとの存在の問い合わせの扱い | 解決済み | Q-005 | LADR-006 |
| [Q-007](261120-small-fixes-discussion.md#q-007-確認する-os) | 確認する OS | 解決済み | Q-004、Q-008 | LADR-007 |
| [Q-008](261120-small-fixes-discussion.md#q-008-実機の-cli-での確認) | 実機の CLI での確認 | 解決済み | Q-003、Q-007 | LADR-007 |
| [Q-009](261120-small-fixes-discussion.md#q-009-作業ブランチと実装の担当) | 作業ブランチと実装の担当 | 解決済み | なし | LADR-008 |
| [Q-010](261120-small-fixes-discussion.md#q-010-許可の画面への応答) | 許可の画面への応答 | 解決済み | Q-004、Q-012 | LADR-004 |
| [Q-011](261120-small-fixes-discussion.md#q-011-完了の判定の直し方) | 完了の判定の直し方 | 解決済み | Q-004、Q-012 | LADR-004 |
| [Q-012](261120-small-fixes-discussion.md#q-012-基盤の直しの実機での確認) | 基盤の直しの実機での確認 | 解決済み | Q-007、Q-008、Q-010、Q-011 | LADR-007 |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261120-small-fixes-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 2

最終仕様 (design.md) の合意版 (第 2 ゲート): 承認済み (2026-11-20 07:15 の版)

更新予定の成果物: `internal/profile/`、`internal/tools/`、`internal/store/`、`internal/server/`、`scripts/interactive/`、`docs/ja/reference.md`、`docs/ja/interactive-verification.md`、`docs/ja/user-guide.md` (最終仕様の各章)

受入条件: 最終仕様の第 9 章 (10 項目)

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-20 06:34 より前 | 案件の開始 | 利用者の依頼 (「次の対象を選定し、作業を行う。小さな問題は、まとめて対処してしまいたい」)。対話セッションで、実施範囲を確認 ([Q-001](261120-small-fixes-discussion.md#q-001-今回の実施範囲)。4 つの組をすべて選択) |
| 2026-11-20 06:34 から 06:40 | 事前の確認 | 一時フォルダーの確認用のサーバー (コミット `b021887`) で、WSL の中のフォルダーでの 5 つの CLI の申告を確認。Grok の申告の差は再現しなかった |
| 2026-11-20 06:44 | 記録の作成 | 全体管理文書、検討用 Markdown (版 1)、簡易 ADR、残課題一覧を作成。前の案件の記録を `docs/prompts/past/` へ移動 |
| 2026-11-20 06:49 より前 | 第 1 ターンの回答の受領 | Q-002 から Q-009 に回答 (Q-004 は C、他は A。Q-003 と Q-009 に補足) |
| 2026-11-20 06:49 から 07:00 | 追加の確認 | 一時フォルダーで、5 つの CLI の許可の画面、入力欄を消すキー、作業中の表示を確認 |
| 2026-11-20 06:55 ごろ | 利用者の指示と疎通 | 「windows で agent-lounge-mcp 担当の cluaude を起動しておいた。セッションを探して問いかけ、疎通を確立してください。必要な作業をwindowsに委譲することができます。」Windows の側のセッション `z8g4-base-polished-glade` と、セッションの間のメッセージで疎通を確認 |
| 2026-11-20 07:02 ごろ | 追加の確認項目の回答の受領 | Q-010 から Q-012 に、対話セッションで回答 (すべて A)。検討用 Markdown の版 2 |
| 2026-11-20 07:15 | 第 1 ゲートの通過と最終仕様の作成 | 未解決の Q と観点の未確認がないことを確認 (合意版 2)。簡易 ADR に LADR-002 から LADR-008 を追加。`261120-small-fixes-design.md` を作成し、第 2 ゲートの承認を依頼 |
| 2026-11-20 07:23 | 利用者の指示 | 「承認の後の main へのマージに push まで含めてよい」。LADR-008 に反映。最終仕様の第 11 章の記載 (マージして push) のとおり |
| 2026-11-20 07:24 | 第 2 ゲートの通過 | 利用者が最終仕様を承認 (「承認します。トークンに余裕がないので、8:05 JST まで小さく待機し、その後着手して。」)。08:05 まで待機し、その後に作業の実施に着手 |
| 2026-11-20 08:05 | 作業の実施に着手 | 作業ブランチ `feature/small-fixes` を作成し、作業前の記録を最初のコミットにした (`f8a5e59`) |
| 2026-11-20 08:05 から 08:30 ごろ | 実装 | サーバー (`efa292d`)、検証の基盤 (`d9f9eb9`)、文書 (`91528c4`)。作業ブランチを GitHub に push し、Windows の側のセッションに確認を依頼 |
| 2026-11-20 08:30 から 09:00 ごろ | 実装のレビューと対応 | サブエージェントのレビュー (指摘 8 件。サーバーの変更に誤りの指摘はなし)。対応をコミット (`659fb8e`) |
| 2026-11-20 09:00 から 09:20 ごろ | 実機で見つけた問題への対応 | WSL の側の Copilot CLI のシナリオ 5 で、続けて出る許可の画面の見落としが判明。見分け方を直してコミット (`24ab07c`)。WSL の側の最終のコードでの実行は、11 件すべて `done` |
| 2026-11-20 13:05 より前 | Windows の側の結果の受領 | コミット `d9f9eb9`、`659fb8e`、`24ab07c` での結果を受領。最終のコードで、自己試験 30 件と Copilot CLI のシナリオ 2、3、5 が成功。作業中の表示の直接の記録は、使用量の上限で未実施 |
| 2026-11-20 13:05 ごろ | 利用者の指示 | 「続けて」 |
| 2026-11-20 13:25 | 第 3 ゲートの承認の依頼 | 文書の検証済みの範囲の更新、確認の記録の作成、簡易 ADR の最終確認 (LADR-002 と LADR-004 を更新)、残課題一覧の整理 (BL-029 を除き、BL-032、BL-033、BL-035 を更新) を終え、成果物の承認を依頼 |

## 反映と検証の記録

| 項目 | 内容 |
| --- | --- |
| コミット (作業ブランチ `feature/small-fixes`) | `f8a5e59` (作業前の記録)、`efa292d` (サーバー)、`d9f9eb9` (検証の基盤)、`91528c4` (文書)、`659fb8e` (レビューの指摘への対応)、`24ab07c` (許可の画面の見分け方)。その後に、文書の検証済みの範囲、確認の記録、記録の整理をコミットした |
| 対象成果物 | `internal/profile/`、`internal/tools/` (`chat.go`、`delivery.go`、`tools.go`、`lounge.go`)、`internal/store/` (`chat.go`、`users.go`、`delivery.go`、テスト)、`internal/server/` (`auth.go`、テスト)、`scripts/interactive/`、`docs/ja/reference.md`、`docs/ja/interactive-verification.md`、`docs/ja/user-guide.md` |
| 機械的検証 | `go vet ./...`、`go test -race ./...`、基盤の自己試験 (30 件)。WSL と Windows で成功 (Windows は `go test ./...`) |
| 確認の結果 | `261120-small-fixes-validation.md`。WSL と Windows の実機で、5 つの CLI のシナリオ 2 と 3、Copilot CLI のシナリオ 5 が `done` |
| 受入条件 (最終仕様の第 9 章) | 1 から 10 を満たすことを確認した。最終仕様との相違が 5 点ある (確認の記録の「最終仕様との相違」) |
| 未完了 | 成果物の承認 (第 3 ゲート)。承認の後に、`main` へのマージと push |
