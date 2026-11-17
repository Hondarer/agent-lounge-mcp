<!--
生成元: agree-before-implement スキル
文書種別: 確認の記録 (validation)
最終更新日時: 2026-11-18 03:46:29 JST
最終更新エージェント: claude code / セッション ID: session_01KJc53Qv4yzq5xHqqMwe5fY
稼働 OS: Linux (WSL)
上流文書: `261118-memory-design.md` (最終仕様)、`261118-memory-discussion.md` (検討用 Markdown)。削除されている場合があります。
-->

# メモリ機能 (保存、検索、忘却) の確認の記録

案件: `memory` (最終仕様は `261118-memory-design.md`)

この文書は、実装のレビュー、WSL の側の 5 つのエージェントでの確認、実際のサーバーでの確認の手順と結果を記録します。操作は、Claude Code が、各 CLI の非対話の実行と、MCP を HTTP で直接呼ぶ補助のスクリプトで行いました。

実際のユーザー名、コンピューター名、パスは書きません。スクラッチパッドのパスは、`<S>` と表します。

## 状況

| 項目 | 状況 |
| --- | --- |
| 到達点 1 (保存と取得) | 検証済 (5 つとも、案内だけで、保存の前に検索してから保存した) |
| 到達点 2 (共有) | 検証済 (5 つとも、別の CLI が保存したメモリで正しく答えた。別のユーザーがリポジトリのスコープに保存したメモリを読み、保存した人を示した) |
| 到達点 3 (忘却) | 検証済 (自動テストと、短い半減期のサーバー) |
| 到達点 4 (検索) | 検証済 (自動テストと、補助のスクリプトからの検索) |
| 到達点 5 (管理コマンド) | 検証済 (サーバーの動作中に、`memory list`、`memory remove`、`export`) |
| 到達点 6 (Windows の側) | 未検証 |
| 実装のレビュー | 実施済み (Opus 5.5 のサブエージェント。指摘はすべて修正) |
| 登録の解除と後片付け | 実施済み |

## 実装のレビュー

コミット `c864af5` を、Opus 5.5 のサブエージェントがレビューしました (ファイルを変更しない条件)。指摘と対処は、次のとおりです。

| 指摘 | 区分 | 対処 |
| --- | --- | --- |
| 忘却の期間の計算が、大きな半減期や小さなしきい値、多い参照回数であふれ、負の値になって、メモリがすぐに薄れる | 不具合 | 期間の上限で止める形にし、定期の処理の基準の時刻の計算も、あふれを確かめる形にした。自動テストを追加 |
| 検索の参照の数え方が、上位 3 件の中に薄れたメモリがあると、4 件目以降を数える | 最終仕様との相違 | 上位 3 件の範囲の中で、薄れていないものだけを数える形にした。自動テストを追加 |
| 3 文字未満の語の検索で、ASCII 以外の文字の大文字と小文字を区別する (SQLite の `lower()` は ASCII だけを変換する) | 不具合 | すべての文字を小文字にする SQL の関数を登録して使う形にした。自動テストを追加 |
| 保持スコアの四捨五入で、薄れたメモリが、しきい値と同じ `0.1` と表示される | 軽微 | 切り捨てにした。直前に触れたメモリが `0.99` と表示されないよう、100 万分の 1 の許容を加えた |
| 案内文に、薄れたメモリの取得は 1 日の制限に関わらず数えることがない | 軽微 | 案内文に加えた |
| 定期の処理の絞り込みが、1[ミリ秒] 未満の差で、削除の時期を過ぎたメモリを次の回に回す | 軽微 | 比較を `<=` にした |
| テストのコメントの誤り | 軽微 | 直した |

## 環境

