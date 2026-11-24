<!--
生成元: agree-before-implement スキル
文書種別: 残課題一覧 (backlog)
最終更新日時: 2026-11-25 07:04:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261124-shared-space-rooms-overview.md` (全体管理文書)、`261124-shared-space-rooms-design.md` (最終仕様)、`261124-shared-space-rooms-discussion.md` (検討用 Markdown)、`261124-shared-space-rooms-lightadr.md` (簡易 ADR)。対策元: `261123-backlog-batch-backlog.md`。いずれも削除されている場合があります。
下流文書: この残課題一覧を対策する継続作業が開始された場合は、その全体管理文書・検討用 Markdown・最終仕様・残課題一覧のパスを追記します。BL-049: `../current/261124-help-text-result-overview.md`、`../current/261124-help-text-result-discussion.md`、`../current/261124-help-text-result-design.md`、`../current/261124-help-text-result-backlog.md`。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。「整理状態」が「随時追記中」であれば、案件の作業中です。気付いた事項を追記し、第 3 ゲートの前に文書を再構築して整理します。上流文書が存在する場合は、各課題の背景と見送りの判断を上流文書で確認します。存在しない場合は、この文書を単体の残課題一覧として扱います。このファイルの残課題を参照して新たな成果物の作成または更新を始めるときは、元案件の再開とは別の作業として扱います。その開始には agree-before-implement を原則として使い、新しい全体管理文書と検討用 Markdown から開始してください。継続作業によってこのファイルの残課題を対策する場合は、このファイルや各項目の記載を削除することなく、継続作業の開始時 (全体管理文書と検討用 Markdown の作成時) から検討・設計・実装・整理の各段階において、このファイルのメタ情報、各項目の状態、および経緯と対処を随時更新することを原則とします。また、フェーズ分けした対策で対処できていない項目がある場合は、当該継続作業の backlog にも必ず転記し、最新作業の backlog を見れば残件が把握できるようにしてください。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。
-->

# 残課題一覧 (バックログ)

ファイル名: `261124-shared-space-rooms-backlog.md`

## 案件の識別

案件名: `shared-space-rooms` (`shared` の空間とルームの作成・参加)  
作成日: `2026-11-24`  
整理状態: 整理済み

この文書は、見送る決定がなされた問題点や、作業中に潜在または別問題として判明したが今回は対策しなかった残課題を記録するものです。各項目は、チケット管理システムへの起票にそのまま利用できる詳細度で記載します。

残課題は、案件ごとに全量を引き継ぎます (利用者の指示)。この文書は、対策元の残課題一覧 (`261123-backlog-batch-backlog.md`) のうち、この案件で対応しなかった 11 件を、同じ課題 ID で転記して作成しました。この案件で対応済みになった BL-004、BL-009、BL-033 は、転記していません。BL-031 は、(6) を仕様として容認し、(2) と (3) を転記しました。BL-048 と BL-049 は、この案件で記録した項目です。

優先度は、Claude Code の提案です。利用者の評価は受けていません。

## 残課題一覧

| 課題 ID | 題名 | 区分 | 優先度 / 影響度 | 状態 |
| --- | --- | --- | --- | --- |
| BL-013 | 英語の文書の作成 (`docs/en/`、`README.md`、GitHub の概要) | 過去 backlog からの転記 | 中 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-014 | 日本語の名前と書き出しのファイル名の、Windows での実機確認 | 過去 backlog からの転記 | 低 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-028 | Windows の側で起動したサーバーへの、WSL からの接続 | 過去 backlog からの転記 | 低 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-030 | 未検証の構成 (物理的に別のマシンからの接続) | 過去 backlog からの転記 | 中 (物理的に別のマシンからの接続とリバース プロキシ経由の通信が未検証) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-031 | トークンの運用の機能 (権限の区分、接続元の制限) | 過去 backlog からの転記 | 中 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-035 | 検証の基盤の残り (他の CLI の許可の画面、画面の再現の細部、WSL ではない Linux ほか) | 過去 backlog からの転記 | 低 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-036 | MCP のプロトコルの版 2026-07-28 への対応 | 過去 backlog からの転記 | 中 (CLI が古い版での接続し直しをやめると、接続できなくなる) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-040 | agy が、閉じられた MCP の接続の後に、ツールを呼べなくなる | 過去 backlog からの転記 | 中 (1 時間使わなかった agy は、起動し直すまでツールを呼べない場合がある) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-042 | Windows の Codex で、出力の途中に作業中の表示が画面にない | 過去 backlog からの転記 | 低 (検証の基盤が、Codex の作業中を見落とす) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-044 | ブランチのスコープの Windows 側での実機確認 | 過去 backlog からの転記 | 中 (Windows の大文字・小文字の動作と PowerShell での申告が未検証) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-047 | MCP を使わない CLI の方式の、Windows での実機確認および Codex サンドボックスの確認 | 過去 backlog からの転記 | 中 (Windows の実機動作が未検証) | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-048 | `shared` とルームの、Claude Code 以外の 4 つの CLI での実機確認 | 検討時見送り | 低 | 次フェーズへ転記 (`../current/261124-help-text-result-backlog.md` の同じ課題 ID) |
| BL-049 | Claude Code 2.1.331 で `lounge_help` の案内が空として扱われる | 作業中摘出 | 高 (Claude Code のエージェントが使い方の案内を読めない) | 対応済み (継続案件 `help-text-result`) |


## BL-013 英語の文書の作成 (`docs/en/`、`README.md`、GitHub の概要)

### 概要と問題点

- **現状の事象**: 利用者向けの文書は `docs/ja/` だけで、英訳を置く `docs/en/` がありません。`README.md` と `docs/README.md` は日本語だけです。GitHub の概要 (About 欄の説明文) も日本語です。文書の構成と `AGENTS.md` の規範は、`docs/ja/` を正本、`docs/en/` を英訳とする前提で作ってあります。
- **本来あるべき姿**: `docs/en/` に `docs/ja/` の英訳があり、`README.md` と `docs/README.md` は英語、日本語の順に併記され、GitHub の概要は英語です。

### 経緯と見送り理由

- **摘出契機**: 前提検討時に見送り (Q-013 の補足、Q-019 の補足、Q-020)。
- **今回の見送り理由**: 英語版は、ある程度検証が進んでから、まとめて作成するという利用者の判断のためです。
- **継続作業での経緯と対処**: `261117-chat-connectivity-backlog.md` の同じ課題 ID から転記しました。この案件 (`chat-autonomy`) では対策しません (検討用 Markdown の Q-001)。`261117-chat-autonomy-backlog.md` の同じ課題 ID から、案件 `repository-space` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 この案件 (`repository-space`) では対策しません (検討用 Markdown の Q-001)。 `261117-repository-space-backlog.md` の同じ課題 ID から、案件 `multi-user-access` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `docs/en/` (新規)、`README.md`、`docs/README.md`、`AGENTS.md` の「Languages」の節、GitHub のリポジトリの設定。
- **放置した場合のリスク**: 日本語を読まない利用者が、使い方を把握できません。英訳の作成が遅れるほど、訳す量が増えます。

### 推奨される対応策

- **恒久対応案**: `docs/ja/reference.md` と `docs/ja/user-guide.md` を訳して `docs/en/` に置き、`README.md` と `docs/README.md` を併記にします。あわせて、`AGENTS.md` の「`docs/en/` はまだ存在しない」という記述を更新します。GitHub の概要は、文面の承認を受けた後、Claude Code が `gh repo edit --description` で設定します (Q-017)。
- **参考情報**: Q-013、Q-017、Q-019、Q-020。

## BL-014 日本語の名前と書き出しのファイル名の、Windows での実機確認

### 概要と問題点

- **現状の事象**: セッション名とユーザー名の規則 (RFC 8265 による検査と正規化) および書き出し (`export`) のファイル名の規則 (`export.SafeName` の Unicode 文字保持と 120 バイト上限) は実装済みで、CI の 3 つの OS での自動テストと、WSL 上の 5 つの CLI・管理コマンドでの実機確認を終えています (`261123-backlog-batch-design.md` の 4 章、`261123-backlog-batch-validation.md` の 1・2.1・2.5 章)。しかし、(1) Windows の側の CLI での日本語の名前の受け渡しの実機確認と、(2) 絞り込んだ書き出し (`export`) で生成される日本語のファイル名の、Windows での実機確認が未実施です (`261123-backlog-batch-validation.md` の 3 章)。
- **本来あるべき姿**: Windows の側の CLI でも日本語のセッション名やユーザー名を正しく受け渡して参加・送信・管理できること、および絞り込んだ `export` の日本語のファイル名が Windows の実機でも問題なく作成・参照できることが確認されています。

### 経緯と見送り理由

- **摘出契機**: 案件 `backlog-batch` の方針決定および検証 (`261123-backlog-batch-lightadr.md` の LADR-001、`261123-backlog-batch-validation.md` の 3 章。2026-11-23)。元の課題は、初回案件の最終仕様の承認時に見送りました。
- **今回の見送り理由**: WSL の環境からは、Windows の側の CLI や、Windows の実機での動作確認を実施できないため、規則の実装、CI の 3 つの OS の自動テスト、および WSL での実機確認までを完了して残しました。
- **継続作業での経緯と対処**: `261117-chat-connectivity-backlog.md` の同じ課題 ID から転記しました。この案件 (`chat-autonomy`) では対策しません (検討用 Markdown の Q-001)。`261117-chat-autonomy-backlog.md` の同じ課題 ID から、案件 `repository-space` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 この案件 (`repository-space`) では対策しません (検討用 Markdown の Q-001)。 `261117-repository-space-backlog.md` の同じ課題 ID から、案件 `multi-user-access` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、対策の検討を始めました (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-018、Q-019)。Windows の側の実機の確認は、その案件の範囲外です。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 4 章) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、セッション名とユーザー名の規則および書き出しのファイル名の規則を実装し、自動テスト (CI の 3 つの OS を含む) と WSL の 5 つの CLI・管理コマンドでの実機確認を完了しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md`)。Windows の側の CLI での日本語の名前の受け渡しの実機確認と、絞り込んだ書き出しの日本語のファイル名の Windows での実機確認が残ります。 macOS の実機の確認は、機材がないため行わず、机上の確認と自動テストだけにすると決めたため (利用者の恒久の決定。2026-11-23。簡易 ADR の LADR-012)、この項目から除きました。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `docs/ja/user-guide.md` の「検証済みの範囲」、`internal/tools/lounge.go`、`internal/export/export.go`、`cmd/agent-lounge-mcp/`。
- **放置した場合のリスク**: Windows の側の CLI や端末の文字コード・引数の扱い、または Windows の実機のファイル システム上での日本語ファイル名の扱いに不都合があっても、気付けません。

