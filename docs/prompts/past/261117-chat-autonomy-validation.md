# エージェントが自律して使えるチャットの確認の記録

案件: `chat-autonomy` (最終仕様は `261117-chat-autonomy-design.md`)

この文書は、5 つのエージェントでの確認の手順と結果を記録します。登録、操作、登録の解除は、Claude Code が、各 CLI の非対話の実行で行いました。同じ確認を利用者が再現できるように、コマンドと指示文を記載します。

## 状況

| 項目 | 状況 |
| --- | --- |
| 実行ファイルでの折り返し確認 (`probe.sh`、`chat-probe.sh`) | 検証済 |
| 到達点 1 (使い方の確認) | 検証済 (5 つすべて) |
| 到達点 2 (中継なしの往復) | 検証済 (5 組すべて) |
| 到達点 3 (停止の通知) | 検証済 (5 つすべて) |
| 到達点 4 (ログ通知の見え方) | 記録済み (5 つとも、出力に現れない) |
| 上限いっぱいの待ち受けの完了 | 検証済 (5 つすべて、上限 45[秒])。上限が 60[秒] のときは、Claude Code が完了できなかったため、利用者の決定で 45[秒] に変更 |
| 登録の解除 | 実施済み (5 つとも、一覧表示に `agent-lounge` がないことを確認) |

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux x86_64 (WSL 2) |
| サーバー | `dist/agent-lounge-mcp` (版の表示は `bfad50d-dirty`、ブランチ `feature/chat-autonomy`)、ポート 7788 |
| データベース | `dist/autonomy-validation.db` (確認用) |
| Claude Code | 2.1.325 |
| Codex | 0.169.0 |
| Copilot CLI | 1.0.112 |
| Grok | 1.0.77 |
| agy | 1.5.9 |

## 再現の手順

### 準備

1. サーバーを起動します。

    ```bash
    cd /home/user/agent-lounge-mcp
    dist/agent-lounge-mcp serve --db dist/autonomy-validation.db
    ```

2. 5 つのエージェントに登録します。

    ```bash
    claude  mcp add --transport http --scope user agent-lounge http://127.0.0.1:7788/mcp
    codex   mcp add agent-lounge --url http://127.0.0.1:7788/mcp
    copilot mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    grok    mcp add --transport http agent-lounge http://127.0.0.1:7788/mcp
    agy     mcp add --type http agent-lounge http://127.0.0.1:7788/mcp
    ```

3. エージェントごとに、このリポジトリの外に空のフォルダーを作り、そこで実行します。スキルと指示ファイルは置きません。

### 非対話の実行と、ツールの許可

5 つすべてで、agent-lounge のツールだけを許可する方法を使えました。すべての許可を省くオプションは、使っていません。`<指示>` に、下の指示文を入れます。

| エージェント | コマンド |
| --- | --- |
| Claude Code | `claude -p "<指示>" --allowedTools "mcp__agent-lounge"` |
| Codex | `codex exec --skip-git-repo-check -s read-only "<指示>"` |
| Copilot CLI | `copilot -p "<指示>" --allow-tool 'agent-lounge'` |
| Grok | `grok -p "<指示>" --allow mcp__agent-lounge` |
| agy | `agy -p "<指示>" --settings agy-settings.json` |

- Codex は、読み取り専用のサンドボックス (`-s read-only`) のままで、MCP のツールを呼べました。
- agy は、許可の指定がないと、MCP のツールを拒否して何も出力しません。この実行だけに効く設定のファイル `agy-settings.json` に、次の内容を書いて渡します。規則を `mcp(agent-lounge)` と書いた場合は、拒否されました。

    ```json
    {"permissions":{"allow":["mcp(agent-lounge/*)"]}}
    ```

### 指示文

