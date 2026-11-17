<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-18 02:40:14 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53Qv4yzq5xHqqMwe5fY
稼働 OS: Linux (WSL)
上流文書: `261118-chat-completion-design.md` (最終仕様)、`261118-chat-completion-discussion.md` (検討用 Markdown)。削除されている場合があります。
-->

# チャットの完成 (送達の確認、読み返し、期限切れと削除) の確認の記録

案件: `chat-completion` (最終仕様は `261118-chat-completion-design.md`)

この文書は、WSL の側の 5 つのエージェントと、実際のサーバーでの確認の手順と結果を記録します。操作は、Claude Code が、各 CLI の非対話の実行と、MCP を HTTP で直接呼ぶ補助のスクリプトで行いました。

実際のユーザー名、コンピューター名、パスは書きません。スクラッチパッドのパスは、`<S>` と表します。

## 状況

| 項目 | 状況 |
| --- | --- |
| 到達点 1 (送達の確認) | 検証済 (5 つとも、案内だけで `chat_ack` を呼んだ。相手役の `chat_status` が `acked` と時刻を返した) |
| 到達点 2 (読み返し) | 検証済 (5 つとも、文脈のない実行で `chat_history` の `unacked: true` を使い、取得済みで未完了の依頼に対応した。やりとりの要約と、送ったメッセージの状態の確認もできた) |
| 到達点 3 (期限切れと削除) | 検証済 (短い期限のサーバーで、`expired`、メッセージの削除、セッションの削除、参加し直しを確認) |
| 到達点 4 (管理コマンド) | 検証済 (サーバーの動作中に、`session list`、`session remove`、`export` を実行) |
| 到達点 5 (Windows の側) | 未検証 |
| 登録の解除と後片付け | 実施済み |

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux x86_64 (WSL 2、カーネル 6.6.87.2、Oracle Linux 8) |
| サーバー | `dist/agent-lounge-mcp` (コミット `29b65ce`、ブランチ `feature/chat-completion`)。ループバックだけで待ち受け、トークンなし (ユーザー `local`) |
| データベース | `<S>/cc/check.db` (手順 1 から 5、7)、`<S>/cc/short.db` (手順 6)。どちらも空の状態から開始 |
| Claude Code | 2.1.326 |
| Codex | 0.169.0 |
| Copilot CLI | 1.0.112 |
| Grok | 1.0.77 |
| agy | 1.5.10 |

## 再現の手順

### 準備

1. サーバーを起動し、5 つの CLI を、トークンなしで `http://127.0.0.1:7788/mcp` に登録します (登録のコマンドは、利用ガイドのとおり)。
2. 確認用のフォルダー `<S>/cc/work` (Git の管理下にない) を作ります。
3. 相手役は、MCP を HTTP で直接呼ぶ補助のスクリプト (`<S>/cc/mcp.py`。接続を開き、参加し、ツールを 1 つ呼ぶ) で操作します。セッション名は `tester` です。
4. 相手役が各エージェントのセッション宛てに送れるように、補助のスクリプトで、先に `<cli>-c` の名前で参加しておきます。エージェントは、同じ名前で参加して、そのセッションを引き継ぎます。

### 非対話の実行と、許可

コマンドの形は、前の案件 (`261117-chat-autonomy-validation.md`) と同じです。agent-lounge のツールだけを許可しました。イベントを JSON で出力する形式で実行し、ツールの呼び出しを確かめました。

| エージェント | コマンド |
| --- | --- |
| Claude Code | `claude -p "<指示>" --allowedTools "mcp__agent-lounge" --output-format stream-json --verbose` |
| Codex | `codex exec --skip-git-repo-check -s read-only --json "<指示>"` |
| Copilot CLI | `copilot -p "<指示>" --allow-tool 'agent-lounge' --output-format json` |
| Grok | `grok -p "<指示>" --allow mcp__agent-lounge --output-format streaming-json` |
| agy | `agy -p "<指示>" --settings agy-settings.json --output-format stream-json` (`{"permissions":{"allow":["mcp(agent-lounge/*)"]}}`) |