### 推奨される対応策

- **恒久対応案**: Windows の実機で CLI から日本語のセッション名・ユーザー名を使った参加・送信を実行して受け渡しを確かめ、あわせて Windows の実機で日本語のユーザーや空間を指定した絞り込み `export` を実行してファイル名と内容を確認し、結果を `docs/ja/user-guide.md` の「検証済みの範囲」に反映します。
- **参考情報**: `261123-backlog-batch-design.md` の 4 章、`261123-backlog-batch-validation.md` の 2.1・2.5・3 章。

## BL-028 Windows の側で起動したサーバーへの、WSL からの接続

### 概要と問題点

- **現状の事象**: Windows の側で起動したサーバーに、WSL の側のエージェントから接続する構成 (元の (3)) を、確かめていません。Windows のファイアウォールが、新しい実行ファイルの受信を、許可の画面で利用者が許可するまで止めるため、利用者が画面を見られる時に確かめる必要があります。
- **本来あるべき姿**: 利用ガイドの「検証済みの範囲」で、この構成が検証済みになります。

### 経緯と見送り理由

- **摘出契機**: Windows の側の Claude Code のセッションとの確認 (利用者の指示で、最終仕様の到達点 5 を検証に変更) で判明しました。
- **今回の見送り理由**: スコープ外。Ctrl+C のキー操作と対話の画面は、非対話の実行では確認できません。Windows の側には Go がなく、サーバーは WSL の側で起動しました。
- **継続作業での経緯と対処**: `261117-repository-space-backlog.md` の同じ課題 ID から、案件 `multi-user-access` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 この案件では、(6) を追加しました。今回の変更は OS によって変わる部分を含まないため、Windows の側での確認を省きました (検討用 Markdown `261118-chat-completion-discussion.md` の Q-015)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 この案件でも、Windows の側での確認を省きます (Q-024)。メモリのツールと管理コマンドを、未検証の項目に加えます。 (7) を追加しました。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 この案件で、Windows で動くエージェントでの `chat_ack`、`chat_status`、`chat_history` とメモリのツールは、OS によって変わる部分を含まないため、確認の必要がない (`-`) と決め、利用ガイドの検証済みの範囲を改めました (2026-11-20、検討用 Markdown `261119-connection-guidance-discussion.md` の Q-014、Q-015)。(6) と (7) から、これらを除きました。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(1)、(2) (疑似端末へのキー入力で確かめ、物理的なキーボードは対象外)、(4)、(6)、(7) を確かめ、検証済みにしました。(5) の `windows-arm64` は、ビルドだけを行い動作を保証しないことにして、除きました (利用者の判断)。(3) は、利用者が画面を見られない時間だったため、確かめずに残しました (利用者の判断。2026-11-21)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `docs/ja/user-guide.md` の「検証済みの範囲」、`internal/tools/help.go`。
- **放置した場合のリスク**: Windows の側でサーバーを動かす使い方で、WSL の側から接続できない場合に、気付けません。

