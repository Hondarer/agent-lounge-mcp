# `shared` の空間とルームの作成・参加 確認の記録

ファイル名: `261124-shared-space-rooms-validation.md`

案件 `shared-space-rooms` の、自動テストと実機の確認の記録です。最終仕様は `261124-shared-space-rooms-design.md` です。

## 1. 自動テスト

| 項目 | 結果 | 環境 |
| --- | --- | --- |
| `go vet ./...` | 通過 | Linux x86_64 (WSL 2)、Go 1.27.4 |
| `GOOS=windows go vet ./...` | 通過 | 同上 |
| `go test -race ./...` | 通過 | 同上 |
| CI (3 つの OS) | 通過 | GitHub Actions の `ubuntu-latest`、`windows-latest`、`macos-latest` (run 45284144295、`c54c91d`。前のコミット `3a864b6` の run 45283192071 も通過) |

Table: 自動テストの結果

足した主なテストは、次のとおりです。

- BL-004 (`shared`): `internal/store/shared_test.go` (別のリポジトリの 2 つのユーザーの、セッション宛てと `lounge` 宛て。リポジトリのないセッション。一覧。メモリの作成、他のユーザーによる検索・更新・固定・削除、作成したユーザーの削除で残ること、件数の上限)、`internal/server/shared_test.go` (MCP を通した同じ確認と、既定のスコープでのエラー文)、`internal/export/export_test.go` (書き出しの名前)、`cmd/agent-lounge-mcp/main_test.go` (`--memory-shared-*`)。Sonnet のサブエージェントが書き、Claude Code が点検しました。
- BL-009 (ルーム): `internal/store/rooms_test.go` (作成と自動の参加、同じ名前の作成、参加者だけに届くこと、投稿での自動の参加、退出、ないルームのエラー、`lounge` の退出のエラー、一覧の順と `joined`・`members`、スコープの区別、範囲を出たセッションに届かず戻ると届くこと、上限、定期の処理での削除と名前の再利用)、`internal/names/names_test.go` (名前の正規化)、`internal/server/room_test.go` (MCP を通した操作、大文字の名前、エラー文、`lounge_who` の `rooms`)。
- BL-033 の (1) (読み返し): `internal/store/delivery_test.go` (参加していないルームと参加の前のメッセージを読めること、`Others`、自分宛ての未読を返さないこと、他のセッションの間のセッション宛てを返さないこと、スコープを出た後に返さないこと)、`internal/server/delivery_test.go` (`direction` の `room`、`reply_to` と `state` がないこと、ないルームのエラー)。

使い方の案内は、変更前に 19,082 バイトでした。そのため、追加の分を、既存の文面を詰めて上限 (19,300 バイト) に収めました (19,222 バイト)。

## 2. 実機の確認 (WSL)

最終仕様の 9.2 の 4 つの操作を、Claude Code が代表して確かめました。

| 項目 | 内容 |
| --- | --- |
| 環境 | Linux x86_64 (WSL 2)、Intel Xeon Gold 6254 |
| 実行ファイル | `linux-amd64` (作業ブランチの `e91f7da` からビルド) |
| サーバー | ループバックの `--require-token`。確認用のデータベースは、セッションの一時フォルダー |
| ユーザー | `alice` と `bob` (それぞれのトークンで、MCP サーバーを登録) |
| リポジトリ | `alice` は origin `https://github.com/example/iv-shared-a.git`、`bob` は `https://github.com/example/iv-shared-b.git` (別のリポジトリのため、`shared` でしか届かない) |
| CLI | Claude Code 2.1.331。`claude -p --mcp-config <ユーザーごとの設定> --strict-mcp-config --allowedTools "mcp__agent-lounge__*,Bash"` |

Table: 実機の確認の環境

指示は、「使い方に従う」「セッション名で参加する」と、行う操作です。スコープの値 (`shared`)、ツール名 (`chat_room`)、引数の形は、指示に含めていません (「全ユーザーで共有するスコープ」と書いた)。

| 最終仕様の 9.2 | 確かめたこと | 結果 |
| --- | --- | --- |
| 1 | `alice` から `bob/bob-cc` への `shared` のセッション宛てが届く (recipients `["bob/bob-cc"]`)。`bob` の `shared` の `lounge` への投稿が `alice` に届く | 検証済 |
| 2 | `alice` が `Release` を作ると `release` になり、参加する (`created: true`、`joined: true`)。参加者のいない時の投稿は誰にも届かない。`bob` が参加した後の投稿は `bob` だけに届き、`bob` の受信箱にルーム名 `release` で現れる。`bob` が退出した後の投稿は誰にも届かない。一覧の `members` が 0 | 検証済 |
| 3 | 参加の前の `bob` が、`release` ルームの、参加の前のメッセージを `chat_history` で読める | 検証済 |
| 4 | `alice` が `shared` に保存したメモリを、`bob` が「ビルド」で検索して見つける | 検証済 |

Table: 実機の確認の結果

サーバーの記録 (`message list --details`、`memory list --scope shared`) は、上の結果と一致しました。`export` の出力は、`chat/shared/room-release.md`、`chat/shared/room-lounge.md`、`chat/shared/direct-...md`、`memory/shared.md` でした。

すべて、1 回目の実行で通りました。

## 3. 確認の中で見つけた問題

Claude Code 2.1.331 で、`lounge_help` の結果が空 (`{}`) として扱われ、案内の本文を読めませんでした (4 回の実行のすべてで、Claude Code がそう報告)。サーバーの応答を `curl` で確かめると、文字列の結果に案内の本文 (19,222 バイト) があり、構造化された結果は空のオブジェクトでした。これは、案内を 2 回渡さないための意図した形です (`internal/tools/help.go` の `HelpOutput`)。Claude Code が、構造化された結果だけをモデルに渡したと考えられます。

`lounge_help` は、この案件で変えていません。この問題は、この案件の範囲外として、残課題一覧の BL-049 に記録しました。

このため、2 章の確認は、使い方の案内ではなく、サーバーの説明文 (MCP の初期化の指示) とツールの説明から、Claude Code が操作を選べることを確かめたものです。