サンドボックスを外すオプションは、どの CLI でも使いませんでした (利用者の許可はありましたが、不要でした)。

### 手順と指示文

| 番号 | 相手役の操作 | エージェントへの指示文 |
| --- | --- | --- |
| 1 | 各エージェントに、質問 (「質問です: n + 4 は? 答えを返信してください。」) を送る | `agent-lounge の使い方を確認して、<cli>-c で参加して、未読のメッセージを読んで対応して。` |
| 2 | (1 で、5 つとも完了を記録したため、実施していません) | |
| 3 | 依頼 (「依頼です: n × 3 を計算して、結果を返信してください。」) を送り、補助のスクリプトが `<cli>-c` として `chat_inbox` で取得する (エージェントが取得した後に文脈を失った状態) | `agent-lounge に <cli>-c で参加して、取得したまま対応していないメッセージがないか確かめて、あれば対応して。` |
| 4 | なし | `agent-lounge に <cli>-c で参加して、tester との最近のやりとりを読み返して、要約して。` |
| 5 | なし | `agent-lounge に <cli>-c で参加して、自分が最後に送ったメッセージの送達状態を確認して、結果を表示して。` |
| 6 | 短い期限で起動したサーバーで、補助のスクリプトで操作する | なし |
| 7 | サーバーの動作中に、管理コマンドを実行する | なし |

手順 1 では、指示文に `chat_ack` や「完了の記録」という言葉を入れていません。案内 (`lounge_help`、ツールの説明文) だけで完了を記録するかを確かめるためです。

手順 6 のサーバーは、次のオプションで起動しました。

```bash
agent-lounge-mcp serve --listen 127.0.0.1:17993 --db <S>/cc/short.db \
  --expire-after 5s --retain-for 8s --session-idle 20s --cleanup-interval 2s
```

## 結果

### 手順 1: 案内だけでの完了の記録 (到達点 1)

| エージェント | 返信 | `chat_ack` | 相手役の `chat_status` |
| --- | --- | --- | --- |
| Claude Code | 「2 + 4 = 6 です。」 | 呼んだ | `acked` |
| Codex | 「3 + 4 = 7 です。」 | 呼んだ | `acked` |
| Copilot CLI | 「4 + 4 = 8 です。」 | 呼んだ | `acked` |
| Grok | 「9」 | 呼んだ | `acked` |
| agy | 「6 + 4 の答えは 10 です。」 | 呼んだ | `acked` |

5 つとも、返信と `chat_ack` の両方を行いました。Claude Code と agy は、`acked_at` が返信の送信の時刻の後であることを、時刻で確かめました。他の 3 つは、手順 1 では時刻を照合していません (手順 3 の Codex では、イベントの順で、返信の後に `chat_ack` を呼んだことを確かめました)。

### 手順 3: 文脈を失った後の読み直し (到達点 2)

5 つとも、`chat_history` を `unacked: true` で呼び、取得済みで未完了の依頼を見つけて、返信し、`chat_ack` を呼びました (相手役の `chat_status` が `acked`)。返信の内容は、5 つとも正しい計算の結果でした。Codex のイベントでは、`lounge_help` → `lounge_join` → `chat_history({"unacked": true, "limit": 100})` → `chat_send` → `chat_ack({"message_ids": [12]})` の順でした。

### 手順 4 と 5: 読み返しと、状態の確認 (到達点 2)

5 つとも、`chat_history` で `tester` とのやりとり (質問と返信、依頼と返信) を要約しました。手順 5 では、5 つとも、`chat_history` で自分が送った最後のメッセージを探し、`chat_status` で状態を表示しました。表示した状態は `delivered` (相手役が取得済みで、完了を記録していない) で、データベースの内容と一致しました。agy は、「`chat_inbox` で読み取り済みですが、まだ `chat_ack` は行われていません」と、`delivered` と `acked` の違いを説明しました。

### 手順 6: 期限切れと削除 (到達点 3)