### 推奨される対応策

- **恒久対応案**: 利用者が Windows の画面を見られる時に、Windows の側で `serve --allow-remote --listen 0.0.0.0:<ポート>` を起動し、ファイアウォールの許可の画面で「プライベート」を許可してから、WSL の側から Windows の側のアドレス (WSL の既定のゲートウェイ) へ接続して確かめます。
- **参考情報**: 確認の記録の「到達点 5」と「Windows 向けの実行ファイル」。

## BL-030 未検証の構成 (物理的に別のマシンからの接続)

### 概要と問題点

- **現状の事象**: 他のマシンからの接続は、同じマシンの Windows から WSL のアドレスへの接続、および WSL の podman で前段に nginx を置いた HTTPS 構成で確認しました。物理的に別のマシンからの接続は、未検証です。
- **本来あるべき姿**: 物理的に別のマシンから接続する構成 (リバース プロキシによる HTTPS 構成を含む) が、実機で検証済みになります。

### 経緯と見送り理由

- **摘出契機**: 前提検討時に見送り (Q-006、Q-010)。
- **今回の見送り理由**: スコープ外。物理的に別のマシンとネットワーク環境の用意には利用者の準備が要り、WSL の環境だけでは進められないためです (`261123-backlog-batch-lightadr.md` の LADR-002)。なお、サーバー自身による HTTPS は案件 `backlog-batch` で取り下げました (LADR-010)。
- **継続作業での経緯と対処**: `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、nginx を置いた構成を、WSL の podman で確かめ、利用ガイドの例を検証済みにしました (2026-11-21)。物理的に別のマシンと、サーバー自身の HTTPS は、残しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、サーバー自身の HTTPS を取り下げるかを検討しています (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-035)。物理的に別のマシンからの接続の検証は、その案件では扱いません。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 サーバー自身の HTTPS は、取り下げました (2026-11-23。この案件の簡易 ADR の LADR-010)。物理的に別のマシンからの接続の検証だけを残します。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `docs/ja/user-guide.md` の「検証済みの範囲」と「リバース プロキシで HTTPS にする」、`internal/server/` (`Host` の検査)、`internal/config/`。
- **放置した場合のリスク**: 物理的に別のマシンとの間やリバース プロキシ経由で、イベントの受信や返信の待ち受けが、プロキシの設定 (バッファリング、時間制限) などで切れる場合に、気付けません。

### 推奨される対応策

- **暫定対応案 (ワークアラウンド)**: 信頼できるネットワーク (社内のネットワーク、VPN) の中だけで使い、HTTPS が必要な場合は前段にリバース プロキシを置きます。
- **恒久対応案**: 物理的に別のマシンから `--allow-remote` のサーバー (および前段にリバース プロキシを置いた HTTPS 構成) へ接続し、参加とメッセージのやりとり、待ち受けが動作することを実機で確認して、`docs/ja/user-guide.md` の「検証済みの範囲」に反映します。
- **参考情報**: 確認の記録、`docs/ja/reference.md` の「認証」と「制限事項」、`docs/ja/user-guide.md` の「リバース プロキシで HTTPS にする」。

## BL-031 トークンの運用の機能 (権限の区分、接続元の制限)

### 概要と問題点

- **現状の事象**: トークンの運用について、次の点が残っています。(2) 権限の区分がなく、どのユーザーも同じことができます (管理は、データベース ファイルを開ける人が管理コマンドで行います)。(3) 接続元のアドレスで制限できません。なお、(1) 有効期限は案件 `expiry-sandbox-verification` で対応済みであり、(4) 設定ファイルの平文のトークンと (5) 履歴に残るトークンは案件 `backlog-batch` で取り下げました。(6) 申告によるリポジトリの単位の選択は、案件 `shared-space-rooms` で仕様として容認しました (このサーバーの情報は共有を目的とし、機密情報を含めない前提)。
- **本来あるべき姿**: 漏れたトークンの影響を、権限の区分と接続元の制限で限定できます。

### 経緯と見送り理由

- **摘出契機**: 設計の時点で、範囲外としました (最終仕様の「今回実装しないこと」)。
- **今回の見送り理由**: スコープ外。企画提案書は、強い安全対策を対象外としています。(2)、(3) は接続の側の項目で、案件 `shared-space-rooms` の対象 (共有の範囲) に含めませんでした。
- **継続作業での経緯と対処**: `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 案件 `interactive-verification` で、5 番の実態を、2 つの OS で調べました (2026-11-19。トークンの形 `alm_` と 43 文字で、各 CLI の設定、履歴、ログのフォルダーを走査)。本物のトークンは、どちらの OS でも、案件 `multi-user-access` の確認のものだけでした。残っていた場所は、確認用のフォルダーで動かした Grok のセッションの履歴 (設定ファイルの `Authorization` の値を、Grok が読んで履歴に残したもの) と、確認を行った Claude Code のセッションの会話の記録 (確認のコマンドに書いたもの) です。トークンは発行し直しまたはデータベースの削除で、無効になっています。Grok の履歴は削除しました。Claude Code の会話の記録は、セッションの再開に使うため、残しています。各エージェントのセッションのデータが残ることは、セキュリティの問題ではなく、各エージェントの動作であるため、今後は原則として対処しません (利用者の判断。2026-11-19)。5 番は、この判断に従い、追加の対策を行いません。 協調テストで使うデータベースはテスト用で、テストの後に削除するため、テストのトークンが会話や確認の記録に残っても問題ない、という判断も受けました (利用者。2026-11-19)。この案件のシナリオ 6 のトークンは、どこにも残っていませんでした。CLI の設定のバックアップ (`.claude/backups/`) は、入れ替わって、トークンを含まなくなっていました。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(1) を対応しました (`--expires-in`、`user expiry`、期限切れの 401。2026-11-21)。(2) から (6) は残しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、(4) と (5) を取り下げるかを検討しています (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-033、Q-034)。(2)、(3)、(6) は、保留を続けます (その案件の簡易 ADR の LADR-002)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 (4) と (5) は、取り下げました (2026-11-23。この案件の簡易 ADR の LADR-010)。利用ガイドの注意は残します。(2)、(3)、(6) は、保留を続けます (LADR-002)。 なお、BL-047 の対応にあわせて、CLI の方式ではトークンを設定ファイルに保存しないことを利用ガイドに書き足しました (2026-11-23。最終仕様 `261123-backlog-batch-design.md` の 10 章)。 継続案件 `shared-space-rooms` (`261124-shared-space-rooms-overview.md`) で、(6) を仕様として容認し、この項目から除くと決めました (2026-11-24。検討用 Markdown `261124-shared-space-rooms-discussion.md` の Q-002、簡易 ADR `261124-shared-space-rooms-lightadr.md` の LADR-002。利用者の判断: この MCP を通して共有される情報には機密情報はなく、接続と情報の共有は別に考える)。(2)、(3) は、保留を続けます。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `internal/auth/`、`internal/store/users.go`、`internal/store/scope.go`、管理コマンド、`docs/ja/user-guide.md`、`docs/ja/reference.md`。
- **放置した場合のリスク**: (2) ユーザーごとに許可する操作を分けられません。(3) 想定外の接続元アドレスからのトークン利用を防げません。