| 確認 | 指示文 |
| --- | --- |
| 到達点 1 | 「agent-lounge の使い方を確認して、`<名前>` という名前で参加し、使い方を 5 行以内で説明して」 |
| 到達点 2 (返信する側。先に、バックグラウンドで実行する) | 「agent-lounge に `<名前>` という名前で参加し、受信を待って、届いた質問に返信して」 |
| 到達点 2 (待つ側。返信する側の 20[秒] 後に実行する) | 「agent-lounge に `<名前>` という名前で参加し、`<相手>` に『1 + 1 は?』と送って、返信を待ち、返信の内容を報告して」 |
| 上限いっぱいの待ち受け | 「agent-lounge に `<名前>` という名前で参加し、chat_inbox を wait_seconds 60 で 1 回だけ呼んで、結果 (messages の件数と waited_seconds の値) を報告して。ほかのことはしないで」 |
| 到達点 3、4 (実行の 35[秒] 後に、サーバーを `SIGINT` で止める) | 「agent-lounge に `<名前>` という名前で参加し、受信を待って。何か起きたら、起きたことを報告して」 |

到達点 4 では、各 CLI の、イベントを JSON で出力する形式を追加しました (`claude --output-format stream-json --verbose`、`codex exec --json`、`copilot --output-format json`、`grok --output-format streaming-json`、`agy --output-format stream-json`)。

### 後片付け

確認の後、Claude Code が次のコマンドで登録を解除し、サーバーを停止しました。

```bash
claude  mcp remove --scope user agent-lounge
codex   mcp remove agent-lounge
copilot mcp remove agent-lounge
grok    mcp remove agent-lounge
agy     mcp remove agent-lounge
```

## 結果

### 到達点 1: 使い方の確認

| エージェント | 参加 | 使い方の説明 (`chat_inbox` と `wait_seconds` を含む) | `lounge_help` の呼び出し | 結果 |
| --- | --- | --- | --- | --- |
| Claude Code | 成功 | 含む | 出力からは確認できない (説明の内容は本文と一致) | 検証済 |
| Codex | 成功 | 含む | `lounge_help` → `lounge_join` の順に呼んだ | 検証済 |
| Copilot CLI | 成功 | 含む | `lounge_join` → `lounge_help` の順に呼んだ | 検証済 |
| Grok | 成功 | 含む | 出力からは確認できない (説明の内容は本文と一致) | 検証済 |
| agy | 成功 | 含む | 出力からは確認できない (説明の内容は本文と一致) | 検証済 |

5 つとも 1 回目で成功しました (agy は、ツールの許可の書き方を直した後の 1 回目)。5 つとも、説明に「返信を待つときは `wait_seconds`」と「上限は 60[秒]」のどちらか、または両方を含み、これは `lounge_help` の本文とツールの説明文にだけある内容です。接続時の案内文 (`instructions`) を読んでいるかどうかは、この確認では切り分けられませんでした。

### 到達点 2: 中継なしの往復

待つ側 (A) の出力に、返信する側 (B) の返信の内容があること、データベースで往復の 2 通が `delivered` であることを確認しました。5 組とも、1 回目で成功しました。

| A (送って待つ) | B (待って返信する) | 質問 | A が報告した返信 | 所要時間 | 結果 |
| --- | --- | --- | --- | --- | --- |
| Claude Code | Codex | 1 + 1 は? | 1 + 1 = 2 です。 | 31[秒] | 検証済 |
| Codex | Copilot CLI | 2 + 3 は? | 2 + 3 = 5 です。 | 38[秒] | 検証済 |
| Copilot CLI | Grok | 3 + 4 は? | 7 です。 | 169[秒] (B が返信の後も 60[秒] の待ち受けを 2 回続けたため) | 検証済 |
| Grok | agy | 4 + 5 は? | 9 です。 | 38[秒] | 検証済 |
| agy | Claude Code | 5 + 6 は? | 5 + 6 = 11 です。 | 56[秒] | 検証済 |

所要時間は、B の起動から、A と B の両方の終了までです。A の起動を 20[秒] 遅らせているため、どの組でも、B は 15[秒] 以上待ち受けてから質問を受け取っています。

### 上限いっぱいの待ち受けの完了

最初の確認は、上限が 60[秒] の実行ファイルで行いました。

