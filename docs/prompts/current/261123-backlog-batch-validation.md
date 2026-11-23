<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-23 17:10:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261123-backlog-batch-overview.md` (全体管理文書)、`261123-backlog-batch-design.md` (最終仕様)、`261123-backlog-batch-lightadr.md` (簡易 ADR)。
-->

# 残課題の一括対策 (WSL で進められる項目) 確認の記録

ファイル名: `261123-backlog-batch-validation.md`

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (WSL 2)。ホストは Windows |
| CPU | Intel Xeon Gold 6254 |
| Go | 1.27.3 (モジュールのキャッシュの道具) |
| Python | 3.12.15 |
| Git | 2.56.0 |
| CLI | Claude Code 2.1.330、Codex CLI 0.171.0、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.14 |
| 対象ブランチ | `feature/backlog-batch` |
| 実機の確認に使った実行ファイル | a から e と h は `7fef6ba` (BL-034 まで)、f と g と Copilot CLI の対話の画面は `fe59fbb` (BL-047 まで) からビルド |

Table: 環境

実機の確認のスクリプトと結果は、セッション用の一時フォルダーに置き、リポジトリには加えていません。確認のために各 CLI に登録した MCP サーバーは、確認の後に登録を解除しました。

## 1. 自動テスト

| 対象 | 方法 | 結果 |
| --- | --- | --- |
| すべてのパッケージ | `go vet ./...` (Linux、`GOOS=windows`、`GOOS=darwin`)、`go test -race ./...`。Linux x86_64 (WSL 2)、Go 1.27.3 | 検証済 |
| 検証の基盤 | `python3 -m unittest discover -s scripts/interactive/tests` (46 件)。Linux x86_64 (WSL 2)、Python 3.12.15。CI は ubuntu-latest だけで動かす | 検証済 |
| 3 つの OS | GitHub Actions の CI (ubuntu-latest、windows-latest、macos-latest)。`gh workflow run CI --ref feature/backlog-batch` で起動。run 45108671488 (BL-011 まで)、run 45111492974 (BL-047 まで)、run 45132690077 (文書とテストの直しまでの、最終のコミット) | 検証済 |

Table: 自動テスト

CI は、`main` への push と pull request でだけ自動で動きます。作業ブランチでは、手動で起動しました。

最終の 1 つ前の run 45132389274 は、ubuntu-latest で `TestReconnectAndUnknownOutcome` が 1 回失敗しました。テストの中のサーバーが、接続を 0.5 秒で閉じる設定で、張り直した接続が、同時の呼び出しの間にもう一度閉じられたためです。テストの設定を 2 秒に延ばし、手元で 3 回、CI で 1 回通ることを確かめました。製品のコードは変えていません。

課題ごとに足した主なテストは、次のとおりです。

- BL-046: `internal/profile/profile_test.go` (最終仕様の 2 章の表の全行、全角空白、タブ、制御文字、`HEAD` と `head` と `Head`、長さ)。
- BL-045: `internal/store/cleanup_test.go` (60 日の直前、ちょうど、直後。件数 0 の検索で作られた行。メモリのあるスコープ (固定、薄れた)。忘却が無効の構成。消した後の再アクセス)、`internal/store/users_test.go` (`user remove` での `user` のスコープの行の削除)。
- BL-014: `internal/names/names_test.go` (正規化の例、拒否する名前、これまでの名前が変わらないこと)、`internal/server/chat_test.go` (正規化した名前での参加と、全角のつづりと `user/name` の形の宛先)、`cmd/agent-lounge-mcp/admin_test.go` (`--`、正規化の前のつづり)、`internal/export/export_test.go` (ファイル名、長さの上限)。
- BL-011: `internal/store/admin_messages_test.go`、`cmd/agent-lounge-mcp/message_test.go` (一覧、`--details`、`--state`、`message show`、絞り込みの境界、続きの取得、使い方の誤り、絞り込みのない `export` が変わらないこと、絞り込んだ `export` のファイル)。
- BL-034: `internal/store/retention_settings_test.go`、`internal/tools/retention_test.go`、`cmd/agent-lounge-mcp/admin_test.go`、`cmd/agent-lounge-mcp/main_test.go` (項目ごとの引き継ぎ、明示した 0、3 つの種類の違い、起動し直しでの割引の引き継ぎ、版 7 から版 8 への移行、設定の記録と `memory list` の既定値と上書き、案内文と `lounge_info` の表示)。
- BL-047: `internal/loungecli/integration_test.go` (実際のサーバーを使う。参加者の分離、名前の省略のエラー、`name` の既定値、プロフィールの収集、閉じられた接続とサーバーの起動し直しの後の張り直しと参加のし直しと送り直し、結果が分からない切断、401、同時の呼び出し、`call` の中断、アイドル終了、残ったソケットの片付け、切り離した実行ファイルの起動と停止、オプションの位置、出力の形)、`internal/loungecli/args_test.go`、`internal/ownerdir/`。
- BL-035 の (17): `scripts/interactive/tests/test_harness.py` (画面の外の部分が変わらない書き直しでは再応答しないこと。既存の、同じコマンドが続く場合と、応答済みの画面が残る場合のテスト)。

## 2. 実機の確認 (WSL)

確認用のサーバーをループバックで起動し、確認用のデータベースと Git のリポジトリ (origin は `https://github.com/example/iv-backlog-batch.git`、ブランチは `Feature/Check` と `other`) を、一時フォルダーに置きました。相手役は、HTTP で直接ツールを呼びました。

