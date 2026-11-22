<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-21 00:59:50 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53SsJ6j1XoXzWU5edaQ
稼働 OS: Linux (WSL)
上流文書: `261120-expiry-sandbox-verification-design.md` (最終仕様)、`261120-expiry-sandbox-verification-discussion.md` (検討用 Markdown)。削除されている場合があります。
-->

# 未検証の消化、トークンの有効期限、サンドボックスのマシン名の確認の記録

案件: `expiry-sandbox-verification` (最終仕様は `261120-expiry-sandbox-verification-design.md`)

## 状況

| 受入条件 (最終仕様の第 10 章) | 結果 |
| --- | --- |
| 1. `go vet`、`go test` (WSL と Windows) | 満たす (「機械的検証」) |
| 2. 期限の機能の自動テスト | 満たす |
| 3. 最後の利用日時の自動テスト | 満たす |
| 4. マシンの名前の自動テスト | 満たす |
| 5. 基盤の自己試験 (WSL と Windows)、Windows で `ResourceWarning` が出ない | 満たす |
| 6. 第 6 章の a から k の実施と記録 (h には、Copilot CLI のシナリオ 5 の 5 回の完了を含む) | 満たす。c は、利用者の判断で確かめず、残課題に回した (「作業中の判断」) |
| 7. 文書の更新 | 満たす |
| 8. Opus のレビューと対応 | 満たす (「実装のレビュー」) |
| 9. 残課題一覧の整理 | 満たす (14 件) |

## 環境

| 項目 | WSL の側 | Windows の側 |
| --- | --- | --- |
| OS | Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (Oracle Linux 8) | Windows 11 Pro for Workstations 10.0.26300 |
| Go | 1.27.3 | 1.27.4 |
| Python | 3.12.15 | 3.14.3 |
| Claude Code | 2.1.329 | 2.1.325 |
| Codex | 0.169.0 | 0.170.0 |
| Copilot CLI | 1.0.114 | 1.0.114 |
| Grok | 1.0.89 | 1.0.89 |
| agy | 1.5.13 | 1.5.11 |
| その他 | podman 4.9.4-rhel、nginx 1.31.9 (`docker.io/library/nginx:latest`) | PowerShell 7.7.0、Git Bash (Git for Windows) |

## 担当

| 担当 | 作業 |
| --- | --- |
| Opus のサブエージェント | トークンの有効期限と最後の利用日時、基盤の (16) とパスへのアクセスの画面、完了の合言葉、実装のレビュー |
| Sonnet のサブエージェント | マシンの名前、基盤の (14) と (15)、WSL の側の確認 (f、i、j、k)、`docs/ja/` の更新 |
| 対話セッション (途中から Opus 5.5。利用者の切り替え) | 指示、差分の点検と取り込み、BL-041 の直し、レビューの指摘の直し、記録 |
| Windows の側のセッション | Windows の側の確認 (a、b、d、e、g、h、BL-041) |

サブエージェントは、別の作業ツリーで作業し、対話セッションが差分を確かめて作業ブランチに取り込みました (`git cherry-pick`)。

## 機械的検証

| 確認 | 結果 |
| --- | --- |
| `go vet ./...` (WSL) | 問題なし |
| `go test -race ./...` (WSL) | すべて ok |
| `go test ./...` (Windows、`0d6b8a7`) | すべて ok |
| 基盤の自己試験 (WSL、`python3 -W error::ResourceWarning`) | 45 件 OK (始めは 30 件) |
| 基盤の自己試験 (Windows、`b5951e8`) | 41 件 OK (1 件はスキップ。前と同じ `PWD` の試験)、`ResourceWarning` は 0 行 |

## 確認の結果

