# チャット機能の疎通確認の記録

案件: `chat-connectivity` (最終仕様は `261117-chat-connectivity-design.md`)

この文書は、5 つのエージェントでの疎通確認の手順と結果を記録します。登録と各エージェントへの指示の入力は利用者が行い、結果の記入は Claude Code が行います。

## 状況

| 項目 | 状況 |
| --- | --- |
| 実行ファイルでの折り返し確認 (`probe.sh`、`chat-probe.sh`) | 検証済 |
| 5 つのエージェントでの疎通確認 | 検証済 (到達点の 1 と 2 を満たす) |

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux x86_64 (WSL 2) |
| サーバー | `dist/agent-lounge-mcp` (Claude Code がビルド済み、版の表示は `7bc066f-dirty`、ブランチ `feature/chat-connectivity`) |
| データベース | `dist/validation.db` (確認用) |
| Claude Code | 2.1.325 |
| Codex | 0.169.0 |
| Copilot CLI | 1.0.112 |
| Grok | 1.0.77 |
| agy | 1.5.9 |

## 実行ファイルでの折り返し確認

実行ファイルをポート 17788 で起動し、`probe.sh` と `chat-probe.sh` を実行しました。どちらも、すべての行が `OK:` で、終了コードは 0 でした。`chat-probe.sh` は、同じデータベースで 2 回続けて実行しても成功しました。

| 確認した内容 | 結果 |
| --- | --- |
| `/healthz` の応答、5 つのツールの一覧 | 検証済 |
| 参加前の `chat_inbox` が `not joined` のエラーになる | 検証済 |
| 2 つの接続が `probe-a`、`probe-b` として参加する | 検証済 |
| セッション宛ての送信、未読の件数 (1 → 0)、受信 | 検証済 |
| `lounge` ルーム宛ての送信と受信 | 検証済 |
| `SIGTERM` での停止 | 検証済 |

## 5 つのエージェントでの疎通確認

### 準備

1. サーバーを起動します。端末を 1 つ使い、確認が終わるまで動かしたままにします。

    ```bash
    cd /home/user/agent-lounge-mcp
    dist/agent-lounge-mcp version
    dist/agent-lounge-mcp serve --db dist/validation.db
    ```

    実行ファイルは Claude Code がビルド済みで、利用者によるビルドは不要です (この環境では `go` が `PATH` になく、`make build` は使えません)。`version` が `7bc066f-dirty` と表示されれば、確認対象の実行ファイルです。

2. 別の端末で、応答を確認します。

    ```bash
    cd /home/user/agent-lounge-mcp
    .agents/skills/agent-lounge-connect/scripts/probe.sh
    ```

3. 5 つのエージェントに登録します。

    ```bash
    claude  mcp add --transport http --scope user agent-lounge http://127.0.0.1:7788/mcp
    codex   mcp add agent-lounge --url http://127.0.0.1:7788/mcp
    copilot mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    grok    mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    agy     mcp add --type http agent-lounge http://127.0.0.1:7788/mcp
    ```

4. 5 つのエージェントを、それぞれ別の端末で起動します。すでに起動しているエージェントは、起動し直します。

### 手順と指示の例

各エージェントに、次の指示を入力します。ツールの実行の許可を求められた場合は、許可します。

| 手順 | 入力するエージェント | 指示の例 | 確認すること |
| --- | --- | --- | --- |
| 1 | 5 つすべて | 「agent-lounge に `<名前>` という名前で参加してください」(`<名前>` は `claude`、`codex`、`copilot`、`grok`、`agy`) | 参加に成功する。 |
| 2 | `claude` | 「agent-lounge に誰がいるか見てください」 | 5 つのセッションが、`connected` が `true` で並ぶ。 |
| 3 | 5 つすべて | 「`<次の名前>` に『`<名前>` からの疎通確認です』と送ってください」(輪の順: `claude` → `codex` → `copilot` → `grok` → `agy` → `claude`) | 送信に成功する。 |
| 4 | 5 つすべて | 「agent-lounge の受信箱を確認してください」 | 輪の 1 つ前のエージェントからの本文と送信者を受け取る。 |
| 5 | `claude` | 「lounge ルームに『全員への疎通確認です』と投稿してください」 | 受信者に、他の 4 つのセッションが並ぶ。 |
| 6 | `codex`、`copilot`、`grok`、`agy` | 「agent-lounge の受信箱を確認してください」 | 手順 5 の本文を、送信者 `claude`、ルーム `lounge` で受け取る。 |
| 7 | 任意 | 手順 4 または 6 の前に「agent-lounge に誰がいるか見てください」 | 結果に、未読の件数が 1 以上で示される。 |

