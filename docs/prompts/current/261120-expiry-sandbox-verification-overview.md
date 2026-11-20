<!--
生成元: agree-before-implement スキル
文書種別: 全体管理文書 (overview)
最終更新日時: 2026-11-21 01:57:07 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53SsJ6j1XoXzWU5edaQ
稼働 OS: Linux (WSL)
上流文書: `../past/261120-small-fixes-backlog.md` (対策元の残課題一覧)。移動または削除されている場合があります。
下流文書: `261120-expiry-sandbox-verification-discussion.md` (検討用 Markdown)、`261120-expiry-sandbox-verification-lightadr.md` (簡易 ADR)、`261120-expiry-sandbox-verification-design.md` (最終仕様)、`261120-expiry-sandbox-verification-backlog.md` (残課題一覧)、`261120-expiry-sandbox-verification-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。この文書と、ユーザーが編集した最新の検討用 Markdown を読み直し、作業 WBS の `← **CURRENT**` の項目から再開します。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。`← **CURRENT**` の項目は処理が中止された項目である場合があり、どこまで進んでいたかは文書だけでは分かりません。対象成果物の内容を確認して実施済みの範囲を特定してから、作業を再開してください。`← **CURRENT**` がなく、作業 WBS がすべて完了 (✅) であれば、この案件の合意フローは完了しています。
-->

# 未検証の消化、トークンの有効期限、サンドボックスのマシン名 全体管理

ファイル名: `261120-expiry-sandbox-verification-overview.md`

この文書は、案件の識別、作業 WBS、対象範囲、確認状況、経緯を管理します。質問と回答の本文は検討用 Markdown (`261120-expiry-sandbox-verification-discussion.md`) に、決定事項は簡易 ADR (`261120-expiry-sandbox-verification-lightadr.md`) に記載し、この文書には重複して記載しません。

## 案件の識別

検討 ID: expiry-sandbox-verification

検討用 Markdown の版: 3

状態: 完了 (成果物の承認済み)

## 作業 WBS と進捗管理

この案件の作業 WBS です。前の案件 `small-fixes` の残課題のうち、未検証の項目の消化、トークンの有効期限、サンドボックスの中でのマシンの名前を対処します ([LADR-001](261120-expiry-sandbox-verification-lightadr.md#ladr-001-今回の実施範囲))。決定事項は、Q を解決済みにした時点、または利用者が決定を明示した時点で、簡易 ADR へ反映します。見送りの決定や範囲外の気付きは、どの段階でも、生じた時点で残課題一覧 (`261120-expiry-sandbox-verification-backlog.md`) に追記します。残課題は全量をこの案件の残課題一覧に引き継いであり、対策元の残課題一覧 (`docs/prompts/past/261120-small-fixes-backlog.md`) は、すべての項目を「次フェーズへ転記」にして `docs/prompts/past/` へ移しました。以後の各課題の状態と経緯は、この案件の残課題一覧で更新します。第 4 章の項目は、回答に応じて、最終仕様の作成の時点で見直します。

凡例: ⏸️ 未着手、🔄 実施中、🚫 ブロック中、⚠️ 要注意、✅ 完了。現在取り組んでいる項目の末尾に `← **CURRENT**` を付けます。

- ✅ 1. 検討用 Markdown のための情報収集と質問の実施
    - ✅ 1.1. 実施範囲の確認 ([Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲)。対話セッションの選択式の質問)
    - ✅ 1.2. 前提条件の調査 (`internal/store/users.go`、`internal/server/auth.go`、`cmd/agent-lounge-mcp/user.go`、`internal/config/duration.go`、`internal/tools/help.go` と `lounge.go` のマシンの名前の案内、`docs/ja/` の「検証済みの範囲」の未検証の項目、Go SDK の版、手元の道具)
    - ✅ 1.3. 一時フォルダーでの事前の確認 (リポジトリは変えない): Codex のサンドボックス (`codex sandbox`) と agy (許可の規則) の中での、マシンの名前を得るコマンドの出力 ([Q-014](261120-expiry-sandbox-verification-discussion.md#q-014-サンドボックスの中でのマシンの名前の決め方))
    - ✅ 1.4. Windows の側のセッションとの疎通と、読み取りだけの問い合わせ (WSL のネットワークのモード、ファイアウォール、CLI が使うシェル。[Q-003](261120-expiry-sandbox-verification-discussion.md#q-003-windows-の側の作業の担当) から [Q-005](261120-expiry-sandbox-verification-discussion.md#q-005-cmd-と-git-bash-での申告))
    - ✅ 1.5. 残課題の全量の引き継ぎ (`261120-expiry-sandbox-verification-backlog.md` の作成、対策元 `261120-small-fixes-backlog.md` の状態とメタ情報の更新、前の案件の記録の `docs/prompts/past/` への移動)
    - ✅ 1.6. 確認項目 ([Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目) から [Q-017](261120-expiry-sandbox-verification-discussion.md#q-017-サンドボックスの中での実機の確認)) の作成
    - ✅ 1.6a. 利用者の指示 (作業は Opus または Sonnet で進める) の記録と、確認項目 ([Q-018](261120-expiry-sandbox-verification-discussion.md#q-018-モデルごとの作業の割り当て)) の追加
    - ✅ 1.7. 第 1 ターンの回答 (Q-002 から Q-018) の受付と記録
    - ✅ 1.8. 回答の検証 (追加の確認項目はなし)
    - ✅ 1.9. 簡易 ADR への決定の反映 (LADR-002 から LADR-007)、残課題一覧の状態の更新
- ✅ 2. 前提合意と残懸念の解消 (第 1 ゲート: 最終仕様作成への着手)
    - ✅ 2.1. 全論点に対する未解決事項および質問の解消確認
    - ✅ 2.2. 方針・前提条件の合意確定 (簡易 ADR の決定事項と、根拠となった解決済み Q の回答または利用者の明示的な指示との一致を確認)
- ✅ 3. 最終決定事項 (最終仕様) の作成と承認の受付
    - ✅ 3.1. `261120-expiry-sandbox-verification-design.md` の作成 (残課題一覧の対象課題の状態と経緯を更新)
    - ✅ 3.2. 最終仕様における受入条件と整合性の確認
    - ✅ 3.3. 対話セッションにおける最終仕様の承認の受付 (第 2 ゲート: 作業着手の合意)
- ✅ 4. 承認された最終仕様に従った作業の実施
    - ✅ 4.1. 作業ブランチの作成と、作業前の記録の最初のコミット ([Q-016](261120-expiry-sandbox-verification-discussion.md#q-016-作業ブランチとマージ))
    - ✅ 4.2. トークンの有効期限 (`internal/store/schema/` の新しい版、`internal/store/users.go`、`internal/server/auth.go`、`internal/tools/`、`cmd/agent-lounge-mcp/user.go`: BL-031 の (1)) とテスト ([Q-008](261120-expiry-sandbox-verification-discussion.md#q-008-有効期限の指定の形) から [Q-012](261120-expiry-sandbox-verification-discussion.md#q-012-期限の知らせ方))
    - ✅ 4.3. 最後の利用日時 (BL-032 の 4 番) とテスト ([Q-013](261120-expiry-sandbox-verification-discussion.md#q-013-参加していない利用の最後の利用日時))
    - ✅ 4.4. サンドボックスの中でのマシンの名前 (`internal/tools/help.go`、`internal/tools/lounge.go`、`internal/profile/`: BL-027) とテスト ([Q-014](261120-expiry-sandbox-verification-discussion.md#q-014-サンドボックスの中でのマシンの名前の決め方)、[Q-015](261120-expiry-sandbox-verification-discussion.md#q-015-マシンの名前として受け付けない値))
    - ✅ 4.5. 検証の基盤の直し (`scripts/interactive/`: BL-035 の (14)、(15)、(16)) と自己試験 ([Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目))
    - ✅ 4.6. 作業ブランチの GitHub への push (記録のコミットを push。以後の変更も、まとまりごとに push)
    - ✅ 4.7. WSL の側の確認 (nginx を置いた構成、agy の確かめ直し、Claude Code のフォルダーの移動、サンドボックスの中での申告。最終仕様の第 6 章の f、i、j、k。[Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目)、[Q-007](261120-expiry-sandbox-verification-discussion.md#q-007-nginx-を置いた構成の確かめ方)、[Q-017](261120-expiry-sandbox-verification-discussion.md#q-017-サンドボックスの中での実機の確認))
    - ✅ 4.8. Windows の側の確認 ([Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目) から [Q-006](261120-expiry-sandbox-verification-discussion.md#q-006-キーボードからの-ctrlc-の確かめ方))
    - ✅ 4.9. 文書 (`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/ja/interactive-verification.md` ほか) の更新
    - ✅ 4.10. 正本ドキュメントおよびコード コメントへの永続的知見の反映
    - ✅ 4.11. 機械的検証 (`go vet ./...`、`go test ./...`、基盤の自己試験)
    - ✅ 4.12. 実装のレビュー (サブエージェント)
    - ✅ 4.13. 確認の記録 (`261120-expiry-sandbox-verification-validation.md`) の作成
    - ✅ 4.14. 受入条件への適合確認 (全作業の完了確認)
    - ✅ 4.15. 簡易 ADR の最終確認
    - ✅ 4.16. 残課題一覧の再構築と整理
- ✅ 5. 成果物の承認の受付 (第 3 ゲート)
    - ✅ 5.1. 成果物・検証結果・残課題の提示
    - ✅ 5.2. 対話セッションにおける成果物の承認の受付 (「承認」または「完了」の受領によるスキル完了)

## 対象と参照資料

対象成果物: サーバー (`internal/store/`、`internal/server/`、`internal/tools/`、`internal/profile/`、`cmd/agent-lounge-mcp/`)、検証の基盤 (`scripts/interactive/`)、文書 (`docs/ja/`)。範囲は、回答の後に、最終仕様 (`261120-expiry-sandbox-verification-design.md`) で確定します。

今回の対象範囲: BL-027、BL-028、BL-030 (nginx を置いた構成)、BL-031 の (1)、BL-032 の 4 番、BL-035 の (12)、(14)、(15)、(16)、BL-040 ([Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲))。

書き込み可能な検討文書: この文書、検討用 Markdown (`261120-expiry-sandbox-verification-discussion.md`)、簡易 ADR (`261120-expiry-sandbox-verification-lightadr.md`)、残課題一覧 (`261120-expiry-sandbox-verification-backlog.md`)

参照資料と版:

- 対策元の残課題一覧: `docs/prompts/past/261120-small-fixes-backlog.md`
- サーバー: コミット `9bb81a0` (`main`)
- Go SDK: v1.8.0 (公開されている最新は v1.9.0-pre.1)
- CLI (WSL の側): Claude Code 2.1.329、Codex 0.169.0、Copilot CLI 1.0.114、Grok 1.0.89、agy 1.5.13
- Windows の側のセッション: 作業フォルダー `D:\Users\tetsuo\Local\repos\agent-lounge-mcp`、`main` の `9bb81a0`
- Codex の週の利用量: 残り 15% (2026-11-20 08:53 の記録)

## 観点別の確認

| 観点 | 状態 | 根拠・関連 Q |
| --- | --- | --- |
| 目的・利用者 | 確認済み | 残課題を進める (利用者の依頼と [Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲)) |
| 範囲・境界 | 確認済み | 対象の組は [Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲)。確かめる項目は [Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目)、期限の機能の範囲は [Q-010](261120-expiry-sandbox-verification-discussion.md#q-010-発行済みのトークンの期限の変更)、[Q-012](261120-expiry-sandbox-verification-discussion.md#q-012-期限の知らせ方)、[Q-013](261120-expiry-sandbox-verification-discussion.md#q-013-参加していない利用の最後の利用日時) |
| 入力・根拠 | 確認済み | 参照資料と版を上に記載。サンドボックスの中のコマンドの出力は、事前の確認で取得 |
| 整合性・依存関係 | 確認済み | 期限の指定の形と既存のオプションの形 ([Q-008](261120-expiry-sandbox-verification-discussion.md#q-008-有効期限の指定の形))、既存のトークンの扱い ([Q-009](261120-expiry-sandbox-verification-discussion.md#q-009-期限を指定しない場合の扱い))、無効なトークンへの応答との関係 ([Q-011](261120-expiry-sandbox-verification-discussion.md#q-011-期限が切れたトークンへの応答)) |
| 実現性・例外 | 確認済み | 作業の担当のモデル ([Q-018](261120-expiry-sandbox-verification-discussion.md#q-018-モデルごとの作業の割り当て))、Windows の側の担当 ([Q-003](261120-expiry-sandbox-verification-discussion.md#q-003-windows-の側の作業の担当))、ファイアウォール ([Q-004](261120-expiry-sandbox-verification-discussion.md#q-004-wsl-から-windows-の側のサーバーへの接続))、シェル ([Q-005](261120-expiry-sandbox-verification-discussion.md#q-005-cmd-と-git-bash-での申告))、nginx の入手 ([Q-007](261120-expiry-sandbox-verification-discussion.md#q-007-nginx-を置いた構成の確かめ方))、マシンの名前の決め方 ([Q-014](261120-expiry-sandbox-verification-discussion.md#q-014-サンドボックスの中でのマシンの名前の決め方)、[Q-015](261120-expiry-sandbox-verification-discussion.md#q-015-マシンの名前として受け付けない値)) |
| 受入条件・検証 | 確認済み | [Q-006](261120-expiry-sandbox-verification-discussion.md#q-006-キーボードからの-ctrlc-の確かめ方)、[Q-007](261120-expiry-sandbox-verification-discussion.md#q-007-nginx-を置いた構成の確かめ方)、[Q-017](261120-expiry-sandbox-verification-discussion.md#q-017-サンドボックスの中での実機の確認) |

## 質問一覧

各 Q の本文は検討用 Markdown にあります。

| Q-ID | 論点 | 状態 | 依存・関連 | 反映先の決定 |
| --- | --- | --- | --- | --- |
| [Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲) | 今回の実施範囲 | 解決済み | Q-002 から Q-017 | LADR-001 |
| [Q-002](261120-expiry-sandbox-verification-discussion.md#q-002-確かめる未検証の項目) | 確かめる未検証の項目 | 解決済み | Q-003 から Q-007 | LADR-003 |
| [Q-003](261120-expiry-sandbox-verification-discussion.md#q-003-windows-の側の作業の担当) | Windows の側の作業の担当 | 解決済み | Q-002、Q-016 | LADR-003 |
| [Q-004](261120-expiry-sandbox-verification-discussion.md#q-004-wsl-から-windows-の側のサーバーへの接続) | WSL から Windows の側のサーバーへの接続 | 解決済み | Q-002 | LADR-003 |
| [Q-005](261120-expiry-sandbox-verification-discussion.md#q-005-cmd-と-git-bash-での申告) | cmd と Git Bash での申告 | 解決済み | Q-002 | LADR-003 |
| [Q-006](261120-expiry-sandbox-verification-discussion.md#q-006-キーボードからの-ctrlc-の確かめ方) | キーボードからの Ctrl+C の確かめ方 | 解決済み | Q-002 | LADR-003 |
| [Q-007](261120-expiry-sandbox-verification-discussion.md#q-007-nginx-を置いた構成の確かめ方) | nginx を置いた構成の確かめ方 | 解決済み | Q-002 | LADR-003 |
| [Q-008](261120-expiry-sandbox-verification-discussion.md#q-008-有効期限の指定の形) | 有効期限の指定の形 | 解決済み | Q-009、Q-010 | LADR-004 |
| [Q-009](261120-expiry-sandbox-verification-discussion.md#q-009-期限を指定しない場合の扱い) | 期限を指定しない場合の扱い | 解決済み | Q-008 | LADR-004 |
| [Q-010](261120-expiry-sandbox-verification-discussion.md#q-010-発行済みのトークンの期限の変更) | 発行済みのトークンの期限の変更 | 解決済み | Q-008 | LADR-004 |
| [Q-011](261120-expiry-sandbox-verification-discussion.md#q-011-期限が切れたトークンへの応答) | 期限が切れたトークンへの応答 | 解決済み | Q-010 | LADR-004 |
| [Q-012](261120-expiry-sandbox-verification-discussion.md#q-012-期限の知らせ方) | 期限の知らせ方 | 解決済み | Q-011 | LADR-004 |
| [Q-013](261120-expiry-sandbox-verification-discussion.md#q-013-参加していない利用の最後の利用日時) | 参加していない利用の最後の利用日時 | 解決済み | Q-008 | LADR-005 |
| [Q-014](261120-expiry-sandbox-verification-discussion.md#q-014-サンドボックスの中でのマシンの名前の決め方) | サンドボックスの中でのマシンの名前の決め方 | 解決済み | Q-015、Q-017 | LADR-006 |
| [Q-015](261120-expiry-sandbox-verification-discussion.md#q-015-マシンの名前として受け付けない値) | マシンの名前として受け付けない値 | 解決済み | Q-014 | LADR-006 |
| [Q-016](261120-expiry-sandbox-verification-discussion.md#q-016-作業ブランチとマージ) | 作業ブランチとマージ | 解決済み | Q-003、Q-018 | LADR-007 |
| [Q-017](261120-expiry-sandbox-verification-discussion.md#q-017-サンドボックスの中での実機の確認) | サンドボックスの中での実機の確認 | 解決済み | Q-014、Q-015 | LADR-006 |
| [Q-018](261120-expiry-sandbox-verification-discussion.md#q-018-モデルごとの作業の割り当て) | モデルごとの作業の割り当て | 解決済み | Q-003、Q-016 | LADR-002 |

## 対象成果物と合意版

最終決定事項 (最終仕様) の設計書: `261120-expiry-sandbox-verification-design.md`

検討用 Markdown の合意版 (第 1 ゲート): 3

最終仕様 (design.md) の合意版 (第 2 ゲート): 承認済み (2026-11-20 16:58 の版)

更新予定の成果物: `internal/store/` (スキーマの版 6 を含む)、`internal/server/`、`internal/tools/`、`internal/profile/`、`cmd/agent-lounge-mcp/`、`scripts/interactive/`、`docs/ja/reference.md`、`docs/ja/user-guide.md`、`docs/ja/interactive-verification.md` (最終仕様の各章)

受入条件: 最終仕様の第 10 章 (9 項目)

## 経緯

| 日時 (JST) | 出来事 | 内容・参照先 |
| --- | --- | --- |
| 2026-11-20 14:20 ごろ | 案件の開始 | 利用者の依頼 (「残件を推進してください」)。対話セッションで、実施範囲を確認 ([Q-001](261120-expiry-sandbox-verification-discussion.md#q-001-今回の実施範囲)。A、B、C を選択) |
| 2026-11-20 14:26 から 14:45 | 事前の調査と確認 | コードと文書の調査。一時フォルダーで、Codex のサンドボックスと agy の中での、マシンの名前を得るコマンドの出力を確認 |
| 2026-11-20 14:45 から 15:00 | Windows の側のセッションとの疎通 | 読み取りだけの問い合わせに、回答を受領 (ネットワークのモード、ファイアウォール、シェル) |
| 2026-11-20 15:15 | 記録の作成 | 全体管理文書、検討用 Markdown (版 1)、簡易 ADR、残課題一覧を作成。前の案件の記録を `docs/prompts/past/` へ移動 |
| 2026-11-20 15:45 | 利用者の指示 | 「作業はOpus(高難易度)またはSonnet(低難易度)で進めたい」。簡易 ADR に LADR-002 を追加。割り当ての確認項目 ([Q-018](261120-expiry-sandbox-verification-discussion.md#q-018-モデルごとの作業の割り当て)) を追加 (検討用 Markdown の版 2) |
| 2026-11-20 16:41 | 第 1 ターンの回答の受領 | Q-002 から Q-018 に回答 (すべて A。Q-002 に補足: `windows-arm64` と macOS は、ビルドはするが保証しないこととして閉じる。物理的に別のマシンと、WSL ではない Linux は、残課題に残す) |
| 2026-11-20 16:44 | 第 1 ゲートの通過 | 未解決の Q と観点の未確認がないことを確認 (合意版 3)。簡易 ADR に LADR-003 から LADR-007 を追加し、LADR-002 を更新。最終仕様の作成に着手 |
| 2026-11-20 16:46 | 最終仕様の作成 | `261120-expiry-sandbox-verification-design.md` を作成し、第 2 ゲートの承認を依頼 |
| 2026-11-20 16:58 | 第 2 ゲートの通過 | 利用者の承認 (「推進」)。作業に着手 |
| 2026-11-20 17:00 ごろ | 作業の開始 | 作業ブランチ `feature/expiry-sandbox-verification` を作成し、記録をコミット (`2c87369`) して push。Opus と Sonnet のサブエージェントに、第 2 章から第 5 章と、WSL の側の確認 (f、i、j) を依頼。Windows の側のセッションに、a、b、c、d、e、g を依頼。対話セッションのモデルは、利用者の切り替えで Opus 5.5 になった |
| 2026-11-20 17:10 ごろ から 18:00 | 作業の中断 | 利用量の上限 (5 時間の枠) で、4 つのサブエージェントが止まった。18:00 の枠の更新の後、止まった所から再開 |
| 2026-11-20 18:14 | 実装の取り込み | マシンの名前 (`966573f`、Sonnet)、基盤の (14) と (15) (`349f471`、Sonnet)、有効期限と最後の利用日時 (`b5951e8`、Opus) を、差分を確かめて作業ブランチに取り込み、push。`go vet`、`go test -race`、基盤の自己試験 (41 件) が通った。基盤の (16) を Opus に、確認の k を Sonnet に依頼。Windows の側のサーバー (項目 c) には、WSL から接続できていない (ファイアウォールの許可を利用者に依頼中) |
| 2026-11-20 夜 | 確認の結果の受領 | Windows の側から a、b、d、e、g、h の結果、WSL の側から f、i、j、k の結果を受領 (確認の記録に記載)。d と g の発見を、残課題 BL-041、BL-042 として追記 |
| 2026-11-21 00:26 | 作業中の判断の受領 | 基盤の (16) が、仕様の条件 (Copilot CLI のシナリオ 5 の 5 回の完了) を満たさなかった (別の許可の画面と、原因不明の時間切れ) ため、扱いを利用者に確認。(16) は取り込み、別の画面にも応答させる。c は残課題に回す。BL-041 はこの案件で直す (LADR-003)。(16) の直しを `d29ef20`、BL-041 の直しを `0d6b8a7` として取り込み |
| 2026-11-21 00:30 ごろ | レビューと直し | Opus のレビュー (重大な問題なし)。指摘 3 点を直した (`6732277`)。文書の更新を点検してコミット (`97d7933`) |
| 2026-11-21 01:15 | 第 3 ゲートの準備 | 最終のコード (`3810702`) で Copilot CLI のシナリオ 5 を 5 回実行し、5 回とも完了。確認の記録を作成し、残課題一覧を整理 (14 件)。成果物の承認を依頼 |
| 2026-11-21 01:57 | 第 3 ゲートの通過 | 利用者の承認 (「承認」)。`main` へマージして push し、作業ブランチを消す (LADR-007)。マージの前の手順の誤りで、GitHub の作業ブランチを先に消した (手元のブランチに全コミットが残っていたため、影響なし) |

## 反映と検証の記録

| 対応箇所 | コミット | 検証 |
| --- | --- | --- |
| マシンの名前 (第 4 章) | `966573f`、`6732277` | 自動テスト、確認 k |
| 基盤の (14)、(15) | `349f471` | 自己試験 (WSL、Windows) |
| トークンの有効期限、最後の利用日時 (第 2 章、第 3 章) | `b5951e8`、`6732277` | 自動テスト (`go test -race`、Windows の `go test`) |
| 基盤の (16)、パスへのアクセスの画面、完了の合言葉 | `d29ef20`、`18d0242`、`3810702` | 自己試験、Copilot CLI のシナリオ 5 (5 回中 5 回完了) |
| Git Bash の `pwd -W` (BL-041) | `0d6b8a7` | 自動テスト、Windows の実機 |
| 文書 | `97d7933` と、この後の記録のコミット | 差分の点検 |

確認の結果の詳細は、確認の記録 (`261120-expiry-sandbox-verification-validation.md`) にあります。未完了の項目は、c (残課題 BL-028) です。
