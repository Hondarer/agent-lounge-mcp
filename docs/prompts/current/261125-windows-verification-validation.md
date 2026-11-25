# Windows の側のセッションでの残課題の確認 確認の記録

ファイル名: `261125-windows-verification-validation.md`

案件 `windows-verification` の、実機の確認と自動テストの記録です。最終仕様は `261125-windows-verification-design.md` です。Windows の側の結果は、Windows 担当 (Windows の側の Claude Code のセッション) の報告によります。

## 1. 環境

| 項目 | WSL の側 | Windows の側 |
| --- | --- | --- |
| OS | Linux x86_64 (WSL 2) | Windows 11 Pro for Workstations 10.0.26300、PowerShell 7.7.0 |
| 実行ファイル | `linux-amd64` (作業ブランチの `1bbfcfb` からビルド) | `windows-amd64` (Windows 担当が `1bbfcfb` から `go build`。Go 1.27.4) |
| CLI | Claude Code 2.1.331、Codex 0.171.0 (GPT-6.1-Sol)、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.17 (確認の時点で 1.5.16 から上がっていた) | Claude Code 2.1.330、Codex 0.171.0 (GPT-6.1-Sol)、Copilot CLI 1.0.114 (GPT-6.1 Terra)、Grok 1.0.92、agy 1.5.17 (Gemini 4 Hydrogen) |
| 確認用のデータ | セッションの一時フォルダー | `%TEMP%\alm-wv` |

Table: 確認の環境

Grok は、両方の側で利用の上限 (`402 Payment Required`、`Grok Build usage balance exhausted`) のため、どの確認も実行できませんでした (簡易 ADR の LADR-004 により未検証)。

## 2. Windows の側の確認

### 2.1. 日本語の名前とブランチの申告 (BL-014 (1)、BL-044 d・e・f)

WSL の側で `--allow-remote --listen 0.0.0.0:7901` のサーバー (S1) を起動し、ユーザー `確認者２` (`確認者2` に正規化) のトークンを Windows 担当へ渡しました。Windows 担当は、origin `https://github.com/hondarer/agent-lounge-mcp.git`、ブランチ `verify/wv-branch` のリポジトリを作り、各 CLI を、WSL のアドレス (`172.28.209.184:7901`) に、トークンを付けて登録しました。

指示は「agent-lounge のツールを使い、使い方の案内 (lounge_help) に従って、`担当-<CLI>２` という名前で参加してください。参加の結果の name、user、repository、branch をそのまま答えてください。」です。

| CLI | 許可 | name | user | repository | branch | 結果 |
| --- | --- | --- | --- | --- | --- | --- |
| Claude Code 2.1.330 | `--allowedTools` に agent-lounge のツールと、プロフィールを調べる PowerShell と Bash のコマンド | `担当-claude2` | `確認者2` | `github.com/hondarer/agent-lounge-mcp` | `verify/wv-branch` | 検証済 |
| Codex 0.171.0 | `-s read-only` と agent-lounge のツールの承認 | `担当-codex2` | `確認者2` | 空 | 空 | 名前は検証済。申告は制限あり |
| Copilot CLI 1.0.114 | `--allow-tool agent-lounge` と、`shell(git remote)` などのサブコマンドの単位の許可 | `担当-copilot2` | `確認者2` | `github.com/hondarer/agent-lounge-mcp` | `verify/wv-branch` | 検証済 |
| Grok 1.0.92 | - | - | - | - | - | 未検証 (利用の上限) |
| agy 1.5.17 | 設定のファイルで、agent-lounge のツールとプロフィールを調べるコマンド | `担当-agy2` | `確認者2` | `github.com/hondarer/agent-lounge-mcp` | `verify/wv-branch` | 検証済 |

Table: Windows の側の申告

