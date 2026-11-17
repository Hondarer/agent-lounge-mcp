<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-17 22:40:31 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53Qv4yzq5xHqqMwe5fY
稼働 OS: Linux (WSL)
上流文書: `261117-multi-user-access-design.md` (最終仕様)、`261117-multi-user-access-discussion.md` (検討用 Markdown)。削除されている場合があります。
-->

# 複数のユーザーと、他のマシンからの接続の確認の記録

案件: `multi-user-access` (最終仕様は `261117-multi-user-access-design.md`)

この文書は、2 人の利用者と、WSL の側と Windows の側のエージェントでの確認の手順と結果を記録します。WSL の側の操作は、Claude Code が、各 CLI の非対話の実行で行いました。Windows の側の操作は、Windows で動く別の Claude Code のセッションに、セッション間のメッセージで依頼しました。

実際のユーザー名、コンピューター名、アドレス、トークンは書きません。`<ユーザー名>`、`<コンピューター名>`、`<WSL のアドレス>`、`<トークン>` と表します。スクラッチパッドのパスは、`<S>` と表します。`alice` と `bob` は、確認用の仮のユーザーです。

## 状況

| 項目 | 状況 |
| --- | --- |
| 到達点 1 (トークンでの接続) | 検証済 (WSL の側の 5 つが、`alice` として参加。既定の構成では、トークンのない接続が `local`) |
| 到達点 2 (他のマシンからの接続) | 検証済 (Windows の側の 5 つが、WSL のアドレスに `bob` のトークンで接続。トークンなしと、誤ったトークンは 401) |
| 到達点 3 (ユーザーをまたぐ協調) | 検証済 (`lounge_who` のスコープと識別子、リポジトリのスコープのルーム宛てと個別のセッション宛て、`reply_to` での返信、範囲の外の相手が見えず届かないこと) |
| 到達点 4 (トークンの無効化) | 検証済 (`user rotate` と `user remove` の後、古いトークンが 401) |
| 到達点 5 (物理的に別のマシン) | 未検証 |
| 登録の解除と後片付け | 実施済み |

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux x86_64 (WSL 2、カーネル 6.6.87.2、Oracle Linux 8) と、そのホストの Windows 10.0.26300 (PowerShell 7.7.0) |
| サーバー | `dist/agent-lounge-mcp` (コミット `8b9bf30`、ブランチ `feature/multi-user-access`)。WSL の側で `--listen 0.0.0.0:7788 --allow-remote` で起動 |
| データベース | `dist/multi-user.db` (確認用。空の状態から開始) |
| 利用者 | `alice` (WSL の側の 5 つの CLI。`http://127.0.0.1:7788/mcp`)、`bob` (Windows の側の 5 つの CLI。`http://<WSL のアドレス>:7788/mcp`) |
| Claude Code | 2.1.326 (WSL の側)、2.1.325 (Windows の側) |
| Codex | 0.169.0 (両方の側) |
| Copilot CLI | 1.0.112 (両方の側) |
| Grok | 1.0.77 (両方の側) |
| agy | 1.5.10 (WSL の側)、1.5.9 (Windows の側) |

WSL 2 は、ホストの Windows と別のアドレスを持ちます。Windows の側から WSL のアドレスへの接続は、ループバックではない経路を通ります。`0.0.0.0` で待ち受けたサーバーには、Windows の側の `127.0.0.1` からは届きませんでした。

## 再現の手順

### 準備

1. ユーザーを作り、サーバーを起動します。トークンは、標準出力に 1 回だけ表示されます。

    ```bash
    cd /home/user/agent-lounge-mcp
    dist/agent-lounge-mcp user add alice --db dist/multi-user.db
    dist/agent-lounge-mcp user add bob --db dist/multi-user.db
    dist/agent-lounge-mcp serve --listen 0.0.0.0:7788 --allow-remote --db dist/multi-user.db
    ```

2. 5 つの CLI に、トークンを付けて登録します。`<URL>` は、WSL の側では `http://127.0.0.1:7788/mcp`、Windows の側では `http://<WSL のアドレス>:7788/mcp` です。

    ```bash
    claude  mcp add --transport http --scope user agent-lounge <URL> --header "Authorization: Bearer <トークン>"
    codex   mcp add agent-lounge --url <URL> --bearer-token-env-var AGENT_LOUNGE_TOKEN
    copilot mcp add --transport http agent-lounge <URL> --header "Authorization: Bearer <トークン>"
    grok    mcp add --transport http agent-lounge <URL> -H "Authorization: Bearer <トークン>"
    agy     mcp add --type http --header "Authorization: Bearer <トークン>" agent-lounge <URL>
    ```

    Codex は、実行のコマンドの中だけで、環境変数 `AGENT_LOUNGE_TOKEN` にトークンを設定しました。