| 項目 | 値 |
| --- | --- |
| OS | Linux x86_64 (WSL 2、カーネル 6.6.87.2、Oracle Linux 8) |
| サーバー | `dist/agent-lounge-mcp` (コミット `f4dbcd7`、ブランチ `feature/memory`)。ループバックだけで待ち受け、トークンなし (ユーザー `local`)。別のユーザー `bob` をトークンで追加 |
| データベース | `<S>/mem/check.db` (手順 1 から 4、6)、`<S>/mem/short.db` (手順 5)。どちらも空の状態から開始 |
| 確認用のフォルダー | `<S>/mem/work`。Git の作業ツリーで、origin は `https://github.com/example-org/lounge-memory-check.git` (実在しない URL) |
| Claude Code | 2.1.326 |
| Codex | 0.169.0 |
| Copilot CLI | 1.0.113 |
| Grok | 1.0.77 |
| agy | 1.5.10 |

## 再現の手順

### 準備

1. サーバーを起動し、5 つの CLI を、トークンなしで `http://127.0.0.1:17995/mcp` に登録します (登録のコマンドは、利用ガイドのとおり)。
2. 確認用のフォルダーを作り、`git init` と `git remote add origin` を行います。
3. 補助のスクリプト (`<S>/mem/mcp.py`。接続を開き、参加し、ツールを 1 つ呼ぶ) を用意します。

### 非対話の実行と、許可

イベントを JSON で出力する形式で実行し、agent-lounge のツールの呼び出しを、順に取り出して確かめました。

| エージェント | コマンド |
| --- | --- |
| Claude Code | `claude -p "<指示>" --allowedTools "mcp__agent-lounge" --output-format stream-json --verbose` |
| Codex | `codex exec --skip-git-repo-check -s read-only -c 'mcp_servers.agent-lounge.default_tools_approval_mode="approve"' --json "<指示>"` |
| Copilot CLI | `copilot -p "<指示>" --allow-tool 'agent-lounge' --output-format json` |
| Grok | `grok -p "<指示>" --allow mcp__agent-lounge --output-format streaming-json` |
| agy | `agy -p "<指示>" --settings agy-settings.json --output-format stream-json` (`{"permissions":{"allow":["mcp(agent-lounge/*)"]}}`) |