- Codex は、サンドボックスの中で `pwsh.exe` を起動できず、5 つのコマンドがすべて `CreateProcessAsUserW failed: 5` になりました。前の確認 (Codex 0.170.0) と違い、`cmd.exe` に切り替えずに、空の値で参加しました。
- Copilot CLI の 1 回目は、許可を `shell(git remote get-url origin)` と引数まで書いたため、git が拒否されました。許可はサブコマンドの単位で判定されるため、書き方を直して実行し直しました。
- **f (往復)**: Windows の Claude Code が、ブランチのスコープの `lounge` に `ping from windows` を送りました。WSL の側の相手役 (CLI の方式。同じ origin とブランチ) が受け取り、全角のつづり `確認者２/担当-claude２` を宛先にして返信しました (recipients は `確認者2/担当-claude2`)。Claude Code は、1 回目の `chat_inbox` で返信を受け取りました。検証済です。ping の受信者に Codex がいなかったのは、ブランチが空のためです。
- **e (大文字と小文字)**: ブランチ `main` のあるリポジトリで `git switch Main` は成功し (終了コード 0)、`git branch --show-current` は `Main` を出しました (`.git\HEAD` は `ref: refs/heads/Main`、参照のファイルは `refs\heads\main` だけ)。Claude Code が参加し直すと、参加の結果とサーバーの記録 (`session list` の `BRANCH`) は `main` でした。検証済です。

### 2.2. 日本語のファイル名の書き出し (BL-014 (2))

Windows の側でループバックのサーバー (S2、`127.0.0.1:7903`) を起動し、`user add -- 確認者２` (`確認者2` で登録)、CLI の方式での参加 (`担当-窓２`、記録は `担当-窓2`)、`shared` の `lounge` への送信とメモリの保存を行いました。

- `export --user 確認者2 --scope shared`: `chat\shared\room-lounge.md` と `memory\shared.md` (`shared` のスコープのファイル名には日本語が入らない)。
- `export --user 確認者2 --scope user --scope-id 確認者2`: `chat\user-確認者2\direct-確認者2_担当-窓2--確認者2_担当-窓d-df376fad.md` (365 バイト)。`Get-ChildItem -Recurse` で日本語の名前のとおりに表示され、`Get-Content -LiteralPath` で開けました。

検証済です。

### 2.3. MCP を使わない CLI の方式 (BL-047 (1))

S2 に対して、PowerShell から実行しました。

| 項目 | 結果 |
| --- | --- |
| 前面の `connection start --token-stdin` と、別の PowerShell からの `connection start --detach --name win-d` | どちらも起動。`--detach` は終了コード 0 で状態の JSON を出した |
| `connection status`、`connection list` | 2 つとも name、url、started、pid を表示 |
| 保存場所の権限 (`icacls`) | `Z8G4-BASE\tetsuo:(OI)(CI)(F)` の 1 行だけ。`AreAccessRulesProtected` は True (継承を切っている)。中のファイルは本人だけ |
| `tools`、`call lounge_help` | 先頭は `agent-lounge from a shell (no MCP server registered)`、`# agent-lounge usage guide` |
| `call --as 担当-窓２ lounge_join` | `os_user`、`machine`、`folder`、`origin`、`branch` が集められ、名前とユーザーは正規化された |
| `chat_send`、`chat_inbox` | `win-d` の `担当-窓d` から `担当-窓２` への送信を受け取れた |
| `--detach` の常駐プロセスの残存 | 起動した PowerShell が終わった後も残った |
| `connection stop` | 2 つとも終了コード 0 で終わり、`.sock` と `.json` が片付いた (`.lock` と `.log` は残る) |

Table: Windows の CLI の方式

検証済です。自分宛ての送信は、仕様どおり拒否されました。

### 2.4. Codex の作業中の表示 (BL-042)

長い英文を書かせる指示で、Codex の対話の画面を 1 秒ごとに保存しました (WSL の側でも同じ確認をした)。