### 2.1. a から d: MCP で登録した 5 つの CLI

各 CLI を、MCP サーバーを登録して、非対話で実行しました。シェルとツールの実行は、一括で許可しました (Claude Code は `--allowedTools` に agent-lounge のツールと `Bash`、Codex は `-s danger-full-access`、Copilot CLI は `--allow-all`、Grok は `--always-approve`、agy は `--dangerously-skip-permissions`)。指示は、「`lounge_help` を読んで従う」「名前 `担当-<CLI>２` (全角の数字) で参加する」「ブランチのルームに送る」「受信箱を待つ」です。プロフィールの調べ方と、ブランチの申告は、指示に含めていません。

| 項目 | 確かめたこと | Claude Code | Codex | Copilot CLI | Grok | agy |
| --- | --- | --- | --- | --- | --- | --- |
| a | 記録の `branch` が `feature/check` (ブランチ `Feature/Check` の小文字) | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| b | 同じブランチの相手役が受け取り、返信が届く。別のブランチの相手役の未読は 0 件 | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| c | `git switch other` の後の指示で、`lounge_join` を呼び直し、記録の `branch` が `other` になる | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| d | サーバーの記録の名前が `担当-<CLI>2` になる。相手役が全角のつづりを宛先にして送ったメッセージが届く | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |

Table: a から d の結果

5 つとも、1 回目の実行で通りました。

### 2.2. e: agy 1.5.14 での、閉じられた接続の後の動作 (BL-040)

対話の画面の agy (検証の基盤の部品を使う) を参加させ、別のセッションからメッセージを送りました。その後、(1) `--connection-idle 30s` のサーバーで 50 秒待つ、(2) サーバーを起動し直す、のそれぞれについて、受信箱を読んで合言葉を答える指示を 2 回まで送りました。各 3 回です。

| 場面 | 1 回目の指示 | 2 回目の指示 |
| --- | --- | --- |
| 閉じられた接続の後 | 3 回とも、ツールを呼べない (`unknown tool`) | 2 回は合言葉を答えた。1 回は呼べない |
| サーバーの起動し直しの後 | 3 回とも、ツールを呼べない (`unknown tool`) | 3 回とも合言葉を答えた |

Table: e の結果

1.5.13 までと同じで、直っていません。BL-040 は、確認した版を 1.5.14 に更新して、残課題に残します。

### 2.3. f: MCP を登録していない 5 つの CLI (BL-047)