Codex の `-c` の設定は、手順 1 の最初の実行の後に加えました ([気付いた事項](#気付いた事項))。サンドボックスを外すオプションは、どの CLI でも使いませんでした。

### 手順と指示文

| 番号 | 操作 | エージェントへの指示文 |
| --- | --- | --- |
| 1 | 各エージェントに、異なる知見を保存させる | `agent-lounge の使い方を確認して、<cli>-m という名前で参加して、次の知見を agent-lounge のメモリに保存して: 「<知見>」` |
| 2 | 各エージェントに、別の CLI が保存した知見で答えさせる | `agent-lounge に <cli>-m という名前で参加して、次の質問に関係するメモリを agent-lounge で調べて、その内容に従って答えて。質問: <質問>` |
| 3 | 補助のスクリプトが、ユーザー `bob` (トークン) として、同じ origin を申告し、リポジトリのスコープにメモリを保存する。各エージェントに読ませる | `agent-lounge に <cli>-m という名前で参加して、デプロイの前にすべきことを agent-lounge のメモリで調べて答えて。その情報を誰 (どのユーザーのどのセッション) が保存したかも示して。` |
| 4 | 補助のスクリプトから検索する | なし |
| 5 | 短い半減期で起動したサーバーで、補助のスクリプトから操作する | なし |
| 6 | サーバーの動作中に、管理コマンドを実行する | なし |

指示文には、ツール名、スコープ、引数を入れていません。

| CLI | 手順 1 の知見 | 手順 2 の質問 (知見を保存した CLI) |
| --- | --- | --- |
| Claude Code | ビルドで「undefined: generated.Schema」が出たら、先に make generate を実行すると直る | 結合テストを実行するには、どうすればよいですか (Codex) |
| Codex | 結合テストは、環境変数 LOUNGE_IT=1 を付けたときだけ実行される | 来週から、ステージング環境の API の基底 URL はどうなりますか (Copilot CLI) |
| Copilot CLI | ステージング環境の API の基底 URL は、来週から /v2 に変わる | Windows で go test が「too many open files」で落ちました。どうすればよいですか (Grok) |
| Grok | Windows でテストが「too many open files」で落ちたら、go test に -p 1 を付けて実行する | リリースのタグは、どのように付ければよいですか (agy) |
| agy | リリースのタグは v で始め、CHANGELOG.md の見出しと同じ版にそろえる | ビルドで「undefined: generated.Schema」が出ました。どうすればよいですか (Claude Code) |

手順 5 のサーバーは、次のオプションで起動しました。

```bash
agent-lounge-mcp serve --listen 127.0.0.1:17996 --db <S>/mem/short.db \
  --memory-half-life 4s --memory-fade-threshold 0.5 --memory-grace 4s --cleanup-interval 1s
```

この設定では、一度も参照されないメモリは 4[秒] で薄れ、さらに 4[秒] で削除されます。1 回参照されたメモリは、8[秒] で薄れます。

## 結果

### 手順 1: 案内だけでの保存 (到達点 1)

| エージェント | 呼び出しの順 | スコープ |
| --- | --- | --- |
| Claude Code | `lounge_help` → `lounge_join` → `memory_search` → `memory_put` | `user` (プロフィールを申告せずに参加したため、リポジトリの単位がなかった) |
| Codex | `lounge_help` → `lounge_join` → `memory_search` → `memory_put` | `repository` |
| Copilot CLI | `lounge_help` → `lounge_join` → `memory_search` → `memory_put` | `repository` |
| Grok | `lounge_help` → `lounge_join` → `memory_search` → `memory_put` | `repository` |
| agy | `lounge_help` → `lounge_join` → `memory_search` (検索語なし、`include_faded`) → `memory_put` | `repository` |

5 つとも、保存の前に検索し、題名とタグを付けて保存しました。

### 手順 2: 別の CLI が保存したメモリでの回答 (到達点 1、2)

5 つとも、`memory_search` (語を変えて 2 回から 3 回) で見つけ、知見のとおりに答えました (「`LOUNGE_IT=1` を付ける」「`/v2` に変わる」「`-p 1` を付ける」「`v` で始め `CHANGELOG.md` にそろえる」「`make generate` を先に実行する」)。Grok と agy は、`memory_get` で全体も読みました。Claude Code が手順 1 で `user` のスコープに保存したメモリは、agy が、`scope` を省いた検索で見つけました (どちらもユーザー `local` のため)。

### 手順 3: 別のユーザーのリポジトリのメモリ (到達点 2)

5 つとも、`bob` が保存したメモリを見つけ、「デプロイの前に `make check-schema` を実行する」と答え、保存した人をユーザー `bob`、セッション `bob-helper` と示しました。Claude Code は、「別のユーザー (bob) が書いた情報です。手順として正しいかは、必要に応じて確認してください」と付け加えました。

### 手順 4: 検索 (到達点 4)

| 検索 | 結果 |
| --- | --- |
| 「結合」(2 文字) | 結合テストのメモリ |
| 「タグ」(2 文字) | リリースのタグのメモリ |
| `MAKE` (大文字) | `make` を含む 2 件 |
| `go test -p` (3 語) | Windows のテストのメモリ |
| 「リリース CHANGELOG」 | リリースのタグのメモリ |
| 「リリース deploy」 | 0 件 (両方を含むメモリはない) |
| タグ `DEPLOY` | `bob` のメモリ |
| `scope: user` (一覧) | Claude Code のメモリだけ |

### 手順 5: 忘却 (到達点 3)

| 時点 | 確認したこと |
| --- | --- |
| 作成の直後 | 2 件 (alpha、固定した beta) が一覧に出た |
| 5[秒] 後 | 一覧は beta だけ。`include_faded` で、alpha が `faded: true` で出た。`memory_get` で alpha が `faded: false`、参照回数 1 に戻った。新しく gamma を作成 |
| 15[秒] 後 | gamma は削除されていた (ログに `1 forgotten memory deleted`)。alpha (参照 1 回) は、薄れた状態で残っていた |
| 19[秒] 後 | alpha も削除され、固定した beta だけが残った |

起動時のログに、`memories: half-life 4s, longer with references; they fade below a retention score of 0.5 and are deleted 4s after they fade` と表示されました。

### 手順 6: 管理コマンド (到達点 5)

| コマンド | 結果 |
| --- | --- |
| `memory list` | 見出しの行と 6 件 (ID、スコープ、識別子、状態、保持スコア、更新日時、作成者、題名)。`bob` のメモリの作成者は `bob/bob-helper` |
| `memory list --scope user` | 見出しの行と 1 件 |
| `memory remove 3` (`--yes` なし) | スコープと題名を表示し、何も削除せず、終了コード 1 |
| `memory remove 3 --yes` | 削除した ID、スコープ、題名を表示。終了コード 0。補助のスクリプトの `memory_get` が `no memory with ID 3` を返した |
| 同じコマンドをもう一度 | `no such memory`。終了コード 1 |
| `export --out <S>/mem/export` | 5 件のメモリを 2 つのファイル (`memory/user-local.md`、`memory/repository-github.com_example-org_lounge-memory-check-<ハッシュ値>.md`) に書き出した。終了コード 0 |

## 後片付け

5 つの CLI の登録を解除し、2 つのサーバーを、待ち受けているポートからプロセスを特定して停止しました。確認用のデータベース、書き出したファイル、確認用のフォルダー、出力を削除しました。Git が無視するフォルダー (`dist/` など) に、確認用のファイルが残っていないことを確かめました。

## 気付いた事項

### 確認の作業で直したこと

- **Codex の承認**: 手順 1 の最初の実行で、Codex の `memory_put` が `MCP tool call requires approval, but approval policy is never` で拒否されました。`memory_put` と `memory_forget` は、データを書き換えることがあるツールとして注記しており (MCP の `destructiveHint`)、Codex は、そのようなツールの呼び出しに承認を求めます。チャットのツールは、追加だけを行うツールとして注記しているため、前の案件では起きませんでした。注記は実際の動作に合っているため変えず、Codex の MCP サーバーの設定 `default_tools_approval_mode` で、agent-lounge のツールを承認済みにして実行し直しました。この設定を、利用ガイドとスキルに書きました。設定ファイルに書く形は、`-c` で渡した値と同じ鍵ですが、設定ファイルでは確かめていません。
- **管理コマンドのエラー文**: `memory remove` で、存在しない ID のエラーが `no such memory in your scopes` と表示されました (エージェント向けの文)。`no such memory` に直しました。エージェントへのエラー文は、ツールが別に組み立てるため、変わりません。

### エージェントごとの気付き

- **申告**: Claude Code は、プロフィールを調べるコマンドの許可がなく (`--allowedTools` で agent-lounge のツールだけを許可したため)、手順 1 ではプロフィールなしで参加しました。手順 2 では、origin とフォルダーを申告しました。agy は、手順 2 で、マシンの名前を `sandbox` と申告しました (BL-027)。Codex は、サンドボックスの中で `cmd.exe` を実行できず、OS のユーザー名とマシンの名前を省きました (BL-027)。どれも origin のあるフォルダーのため、リポジトリの単位は同じでした。
- **許可の効き方**: agent-lounge のツールだけを許可しましたが、Grok と agy は、プロフィールを調べる読み取りのコマンドを実行していました。前の案件と同じです。
- **検索の仕方**: Claude Code と Copilot CLI は、日本語と英語の両方の語で検索しました。agy は、検索語なしの一覧 (`include_faded: true`) で全体を見てから選ぶ使い方をしました。
- **保持スコアの表示**: 数分前に参照したメモリの保持スコアは、切り捨てのため `0.99` と表示されます。仕様どおりです。