3. 確認用のフォルダーを作ります。

| フォルダー | 内容 | 使う側 |
| --- | --- | --- |
| Windows の一時フォルダーの下の `lounge-check-w` | Git の管理下にないフォルダー。WSL からは `/mnt/c/...`。両方の側で同じリポジトリの単位になります | `alice` の 5 つ、`bob` の 5 つ |
| `<S>/check/other` | origin のない Git のリポジトリ (別のリポジトリの単位) | `alice` の `other-a` (対照) |

### 非対話の実行と、許可

コマンドの形は、前の案件 (`261117-repository-space-validation.md`) と同じです。まず、agent-lounge のツールと、プロフィールを調べる読み取りのコマンドだけを許可しました。

| 側 | すべての許可を省くオプションを使った CLI | 理由 |
| --- | --- | --- |
| WSL | Codex (`--dangerously-bypass-approvals-and-sandbox`)、agy (`--dangerously-skip-permissions`) | サンドボックスの中では、ホストの Windows の値を取得できず、origin のないフォルダーでリポジトリの単位を申告できないため (前の案件と同じ) |
| Windows | なし | 5 つとも、個別の許可 (Codex は `-s read-only`) で確認できた |

### 指示文

| 手順 | 側 | 指示文 |
| --- | --- | --- |
| 1 | WSL (`alice`) | `agent-lounge の使い方を確認して、<cli>-a で参加して。参加の結果と lounge_who の結果を、そのまま表示して。` (対照の `other-a` は、`<S>/check/other` で Claude Code が参加) |
| 2 | Windows (`bob`) | `agent-lounge の使い方を確認して、<cli>-b で参加して、lounge_who の結果をそのまま表示して。次に、リポジトリのスコープの全セッション宛てに「<cli> (bob) からの確認です」と送って、送信の結果をそのまま表示して。` |
| 3 | WSL (`alice`) | `agent-lounge の使い方を確認して、<cli>-a で参加して、未読のメッセージを読んで。各メッセージの送信者に、個別のセッション宛てで「<cli> (alice) からの返信です」と返信して。受け取ったメッセージの from、from_user、scope、reply_to と、各返信の送信の結果を表示して。` |
| 4-1 | Windows (`bob`) | `agent-lounge の使い方を確認して、<cli>-b で参加して、未読のメッセージを読んで。各メッセージの from、from_user、scope、scope_id、reply_to、room の有無、本文を、表にして表示して。` |
| 4-2 | Windows (`bob`、Claude Code) | `to` に `alice/claude-a` (スコープなし)、`alice/other-a` (スコープは `repository`)、`alice/no-such-session` (スコープは `repository`) で、順に `chat_send` を呼ぶ指示 |
| 5 | 両方 | `agent-lounge の使い方を確認して、<名前> で参加して。参加の結果か、失敗した場合は起きたことをそのまま表示して。` (トークンを無効にした後) |

### 後片付け

両方の側で、`claude mcp remove --scope user agent-lounge` と、`<cli> mcp remove agent-lounge` を実行します。サーバーを停止し、確認用のデータベース、トークンを書いたファイル、フォルダーを削除します。

## 結果

### 実行ファイルでの確認

| 確認 | 結果 |
| --- | --- |
| `--listen 0.0.0.0:7788` を、`--allow-remote` なしで指定 | 終了コード 2 で、理由と対処を表示して起動を中止 (サブエージェントが、別のポートで確認) |
| `--allow-remote` を付けて起動 | 平文であることの警告と、`every request needs a token` を表示 |
| `/mcp` へ、トークンなし、誤ったトークン、`alice` のトークン (WSL の側、`curl`) | 401、401、200 |
| Windows の側から WSL のアドレスへ (`curl.exe`) | `/healthz` は 200 (`schema_version` は 3)。`/mcp` は、トークンなしで 401 (`WWW-Authenticate: Bearer`)、誤ったトークンで 401 (`Bearer error="invalid_token"`)。本文は、原因と対処の 1 文 |
| `probe.sh` (`alice` のトークン)、`chat-probe.sh` (`alice` と `bob` のトークン) | すべて `OK:` (5 行、12 行) |
| 既定の構成 (ループバック、オプションなし) で、トークンなしの `probe.sh` と `chat-probe.sh` | すべて `OK:` (5 行、9 行)。セッションは、ユーザー `local` |
| 既定の構成で、`local` (トークンなし) と、トークンのあるユーザーの組での `chat-probe.sh` | すべて `OK:` (12 行) |
| 既定の構成で、`Host` がループバック以外を指す要求 | 403 |