| 記号 | 項目 | 結果 | 要点 |
| --- | --- | --- | --- |
| a | Windows の、origin のあるフォルダーでの、4 つの CLI の申告 | 検証済 | 4 つとも、リポジトリの単位は `github.com/hondarer/agent-lounge-mcp`。Codex の `-s read-only` では、`pwsh.exe` の起動が拒否され (CreateProcessAsUserW 5)、モデルが `cmd.exe /c` に切り替えた。Copilot CLI は、シェルの許可の規則がないと、プロフィールが空のまま参加する (許可の規則を足すと正しい値。`$env:USERNAME` は規則に当てはまらず、`os_user` は空) |
| b | Windows の側で起動したサーバーでの、期限切れ、削除、管理コマンド | 検証済 | `--expire-after 20s --retain-for 20s --session-idle 60s --cleanup-interval 5s`。未読は 21[秒] で `expired`、その約 20[秒] 後に削除。使わないセッションは 101[秒] の時点で削除。`session list`、`session remove`、`memory list`、`memory remove`、`export` (`C:\...\Temp\alm-esv\b\export-1`) は、すべて終了コード 0。実行ファイルは `2c87369` (トークンの有効期限の前) |
| c | Windows の側で起動したサーバーへの、WSL からの接続 | 未検証 | Windows の側で `--allow-remote --listen 0.0.0.0:7898` で起動。WSL から `172.28.208.1:7898` への接続は、時間切れ。ファイアウォールに `alm.exe` の規則がなく、許可の画面に利用者が応答できなかった。利用者の判断で残課題 (BL-028) に回した |
| d | `cmd` と Git Bash での申告 | 検証済 (値) | 使うシェルは、Claude Code、Copilot CLI、Grok が PowerShell 7、agy が Windows PowerShell 5.1、Codex は `pwsh.exe` だが `-s read-only` では `cmd.exe /c`。Claude Code (Git Bash)、Codex (cmd) の申告の値は、PowerShell と同じ。Git Bash の `pwd` が `/tmp/...` を返す点を見つけ、BL-041 として直した |
| BL-041 | Git Bash の `pwd -W` | 検証済 | `0d6b8a7`。Claude Code (Git Bash) が、案内どおりに `pwd -W` を実行し、その出力を渡した。記録されたフォルダーとリポジトリの単位は、PowerShell と同じ |
| e | Windows での Ctrl+C と、2 回目の Ctrl+C | 検証済 (疑似端末へのキー入力) | ConPTY に 0x03 を送った。1 回: 停止の表示、5.0[秒] で終了コード 0、待ち受け中の `chat_inbox` は停止のエラー。0.5[秒] あけて 2 回: `stopped (forced)`、0.6[秒] で終了コード 0 |
| f | nginx 経由の HTTPS | 検証済 | 利用ガイドの例は、直さずに動いた (変えたのは、待ち受けのポート、名前、証明書のパス、転送先のポート)。`--allowed-host lounge.test` の経路は `curl --resolve` で確かめた (`/mcp` はトークンありで 200、なしで 401、別の名前で 403)。`probe.sh`、`chat-probe.sh` (待ち受け中の受信を含む)、Claude Code (`NODE_EXTRA_CA_CERTS`) は、`https://localhost:7912` で確かめた |
| g | Windows の作業中の表示と Ctrl+U | Copilot CLI と Ctrl+U は検証済、Codex は制限あり | Copilot CLI の `ctrl+c interrupt` は、作業中だけに出た。Codex の `esc to interrupt` は、最初の数秒だけで、作業を続けている間 (7[秒] から 63[秒]) は画面になかった (BL-042)。Ctrl+U は 5 つの CLI とも入力欄を消した |
| h | 検証の基盤の直し | 検証済 | 自己試験 (上の表)。Copilot CLI のシナリオ 5 は、下の「基盤の (16)」 |
| i | agy 1.5.13 の、閉じられた接続の後の動作 | 制限あり (前の版と同じ) | `--connection-idle 30s`、50[秒] 待ち。1 回目の指示は 5 回とも失敗 (`unknown tool: "call_mcp_tool"`)、2 回目の指示は 5 回のうち 2 回で使えた。サーバーを起動し直した後も同じ。起動し直した agy は、同じ名前で参加でき、セッションは増えなかった |
| j | Claude Code のフォルダーの移動 | 検証済 | 1 回目は、作業フォルダーの外への `cd` を Claude Code が止めた。2 回目は `--add-dir` で許可し、Claude Code が自分から `lounge_join` を呼び直して、リポジトリの単位が移った先のフォルダーになった |
| k | サンドボックスの中での申告 | 検証済 | Codex (`-s read-only`) と agy (許可の規則) は、`cmd.exe` が使えないことに気付き、`cat /etc/hostname` の値を `machine` に、`os_user` を省いて申告した。サンドボックスなしの Claude Code と、リポジトリのルームで往復できた。サンドボックスなしの 5 つの CLI の申告は、変わらなかった。Codex の週の使用率は 85% から 86% (残り 14%) |

### 基盤の (16) (Copilot CLI のシナリオ 5、WSL)