### 推奨される対応策

- **暫定対応案 (ワークアラウンド)**: トークンが漏れた場合は `user rotate` を使い、信頼できる利用者に限定して運用します。このサーバーで機密情報を扱わないでください。
- **恒久対応案**: 必要に応じて、(2) 権限の区分、(3) 接続元のアドレスによる制限を検討します。
- **参考情報**: `docs/ja/user-guide.md` の「複数の利用者で使う」の「注意」、`261123-backlog-batch-lightadr.md` の LADR-002・LADR-010。

## BL-035 検証の基盤の残り (他の CLI の許可の画面、画面の再現の細部、WSL ではない Linux ほか)

### 概要と問題点

- **現状の事象**: 検証の基盤 (`scripts/interactive/`) に、次の点が残っています。(1) 許可の画面への応答は、Copilot CLI のシェルのコマンドとパスへのアクセスだけを定義しています。他の 4 つの CLI は、確認した環境の設定では許可の画面を出さないため、定義がありません。許可を求める設定の環境では、応答が時間切れになります。(3) 端末の画面の再現に、細かな相違があります (行の端での折り返しの遅延、代替画面でのカーソルの保存、幅の広い文字の半分の上書き)。(7) シナリオ 6 のトークンの残りの確認は、決めたフォルダーだけを調べます。(9) WSL ではない Linux で確かめていません。(10) Claude Code は、Claude Code のセッションの作業用の一時フォルダーの下の `CLAUDE.md` を読みません。(13) Windows の Copilot CLI のシナリオ 5 で、指示ファイルを確かめる質問の 2 回とも、画面の記録に会話の部分がなく、`unverified` になったことが 1 回ありました (原因は分かっていません)。このほか、実機の CLI での、時間切れでのプロセスのツリーの終了は、起きた場面がなく、確かめていません。 (17) の対策 (同じ表示への再応答を、画面の外の部分が変わった場合に限る) は、自動テストの模擬の CLI と、Copilot CLI 1.0.114 の対話の画面 (シナリオ 2 と 5) で確かめましたが、同じコマンドの許可が続く場面が実機で起きたかは、記録から判別できず、実機では未検証です。
- **本来あるべき姿**: CLI の版や環境が変わっても、基盤の直しが少なく済み、結果を誤って記録せず、すべての対象環境で確認できています。

### 経緯と見送り理由