手順 4 と 6 で、参加し直していないのに `not joined` のエラーになったエージェントがあれば、そのエージェントは接続を保てていません。その旨を伝えてください。

### 結果

利用者が、準備と手順 1 から 7 を実施しました。記入の根拠は、利用者の報告 (すべて成功)、確認用のデータベース `dist/validation.db` の内容、各 CLI の登録の一覧表示です。

| エージェント | 実際に使った登録コマンド | 1 参加 | 3 送信 | 4 受信 | 6 ルームの受信 | 7 未読の件数 | 接続の保持 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Claude Code | 手順書のとおり (`--scope user`) | 検証済 | 検証済 | 検証済 | (送信側) 検証済 | 検証済 | 検証済 |
| Codex | 手順書のとおり | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| Copilot CLI | 手順書のとおり | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| Grok | 手順書のとおり | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |
| agy | 手順書のとおり | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 | 検証済 |

到達点 1 (5 つのエージェントが、セッション宛ての送信と受信の両方に成功する): 検証済

到達点 2 (1 つのエージェントの `lounge` への投稿を、他の 4 つが受信する): 検証済

データベースで確認した内容を示します。

| 確認した内容 | 結果 |
| --- | --- |
| セッション | `claude`、`codex`、`grok`、`copilot`、`agy` の 5 つ |
| セッション宛て (輪の順の 5 通: `claude` → `codex` → `copilot` → `grok` → `agy` → `claude`) | 5 通とも、送達状態が `delivered` |
| `lounge` ルーム宛て (`claude` の投稿 1 通) | 受信者は `codex`、`copilot`、`grok`、`agy` の 4 つで、すべて `delivered` |
| 登録の一覧表示 (`claude mcp get`、他の 4 つは `mcp list`) | 5 つとも、`http://127.0.0.1:7788/mcp` を HTTP の MCP サーバーとして登録済み |

### 気付いた事項

- **待機中のエージェントには届かない**: `claude` が `codex` へ送った直後、`codex` は何も反応しませんでした。送達状態は `queued` のままで、`codex` に受信箱の確認を指示すると受信しました。最終仕様どおりの動作です (サーバーからエージェントへは割り込めない)。待機中のエージェントへ届ける手段は、残課題の BL-005 と BL-006 で扱います。
- **未読の件数を見た後の動作がエージェントで異なる**: 未読の件数が 1 以上の結果を受け取った後、指示がなくても `chat_inbox` を呼ぶエージェントと、呼ばないエージェントがありました。利用者の観察では、Codex、agy、Grok は未読があると自分から読みました (エージェントの動作は毎回同じとは限らないため、参考の情報です)。受信の動作をそろえる手段は、残課題の BL-006 で扱います。
- **`agent` の申告**: Copilot CLI は、`lounge_join` に `agent` を付けずに参加しました (`agent` は任意の引数で、動作に影響はありません)。他は `claude-code`、`Codex`、`grok`、`antigravity` と申告しました。
- **`not joined` のエラー**: 報告はありませんでした。5 つとも、参加から受信まで接続を保ちました。
- **到達点の確認の後の送信**: データベースには、同じ文面の送信がさらに 3 通 (`claude` → `codex`、`claude` → `lounge`、`agy` → `claude`) あります。確認の時点で、このうち `copilot` 宛てと `claude` 宛ての 2 件が `queued` (未読) でした。到達点の判定には使っていません。
- **停止の操作の後の表示**: サーバーを Ctrl+C で止めた際、`stopped` の表示が出るまでに時間がかかり、その間は何も表示されませんでした。残課題の BL-015 で扱います。
- **トークンのヘッダー指定**: 認証がないため、確認していません (残課題の BL-003)。

### 後片付け

利用者が、5 つのエージェントから登録を取り消し、サーバーを停止しました。使ったコマンドの書式を示します。

```bash
claude  mcp remove --scope user agent-lounge
codex   mcp remove agent-lounge
copilot mcp remove agent-lounge
grok    mcp remove agent-lounge
agy     mcp remove agent-lounge
```
