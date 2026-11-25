<!--
生成元: agree-before-implement スキル
文書種別: 最終決定事項 (最終仕様)
最終更新日時: 2026-11-25 19:23:00 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53ReNCYHwdPaqjNwsEV
稼働 OS: Linux (WSL)
上流文書: `261125-windows-verification-overview.md` (全体管理文書)、`261125-windows-verification-discussion.md` (検討用 Markdown)、`261125-windows-verification-lightadr.md` (簡易 ADR)。いずれも削除されている場合があります。
下流文書: `261125-windows-verification-backlog.md` (残課題一覧)、`261125-windows-verification-validation.md` (確認の記録)。未作成の場合や、削除されている場合があります。
再開手順: このファイルを開いたエージェントは、作業の前に agree-before-implement スキルを読み込んでください。上流文書の全体管理文書が存在する場合は、その再開手順に従って作業 WBS の `← **CURRENT**` の項目から再開します。存在しない場合は、この文書を単体の最終仕様として扱います。最終更新エージェントのセッション情報や稼働 OS・最終更新日時が参照可能であれば再開時のコンテキストや環境差分の把握に利用し、別エージェントや別環境・別 OS からの引き継ぎ、担当変更によりセッションが見つからない場合も異常とせず、文書の記載から継続してください。
-->

# Windows の側のセッションでの残課題の確認 最終仕様

ファイル名: `261125-windows-verification-design.md`

## 1. 目的と範囲

Windows の側のセッションと WSL の側のセッションで、実機の確認が残っている次の 7 つの残課題を確かめ、利用者向けの文書の「検証済みの範囲」を更新します。

| 課題 ID | 確かめること | 主に確かめる側 |
| --- | --- | --- |
| BL-014 | Windows の CLI での日本語の名前の受け渡し、Windows での絞り込んだ `export` の日本語のファイル名 | Windows |
| BL-044 | Windows のブランチのスコープ (d: PowerShell での申告、e: 大文字と小文字を変えたブランチ名、f: WSL の側のセッションとの往復) | Windows (f は両方) |
| BL-047 (1) | Windows での MCP を使わない CLI の方式 | Windows |
| BL-047 (2) | Codex のサンドボックスで、エージェントが `--detach` で起動した常駐プロセスが残るか | WSL |
| BL-042 | Windows の Codex の、作業中の表示 | Windows |
| BL-040 | agy の新しい版での、閉じられた接続の後とサーバーの起動し直しの後の動作 | 両方 |
| BL-048 | `shared` とルームの、Codex、Copilot CLI、agy での確認 (Grok は未検証) | WSL |

Table: 確かめる残課題