- **摘出契機**: (1) は、前提検討時に見送り (検討用 Markdown `261118-interactive-verification-discussion.md` の Q-006、Q-015)。(3) と (7) は、実装のレビューで判明しました。(9) と (10) は、実装中に判明しました。(13) は、案件 `small-fixes` の確認とレビューで判明しました (2026-11-20)。
- **今回の見送り理由**: 確認の結果に影響しなかったためです。(1) の他の CLI の画面は、画面を出す設定にしないと、文字列を調べられません。案件 `backlog-batch` では (17) だけを対策し、残る番号は扱いませんでした (`261123-backlog-batch-design.md` の 1.2)。
- **継続作業での経緯と対処**: 案件 `interactive-verification` で追加しました (2026-11-19)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 この案件で、一部の番号を検討します (Q-001、Q-004)。 (1)、(2)、(4)、(5)、(6)、(8) を直し、(11) を記録して除きます。(1) は Copilot CLI の画面だけを定義します (Q-004、Q-010、Q-011、最終仕様の第 6 章)。(3)、(7)、(9)、(10) と、他の 4 つの CLI の許可の画面は、この項目に残します。 (2)、(4)、(5)、(6)、(8) を対応し、(11) は残ったプロセスがないことを確かめて除きました (2026-11-20)。(1) は、Copilot CLI のシェルのコマンドの許可の画面に応答する形にし、一括の許可 (`--allow-tool shell`) をやめました。WSL と Windows の実機で確かめました (確認の記録 `261120-small-fixes-validation.md`)。(12) から (16) を追加しました。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(14)、(15)、(16) を直し、(12) を確かめました (2026-11-21)。(12) のうち、Windows の Codex の作業中の表示は、BL-042 に分けました。(16) の確認の中で、Copilot CLI のパスへのアクセスの許可の画面に応答する定義と、完了の合言葉の綴りの誤りへの対処を足しました。(9) の macOS は、動作を保証しないことにして除きました (利用者の判断)。(17) を追加しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、(17) の対策の検討を始めました (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-014)。その他の番号は、その案件では扱いません。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 8 章 ((17))) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、(17) の許可の画面への再応答の条件を実装し、自動テスト (Python の 46 件、CI の 3 つの OS を含む) と Copilot CLI の対話の画面で確認して対応を完了しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md`)。残るのは、(1)、(3)、(7)、(9)、(10)、(13) と、実機の CLI での時間切れでのプロセスのツリーの終了の未確認です。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `scripts/interactive/harness/` (`terminal.py`、`screen.py`、`agents.py`、`scenarios/`)、`docs/ja/interactive-verification.md` の制限事項。製品 (サーバー) には影響しません。
- **放置した場合のリスク**: 許可を求める設定の環境や、画面描画・起動条件が変わった環境で、結果が `unverified` になるか、まれに誤った値を記録します。

### 推奨される対応策

- **恒久対応案**: (1) は、許可を求める設定で各 CLI を起動し、画面の文字列とキーを定義に加えます。(3) は、端末再現の不一致が実機シナリオに影響する場合に修正します。(7) は、必要に応じて走査対象のフォルダーを広げます。(9) は、WSL ではない Linux で確認します。(10) は、Claude Code の指示ファイルの読み込み仕様に合わせてシナリオを見直します。(13) は、再発したときに、端末の出力の全体を保存して調べます。実機の CLI での時間切れでのプロセスのツリーの終了は、時間切れを発生させて動作を確かめます。
- **参考情報**: 確認の記録の「実装のレビュー」と「基盤で見つけた問題と対処」、`docs/ja/interactive-verification.md`。

## BL-036 MCP のプロトコルの版 2026-07-28 への対応

### 概要と問題点

- **現状の事象**: 5 つの CLI のうち、Claude Code、Copilot CLI、agy (検証した版) は、接続のたびに、まずプロトコルの版 2026-07-28 で `server/discover` を送ります。サーバー (Go SDK v1.8.0 の、状態を持つ HTTP の受け口) は、対応する版 (2025-11-25 から 2024-11-05) を返し、CLI は 2025-11-25 の初期化で接続し直します。2026-07-28 のそれ以外の要求には、エラー (`-32022`) を返します。この動作は、`internal/server/protocol_test.go` で確かめています。
- **本来あるべき姿**: サーバーが、CLI の使う新しいプロトコルの版に応えます。

### 経緯と見送り理由

- **摘出契機**: 案件 `interactive-verification` の実装中に判明しました (接続の診断の出力)。
- **今回の見送り理由**: 2026-07-28 に対応する Go SDK は先行版 (v1.9.0-pre.1) だけで、対応は状態を持たない受け口に限られます。参加を MCP の接続に結び付けている今の設計 (`internal/tools/presence.go`) を作り直す必要があります。古い版での接続し直しで、5 つの CLI は使えています (検討用 Markdown `261119-connection-guidance-discussion.md` の Q-003)。
- **継続作業での経緯と対処**: 案件 `interactive-verification` で追加しました (2026-11-19)。案件 `connection-guidance` で、元の項目の 2 番 (停止の通知の手段) を対応しました。停止のログ通知とログの機能の宣言をやめ、停止はツールの呼び出しのエラーだけで伝えます (2026-11-20、Q-004、最終仕様の第 2 章)。1 番 (版への対応) は、今の動作をテストで保ち、この項目に残しました (Q-003、最終仕様の第 3 章)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002。Go SDK の公開されている最新は v1.9.0-pre.1 で、正式版は出ていません。2026-11-23 に確認)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `internal/server/` (HTTP の受け口)、`internal/tools/presence.go` (参加と接続の結び付け)、`docs/ja/reference.md` (MCP サーバーとしての振る舞い、制限事項)。
- **放置した場合のリスク**: CLI が、古い版での接続し直しをやめると、接続できなくなります。`protocol_test.go` は、SDK を上げたときの動作の変化を検出しますが、CLI の側の変化は検出しません。

### 推奨される対応策

- **恒久対応案**: Go SDK の正式版 (v1.9.0 以降) が出た時点で、状態を持たない受け口での参加の扱い (接続ではなく、要求ごとの識別での結び付け) を設計します。あわせて、各 CLI の版の上がり方 (古い版での接続し直しを続けるか) を、検証の基盤の接続の診断で確かめます。
- **参考情報**: Go SDK v1.9.0-pre.1 の `docs/protocol.md` と `docs/mcpgodebug.md`。`internal/server/server.go` の `Handler` のコメント。

## BL-040 agy が、閉じられた MCP の接続の後に、ツールを呼べなくなる

### 概要と問題点

- **現状の事象**: サーバーが要求のない MCP の接続を閉じた後 (`--connection-idle`、既定は 1 時間。閉じた接続への要求は HTTP の 404)、agy (1.5.12、1.5.13、1.5.14) の対話の画面で指示をすると、agy は新しい接続を作り、初期化とツールの一覧の取得を終えます。しかし、モデルが MCP のツールを呼ぶための道具 (`call_mcp_tool`) が「unknown tool」になり、ツールを呼べません。確認した 5 回で、1 回目の指示は 5 回とも失敗しました。2 回目の指示を送った 2 回では、1 回は参加し直して使え、1 回は再び失敗しました。1.5.13 でも、1 回目の指示は 5 回とも失敗し、2 回目の指示は 5 回のうち 2 回で使えました。サーバーを起動し直した後も、1 回目の指示は失敗し、2 回目で使えました。さらに、1.5.14 で (1) 閉じられた接続の後と (2) サーバーの起動し直しの後を各 3 回確認したところ、1 回目の指示は 6 回とも `unknown tool` で失敗し、2 回目の指示は 5 回成功しました (閉じられた接続の後は 3 回中 2 回が合言葉を答え、1 回は失敗。サーバーの起動し直しの後は 3 回とも合言葉を答えました)。起動し直した agy は、同じ名前で参加でき、使えました。
- **本来あるべき姿**: 接続し直した後の 1 回目の指示から、ツールを使えます (Claude Code、Codex、Copilot CLI、Grok は、そう動きました)。

### 経緯と見送り理由

- **摘出契機**: 案件 `connection-guidance` の前提調査で判明しました (2026-11-19)。BL-039 の対策の前に、使われない接続を 30 秒で閉じる確認用のサーバーで、5 つの CLI の動作を確かめました (検討用 Markdown `261119-connection-guidance-discussion.md` の Q-016)。
- **今回の見送り理由**: agy の側の不具合と見られ、サーバーの側では直せません。利用者は、使われない接続を閉じる設定を、この制限を文書に書いたうえで採用しました (Q-016、Q-017)。
- **継続作業での経緯と対処**: 案件 `connection-guidance` で追加しました (2026-11-19)。改善の後のサーバー (`--connection-idle 30s`) で確かめ直し、2 回目の指示でも失敗する場合があることが分かりました (2026-11-20)。利用ガイドの会話の手順と検証済みの範囲、スキルの文書に、agy を起動し直して同じ名前で参加させることを書きました。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、agy 1.5.13 で確かめ直しました。直っていません (2026-11-21)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、agy 1.5.14 で確かめ直す予定です (2026-11-23)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 9.2 の e) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、agy 1.5.14 を使って閉じられた接続の後とサーバーの起動し直しの後を各 3 回確かめ直しましたが、1 回目の指示は 6 回とも失敗し、2 回目の指示も閉じられた接続の後は 3 回中 1 回失敗しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md` の 2.2)。確認した版を 1.5.14 に更新し、未着手のまま残します。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: agy の利用者。1 時間以上ツールを呼ばなかった agy と、サーバーを起動し直した場合。
- **放置した場合のリスク**: 長く使わなかった agy が、起動し直すまでツールを呼べず、メッセージを受け取れません。