| 環境 | 応答の時間 | 保存した画面 | `esc to interrupt` がある画面 | 完了の後の画面 |
| --- | --- | --- | --- | --- |
| WSL | 71.5[秒] | 72 枚 | 2 枚 (2 秒、3 秒) | `Worked for 1m 6s • 7:42 PM` |
| Windows | 67.8[秒] | 68 枚 | 2 枚 (5 秒、6 秒) | `Worked for 1m 0s • 19:44` |

Table: Codex の画面の記録

作業中だけ出る文字列は、どちらにもありませんでした。最終仕様の 7 章に従って作業を止め、利用者の回答 (検討用 Markdown の Q-006 の A) で、検証の基盤に完了の後だけ出る文字列 (`done`) を足しました (`c4613ad`)。

### 2.5. agy の閉じられた接続の後 (BL-040)

前の確認と同じスクリプトで、`--connection-idle 30s` のサーバーで 50 秒待つ場合 (idle) と、サーバーを起動し直す場合 (restart) を、各 3 回確かめました。Windows の側は、ポート 7793 が WSL の側の確認と重なったため (WSL の localhost の転送)、`127.0.0.1:7811` で実行しました。

| 環境 | 1 回目の指示 | 2 回目の指示で合言葉を答えた回数 |
| --- | --- | --- |
| WSL (agy 1.5.17) | 6 回とも `unknown tool` で呼べない | 6 回中 4 回 (idle 2 回、restart 2 回) |
| Windows (agy 1.5.17) | 6 回とも `unknown tool` で呼べない | 6 回中 5 回 (idle 2 回、restart 3 回) |

Table: agy の結果

直っていません。スクリプトの最後の `chat_status` は、今の版で `message_id` が必須のためエラーになり、送達状態は取れていません (結果の判定には影響しない)。`ask*_inbox_calls` は、合言葉を答えた場合も 0 で、数え方が働いていません (判定は画面の合言葉で行った)。

## 3. WSL の側の確認

### 3.1. `shared` とルーム (BL-048)

ループバックの `--require-token` のサーバー (S4) に、ユーザー `alice` (確かめる CLI) と `bob` (相手役。CLI の方式) を作りました。`alice` は origin `https://github.com/example/wv-shared-a.git`、`bob` は `...-b.git` のリポジトリです。相手役は、事前に `shared` のルーム `plan-<CLI>` を作り、参加の前のメッセージを投稿しました。指示は 3 回に分け、スコープの値とツール名は含めていません (「全ユーザーで共有するスコープ」と書いた)。

| 指示 | 確かめたこと | Codex 0.171.0 | Copilot CLI 1.0.114 | agy 1.5.17 |
| --- | --- | --- | --- | --- |
| 1 | `bob/bob-p` へのセッション宛て (recipients `["bob/bob-p"]`)、`Release-<CLI>` の作成 (`release-<CLI>`、`created`、`joined`)、参加者のいない投稿 (recipients `[]`) | 通過 | 通過 | 通過 |
| 2 | 相手役の `shared` の `lounge` への投稿の受信、相手役の参加の後の投稿 (recipients `["bob/bob-p"]`) | 通過 | 通過 | 通過 |
| 3 | 相手役の退出の後の投稿 (recipients `[]`)、参加していない `plan-<CLI>` の読み返し (合言葉の本文)、`shared` のメモリの保存と、相手役の「ビルド」での検索 | 通過 | 通過 | 通過 |

Table: `shared` とルームの結果

すべて 1 回目の実行で通りました。許可は、Codex は `-s read-only` と agent-lounge のツールの承認、Copilot CLI は `--allow-tool agent-lounge` と `shell(git:*)`、agy は agent-lounge のツールだけの設定のファイルです。Copilot CLI は、WSL のホストのユーザー名とコンピューター名を調べるコマンドが許可されず、`os_user` と `machine` を空にして参加しました (この確認の対象外)。

### 3.2. Codex のサンドボックスの `--detach` (BL-047 (2))

