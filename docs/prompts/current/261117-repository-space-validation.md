<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-17 19:17:52 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53Qv4yzq5xHqqMwe5fY
稼働 OS: Linux (WSL)
上流文書: `261117-repository-space-design.md` (最終仕様)、`261117-repository-space-discussion.md` (検討用 Markdown)。削除されている場合があります。
-->

# 参加時のプロフィールと、リポジトリの空間の確認の記録

案件: `repository-space` (最終仕様は `261117-repository-space-design.md`)

この文書は、エージェントでの確認の手順と結果を記録します。WSL の側の登録、操作、登録の解除は、Claude Code が、各 CLI の非対話の実行で行いました。Windows の側の操作は、Windows で動く別の Claude Code のセッションに、セッション間のメッセージで依頼しました。同じ確認を利用者が再現できるように、コマンドと指示文を記載します。

実際のユーザー名とコンピューター名は、`<ユーザー名>`、`<コンピューター名>` と表します。スクラッチパッドのパスは、`<S>` と表します。

## 状況

| 項目 | 状況 |
| --- | --- |
| 実行ファイルでの折り返し確認 (`probe.sh`、`chat-probe.sh`) | 検証済 |
| 到達点 1 (プロフィールの申告) | 検証済 (5 つすべて。Codex と agy は、サンドボックスを外した実行) |
| 到達点 2 (リポジトリのルーム、origin あり) | 検証済 (5 つすべて、送信と受信) |
| 到達点 3 (origin のないフォルダー) | 検証済 (5 つすべて。WSL の中のフォルダーと、Windows のドライブの中のフォルダー) |
| 到達点 4 (フォルダーを移ったときの更新) | 記録済み (4 つが自分で更新。Claude Code は、別のフォルダーへ移る操作が許可されず、更新しなかった) |
| 到達点 5 (Windows の側のエージェント) | 検証済 (Claude Code、PowerShell)。他の 4 つの CLI は未検証 |
| Windows 向けの実行ファイル (`windows-amd64`) | 検証済 (起動、`/healthz`、`lounge_join`、停止の表示) |
| 登録の解除と後片付け | 実施済み (WSL の側の 5 つと、Windows の側の Claude Code。確認用のフォルダーを削除) |

## 環境

| 項目 | 値 |
| --- | --- |
| OS (WSL の側) | Linux x86_64 (WSL 2、カーネル 6.6.87.2、Oracle Linux 8 のディストリビューション) |
| OS (Windows の側) | Windows 10.0.26300、PowerShell 7.7.0 |
| サーバー | `dist/agent-lounge-mcp` (ブランチ `feature/repository-space`、コミット `e768743` に、`cmd.exe` の入力を閉じる案内と、最後の行を値にする変更を加えたもの)、ポート 7788、WSL の側で起動 |
| データベース | `dist/repository-final.db` (最終の通しの確認用。空の状態から開始) |
| Claude Code | 2.1.326 (WSL の側)、2.1.325 (Windows の側) |
| Codex | 0.169.0 |
| Copilot CLI | 1.0.112 |
| Grok | 1.0.77 |
| agy | 1.5.10 (確認の開始時は 1.5.9) |

## 再現の手順

### 準備

1. サーバーを起動します。

    ```bash
    cd /home/user/agent-lounge-mcp
    dist/agent-lounge-mcp serve --db dist/repository-final.db
    ```

2. 5 つのエージェントに登録します。

    ```bash
    claude  mcp add --transport http --scope user agent-lounge http://127.0.0.1:7788/mcp
    codex   mcp add agent-lounge --url http://127.0.0.1:7788/mcp
    copilot mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    grok    mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    agy     mcp add --type http agent-lounge http://127.0.0.1:7788/mcp
    ```

3. 確認用のフォルダーを作ります。origin は実在しない URL で、通信は行いません。スキルと指示ファイルは置きません。

    ```bash
    mkdir -p <S>/check && cd <S>/check
    for d in a1 a2; do git init -q $d && git -C $d remote add origin https://github.com/example-org/lounge-check-a.git; done
    git init -q b && git -C b remote add origin git@github.com:example-org/lounge-check-b.git
    git init -q n && mkdir -p n/sub/deep
    mkdir "$(wslpath -u "$(cd /mnt/c && cmd.exe /c 'echo %TEMP%' | tr -d '\r')")/lounge-check-w"
    ```