### 推奨される対応策

- **恒久対応案**: agy の新しい版で、同じ確認を繰り返します (確認の記録の「閉じられた接続の後の動作」の手順)。直らない場合は、agy の開発元に報告することを検討します。agy の利用者が多い場合は、`--connection-idle` の既定値を長くすることも検討します。
- **参考情報**: 確認の記録 `261119-connection-guidance-validation.md` の「閉じられた接続の後の動作」、`261123-backlog-batch-validation.md` の 2.2。Go SDK の `StreamableHTTPOptions.SessionTimeout`。

## BL-042 Windows の Codex で、出力の途中に作業中の表示が画面にない

### 概要と問題点

- **現状の事象**: Windows の Codex (0.170.0) に長い文章を出力させると、作業中の表示 (`esc to interrupt`) は最初の数秒だけ画面にあり、その後、作業を続けている間 (7[秒] から 63[秒]) は画面にありませんでした。検証の基盤は、この文字列で作業中を判定するため、Codex の作業の途中で完了と判定するおそれがあります。WSL の Codex では、作業中に表示が出ることを確かめています。
- **本来あるべき姿**: 検証の基盤が、Windows の Codex の作業中を見分けられます。

### 経緯と見送り理由

- **摘出契機**: 案件 `expiry-sandbox-verification` の確認 g (Windows の側のセッション。2026-11-20)。BL-035 の (12) から分けました。
- **今回の見送り理由**: 作業の途中の画面を保存していなかったため、代わりの文字列を決められませんでした。Windows の Codex のシナリオ 2 と 3 は、完了と判定されています。
- **継続作業での経緯と対処**: 案件 `expiry-sandbox-verification` で追加しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `scripts/interactive/harness/agents.py` の Codex の `busy`、`docs/ja/interactive-verification.md` の検証済みの範囲。製品 (サーバー) には影響しません。
- **放置した場合のリスク**: Windows の Codex で、長い応答の途中を完了と判定し、結果を `unverified` にするか、途中の画面で判定します。

### 推奨される対応策

- **恒久対応案**: 作業の途中の画面を保存して、作業中に必ず出る別の文字列を探します。見つからない場合は、Codex の完了の判定に、画面の変化が止まったことを加えます。
- **参考情報**: 確認の記録の「Windows の側の確認」。

## BL-044 ブランチのスコープの Windows 側での実機確認

### 概要と問題点

- **現状の事象**: ブランチのスコープ (`branch`) の機能は、自動テスト (`go test -race ./...`)、実行ファイル、および WSL 側の 5 つの CLI (Claude Code、Codex、Copilot CLI、Grok、agy) による非対話実行での確認 (`../past/261121-branch-scope-design.md` の 5.2 の a〜c) まで検証済みです。残っているのは、Windows 側のセッションでの確認 (`../past/261121-branch-scope-design.md` の 5.3 の d〜f: d. PowerShell を使う CLI が案内に従ってブランチを申告する、e. ブランチ `main` があるリポジトリで大文字と小文字を変えた名前 `Main` に切り替えたときの `git branch --show-current` の出力とサーバーが記録する値、f. 同じブランチを申告した WSL 側のセッションとブランチのルームで 1 往復できる) です。
- **本来あるべき姿**: Windows 側の各 CLI が、`lounge_help` の案内文に従って PowerShell 上で `git branch --show-current` を実行してブランチを申告し、Windows 上でブランチ名の大文字と小文字の違いが小文字にそろえられること、および同じブランチを申告した WSL 側のセッションとブランチのルームで 1 往復できることが実機で確認されています。