製品のコードは、変えないことを前提にします。確認で製品や検証の基盤の不具合が見つかった場合の扱いは、[7 章](#7-作業を止める条件) のとおりです。

## 2. 環境

| 項目 | WSL の側 | Windows の側 |
| --- | --- | --- |
| OS | Linux x86_64 (WSL 2) | Windows 11 Pro for Workstations 10.0.26300、PowerShell 7.7.0 |
| 実行ファイル | 作業ブランチから `make build` (`linux-amd64`) | Windows 担当が、同じコミットから `go build` (`windows-amd64`) |
| CLI | Claude Code 2.1.331、Codex 0.171.0、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.16 | Claude Code 2.1.330、Codex 0.171.0、Copilot CLI 1.0.114、Grok 1.0.92、agy 1.5.17 |
| 確認用のデータ | セッションの一時フォルダー | `%TEMP%\alm-wv` (リポジトリの外) |

Table: 確認の環境

- 版は、確認の時点のものを記録します。確認の途中で CLI の版が上がった場合は、上がった版を記録します。
- CLI への agent-lounge の登録は、確認のために足し、確認の後に外します。許可は、確認に要るツールだけにします (前の案件と同じ、CLI ごとの個別の許可)。
- 利用の上限に達した CLI は、未検証として記録し、残課題一覧に残して先へ進みます。Grok は、確認の時点で上限であれば、すべての項目で未検証にします。

### 2.1. サーバーの構成

| 名前 | 起動する側 | 構成 | 使う確認 |
| --- | --- | --- | --- |
| S1 | WSL | `--allow-remote --listen 0.0.0.0:<ポート>`、トークンあり。Windows の側からは WSL のアドレスへ接続 | BL-014 (1)、BL-044 |
| S2 | Windows | ループバック、トークンなし | BL-014 (2)、BL-047 (1) |
| S3 | WSL と Windows のそれぞれ | ループバック、`--connection-idle 30s` | BL-040 |
| S4 | WSL | ループバック、2 つのユーザーのトークン | BL-048、BL-047 (2) |

Table: サーバーの構成

Windows の側のサーバーは、ループバックだけで待ち受けます (ファイアウォールの許可の画面を出さないため)。

## 3. Windows の側の確認

Windows 担当が行い、結果 (実行したコマンド、出力の要点、版、判定) を WSL の側へメッセージで報告します。

### 3.1. 日本語の名前とブランチの申告 (BL-014 (1)、BL-044 d・e・f)

1. WSL の側で S1 を起動し、日本語と全角の数字を含むユーザー名 (例: `確認者２`) のユーザーを `user add` で作り、トークンを Windows 担当へ渡します。
2. Windows 担当は、`%TEMP%\alm-wv` の下に、origin が `https://github.com/hondarer/agent-lounge-mcp.git` の Git のリポジトリを作り、ブランチ `verify/wv-branch` に切り替えます。
3. Windows の 5 つの CLI (Grok は上限であれば未検証) を、トークンを付けて S1 に登録し、それぞれ非対話の実行で、このリポジトリのフォルダーから、使い方の案内に従って、日本語と全角の数字を含むセッション名 (`担当-<CLI>２`) で参加させます。プロフィールの調べ方とブランチの申告は、指示に含めません。確かめること:
    - 参加の結果の名前が正規化の後の値 (`担当-<CLI>2`) で、ユーザーが正規化の後の `確認者2` であること (BL-014 (1))。
    - `branch` が `verify/wv-branch` であること (BL-044 d)。
4. WSL の側の相手役 (HTTP で直接ツールを呼ぶ) が、同じ origin とブランチを申告して参加し、ブランチのルームで Windows の 1 つの CLI (Claude Code) と 1 往復します。相手役が全角のつづりを宛先にして送ったメッセージを、その CLI が受け取ることも確かめます (BL-044 f、BL-014 (1))。
5. Windows 担当は、ブランチ `main` があるリポジトリで、`git switch Main` (大文字と小文字を変えた名前) を試し、`git branch --show-current` の出力を記録します。切り替えられた場合は、Claude Code を参加し直させ、サーバーが記録した `branch` (`session list` の `BRANCH` 列) が `main` であることを確かめます (BL-044 e)。

### 3.2. 日本語のファイル名の書き出し (BL-014 (2))

1. Windows 担当は、S2 を起動し、日本語のユーザー名 (`確認者２`) を `user add` で作り、そのユーザーで日本語のセッション名で参加し、メッセージとメモリを 1 件ずつ作ります (`call` または HTTP で直接)。
2. 日本語のユーザーと空間を指定した絞り込みの `export` を、`%TEMP%\alm-wv` の下へ実行し、作られたファイル名 (日本語) を `Get-ChildItem` で確かめ、内容を開けることを確かめます。

### 3.3. MCP を使わない CLI の方式 (BL-047 (1))

S2 に対して、Windows 担当が、PowerShell から次を実行します (エージェントの CLI への登録はしません)。

1. `connection start` (前面) と、別の名前での `connection start --detach`。`connection status`、`connection list` の出力。
2. 保存場所 `%LOCALAPPDATA%\agent-lounge-mcp\connections` の権限を `icacls` で確かめ、本人 (と SYSTEM、Administrators の扱い) だけの保護された DACL であることを記録します。
3. `tools`、`call lounge_help`、`call --as <名前> lounge_join` (プロフィールの `os_user`、`machine`、`folder`、`origin`、`branch` が集められる)、`chat_send`、`chat_inbox`。
4. `connection stop` の後に、常駐プロセスが終わり、ソケットのファイルが片付くこと。`--detach` の常駐プロセスが、起動した PowerShell を閉じた後も残ること。

### 3.4. Windows の Codex の作業中の表示 (BL-042)

1. Windows 担当は、検証の基盤 (`scripts/interactive/`) で Codex の対話の画面を起動し、長い文章の出力を指示します。
2. 作業の間、1 秒ごとに画面の全体を保存します (確認用のデータの下)。
3. 保存した画面から、作業の間だけ出る文字列 (または、作業の終わりにだけ出る文字列) を探します。
4. 見つかった場合は、その文字列で `scripts/interactive/harness/agents.py` の Codex の作業中の判定を直し、基盤の自己試験 (WSL と Windows) と、Windows の Codex のシナリオ (2 と 3) で確かめます。見つからない場合は、作業を止めて利用者に扱いを確かめます ([7 章](#7-作業を止める条件))。

### 3.5. agy 1.5.17 の確かめ直し (BL-040、Windows)

Windows の S3 で、前の確認 (`261123-backlog-batch-validation.md` の 2.2) と同じ手順を行います。対話の画面の agy を参加させ、別のセッションからメッセージを送り、(1) 50 秒待つ、(2) サーバーを起動し直す、のそれぞれについて、受信箱を読んで合言葉を答える指示を 2 回まで送ります。各 3 回です。

## 4. WSL の側の確認

WSL の側 (このセッション) が行います。

### 4.1. agy 1.5.16 の確かめ直し (BL-040、WSL)

WSL の S3 で、3.5 と同じ手順を行います。

### 4.2. `shared` とルーム (BL-048)

S4 で、`../past/261124-shared-space-rooms-design.md` の 9.2 の 4 つの操作を、Codex、Copilot CLI、agy の非対話の実行で確かめます (各 1 回。失敗した場合は 3 回まで)。相手役は、別のユーザーのトークンで、HTTP で直接ツールを呼びます。Grok は、上限であれば未検証にします。

1. 別のリポジトリのセッションの間の、`shared` のセッション宛てと `lounge` 宛ての送受信。
2. ルームの作成、別のユーザーのセッションの参加、投稿、受信、退出の後に届かないこと。
3. 後から参加したセッションによる、参加の前のやりとりの読み返し。
4. `shared` のメモリの作成と、別のユーザーのセッションからの検索。

### 4.3. Codex のサンドボックスの `--detach` (BL-047 (2))

S4 を利用者の側 (このセッション) で起動し、Codex を `-s workspace-write -c sandbox_workspace_write.network_access=true` で、確認用のフォルダーから非対話で実行します。指示は、`connection start --detach` で常駐プロセスを起動し、`call` で参加して終わることです。Codex の終了の後に、常駐プロセスが残っているか (`connection list` と `ps`) を確かめます。確かめた後、常駐プロセスを止めます。

## 5. 文書の更新

結果に合わせて、次の「検証済みの範囲」の行を書き換えます。確かめられなかった項目は「未検証」のまま、理由を書きます。

| ファイル | 行 |
| --- | --- |
| `docs/ja/user-guide.md` | Windows のブランチの申告・大文字と小文字・往復、Windows の日本語の名前の受け渡し、Windows の日本語のファイル名の `export`、agy の閉じられた接続の後の動作、Codex・Copilot CLI・Grok・agy での `shared` とルーム |
| `docs/ja/cli-without-mcp.md` | Windows での実機の動作、`--detach` の常駐プロセスが残るか (70 行の「未検証です」の文も合わせる) |
| `docs/ja/interactive-verification.md` | Windows の Codex の作業中の表示 (3.4 で直した場合) |

Table: 文書の更新

- BL-040 が直っていた場合は、利用ガイドの「エージェントの接続」の注意 (agy を起動し直す案内) と、スキルの文書の同じ案内を改めます。直っていない場合は、確認した版を更新します。
- 確認の記録は、`docs/prompts/current/261125-windows-verification-validation.md` に書きます。Windows 担当の報告は、出典 (Windows 担当の報告) を付けて要点を書きます。

## 6. 作業の進め方

1. WSL の側で、作業ブランチ `verify/windows-verification` を作り、作業前の記録をコミットします。
2. Windows 担当へ、3 章の手順を送ります。Windows 担当は、`git fetch` と作業ブランチへの切り替えだけを行い (ファイルは変えません)、実行ファイルをビルドします。
3. Windows の側の確認 (3 章) と WSL の側の確認 (4 章) を、交互に進めます。3.1 の 4 は、両方の側で同時に行います。
4. 3.4 で基盤を直す場合は、WSL の側で直してコミットと push を行い、Windows 担当が取り込んで確かめます。
5. 文書の更新、確認の記録、残課題の整理を行い、第 3 ゲートで承認を受けます。push は作業ブランチに行い、`main` へのマージは、第 3 ゲートで利用者の指示を受けてから行います。

## 7. 作業を止める条件

次の場合は、作業を止めて利用者に確かめます。

- 製品のコード (`internal/`、`cmd/`) の不具合が見つかった場合、または使い方の案内 (`internal/tools/help.go`) の表現を変える必要がある場合。
- 3.4 で、作業中を見分ける文字列が見つからない場合。
- 利用者の操作 (ファイアウォールの許可の画面など) が必要になった場合。

## 8. 受入条件

1. 1 章の 7 つの課題ごとに、確かめた結果 (検証済、未検証と理由、制限あり) が確認の記録にあること。
2. 5 章の文書が、結果どおりの「検証済みの範囲」だけを書いていること。
3. 確認の後に、CLI への agent-lounge の登録と、確認用のサーバー・常駐プロセスが残っていないこと (両方の側)。
4. 基盤を直した場合は、基盤の自己試験 (WSL と Windows) と CI が通ること。
5. 確かめられなかった項目と、選ばなかった項目が、この案件の残課題一覧にあること。

## 9. 第 2 ゲートで確認を受ける細部

決定から一意に決まらない細部です。

- BL-047 (2) は、WSL の側の Codex だけで確かめます (前に失敗したのが WSL の側のため)。
- BL-044 f の往復は、Windows の Claude Code 1 つで行います (他の CLI は、d の申告だけ)。
- BL-047 (1) は、エージェントの CLI を使わず、Windows 担当が PowerShell からコマンドを実行して確かめます。
- 確認用のユーザー名とセッション名は、前の確認と同じ形 (`確認者２`、`担当-<CLI>２`) にします。
- 3.4 で作業中の文字列が見つかった場合の基盤の直しは、この案件で行います (それ以外の基盤の直しは行いません)。