このスクリプト (利用者の役) が常駐プロセスを `connection start --detach` で起動し (保存場所は既定の `/run/user/<UID>/agent-lounge-mcp/connections`)、各 CLI を非対話で実行しました。許可は、2.1 と同じです。指示は、「`agent-lounge-mcp tools` を実行して、その案内に従う」「参加者の名前は `cli-<CLI>`」「参加、リポジトリのルームへの送信、返信の待ち受け、メモリの保存と検索」です。コマンドの形は、指示に含めていません。

| 確かめたこと | Claude Code | Codex | Copilot CLI | Grok | agy |
| --- | --- | --- | --- | --- | --- |
| 参加し、プロフィール (ホストの Windows のユーザー名とコンピューター名、リポジトリの単位、ブランチ) が記録される | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| リポジトリのルームに送り、相手役の返信を受け取って報告する | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| メモリを保存し、検索して題名を報告する | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |

Table: f の結果

Claude Code、Copilot CLI、Grok、agy は、1 回目の実行で通りました。Codex は、次の 2 つの原因でやり直しました。どちらも、確認の手順の側の原因です。

- 最初の 3 回は、確認のスクリプトが、登録していない MCP サーバーの設定の上書き (`-c mcp_servers.agent-lounge...`) を付けていたため、Codex が設定の誤りで起動しませんでした。
- その次の実行は、Codex の側の操作はすべて成功しましたが、相手役が前の実行の未読に返信したため、返信が届きませんでした。相手役が古い未読を先に読み捨てるように直して、通りました。

Codex は、サンドボックスなし (`-s danger-full-access`) と、ネットワークを許可したサンドボックス (`-s workspace-write -c sandbox_workspace_write.network_access=true`) の両方で通りました。

### 2.4. g: Codex のサンドボックス (BL-047)

| 構成 | 結果 |
| --- | --- |
| `-s workspace-write` (ネットワークは既定の不許可)。利用者が起動した常駐プロセスへの `call` | 接続できない (終了コード 3)。保存場所を `/tmp` の下にしても同じ。サンドボックスの中の `connect()` が、Unix ドメイン ソケットに対しても `Operation not permitted` になる (Python で確認) |
| `-s workspace-write` (同上)。エージェントによる `connection start --detach` | 失敗する。既定の保存場所はログを書けず (`read-only file system`)、`/tmp` の下ではサーバーに接続できない |
| `-s workspace-write -c sandbox_workspace_write.network_access=true`。利用者が起動した常駐プロセスへの `call` | 使える (2.3 の確認が通った) |

Table: g の結果

Codex のサンドボックス (`workspace-write`) では、ネットワークを許可する設定にしたうえで、利用者が自分の端末で常駐プロセスを起動する必要があります。エージェントが `--detach` で起動した常駐プロセスが、コマンドの後に残るかは、起動そのものが失敗するため、確かめられませんでした。

### 2.5. h: 管理コマンド

確認用のデータベースで、実行ファイルの出力を確かめました。

| 確かめたこと | 結果 |
| --- | --- |
| `user add 本田`、`user add Ａlice` (全角の A。`alice` として登録される)、`user list` | 検証済 |
| 日本語のユーザー名と、全角の数字を含むセッション名 (`レビュー担当２`) での参加。宛先 `本田/レビュー担当２` への送信 | 検証済 |
| `message list` (件数の列)、`message list --details --state queued --user alice`、`message show 2` | 検証済 |
| `export --user 本田 --scope repository` (選ばれたファイルだけ。ファイル名に日本語が残る) と、絞り込みのない `export` | 検証済 |
| `serve --memory-branch-half-life 3d --memory-repository-grace 0` での `lounge_info` の種類ごとの値と、オプションなしの `memory list` | 検証済 |
| `message list --scope-id x` (`--scope` なし) が、使い方の誤り (終了コード 2) になる | 検証済 |

Table: h の結果

### 2.6. BL-047 の手動の確認

実行ファイルで、次を確かめました。