### 経緯と見送り理由

- **摘出契機**: 案件 `branch-scope` の第 2 ゲート承認時の利用者方針および検証段階 (2026-11-22)。
- **今回の見送り理由**: 案件 `branch-scope` では LADR-007 に従い必要最小限の実機確認までを実施し、続く案件 `backlog-batch` では WSL の側 (a〜c) を検証して、WSL の環境だけでは進められない Windows の側 (d〜f) は扱わず残しました (`261123-backlog-batch-lightadr.md` の LADR-002)。
- **継続作業での経緯と対処**: 案件 `branch-scope` で新規起票しました (2026-11-22)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、WSL の側 (最終仕様 5.2 の a から c) を確かめる予定です (2026-11-23)。Windows の側 (5.3 の d から f) は、その案件では扱いません。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 9.2 の a から c) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、WSL 側の 5 つの CLI での実機確認 (a から c) を実施し、5 つとも 1 回目の実行で検証済みになりました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md` の 2.1)。Windows 側 (`../past/261121-branch-scope-design.md` の 5.3 の d から f) が残ります。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `internal/tools/help.go` (PowerShell 上での案内文の実効性)、`docs/ja/user-guide.md` と `.agents/skills/agent-lounge-connect/SKILL.md` の「検証済みの範囲」。
- **放置した場合のリスク**: Windows 環境の PowerShell やケース非区別ファイルシステム上で、特定の CLI が `lounge_help` のブランチ案内を見落としたり、想定外のブランチ文字列を渡したり、WSL 側とブランチのルームでやりとりできない問題に気づくのが遅れます。

### 推奨される対応策

- **恒久対応案**: 最終仕様 (`../past/261121-branch-scope-design.md`) の 5.3 (Windows 側の d〜f) を実行し、結果を `docs/ja/user-guide.md` などの「検証済みの範囲」に反映します。案内文で申告が漏れる CLI があれば、`internal/tools/help.go` の表現を改善します。
- **参考情報**: `../past/261121-branch-scope-design.md` の 5.3、`261123-backlog-batch-validation.md` の 2.1・3 章。

## BL-047 MCP を使わない CLI の方式の、Windows での実機確認および Codex サンドボックスの確認

### 概要と問題点

- **現状の事象**: MCP を直接使わないエージェントのための CLI の方式 (`connection start|status|list|stop`、`tools`、`call`) は実装を完了し、CI の 3 つの OS の自動テストおよび WSL での 5 つの CLI と手動確認で検証済みです (`261123-backlog-batch-design.md` の 7 章、`261123-backlog-batch-validation.md` の 1・2.3・2.4・2.6 章)。残っているのは、(1) Windows での実機の確認 (常駐プロセス、保存場所の権限、プロフィールの収集、切り離した起動) と、(2) Codex のサンドボックス (`workspace-write`) でエージェントが `--detach` で起動した常駐プロセスがコマンドの後に残るかの確認 (ネットワークを許可していないサンドボックスでは、起動そのものが失敗するため、確かめられませんでした) です。
- **本来あるべき姿**: Windows の実機でも常駐プロセス、保存場所の権限 (本人だけの保護された DACL)、プロフィールの収集、切り離した起動が正しく動くこと、および Codex のサンドボックスでの `--detach` 起動時のプロセス残存の有無が実機で確認され、文書に検証済みとして記載されています。

### 経緯と見送り理由

- **摘出契機**: 案件 `backlog-batch` における実装と検証 (`261123-backlog-batch-design.md` の 7.10・9.2、`261123-backlog-batch-validation.md` の 2.4・3 章。2026-11-23)。元の要望は、2026-11-22 に新規起票しました。
- **今回の見送り理由**: WSL の環境からは Windows の実機確認を実施できないためです。また、ネットワークを許可していない Codex のサンドボックス (`workspace-write`) では、既定の保存場所はログを書けず (`read-only file system`)、`/tmp` の下ではサーバーに接続できないため、`connection start --detach` の起動そのものが失敗し、コマンドの後に常駐プロセスが残るかを確かめられませんでした。
- **継続作業での経緯と対処**: この残課題一覧を新規に作成して追加しました (2026-11-22)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、対策の検討を始めました (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-020 から Q-031、Q-036)。推奨される対応策のコマンドの例 (`session start` など) は、既存の管理コマンド `session list`、`session remove` と名前が重なるため、名前を検討しています (Q-020)。 `261122-cli-without-mcp-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 7 章) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、CLI の方式 (`connection`、`tools`、`call`、スキルの文書、利用者向け文書) を実装し、CI の 3 つの OS の自動テストと WSL での 5 つの CLI・手動確認を完了しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md`)。Windows での実機の確認 (常駐プロセス、保存場所の権限、プロフィールの収集、切り離した起動) と、Codex のサンドボックスでエージェントが `--detach` で起動した常駐プロセスがコマンドの後に残るかの確認が残ります。 macOS の実機の確認は、機材がないため行わず、机上の確認と自動テストだけにすると決めたため (利用者の恒久の決定。2026-11-23。簡易 ADR の LADR-012)、この項目から除きました。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `internal/loungecli/`、`internal/ownerdir/`、`cmd/agent-lounge-mcp/`、`docs/ja/cli-without-mcp.md`、`docs/ja/user-guide.md` の「検証済みの範囲」。
- **放置した場合のリスク**: Windows の本人だけの保護された DACL や Unix ドメイン ソケットの扱い、Windows でのプロフィール収集に実機固有の問題がある場合、気付けません。また、Codex のサンドボックス (ネットワークを許可した構成など) でエージェント自身が `--detach` を実行したときに、コマンド終了後も常駐プロセスが残るかどうかが不明なままになります。

### 推奨される対応策

- **恒久対応案**: Windows の実機で `connection start` (前面と `--detach`)、保存場所の権限、CLI による操作 (`tools`、`call --as NAME lounge_join` のプロフィールの収集、`chat_send`、`chat_inbox`、`connection status`、`connection stop`) を確認し、結果を `docs/ja/cli-without-mcp.md` と `docs/ja/user-guide.md` の「検証済みの範囲」に反映します。あわせて、Codex のサンドボックス (ネットワークを許可した設定) などでエージェントが `connection start --detach` を実行したときに、コマンドの後に常駐プロセスが残るかを実機で確認します。
- **参考情報**: `261123-backlog-batch-design.md` の 7 章・9.2、`261123-backlog-batch-validation.md` の 2.3・2.4・2.6・3 章、`docs/ja/cli-without-mcp.md`。
## BL-048 `shared` とルームの、Claude Code 以外の 4 つの CLI での実機確認

### 概要と問題点

- **現状の事象**: `shared` のスコープ、ルームの作成と参加、参加の前のやりとりの読み返し、`shared` のメモリを、実機の CLI では Claude Code だけで確かめます。Codex、Copilot CLI、Grok、agy では確かめません。
- **本来あるべき姿**: 5 つの CLI のすべてで、使い方の案内だけから、最終仕様 `261124-shared-space-rooms-design.md` の 9.2 の操作を行えることが確かめられています。

### 経緯と見送り理由

- **摘出契機**: 前提検討時に見送り (検討用 Markdown `261124-shared-space-rooms-discussion.md` の Q-025、簡易 ADR の LADR-007)。
- **今回の見送り理由**: Grok はトークンの上限に達し、Copilot CLI も上限が近いため、Claude Code が代表して確かめると利用者が決めました。エージェントによって動作は変わらないという利用者の見込みによります。
- **継続作業での経緯と対処**: なし。 案件 `help-text-result` では対策せず、`../current/261124-help-text-result-backlog.md` の同じ課題 ID へ転記しました (2026-11-25)。

### 影響範囲とリスク

- **影響対象**: `docs/ja/user-guide.md` の「検証済みの範囲」、`internal/tools/help.go` (案内の分かりやすさ)。
- **放置した場合のリスク**: 一部の CLI が、新しいツール `chat_room` や `shared` のスコープを、案内だけでは正しく使えない場合に、気付けません。

### 推奨される対応策

- **恒久対応案**: トークンの上限に余裕がある時に、WSL の側で、Codex、Copilot CLI、Grok、agy の非対話の実行により、最終仕様の 9.2 の 4 つの操作を確かめ、利用ガイドの「検証済みの範囲」を更新します。
- **参考情報**: `261124-shared-space-rooms-design.md` の 9.2・9.3。

## BL-049 Claude Code 2.1.331 で `lounge_help` の案内が空として扱われる

### 概要と問題点

- **現状の事象**: Claude Code 2.1.331 の非対話の実行で、`lounge_help` の結果が空 (`{}`) として扱われ、案内の本文を読めません。サーバーは、文字列の結果に案内の本文を、構造化された結果に空のオブジェクト (`unread` だけを持つ形。参加の前は空) を返します (`internal/tools/help.go` の `HelpOutput`)。Claude Code は、構造化された結果だけをモデルに渡したと考えられます。
- **本来あるべき姿**: どの CLI でも、`lounge_help` で使い方の案内の全文を読めます。

### 経緯と見送り理由

- **摘出契機**: 案件 `shared-space-rooms` の実機の確認で判明しました (確認の記録 `261124-shared-space-rooms-validation.md` の 3 章)。4 回の実行のすべてで再現しました。`curl` での確認では、サーバーの応答に案内の本文がありました。
- **今回の見送り理由**: この案件の範囲外です (`lounge_help` は変えていない)。構造化された結果に案内を入れると、両方をモデルに渡す CLI で案内が 2 回渡り、Copilot CLI が一度に読める大きさを超えるため (`maxGuideBytes` の注記)、直し方に判断が要ります。
- **継続作業での経緯と対処**: 案件 `help-text-result` で対策を始めました (2026-11-24。全体管理文書 `../current/261124-help-text-result-overview.md`)。Claude Code 2.1.331 が構造化された結果をモデルに渡すことを `stream-json` の記録で確かめ、出力スキーマを持たない形の試作で本文が渡ることを確かめました。`lounge_help` だけ出力スキーマを持たず、テキストの結果だけを返す方針に決めました (`../current/261124-help-text-result-lightadr.md` の LADR-002)。 実装 (コミット `d3a5768`) と 4 つの CLI での実機の確認 (`../current/261124-help-text-result-validation.md`) を終え、対応済みとしました (2026-11-25)。Grok は利用の上限のため未検証とし、BL-050 として転記先に記録しました。

### 影響範囲とリスク

- **影響対象**: `internal/tools/help.go` (`HelpOutput`、`resultText`)、`internal/tools/tools.go` (構造化された結果の扱い)、Claude Code を使う全利用者。
- **放置した場合のリスク**: Claude Code のエージェントが、プロフィールの調べ方、スコープ、ルーム、受信箱を読む時点などの規則を、案内から得られません。ツールの説明だけで動くため、誤った使い方に気付きにくくなります。

### 推奨される対応策

- **恒久対応案**: Claude Code の版ごとに、文字列の結果と構造化された結果のどちらをモデルに渡すかを確かめます。そのうえで、`lounge_help` の構造化された結果の形 (案内を入れるか、出力スキーマを持たないツールにするか) を決めます。Copilot CLI の大きさの制約を合わせて確かめます。
- **参考情報**: `261124-shared-space-rooms-validation.md` の 3 章、`internal/tools/help.go` の `maxGuideBytes` と `HelpOutput` の注記。