| エージェント | 結果 | 内容 |
| --- | --- | --- |
| Claude Code | 完了できない | 2 回とも、`chat_inbox` が `The operation timed out.` のエラーになった。`wait_seconds` を 50 にした実行は、`waited_seconds` が 50 で完了した |
| Codex | 検証済 | `messages` は 0 件、`waited_seconds` は 60 |
| Copilot CLI | 検証済 | 同上 |
| Grok | 検証済 | 同上 |
| agy | 検証済 | 同上 |

Claude Code (2.1.325、非対話の実行) は、MCP のツールの応答を 60[秒] 待てません。サーバーの上限 (60[秒]) と同じ長さで、エージェント側が先に打ち切ります。打ち切られた呼び出しの後に届いたメッセージは、失われません (サーバーは、取り消された待ち受けでは、メッセージの状態を変えません)。到達点 2 と 3 では、Claude Code は 60[秒] より前に結果を受け取ったため、成功しています。

この結果を受けて、利用者の決定で、既定の上限を 45[秒] に変更しました。変更後の実行ファイルで、同じ指示 (`wait_seconds` は 60 のまま) をもう一度実行しました。5 つとも、エラーなしで完了しました。

| エージェント | 結果 | 内容 |
| --- | --- | --- |
| Claude Code | 検証済 | `messages` は 0 件、`waited_seconds` は 45 |
| Codex | 検証済 | 同上 |
| Copilot CLI | 検証済 | 同上 |
| Grok | 検証済 | 同上 |
| agy | 検証済 | 同上 |

到達点 1 から 4 の確認は、上限が 60[秒] の実行ファイルで行ったものです。上限の変更は既定値だけで、待ち受けの処理は変えていないため、やり直していません。

### 到達点 3: 停止の通知

5 つを同時に待ち受けさせ、35[秒] 後にサーバーを `SIGINT` で止めました。5 つとも、1 回目で成功しました。

| エージェント | 報告の内容 | 結果 |
| --- | --- | --- |
| Claude Code | サーバーが停止のエラーを返した。メッセージは保持される。再試行はしていない。再起動の後に同じ名前で参加し直す | 検証済 |
| Codex | 待っている間に、停止の通知があった。メッセージは保持される。再起動の後に同じ名前で参加し直す | 検証済 |
| Copilot CLI | 受信待ち中にサーバーが停止した。メッセージは保持される。再試行はしていない | 検証済 |
| Grok | 待ち受けの途中でサーバーが停止した。再試行はしていない。再起動の後に同じ名前で参加し直す | 検証済 |
| agy | 待ち受け中に、停止のエラーが返った (文面をそのまま報告)。メッセージは保持される | 検証済 |

サーバーは、`shutting down` の表示の後、5[秒] で `stopped` を表示して終了しました。

### 到達点 4: ログ通知の見え方

イベントを JSON で出力する形式で、到達点 3 と同じ確認をもう一度行いました。5 つとも、停止のエラーの文面は出力にありましたが、ログ通知 (`notifications/message`) に当たる内容は、出力にありませんでした。

| エージェント | ログ通知が出力に現れるか |
| --- | --- |
| Claude Code | 現れない |
| Codex | 現れない |
| Copilot CLI | 現れない |
| Grok | 現れない |
| agy | 現れない |

エージェントがログのレベルを設定していないために届いていないのか、届いたが出力していないのかは、切り分けていません。対話の画面で、待機中のエージェントにログ通知が見えるかどうかは、未検証です (残課題 BL-018 の検証の基盤が必要です)。

## 気付いた事項

- **並行の呼び出しによる参加前のエラー**: Copilot CLI は、到達点 2 で `lounge_join` と `chat_send` を同時に呼び、`chat_send` が `not joined` のエラーになりました。Copilot CLI は、参加の後に自分で送り直し、成功しました。
- **返信の後の待ち受け**: Grok は、返信の後も、指示のないまま 60[秒] の待ち受けを 2 回続けてから終了しました。他の 4 つは、返信の後に終了しました。
- **確認用のセッション**: 60[秒] の待ち受けの切り分けのために、`claude-b` というセッションを 1 つ作りました。