| フォルダー | 内容 |
| --- | --- |
| `a1`、`a2` | 同じ origin を設定した、別々の Git のリポジトリ |
| `b` | 別の origin を設定した Git のリポジトリ |
| `n`、`n/sub/deep` | origin のない Git のリポジトリと、そのサブフォルダー |
| `lounge-check-w` | Windows の一時フォルダーの下の、Git の管理下にないフォルダー (WSL からは `/mnt/c/Users/<ユーザー名>/AppData/Local/Temp/lounge-check-w`) |

### 非対話の実行と、許可

各エージェントを、確認用のフォルダーをカレント フォルダーにして実行します。`<指示>` に、下の指示文を入れます。

| エージェント | コマンド |
| --- | --- |
| Claude Code | `claude -p "<指示>" --allowedTools "mcp__agent-lounge" "Bash(git remote get-url origin)" "Bash(git rev-parse --show-toplevel)" "Bash(pwd)" "Bash(id -un)" "Bash(hostname)" "Bash(uname -n)" 'Bash(echo $WSL_DISTRO_NAME)' "Bash(cmd.exe /c:*)"` |
| Codex | `codex exec --skip-git-repo-check --dangerously-bypass-approvals-and-sandbox "<指示>"` |
| Copilot CLI | `copilot -p "<指示>" --allow-tool 'agent-lounge' 'shell(git remote get-url origin)' 'shell(git rev-parse --show-toplevel)' 'shell(pwd)' 'shell(id -un)' 'shell(hostname)' 'shell(uname -n)' 'shell(echo:*)' 'shell(cmd.exe:*)'` |
| Grok | `grok -p "<指示>" --allow mcp__agent-lounge --allow 'Bash(git remote get-url origin)' --allow 'Bash(git rev-parse --show-toplevel)' --allow 'Bash(pwd)' --allow 'Bash(id -un)' --allow 'Bash(hostname)' --allow 'Bash(uname -n)' --allow 'Bash(echo $WSL_DISTRO_NAME)' --allow 'Bash(cmd.exe /c:*)'` |
| agy | `agy -p "<指示>" --dangerously-skip-permissions --print-timeout 200s` |

Claude Code、Copilot CLI、Grok は、agent-lounge のツールと、プロフィールを調べる読み取りのコマンドだけを許可しました。

Codex と agy は、すべての許可を省くオプションを使いました (利用者の許可による)。理由は、次のとおりです。

- **Codex**: サンドボックス (`-s read-only`、`-s workspace-write`) の中では、WSL から Windows のプログラムを呼ぶ通信が止められ、`cmd.exe` が `UtilBindVsockAnyPort ... socket failed` で失敗します。`os_user` と `machine` を省略して参加するため、origin のあるリポジトリでは使えますが、origin のないフォルダーでは `lounge_join` がエラーになります。
- **agy**: 許可の規則 (`--settings` の `permissions.allow`) で実行すると、コマンドが隔離された環境で動きます。`/mnt/c` が見えず、`uname -n` は `sandbox` を返し、`cmd.exe` は見つかりません。

### 指示文

`<名前>` は、`<エージェント>-a` (origin あり)、`<エージェント>-n` (origin なし、WSL の中)、`<エージェント>-w` (origin なし、Windows のドライブの中) です。指示文では、値の調べ方と引数を教えません。

| 到達点 | フォルダー | 指示文 |
| --- | --- | --- |
| 1 | `a1` または `a2` | `agent-lounge の使い方を確認して、<名前> で参加して。参加の結果と lounge_who の結果を、そのまま表示して。` |
| 1 (対照) | `b` | 同じ (名前は `claude-b`) |
| 2 (送信) | `a2` または `a1` | `agent-lounge の使い方を確認して、<名前> で参加して、リポジトリのルームに「<エージェント> からの確認です」と送って。送信の結果を、そのまま表示して。` |
| 2 (受信) | `a1` または `a2` | `agent-lounge の使い方を確認して、<名前> で参加して、未読のメッセージを読んで。各メッセージの送信者、scope、本文を表示して。` |
| 3 | `n` と `n/sub/deep`、`lounge-check-w` | 到達点 1 と 2 と同じ 3 つの指示文 (参加、送信、受信) |
| 4 | `a1` | `agent-lounge の使い方を確認して、<エージェント>-u で参加して。次に、別のフォルダー <S>/check/b に移って作業を続けて (そのフォルダーの README の有無を確かめるだけでよい)。最後に lounge_who を呼んで、自分の行を表示して。` |

合否は、エージェントの出力と、データベースの内容 (`sessions` のプロフィール、`deliveries` の送達状態) の両方で判定しました。

### 後片付け

```bash
claude  mcp remove --scope user agent-lounge
codex   mcp remove agent-lounge
copilot mcp remove agent-lounge
grok    mcp remove agent-lounge
agy     mcp remove agent-lounge
```