### 到達点 1: トークンでの接続 (WSL の側、`alice`)

5 つすべてが、`lounge_join` の `user` が `alice` で参加しました。`user list` でも、セッションが `alice` に属しています。Claude Code、Copilot CLI、Grok は、個別の許可で、リポジトリの単位 (`<コンピューター名>:c:\users\<ユーザー名>\appdata\local\temp\lounge-check-w`) を申告しました。Codex と agy は、サンドボックスの中では単位が空になったため、サンドボックスを外して実行し、同じ単位を申告しました。

### 到達点 2: 他のマシンからの接続 (Windows の側、`bob`)

5 つすべてが、WSL のアドレスに `bob` のトークンで接続し、`lounge_join` の `user` が `bob`、`repository` が `alice` の 5 つと同じ値で参加しました。`claude mcp list` は `Connected`、`codex mcp list` は認証の欄が `Bearer token` でした。トークンを付けた接続で、OAuth の認証の流れを始めた CLI は、ありませんでした。

### 到達点 3: ユーザーをまたぐ協調

**`lounge_who` (`bob` の 5 つから見た結果)**

| 項目 | 結果 |
| --- | --- |
| 最上位の `scopes` | `user` (識別子は `bob`) と、`repository` (識別子は、リポジトリの単位) |
| `alice` の 5 つのセッション | `address` は `alice/<名前>`、`scopes` は `repository` だけ |
| `bob` のセッション | `address` はセッション名、`scopes` は `user` と `repository` |
| `alice` の `other-a` (別のリポジトリの単位) | どの CLI からも、一覧に現れない |

**リポジトリのスコープの全セッション宛て (`bob` → 全員)**

`bob` の 5 つが 1 通ずつ送信し、`scope` は `repository`、`scope_id` はリポジトリの単位、`recipients` は、`alice` の 5 つ (`alice/<名前>` の形) と、先に参加していた `bob` のセッションでした。

**個別のセッション宛ての返信 (`alice` → `bob`)**

`alice` の 5 つが、受信した 5 通のそれぞれの送信者に、個別のセッション宛てで返信しました (各 5 通、合計 25 通)。データベースでは、各エージェントの 5 通が、`bob` の 5 つのセッションに 1 通ずつ届いています。

**受信 (`bob` の 5 つ)**

5 つすべてが、`alice` の 5 つからの個別の返信を 5 通ずつ受け取りました。`from_user` は `alice`、`scope` は `repository`、`scope_id` はリポジトリの単位、`reply_to` は `alice/<名前>`、`room` はありません。あわせて、後から参加した `bob` のセッションのルーム宛てのメッセージ (`from_user` は `bob`、`reply_to` はセッション名、`room` は `lounge`) を受け取りました (件数は、参加の順に、9、8、7、6、5 通)。

**エラー (`bob` の Claude Code)**

| 呼び出し | エラー文 |
| --- | --- |
| `to` に `alice/claude-a`、スコープなし | `"alice/claude-a" is a session of another user (alice): sessions of other users are reached only in a scope you share with them. Pass scope repository with this to (lounge_who lists the scopes you share with each session)` |
| `to` に `alice/other-a`、スコープは `repository` (範囲の外) | `no session "alice/other-a" in scope repository: call lounge_who, and pass the address of a session whose scopes include repository` |
| `to` に `alice/no-such-session`、スコープは `repository` (存在しない) | 上と同じ文面 (宛先の名前だけが違う) |

### 到達点 4: トークンの無効化