| 段階 | コード | 結果 |
| --- | --- | --- |
| 1 回目の 5 回 | `d29ef20` (SETTLE の規則) | 5 回とも `done`。中の 5 回をすべて終えたのは 2 回。止まった原因は、応答の定義のない「Allow path access」の画面 (3 回) と、原因の分からない時間切れ (1 回) |
| 2 回目の 5 回 | `18d0242` (パスへのアクセスの画面に応答) | 5 回とも `done`。中の 5 回をすべて終えたのは 4 回。残る 1 回は、モデルが完了の合言葉 (`IVDONE` と、`e` で始まる文字列) の `E` を 1 つ落として書いたため。1 回目の原因の分からない時間切れも、同じ原因だった。`3810702` で、文字列を `e` で始めないようにした |
| 3 回目の 5 回 | `3810702` (最終のコード) | 5 回とも `done` で、中の 5 回をすべて終えた (5 回中 5 回)。応答した画面の数は 7、9、8、9、10。1 回は 204[秒] から 223[秒] |

どの段階でも、15[秒] の遅れと、答えた画面への 2 回目の応答は、ありませんでした。同じコマンドの許可を続けて求められる場面は、実機では起きず、自己試験と、確認用の実行 (`touch same.txt` を 2 つ同時に求めさせた) で確かめました (2 回目への応答は、15.4[秒] から 2.5[秒] になった)。

## 作業中の判断

| 時点 | 判断 | 記録 |
| --- | --- | --- |
| 第 1 ターンの回答 | `windows-arm64` と macOS は、ビルドはするが保証しないこととして閉じる。物理的に別のマシンと、WSL ではない Linux は、残課題に残す | 簡易 ADR の LADR-003 |
| 作業中 | 作業は Opus (高難易度) と Sonnet (低難易度) のサブエージェントで進める | LADR-002 |
| 作業中 | (16) の直しは、仕様の条件 (5 回の完了) を満たさないまま取り込み、別の許可の画面にも応答させる | LADR-003 |
| 作業中 | c は、利用者が画面を見られないため、残課題に回す | LADR-003 |
| 作業中 | BL-041 (Git Bash の `pwd`) は、この案件で直す | LADR-003 |

## 実装のレビュー

Opus のサブエージェントが、`9bb81a0..0d6b8a7` (文書を除く) を読み、正しさを確かめました。重大な問題はありませんでした。

| 指摘 | 扱い |
| --- | --- |
| ホスト名が `localhost` の Linux で、Git の外のフォルダーにいると、拒否のエラー文と `machine` がない場合のエラー文の間を行き来する | 直した (`6732277`)。拒否のエラー文に、WSL の場合であることと、`machine` を省き、origin がなければ `folder` も省くことを足した |
| 1 秒未満の期間 (`500ms`) で、発行した時点で切れたトークンができる | 直した。1 秒未満を書式の誤りにした |
| 試験のコメントの位置のずれ | 直した |
| SETTLE の規則で、答えた画面を CLI が 2[秒] より長く残し、その間に英字か数字を出力すると、2 回応答する | 直さない。残課題 (BL-035 の (17)) に記録した |
| 最後の利用日時の書き込みは、要求の処理の中で行う (同じユーザーで 1 分に 1 回) | 直さない。仕様どおり |
| 時計が戻った場合、`_Prompts.answered` の古い競合、孫のプロセスが標準エラーを持ち続ける場合、`lounge_help` の大きさの余裕 (93 バイト) | 直さない (影響が小さい) |

## 最終仕様との相違

- 第 5 章の BL-035 の (16) は、条件 (2) (Copilot CLI のシナリオ 5 の 5 回の完了) を満たさない場合は今の動作を保つ、としていました。利用者の判断で、直しを取り込み、別の許可の画面 (パスへのアクセス) への応答と、完了の合言葉の綴りの誤りへの対処を加えました。
- 第 6 章の c は、確かめていません (利用者の判断)。
- 第 1 章の範囲に、BL-041 (Git Bash の `pwd -W`) を加えました (利用者の判断)。
- 第 2 章の期間は「0 以下は誤り」としていましたが、レビューの指摘で「1 秒未満は誤り」にしました (保存の単位がミリ秒で、それより短い期間は、発行した時点で切れるため)。
- 第 4 章の拒否のエラー文に、`folder` も省くことを足しました (レビューの指摘)。

## 後片付け

- nginx のコンテナーは消し、イメージ (`docker.io/library/nginx:latest`) は残しました。
- 確認用のサーバーは、すべて止めました。各 CLI への agent-lounge の登録は、確認の後に外しました (Windows の側は `mcp list` で確認)。
- agy の設定 (`~/.gemini/config/mcp_config.json`) は、並行した確認の間で書き換えがぶつかりました。今は `mcpServers` が空です。案件の前の内容は、記録が残っていません。
- 確認用のデータは、WSL の側はセッションの一時フォルダーに、Windows の側は `%TEMP%\alm-esv` にあります。