サーバーを停止し、確認用のフォルダー (`<S>/check`、Windows の一時フォルダーの `lounge-check-w` と `lounge-check-bin`) を削除します。

## 結果

### 到達点 1: プロフィールの申告

5 つすべてが、`lounge_help` を読み、WSL であることを確かめて、ホストの Windows の値を調べ、申告しました。

| セッション | `os_user` | `machine` | `repository` |
| --- | --- | --- | --- |
| `claude-a`、`codex-a`、`copilot-a`、`grok-a`、`agy-a` | `<ユーザー名>` (Windows) | `<コンピューター名>` (小文字) | `github.com/example-org/lounge-check-a` |
| `claude-b` | 同じ | 同じ | `github.com/example-org/lounge-check-b` |

`folder` は、5 つとも `\\wsl.localhost\oraclelinux8-dev\...\check\a1` (または `a2`) の形で記録されました。

### 到達点 2: リポジトリのルーム (origin あり)

5 つのエージェントが 1 通ずつ送信し、それぞれ、同じ origin の他の 4 つに届きました。別の origin の `claude-b` には、1 通も届いていません。5 つとも、4 通を受信して表示しました (送達状態は、すべて `delivered`)。

### 到達点 3: origin のないフォルダー

| フォルダー | `repository` | 結果 |
| --- | --- | --- |
| `n` とそのサブフォルダー | `<コンピューター名>:\\wsl.localhost\oraclelinux8-dev\...\check\n` | 5 つすべてが同じ単位。サブフォルダーから起動したエージェントも、作業ツリーの最上位を申告。5 つとも、送信し、他の 4 通を受信 |
| `lounge-check-w` | `<コンピューター名>:c:\users\<ユーザー名>\appdata\local\temp\lounge-check-w` | 5 つすべてが同じ単位 (`/mnt/c/...` が `c:\...` に変換)。5 つとも、送信し、他の 4 通を受信 |

### 到達点 4: フォルダーを移ったときの更新

| エージェント | 結果 |
| --- | --- |
| Codex、Copilot CLI、Grok、agy | 自分で `lounge_join` を呼び直し、`repository` が `github.com/example-org/lounge-check-b` に変わった |
| Claude Code | 別のフォルダーへ移る操作 (`cd`、絶対パスの `ls`) が許可されず、移っていないため、プロフィールを更新しなかった。「移れたら、同じ名前で `lounge_join` をやり直す」と報告した |

### 到達点 5: Windows の側のエージェント

Windows の Claude Code (PowerShell) で、次を確認しました。確認の時点のサーバーは、コミット `afa7962` から `e768743` の間の版です。

| 確認 | 結果 |
| --- | --- |
| Windows から、WSL の側のサーバー (`127.0.0.1:7788`) への接続 | 検証済 |
| リポジトリのフォルダー (`D:\Users\<ユーザー名>\...\agent-lounge-mcp`) での申告 | `folder` は `d:\users\<ユーザー名>\...\agent-lounge-mcp`、`origin` と `repository` は `github.com/hondarer/agent-lounge-mcp`。`os_user` は `$env:USERNAME`、`machine` は `hostname` で取得 |
| `%TEMP%\lounge-check-w` (origin なし) での申告 | `repository` は `<コンピューター名>:c:\users\<ユーザー名>\appdata\local\temp\lounge-check-w`。WSL の側から同じフォルダーで参加した `claude-w` と、同じ値 |
| Windows の側と WSL の側の間の、リポジトリのルームの往復 | `win-w` の送信が `claude-w` に届き、`claude-w` の返信が、待ち受け中の `win-w` に約 1[秒] で届いた |

非対話の実行のコマンドは、`claude -p '<指示>' --allowedTools "mcp__agent-lounge" "Bash(git remote get-url origin)" "Bash(git rev-parse --show-toplevel)" "Bash(pwd)" "Bash(hostname)" "Bash(echo $USERNAME)" "PowerShell(git remote get-url origin)" "PowerShell(git rev-parse --show-toplevel)" "PowerShell(hostname)" "PowerShell(pwd)" "PowerShell($env:USERNAME)"` の形です。登録と登録の解除は、WSL の側と同じコマンドです。

### Windows 向けの実行ファイル

`agent-lounge-mcp-windows-amd64.exe` (コミット `e768743`) を、Windows で直接起動して確認しました。