| 確かめたこと | 結果 |
| --- | --- |
| `connection start --detach`、`tools`、`call --as NAME lounge_join` (プロフィールが集められる)、`chat_send`、`chat_inbox`、`connection status`、`connection stop` | 検証済 |
| `--connection-idle 20s` のサーバーで 25 秒待った後の `call chat_inbox` が、張り直し、参加のし直し、送り直しを経て、届いていたメッセージを返す | 検証済 |
| サーバーが止まっている間の `call` が終了コード 3 になり、起動し直した後の `call chat_send` が、参加し直して成功する | 検証済 |
| 未知のツールと未知のキーが終了コード 2、ツールのエラーが終了コード 1 になる。`--json -` で標準入力から引数を渡せる | 検証済 |
| トークンが必須のサーバーに、トークンなしでは起動できない。`--token-stdin` で起動でき、トークンが保存場所のファイルとプロセスの引数にない。`user rotate` の後の `call` が、終了コード 3 でトークンが無効であることを伝える | 検証済 |
| 保存場所のパスが長い場合に、`AGENT_LOUNGE_CONNECTION_DIR` を短くするよう案内するエラーになる | 検証済 |

Table: 手動の確認の結果

### 2.7. Copilot CLI の対話の画面 (BL-035 の (17))

再応答の条件を変えた検証の基盤で、Copilot CLI 1.0.114 のシナリオ 2 (`several-instructions`) と 5 (`instruction-files`) を実行しました (`verify.py run --agents copilot --scenarios 2,5`)。

| 確かめたこと | 結果 |
| --- | --- |
| シナリオ 2 が `done` になる (許可の画面は出なかった) | 検証済 |
| シナリオ 5 が `done` になる。5 回の作業のすべてで、参加し、メモを作り、途中のメッセージを読む。応答した許可の画面は、合計 6 | 検証済 |
| 同じコマンドの許可が続く場面 (次の要求を、画面の外の部分の変化で見分ける場面) が、実機で起きたか | 未検証 (記録からは判別できない。この場面は、自動テストの模擬の CLI で確かめている) |

Table: Copilot CLI の対話の画面の結果

最終仕様の 8 章は、実機で次の要求を見分けられない場合に作業を止めるとしています。実機の実行で、応答の取りこぼしによる時間切れや、作業の未完了は起きなかったため、作業を続けました。同じコマンドの許可が続く場面を実機で起こして確かめることは、残課題 (BL-035) に残します。

## 3. 未検証の項目

| 項目 | 理由 |
| --- | --- |
| BL-047 の Windows での実機の動作 (常駐プロセス、保存場所の権限、プロフィールの収集、切り離した起動) | この環境 (WSL) では確かめられない。CI の自動テストは通っている |
| BL-014 の Windows の側の CLI での、日本語の名前の受け渡し | 同上 |
| BL-044 の Windows の側 (PowerShell での申告、大文字と小文字、WSL との往復) | 同上 |
| 絞り込んだ `export` の、Windows での日本語のファイル名 | 実機では確かめていない。CI の自動テストは通っている |

Table: 未検証の項目

macOS の実機の確認は、行いません (機材がないため。利用者の恒久の決定、2026-11-23。簡易 ADR の LADR-012)。macOS は、自動テスト (CI の macos-latest) と机上の確認だけで扱い、未検証の項目には数えません。

## 4. 点検で直した点

実装の点検と確認の中で、次の点を直しました。

| 課題 | 直した点 |
| --- | --- |
| BL-035 の (17) | 静止の判定を、すべての出力ではなく、英字と数字の出力に戻した。再応答の条件に、応答の後に英字か数字の出力があることを加えた (記号だけの作業中の表示が画面の外の部分を変える、既存のテストが失敗したため)。最終仕様の 8 章に反映した |
| BL-014 | 空の `name` を、省略と同じ扱い (サーバーが名前を決める) に戻した。宛先のエラーの文に、宛先の形と `lounge_who` の案内を戻した。書き出しのファイル名に、120 バイトの上限を足した (64 文字の日本語の名前が、ファイル名の長さの上限を超えるため)。最終仕様の 4.3 に反映した |
| BL-047 | `call` の出力を、ツールの構造化された結果 (テキストだけのツールは、そのテキスト) にした。オプションを、ツール名の後にも書けるようにした。`tools` の案内文を書き直した。最終仕様の 7 章に反映した |

Table: 点検で直した点