Codex を `-s workspace-write -c sandbox_workspace_write.network_access=true` で実行し、`connection start --detach` と `call --as codex-sb lounge_join` を、それぞれ 1 回のシェルの呼び出しで実行させました。

- `connection start --detach` は終了コード 0 で、状態の JSON (サンドボックスの中の `pid` 13) を出しました。
- 次のシェルの呼び出しの `call` は、`cannot connect to daemon "cx"` (終了コード 3) でした。
- Codex の終了の後、`connection list` に `cx` はなく、プロセスもありませんでした。ソケットなどのファイルは残り、次の `connection start` で片付きました。

常駐プロセスは、起動したシェルの呼び出しの終わりで残りません。制限ありとして、文書に書きました。

## 4. 検証の基盤の直し (BL-042)

- `scripts/interactive/harness/agents.py`: `Definition.done` を足し、Codex に `Worked for` を設定しました。`Agent.ask` は、`done` がある CLI では、指示の後の出力の、最後の完了の語より後に、その文字列が出たことを完了の条件に加えます。実装は Sonnet のサブエージェントが書き、Claude Code が点検しました。点検で、完了の語と `Worked for` が同じ出力の塊で届く場合と、画面の書き直しで前の指示の行が後から書かれる場合に備え、判定の位置を「最後の完了の語の後」に直しました。
- 自己試験: `scripts/interactive/tests/test_harness.py` に 4 件 (`done` を待つこと、`done` がないと早く完了すること、`done` が出なければ時間切れで未完了、前の指示の `done` を数えないこと) を足しました。

| 項目 | 結果 | 環境 |
| --- | --- | --- |
| 自己試験 (`python3 -m unittest discover -s scripts/interactive/tests`) | 50 件が通過 | Linux x86_64 (WSL 2)、Python 3.12 |
| 自己試験 (同じ。Windows 担当) | 49 件が通過、1 件は省略 (`PWD` の試験)。`ResourceWarning` の行は 0 件 | Windows 10.0.26300、Python 3.14.3、`c4613ad` |
| Codex の長い応答 (2.4 と同じ指示) | 完了と判定 (73.5 秒)。`Worked for 1m 8s • 8:11 PM` の後。`esc to interrupt` は 74 枚中 3 枚 (2 秒から 4 秒) | Linux x86_64 (WSL 2)、Codex 0.172.0 |
| Codex のシナリオ 2 と 3 (`verify.py run --agents codex --scenarios 2,3`。Windows 担当) | どちらも `done` (シナリオ 3 は 48.5 秒、シナリオ 2 の実行し直しは 39.9 秒) | Windows 10.0.26300、Codex 0.172.0 |

Table: 基盤の自己試験

基盤の直しの確認の中で、次のことが分かりました。

- Codex は、起動したときに新しい版があると、更新の案内を出します。基盤がこれに答えたとみられ、Codex が自分を更新しました (Windows は `npm install -g @openai/codex` で 0.171.0 から 0.172.0。WSL は公式のインストールのスクリプトで同じ版)。更新の間の指示は応答されず、Windows のシナリオ 2 の 1 回目は `unverified` (`the agent did not call lounge_join`)、WSL の長い応答の 1 回目は未完了になりました。どちらも、更新の後に実行し直して通りました。残課題に記録しました (BL-052)。

## 5. 後片付け

- WSL の側: 確認用のサーバー (S1、S4) と常駐プロセスを止めました。5 つの CLI の `mcp list` に agent-lounge はありません。確認用のデータは、セッションの一時フォルダーにあります。
- Windows の側 (Windows 担当の報告): 5 つの CLI の `mcp list` に agent-lounge はなく、`alm.exe` と agy のプロセスは残っていません。確認用のデータは `%TEMP%\alm-wv` にあります。
- 確認の途中で、Codex は両方の側で 0.172.0 に、agy は WSL の側で 1.5.17 に上がりました。