| 確認 | 結果 |
| --- | --- |
| `version` | `agent-lounge-mcp e768743` |
| `serve --listen 127.0.0.1:7799` と `/healthz` | `{"schema_version":2,"state":"ok","version":"e768743","waiting":0}` |
| `initialize`、`tools/list` (`curl`) | 6 個のツール |
| `lounge_join` (`machine` が `PC-A`、`folder` が `C:\Work\App`) | `folder` は `c:\work\app`、`repository` は `pc-a:c:\work\app` |
| Ctrl+C (`GenerateConsoleCtrlEvent`) | `shutting down (waiting up to 5s for open connections; press Ctrl+C again to stop now)`、`stopped`、終了コード 0 |

キーボードからの Ctrl+C と、2 回目の Ctrl+C は、Windows では確認していません。

### 実行ファイルでの確認 (WSL の側)

- `probe.sh` (4 行) と `chat-probe.sh` (7 行) は、すべての行が `OK:` でした。`chat-probe.sh` は、同じデータベースで 2 回実行しても成功しました。
- 接続を 1 つ残した状態で、`SIGINT` を 0.5[秒] の間隔で 2 回送ると、2 回目の約 0.04[秒] 後に終了コード 0 で終了し、`stopped (forced)` と表示しました。
- 前の案件の確認用のデータベース (版 1) の写しを指定して起動すると、版 2 に移行して起動しました (`/healthz` の `schema_version` が 2)。データが保たれることは、自動テスト (版 1 のデータベースを作り、移行の前後で全行を比べる) で確認しています。

## 気付いた事項

### 確認の中で直した点

最初の文面と実装では、次の問題が現れ、案内文とサーバーを直しました。上の結果は、直した後のものです。

| 現れた問題 | 直した内容 |
| --- | --- |
| Copilot CLI と Codex が、WSL を普通の Linux と見なし、`id -un` と `hostname` を実行した | `lounge_help` の Profile を番号付きの手順にし、最初に `echo $WSL_DISTRO_NAME` で確かめさせる。`folder` が Linux のパスの形のとき、`lounge_join` の結果に `hint` を付ける |
| このディストリビューションに `hostname` コマンドがない | Linux と macOS の `machine` の調べ方を `uname -n` にする |
| Windows の Git Bash では、`cmd.exe /c` の `/c` がパスとして変換され、`cmd.exe` が対話モードで起動する (Windows の側のセッションと利用者が確認) | Windows の `os_user` は、シェルごとの `USERNAME` の調べ方を案内する。`cmd.exe /c` は WSL だけの案内にする |
| PowerShell で `echo %USERNAME%` を実行すると、展開されない文字列が返る | 展開されていない変数の形の `os_user` と `machine` を、エラーにする |
| エージェントが、コマンドを `;` でつないで実行し、許可の規則で拒否される。値を作業フォルダーのパスから推測する (Windows の側で 1 回) | 「1 つずつ実行する」「調べられない値は、推測せずに省略する」と明記する |
| agy で、`cmd.exe /c hostname` が入力を待って止まる | 案内のコマンドに `</dev/null` を付ける |
| WSL の `cmd.exe` が、値の前に UNC パスについての警告の行を表示する | サーバーが、`os_user` と `machine` の、空でない最後の行を値として使う |
| `wsl_distro` だけを渡すと、場所が消える (最終仕様どおりの動作) | エージェントが `wsl_distro` をマシンの性質として単独で渡すおそれがあるため、`folder` のない `wsl_distro` をエラーにする |

### 確認の作業での誤り

確認の途中で、再ビルドのコマンドを `make build | tail -0` と書いたために、ビルドが途中で打ち切られ、実行ファイルが更新されていませんでした。このため、案内文を直した後の 3 回のやり直しと、Windows の側の 2 回の実行が、古い案内文で動いていました。実行ファイルの時刻と中身で気付き、ビルドし直して、やり直しました。上の結果は、最終のビルドと、空のデータベースで、最初から通して実行したものです。

### エージェントごとの気付き

- Claude Code は、WSL の確認のコマンドが許可されなかった回に、「フォルダーが WSL の形のまま記録された。Windows の側のセッションと同じ単位にならないかもしれない」と自分で報告しました。
- Copilot CLI は、許可されていないコマンド (`cat /etc/hostname`) を試して拒否された後、別の方法に切り替えました。
- Grok は、申告する `agent` の値が、実行のたびに変わりました (`grok-4.6`、`grok@<コンピューター名>`、`grok` など)。
- agy は、確認の途中で 1.5.9 から 1.5.10 に変わりました。
- 受信の指示への出力の形 (表、箇条書き) は、エージェントによって異なります。件数の判定には、データベースの送達状態を使いました。