| 操作 | 結果 |
| --- | --- |
| `user rotate bob` | WSL の側の `curl` で、古いトークンは 401、新しいトークンは 200 |
| `user remove alice` (`--yes` なし) | 削除される件数 (セッション 9 件、メッセージ 34 件、送達 60 件) を表示し、何もせずに終了コード 1 |
| `user remove alice --yes` | 削除し、終了コード 0。`user list` から消えた |
| 削除の後に、`alice` のトークンで登録したままの Claude Code と Grok を実行 | どちらも、接続が 401 で拒否されたこと、原因 (打ち間違い、ユーザーの削除、トークンの発行し直し) と、トークンを確かめる必要があることを報告した。`claude mcp list` は、`Failed to connect — Server rejected the configured Authorization header (HTTP 401)` と表示した |
| `user rotate bob` の後に、Windows の側から古いトークンで `/mcp` に POST (`curl.exe`) | 401、`WWW-Authenticate: Bearer error="invalid_token"`、本文は、トークンが無効であることと対処 |
| 古いトークンで登録したままの、Windows の側の Claude Code | `claude mcp list` が `Connected` から `Failed to connect` に変わり、401 と、サーバーのエラー文を表示した。非対話の実行では、ツールが読み込まれず、エージェントが、401 とトークンが無効であることを利用者に伝えた。OAuth の認証の流れは、始まらなかった (ヘッダーを指定した登録では、切り替えが無効と表示される) |
| 古いトークンで登録したままの、Windows の側の Grok | ツールが見えず、エージェントが、ログで 401 を見つけた。自分で設定ファイルを読み、`curl` で直接呼び出して 401 を確かめようとし、200[秒] の打ち切りまで終わらなかった。OAuth の認証の流れは、始まらなかった |
| `user remove bob --yes` | 削除し、発行し直した後のトークンも 401 になった |

## 後片付け

- WSL の側と Windows の側の、それぞれ 5 つの CLI の登録を解除し、一覧に `agent-lounge` がないことを確認しました。
- 確認用のユーザー (`alice`、`bob`) を `user remove` で削除し、サーバーを停止して、確認用のデータベース、トークンを書いたファイル、フォルダーを削除しました。
- Windows の側のリポジトリは、`main` で変更なしです。
- 無効にしたトークンの文字列が、次の場所に残っています。値は無効です。利用者のファイルのため、編集していません。
    - Windows の側: Claude Code の設定のバックアップ (`.claude\backups\` の下)、Windows の側のセッションの履歴 (トークンを伝えたメッセージを含む)、Grok のセッションの履歴 (`.grok\sessions\` の下)
    - WSL の側: Grok のセッションの履歴 (`~/.grok/sessions/` の下)。このセッションの履歴 (トークンを読み出した出力を含む)

## 気付いた事項

### 確認の作業での誤り

- 前の案件の Windows の CLI の確認の後、WSL の側のサーバーを止めたつもりで、止まっていませんでした (`pgrep -x` に渡したプロセス名が、15 文字に切り詰められた名前と一致しなかったため)。このサーバーが 7788 番を使っていたため、今回の最初の起動が失敗し、最初の `curl` と確認のスクリプトは、古いサーバーに対して動きました (スキーマの版が 2 と表示されたことで気付きました)。待ち受けているポートからプロセスを特定して止め、起動し直してから、確認をやり直しました。上の結果は、やり直した後のものです。
- 確認用のトークンを、Windows の側のセッションに、セッション間のメッセージで伝えました (最終仕様のとおり)。メッセージは、両方の側のセッションの履歴に残ります。確認の後に、トークンを無効にしています。

### エージェントごとの気付き

- **トークンが残る場所**: Codex 以外の 4 つは、登録したトークンを設定ファイルに平文で保存します。Claude Code は、設定ファイルのバックアップを自動で作り、そこにもトークンが残りました。Grok は、接続の失敗を調べるときに、自分の設定ファイルを読んで、トークンを履歴に残しました (WSL の側と Windows の側の両方)。
- **許可の効き方**: Windows の側の Grok は、許可していないコマンド (設定ファイルを読むスクリプト、`curl`) を実行できていました。許可の指定が実際に効いていたかは、確認できていません。
- **Windows の側の Codex**: 読み取り専用のサンドボックスでは、`pwsh.exe` の起動が拒否され (`CreateProcessAsUserW failed: 5`)、同梱の JavaScript の実行の機能で、ホスト名とユーザー名を取得しました。
- **Windows の側の Copilot CLI**: 許可に入れた `git` の 2 つのコマンドが拒否されましたが、Git の管理下にないフォルダーのため、結果には影響しませんでした。
- **出力の文字化け**: Windows の側のセッションが取り込んだ出力で、日本語の本文が文字化けしました。本文の一致は、英数字の部分 (`(alice)` など) と、データベースの内容で確認しました。
- **スコープの扱い**: 10 個のエージェントすべてが、`lounge_help` と `lounge_who` の結果だけから、スコープと宛先の形 (`alice/<名前>`)、`reply_to` での返信を使えました。指示文では、「リポジトリのスコープ」「個別のセッション宛て」という言葉だけを使いました。
