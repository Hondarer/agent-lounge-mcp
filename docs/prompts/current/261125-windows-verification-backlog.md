<!--
生成元: agree-before-implement スキル
文書種別: 残課題一覧 (backlog)
最終更新日時: 2026-11-25 20:13:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261125-windows-verification-overview.md` (全体管理文書)、`261125-windows-verification-design.md` (最終仕様)、`261125-windows-verification-discussion.md` (検討用 Markdown)、`261125-windows-verification-lightadr.md` (簡易 ADR)。対策元: `../past/261124-help-text-result-backlog.md`。いずれも削除されている場合があります。
下流文書: この残課題一覧を対策する継続作業が開始された場合は、その全体管理文書・検討用 Markdown・最終仕様・残課題一覧のパスを追記します。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。「整理状態」が「随時追記中」であれば、案件の作業中です。気付いた事項を追記し、第 3 ゲートの前に文書を再構築して整理します。上流文書が存在する場合は、各課題の背景と見送りの判断を上流文書で確認します。存在しない場合は、この文書を単体の残課題一覧として扱います。このファイルの残課題を参照して新たな成果物の作成または更新を始めるときは、元案件の再開とは別の作業として扱います。その開始には agree-before-implement を原則として使い、新しい全体管理文書と検討用 Markdown から開始してください。継続作業によってこのファイルの残課題を対策する場合は、このファイルや各項目の記載を削除することなく、継続作業の開始時 (全体管理文書と検討用 Markdown の作成時) から検討・設計・実装・整理の各段階において、このファイルのメタ情報、各項目の状態、および経緯と対処を随時更新することを原則とします。また、フェーズ分けした対策で対処できていない項目がある場合は、当該継続作業の backlog にも必ず転記し、最新作業の backlog を見れば残件が把握できるようにしてください。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。
-->

# 残課題一覧 (バックログ)

ファイル名: `261125-windows-verification-backlog.md`

## 案件の識別

案件名: `windows-verification` (Windows の側のセッションでの残課題の確認)  
作成日: `2026-11-25`  
整理状態: 整理済み

この文書は、見送る決定がなされた問題点や、作業中に潜在または別問題として判明したが今回は対策しなかった残課題を記録するものです。各項目は、チケット管理システムへの起票にそのまま利用できる詳細度で記載します。

残課題は、案件ごとに全量を引き継ぎます (利用者の指示)。この文書は、対策元の残課題一覧 (`../past/261124-help-text-result-backlog.md`) のうち、この案件で対応済みにならなかった項目を、同じ課題 ID で転記して作成しました。BL-014、BL-042、BL-044、BL-047、BL-048 は、この案件で対応済みです (Grok の未検証の分は BL-050 にまとめた)。BL-051 と BL-052 は、この案件で記録した項目です。

優先度は、Claude Code の提案です。利用者の評価は受けていません。

## 残課題一覧

| 課題 ID | 題名 | 区分 | 優先度 / 影響度 | 状態 |
| --- | --- | --- | --- | --- |
| BL-013 | 英語の文書の作成 (`docs/en/`、`README.md`、GitHub の概要) | 過去 backlog からの転記 | 中 | 未着手 |
| BL-028 | Windows の側で起動したサーバーへの、WSL からの接続 | 過去 backlog からの転記 | 低 | 未着手 |
| BL-030 | 未検証の構成 (物理的に別のマシンからの接続) | 過去 backlog からの転記 | 中 (物理的に別のマシンからの接続とリバース プロキシ経由の通信が未検証) | 未着手 |
| BL-031 | トークンの運用の機能 (権限の区分、接続元の制限) | 過去 backlog からの転記 | 中 | 一部対応 ((1) を対応。(6) を容認。(2)、(3) が残る) |
| BL-035 | 検証の基盤の残り (他の CLI の許可の画面、画面の再現の細部、WSL ではない Linux ほか) | 過去 backlog からの転記 | 低 | 一部対応 ((17) を対応。(1)、(3)、(7)、(9)、(10)、(13) などが残る) |
| BL-036 | MCP のプロトコルの版 2026-07-28 への対応 | 過去 backlog からの転記 | 中 (CLI が古い版での接続し直しをやめると、接続できなくなる) | 一部対応 (停止の通知を対応。版への対応が残る) |
| BL-040 | agy が、閉じられた MCP の接続の後に、ツールを呼べなくなる | 過去 backlog からの転記 | 中 (1 時間使わなかった agy は、起動し直すまでツールを呼べない場合がある) | 未着手 (1.5.17 でも再現) |
| BL-050 | Grok での実機確認 (`lounge_help` の本文、`shared` とルーム、Windows の日本語の名前とブランチの申告) | 過去 backlog からの転記 | 低 | 未着手 |
| BL-051 | Windows の Codex がサンドボックスの中で PowerShell を起動できず、プロフィールを申告できない | 作業中摘出 | 中 (Windows のサンドボックスの Codex が、リポジトリとブランチのスコープに入れない) | 未着手 |
| BL-052 | 検証の基盤が、Codex の起動時の更新の案内に答えてしまう | 作業中摘出 | 低 (検証の基盤だけ。確認の途中で Codex の版が変わる) | 未着手 |

## BL-013 英語の文書の作成 (`docs/en/`、`README.md`、GitHub の概要)

### 概要と問題点

- **現状の事象**: 利用者向けの文書は `docs/ja/` だけで、英訳を置く `docs/en/` がありません。`README.md` と `docs/README.md` は日本語だけです。GitHub の概要 (About 欄の説明文) も日本語です。文書の構成と `AGENTS.md` の規範は、`docs/ja/` を正本、`docs/en/` を英訳とする前提で作ってあります。
- **本来あるべき姿**: `docs/en/` に `docs/ja/` の英訳があり、`README.md` と `docs/README.md` は英語、日本語の順に併記され、GitHub の概要は英語です。

### 経緯と見送り理由

- **摘出契機**: 前提検討時に見送り (Q-013 の補足、Q-019 の補足、Q-020)。
- **今回の見送り理由**: 英語版は、ある程度検証が進んでから、まとめて作成するという利用者の判断のためです。
- **継続作業での経緯と対処**: `261117-chat-connectivity-backlog.md` の同じ課題 ID から転記しました。この案件 (`chat-autonomy`) では対策しません (検討用 Markdown の Q-001)。`261117-chat-autonomy-backlog.md` の同じ課題 ID から、案件 `repository-space` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 この案件 (`repository-space`) では対策しません (検討用 Markdown の Q-001)。 `261117-repository-space-backlog.md` の同じ課題 ID から、案件 `multi-user-access` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

### 影響範囲とリスク

- **影響対象**: `docs/en/` (新規)、`README.md`、`docs/README.md`、`AGENTS.md` の「Languages」の節、GitHub のリポジトリの設定。
- **放置した場合のリスク**: 日本語を読まない利用者が、使い方を把握できません。英訳の作成が遅れるほど、訳す量が増えます。

### 推奨される対応策

- **恒久対応案**: `docs/ja/reference.md` と `docs/ja/user-guide.md` を訳して `docs/en/` に置き、`README.md` と `docs/README.md` を併記にします。あわせて、`AGENTS.md` の「`docs/en/` はまだ存在しない」という記述を更新します。GitHub の概要は、文面の承認を受けた後、Claude Code が `gh repo edit --description` で設定します (Q-017)。
- **参考情報**: Q-013、Q-017、Q-019、Q-020。

## BL-028 Windows の側で起動したサーバーへの、WSL からの接続

### 概要と問題点

- **現状の事象**: Windows の側で起動したサーバーに、WSL の側のエージェントから接続する構成 (元の (3)) を、確かめていません。Windows のファイアウォールが、新しい実行ファイルの受信を、許可の画面で利用者が許可するまで止めるため、利用者が画面を見られる時に確かめる必要があります。
- **本来あるべき姿**: 利用ガイドの「検証済みの範囲」で、この構成が検証済みになります。

### 経緯と見送り理由

- **摘出契機**: Windows の側の Claude Code のセッションとの確認 (利用者の指示で、最終仕様の到達点 5 を検証に変更) で判明しました。
- **今回の見送り理由**: スコープ外。Ctrl+C のキー操作と対話の画面は、非対話の実行では確認できません。Windows の側には Go がなく、サーバーは WSL の側で起動しました。
- **継続作業での経緯と対処**: `261117-repository-space-backlog.md` の同じ課題 ID から、案件 `multi-user-access` の残課題一覧 (この文書) へ転記しました (2026-11-17)。 `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 この案件では、(6) を追加しました。今回の変更は OS によって変わる部分を含まないため、Windows の側での確認を省きました (検討用 Markdown `261118-chat-completion-discussion.md` の Q-015)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 この案件でも、Windows の側での確認を省きます (Q-024)。メモリのツールと管理コマンドを、未検証の項目に加えます。 (7) を追加しました。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 この案件で、Windows で動くエージェントでの `chat_ack`、`chat_status`、`chat_history` とメモリのツールは、OS によって変わる部分を含まないため、確認の必要がない (`-`) と決め、利用ガイドの検証済みの範囲を改めました (2026-11-20、検討用 Markdown `261119-connection-guidance-discussion.md` の Q-014、Q-015)。(6) と (7) から、これらを除きました。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(1)、(2) (疑似端末へのキー入力で確かめ、物理的なキーボードは対象外)、(4)、(6)、(7) を確かめ、検証済みにしました。(5) の `windows-arm64` は、ビルドだけを行い動作を保証しないことにして、除きました (利用者の判断)。(3) は、利用者が画面を見られない時間だったため、確かめずに残しました (利用者の判断。2026-11-21)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 案件 `windows-verification` (`261125-windows-verification-overview.md`) で、対象にするかを検討し、対象外にしました (2026-11-25。検討用 Markdown `261125-windows-verification-discussion.md` の Q-001 で選ばれなかった)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

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
- **継続作業での経緯と対処**: `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、nginx を置いた構成を、WSL の podman で確かめ、利用ガイドの例を検証済みにしました (2026-11-21)。物理的に別のマシンと、サーバー自身の HTTPS は、残しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、サーバー自身の HTTPS を取り下げるかを検討しています (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-035)。物理的に別のマシンからの接続の検証は、その案件では扱いません。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 サーバー自身の HTTPS は、取り下げました (2026-11-23。この案件の簡易 ADR の LADR-010)。物理的に別のマシンからの接続の検証だけを残します。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

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
- **継続作業での経緯と対処**: `261117-multi-user-access-backlog.md` の同じ課題 ID から、案件 `chat-completion` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-chat-completion-backlog.md` の同じ課題 ID から、案件 `memory` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 `261118-memory-backlog.md` の同じ課題 ID から、案件 `interactive-verification` の残課題一覧 (この文書) へ転記しました (2026-11-18)。 案件 `interactive-verification` で、5 番の実態を、2 つの OS で調べました (2026-11-19。トークンの形 `alm_` と 43 文字で、各 CLI の設定、履歴、ログのフォルダーを走査)。本物のトークンは、どちらの OS でも、案件 `multi-user-access` の確認のものだけでした。残っていた場所は、確認用のフォルダーで動かした Grok のセッションの履歴 (設定ファイルの `Authorization` の値を、Grok が読んで履歴に残したもの) と、確認を行った Claude Code のセッションの会話の記録 (確認のコマンドに書いたもの) です。トークンは発行し直しまたはデータベースの削除で、無効になっています。Grok の履歴は削除しました。Claude Code の会話の記録は、セッションの再開に使うため、残しています。各エージェントのセッションのデータが残ることは、セキュリティの問題ではなく、各エージェントの動作であるため、今後は原則として対処しません (利用者の判断。2026-11-19)。5 番は、この判断に従い、追加の対策を行いません。 協調テストで使うデータベースはテスト用で、テストの後に削除するため、テストのトークンが会話や確認の記録に残っても問題ない、という判断も受けました (利用者。2026-11-19)。この案件のシナリオ 6 のトークンは、どこにも残っていませんでした。CLI の設定のバックアップ (`.claude/backups/`) は、入れ替わって、トークンを含まなくなっていました。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(1) を対応しました (`--expires-in`、`user expiry`、期限切れの 401。2026-11-21)。(2) から (6) は残しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、(4) と (5) を取り下げるかを検討しています (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-033、Q-034)。(2)、(3)、(6) は、保留を続けます (その案件の簡易 ADR の LADR-002)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 (4) と (5) は、取り下げました (2026-11-23。この案件の簡易 ADR の LADR-010)。利用ガイドの注意は残します。(2)、(3)、(6) は、保留を続けます (LADR-002)。 なお、BL-047 の対応にあわせて、CLI の方式ではトークンを設定ファイルに保存しないことを利用ガイドに書き足しました (2026-11-23。最終仕様 `261123-backlog-batch-design.md` の 10 章)。 継続案件 `shared-space-rooms` (`../past/261124-shared-space-rooms-overview.md`) で、(6) を仕様として容認し、この項目から除くと決めました (2026-11-24。検討用 Markdown `../past/261124-shared-space-rooms-discussion.md` の Q-002、簡易 ADR `../past/261124-shared-space-rooms-lightadr.md` の LADR-002。利用者の判断: この MCP を通して共有される情報には機密情報はなく、接続と情報の共有は別に考える)。(2)、(3) は、保留を続けます。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

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
- **継続作業での経緯と対処**: 案件 `interactive-verification` で追加しました (2026-11-19)。 `261118-interactive-verification-backlog.md` の同じ課題 ID から、案件 `connection-guidance` の残課題一覧 (この文書) へ転記しました (2026-11-19)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 この案件で、一部の番号を検討します (Q-001、Q-004)。 (1)、(2)、(4)、(5)、(6)、(8) を直し、(11) を記録して除きます。(1) は Copilot CLI の画面だけを定義します (Q-004、Q-010、Q-011、最終仕様の第 6 章)。(3)、(7)、(9)、(10) と、他の 4 つの CLI の許可の画面は、この項目に残します。 (2)、(4)、(5)、(6)、(8) を対応し、(11) は残ったプロセスがないことを確かめて除きました (2026-11-20)。(1) は、Copilot CLI のシェルのコマンドの許可の画面に応答する形にし、一括の許可 (`--allow-tool shell`) をやめました。WSL と Windows の実機で確かめました (確認の記録 `261120-small-fixes-validation.md`)。(12) から (16) を追加しました。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、(14)、(15)、(16) を直し、(12) を確かめました (2026-11-21)。(12) のうち、Windows の Codex の作業中の表示は、BL-042 に分けました。(16) の確認の中で、Copilot CLI のパスへのアクセスの許可の画面に応答する定義と、完了の合言葉の綴りの誤りへの対処を足しました。(9) の macOS は、動作を保証しないことにして除きました (利用者の判断)。(17) を追加しました。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、(17) の対策の検討を始めました (2026-11-23。検討用 Markdown `261123-backlog-batch-discussion.md` の Q-014)。その他の番号は、その案件では扱いません。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 8 章 ((17))) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、(17) の許可の画面への再応答の条件を実装し、自動テスト (Python の 46 件、CI の 3 つの OS を含む) と Copilot CLI の対話の画面で確認して対応を完了しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md`)。残るのは、(1)、(3)、(7)、(9)、(10)、(13) と、実機の CLI での時間切れでのプロセスのツリーの終了の未確認です。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

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
- **継続作業での経緯と対処**: 案件 `interactive-verification` で追加しました (2026-11-19)。案件 `connection-guidance` で、元の項目の 2 番 (停止の通知の手段) を対応しました。停止のログ通知とログの機能の宣言をやめ、停止はツールの呼び出しのエラーだけで伝えます (2026-11-20、Q-004、最終仕様の第 2 章)。1 番 (版への対応) は、今の動作をテストで保ち、この項目に残しました (Q-003、最終仕様の第 3 章)。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 この案件では、扱いません (簡易 ADR の LADR-002。Go SDK の公開されている最新は v1.9.0-pre.1 で、正式版は出ていません。2026-11-23 に確認)。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 この案件では、扱いません (簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。

### 影響範囲とリスク

- **影響対象**: `internal/server/` (HTTP の受け口)、`internal/tools/presence.go` (参加と接続の結び付け)、`docs/ja/reference.md` (MCP サーバーとしての振る舞い、制限事項)。
- **放置した場合のリスク**: CLI が、古い版での接続し直しをやめると、接続できなくなります。`protocol_test.go` は、SDK を上げたときの動作の変化を検出しますが、CLI の側の変化は検出しません。

### 推奨される対応策

- **恒久対応案**: Go SDK の正式版 (v1.9.0 以降) が出た時点で、状態を持たない受け口での参加の扱い (接続ではなく、要求ごとの識別での結び付け) を設計します。あわせて、各 CLI の版の上がり方 (古い版での接続し直しを続けるか) を、検証の基盤の接続の診断で確かめます。
- **参考情報**: Go SDK v1.9.0-pre.1 の `docs/protocol.md` と `docs/mcpgodebug.md`。`internal/server/server.go` の `Handler` のコメント。

## BL-040 agy が、閉じられた MCP の接続の後に、ツールを呼べなくなる

### 概要と問題点

- **現状の事象**: サーバーが要求のない MCP の接続を閉じた後 (`--connection-idle`、既定は 1 時間。閉じた接続への要求は HTTP の 404)、agy (1.5.12 から 1.5.14、および 1.5.17) の対話の画面で指示をすると、agy は新しい接続を作り、初期化とツールの一覧の取得を終えます。しかし、モデルが MCP のツールを呼ぶための道具 (`call_mcp_tool`) が「unknown tool」になり、ツールを呼べません。確認した 5 回で、1 回目の指示は 5 回とも失敗しました。2 回目の指示を送った 2 回では、1 回は参加し直して使え、1 回は再び失敗しました。1.5.13 でも、1 回目の指示は 5 回とも失敗し、2 回目の指示は 5 回のうち 2 回で使えました。サーバーを起動し直した後も、1 回目の指示は失敗し、2 回目で使えました。さらに、1.5.14 で (1) 閉じられた接続の後と (2) サーバーの起動し直しの後を各 3 回確認したところ、1 回目の指示は 6 回とも `unknown tool` で失敗し、2 回目の指示は 5 回成功しました (閉じられた接続の後は 3 回中 2 回が合言葉を答え、1 回は失敗。サーバーの起動し直しの後は 3 回とも合言葉を答えました)。起動し直した agy は、同じ名前で参加でき、使えました。1.5.17 でも、WSL の側と Windows の側の両方で、1 回目の指示は 12 回とも `unknown tool` で失敗しました (2 回目の指示は 12 回中 9 回で合言葉を答えた)。
- **本来あるべき姿**: 接続し直した後の 1 回目の指示から、ツールを使えます (Claude Code、Codex、Copilot CLI、Grok は、そう動きました)。

### 経緯と見送り理由

- **摘出契機**: 案件 `connection-guidance` の前提調査で判明しました (2026-11-19)。BL-039 の対策の前に、使われない接続を 30 秒で閉じる確認用のサーバーで、5 つの CLI の動作を確かめました (検討用 Markdown `261119-connection-guidance-discussion.md` の Q-016)。
- **今回の見送り理由**: agy の側の不具合と見られ、サーバーの側では直せません。利用者は、使われない接続を閉じる設定を、この制限を文書に書いたうえで採用しました (Q-016、Q-017)。
- **継続作業での経緯と対処**: 案件 `connection-guidance` で追加しました (2026-11-19)。改善の後のサーバー (`--connection-idle 30s`) で確かめ直し、2 回目の指示でも失敗する場合があることが分かりました (2026-11-20)。利用ガイドの会話の手順と検証済みの範囲、スキルの文書に、agy を起動し直して同じ名前で参加させることを書きました。 `261119-connection-guidance-backlog.md` の同じ課題 ID から、案件 `small-fixes` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 `261120-small-fixes-backlog.md` の同じ課題 ID から、案件 `expiry-sandbox-verification` の残課題一覧 (この文書) へ転記しました (2026-11-20)。 案件 `expiry-sandbox-verification` で、agy 1.5.13 で確かめ直しました。直っていません (2026-11-21)。 `261120-expiry-sandbox-verification-backlog.md` の同じ課題 ID から、案件 `branch-scope` の残課題一覧 (この文書) へ転記しました (2026-11-21)。 案件 `backlog-batch` (`261123-backlog-batch-overview.md`) で、agy 1.5.14 で確かめ直す予定です (2026-11-23)。 `261121-branch-scope-backlog.md` の同じ課題 ID から、案件 `backlog-batch` の残課題一覧 (この文書) へ転記しました (2026-11-23)。 最終仕様 (`261123-backlog-batch-design.md` の 9.2 の e) を作成しました (2026-11-23)。承認の後に着手します。 案件 `backlog-batch` で、agy 1.5.14 を使って閉じられた接続の後とサーバーの起動し直しの後を各 3 回確かめ直しましたが、1 回目の指示は 6 回とも失敗し、2 回目の指示も閉じられた接続の後は 3 回中 1 回失敗しました (2026-11-23。確認の記録 `261123-backlog-batch-validation.md` の 2.2)。確認した版を 1.5.14 に更新し、未着手のまま残します。 `261123-backlog-batch-backlog.md` の同じ課題 ID から、案件 `shared-space-rooms` の残課題一覧 (この文書) へ転記しました (2026-11-24)。 `../past/261124-shared-space-rooms-backlog.md` の同じ課題 ID から、案件 `help-text-result` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件では、扱いません (簡易 ADR の LADR-001)。 案件 `windows-verification` (`261125-windows-verification-overview.md`) で、対象にしました (2026-11-25。検討用 Markdown `261125-windows-verification-discussion.md` の Q-001、簡易 ADR `261125-windows-verification-lightadr.md` の LADR-001)。最終仕様 `261125-windows-verification-design.md` を作成しました。承認の後に着手します。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。 案件 `windows-verification` で、agy 1.5.17 を使い、WSL の側と Windows の側で、閉じられた接続の後とサーバーの起動し直しの後を各 3 回確かめ直しました (2026-11-25。確認の記録 `261125-windows-verification-validation.md` の 2.5)。1 回目の指示は 12 回とも `unknown tool` で失敗し、2 回目の指示は WSL の側で 6 回中 4 回、Windows の側で 6 回中 5 回、合言葉を答えました。直っていないため、確認した版を 1.5.17 に更新し、残します。

### 影響範囲とリスク

- **影響対象**: agy の利用者。1 時間以上ツールを呼ばなかった agy と、サーバーを起動し直した場合。
- **放置した場合のリスク**: 長く使わなかった agy が、起動し直すまでツールを呼べず、メッセージを受け取れません。

### 推奨される対応策

- **恒久対応案**: agy の新しい版で、同じ確認を繰り返します (確認の記録の「閉じられた接続の後の動作」の手順)。直らない場合は、agy の開発元に報告することを検討します。agy の利用者が多い場合は、`--connection-idle` の既定値を長くすることも検討します。
- **参考情報**: 確認の記録 `261119-connection-guidance-validation.md` の「閉じられた接続の後の動作」、`261123-backlog-batch-validation.md` の 2.2、`261125-windows-verification-validation.md` の 2.5。Go SDK の `StreamableHTTPOptions.SessionTimeout`。

## BL-050 Grok での実機確認 (`lounge_help` の本文、`shared` とルーム、Windows の日本語の名前とブランチの申告)

### 概要と問題点

- **現状の事象**: Grok 1.0.92 は、利用の上限 (`402 Payment Required`、`Grok Build usage balance exhausted`) のため、次の確認を実行できず、未検証です。(1) `lounge_help` の本文を、参加の前と後に全文読めること (案件 `help-text-result`。`../past/261124-help-text-result-validation.md` の 2 章)。(2) `shared` のスコープとルームの 4 つの操作 (元の BL-048。`../past/261124-shared-space-rooms-design.md` の 9.2)。(3) Windows の側での、日本語と全角の数字を含む名前の正規化と、PowerShell でのブランチの申告 (元の BL-014 と BL-044 の Grok の分。`261125-windows-verification-design.md` の 3.1)。他の 4 つの CLI は、これらを検証済みです (Windows の Codex のブランチの申告は BL-051)。
- **本来あるべき姿**: Grok でも、(1) から (3) が確かめてあり、利用ガイドの「検証済みの範囲」の Grok の未検証の記載がなくなります。

### 経緯と見送り理由

- **摘出契機**: 案件 `help-text-result` の実機の確認で、Grok が利用の上限で実行できませんでした ((1))。案件 `windows-verification` の確認でも、WSL の側 (2026-11-25 19:14) と Windows の側の両方で、上限のままでした ((2)、(3))。
- **今回の見送り理由**: 利用者の指示 (「トークンリミットの場合は未検証としてください。」、および `261125-windows-verification-lightadr.md` の LADR-004) により、未検証としました。
- **継続作業での経緯と対処**: 案件 `help-text-result` で記録しました。 `../past/261124-help-text-result-backlog.md` の同じ課題 ID から、案件 `windows-verification` の残課題一覧 (この文書) へ転記しました (2026-11-25)。この案件の確認でも Grok は上限のままだったため、元の BL-048 と、BL-014 と BL-044 の Grok の分を、この項目にまとめました。

### 影響範囲とリスク

- **影響対象**: `docs/ja/user-guide.md` の検証済みの範囲 (Grok の未検証の行と記載)。
- **放置した場合のリスク**: Grok が、案内だけでは `shared` のスコープやルームを正しく使えない場合や、Windows の側で名前やブランチの申告に不都合がある場合に、気付けません。(1) のリスクは小さいと見込みます (変更の前から、Grok にはテキストの結果が渡っていた)。

### 推奨される対応策

- **恒久対応案**: Grok の利用の上限が戻った後に、(1) は `../past/261124-help-text-result-design.md` の 5 章、(2) は `261125-windows-verification-validation.md` の 3.1 の 3 回の指示、(3) は同じく 2.1 の指示で確かめ、`docs/ja/user-guide.md` の該当の行を更新します。
- **参考情報**: `../past/261124-help-text-result-validation.md` の 2 章、`261125-windows-verification-validation.md` の 2.1・3.1。

## BL-051 Windows の Codex がサンドボックスの中で PowerShell を起動できず、プロフィールを申告できない

### 概要と問題点

- **現状の事象**: Windows の Codex 0.171.0 を `-s read-only` で実行すると、サンドボックスの中で `pwsh.exe` を起動できず (`CreateProcessAsUserW failed: 5 (アクセスが拒否されました)`)、使い方の案内に従ってプロフィールを調べる 5 つのコマンドがすべて失敗しました。Codex は `cmd.exe` に切り替えずに、リポジトリの単位とブランチが空のまま参加しました。このため、リポジトリとブランチのスコープに入れません。前の確認 (Codex 0.170.0) では、モデルが `cmd.exe /c` に切り替えて申告できました (`docs/ja/user-guide.md` の検証済みの範囲)。
- **本来あるべき姿**: Windows のサンドボックスの中の Codex も、リポジトリの単位とブランチを申告できます。または、申告できない場合の利用者の対処が文書にあります。

### 経緯と見送り理由

- **摘出契機**: 案件 `windows-verification` の Windows の側の確認 (2026-11-25。`261125-windows-verification-validation.md` の 2.1)。
- **今回の見送り理由**: この案件は確認だけを範囲とし、使い方の案内 (`internal/tools/help.go`) の表現を変える場合は利用者に確かめる (最終仕様の 7 章) としていました。モデルの判断による違いか、案内で `cmd.exe` を示すべきかを、まだ見極めていません。
- **継続作業での経緯と対処**: 案件 `windows-verification` で記録しました。

### 影響範囲とリスク

- **影響対象**: `internal/tools/help.go` (Windows のシェルでの申告の案内)、`docs/ja/user-guide.md` の検証済みの範囲と Codex の登録の説明。
- **放置した場合のリスク**: Windows のサンドボックスの中の Codex が、リポジトリやブランチのルームのやりとりに加われず、利用者が気付きにくいです。

### 推奨される対応策

- **暫定対応案 (ワークアラウンド)**: Windows の Codex は、サンドボックスを外すか、PowerShell の起動を許す設定で使います。
- **恒久対応案**: 同じ指示を数回実行して、`cmd.exe` に切り替えるかの割合を確かめます。切り替えない場合が多ければ、案内に「PowerShell を起動できない場合は `cmd.exe /c` で同じ値を調べる」ことを足すかを検討し、5 つの CLI で確かめます。
- **参考情報**: `261125-windows-verification-validation.md` の 2.1。

## BL-052 検証の基盤が、Codex の起動時の更新の案内に答えてしまう

### 概要と問題点

- **現状の事象**: Codex は、起動したときに新しい版があると、更新の案内を出します。検証の基盤 (`scripts/interactive/harness/agents.py` の Codex の定義の `dialogs` と `ready` の扱い) がこれに答えたとみられ、Codex が自分を更新しました (Windows は `npm install -g @openai/codex`、WSL は公式のインストールのスクリプト。どちらも 0.171.0 から 0.172.0)。更新の間は指示に応答せず、シナリオの結果が `unverified` になるか、応答が未完了になりました。利用者の環境の CLI が、確認の途中で上がることにもなります。
- **本来あるべき姿**: 基盤は、更新の案内を見分けて「更新しない」と答えるか、更新の確認を切る設定で Codex を起動し、確認の途中で CLI の版を変えません。

### 経緯と見送り理由

- **摘出契機**: 案件 `windows-verification` の、基盤の直しの確認 (2026-11-25。`261125-windows-verification-validation.md` の 4 章)。
- **今回の見送り理由**: この案件で直す基盤は、完了の判定 (BL-042) だけと決めていたためです (最終仕様の 9 章)。
- **継続作業での経緯と対処**: 案件 `windows-verification` で記録しました。

### 影響範囲とリスク

- **影響対象**: `scripts/interactive/harness/agents.py` (Codex の定義)、`docs/ja/interactive-verification.md`。製品 (サーバー) には影響しません。
- **放置した場合のリスク**: Codex の新しい版が出るたびに、最初の確認が失敗し、利用者の環境の Codex が予告なく更新されます。

### 推奨される対応策

- **恒久対応案**: 更新の案内の画面を保存して文字列と選択肢を調べ、定義の `dialogs` に「更新しない」を選ぶキーを足すか、Codex の更新の確認を切る設定 (設定のファイルまたは起動のオプション) を `interactive` に足します。自己試験とシナリオで確かめます。
- **参考情報**: `261125-windows-verification-validation.md` の 4 章。