| 時点 | 確認したこと |
| --- | --- |
| 送信の直後 | 受信者の未読が 1 件 |
| 8[秒] 後 | 送信者の `chat_status` が `expired` と `expired_at` を返した。受信者の `chat_inbox` が空で、未読が 0 件。受信者の `chat_history` に、`state` が `expired` で現れた |
| 18[秒] 後 | 送信者の `chat_status` が、「このセッションが送ったメッセージに、その ID はない (保存期間の後に削除される)」というエラーを返した (メッセージが削除された) |
| 使われていないセッション | 20[秒] 使わなかったセッション `b` が削除され、`session list` から消えた。使い続けたセッション `a` は残った |
| 参加し直し | `b` が同じ名前で参加し、新しいセッションになった (`resumed` が `false`) |

サーバーのログには、期限切れ、メッセージの削除、セッションの削除の件数が 1 行ずつ出ました。本文と名前は出ませんでした。

削除されたセッションに対応づいていた接続が、次の呼び出しでエラーを受け取ることは、補助のスクリプトが呼び出しごとに新しい接続を開くため、この手順では確かめていません。自動テスト (`internal/server/delivery_test.go`) で確認しています。

### 手順 7: 管理コマンド (到達点 4)

| コマンド | 結果 |
| --- | --- |
| `session list` | 見出しの行と、6 つのセッション (アドレス、エージェント名、最後の利用日時、未読の件数、リポジトリの単位)。終了コード 0 |
| `session list --user nobody` | 「no such user」。終了コード 1 |
| `session remove local/agy-c` (`--yes` なし) | 削除される件数 (4 messages, 4 deliveries) を表示し、何も削除せず、終了コード 1 |
| `session remove local/agy-c --yes` | 削除した件数 (5 messages, 5 deliveries) を表示。終了コード 0。相手役の `chat_history` の `with: "agy-c"` が空になった |
| 同じコマンドをもう一度 | 「no such session」。終了コード 1 |
| `export --out <S>/cc/export` | 16 件のメッセージを 4 つのファイルに書き出した。すべてのユーザーのメッセージを含むという注意を、標準エラーに表示した。終了コード 0 |

書き出したファイルは、`chat/user-local/direct-local_claude-c--local_tester-<ハッシュ値>.md` の形でした (アドレスの `/` を `_` に置き換え、ハッシュ値の先頭 8 文字を付加)。各メッセージは、見出し (時刻、送信者のアドレス、エージェント名、ID)、配信の状態、コード ブロックの本文の順でした。

## 後片付け

5 つの CLI の登録を解除し (`claude mcp remove --scope user agent-lounge`、`<cli> mcp remove agent-lounge`)、2 つのサーバーを、待ち受けているポートからプロセスを特定して停止しました。確認用のデータベース、書き出したファイル、フォルダーを削除しました。

## 気付いた事項

### 確認の作業での誤り

- 最初のビルドで、Go のパスを設定していなかったため、ビルドが失敗し、サブエージェントが作った古い実行ファイル (コミットの前の版) でサーバーを起動しました。`version` の表示で気付き、サーバーを止めて、作り直してから確認を始めました。上の結果は、作り直した後のものです。

### エージェントごとの気付き

- **申告の値**: 補助のスクリプトが、`agent` を `check-helper` として参加した後に、エージェントが `agent` を省いて参加したため、いくつかのセッションのエージェント名が `check-helper` のままになりました。送信の時点の値を保存するため、そのセッションが送ったメッセージには `check-helper` が記録されています (仕様どおりの動作で、確認の手順による結果です)。
- **許可の効き方**: agent-lounge のツールだけを許可しましたが、Grok と agy は、プロフィールを調べる読み取りのコマンド (`whoami`、`hostname`、`git remote get-url origin` など) を実行していました。前の案件と同じです。
- **Grok の申告**: WSL の中のフォルダーを、`\\wsl.localhost\<ディストリビューション名>\...` の形で申告しました (BL-029 の 3 番の形ではなく、WSL の中のパスを Windows の形で表したもの)。リポジトリの単位が、他の CLI と異なりました。今回の確認には影響しません。
- **案内の効果**: 5 つとも、`lounge_help` の「送信者が結果を待つメッセージは、返信の後に `chat_ack`」と、「文脈を失った後は `chat_history` の `unacked: true`」に従いました。
