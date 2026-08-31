# AGENTS.md

このファイルはコーディングエージェント（Claude Code など）がこのリポジトリで作業するための
ガイドです。人間向けの機能概要・セットアップ手順は [README.md](README.md) を参照してください。

## リポジトリ概要

ポケモンカード対戦記録サービス **バトレコ (vsrecorder)** のコアAPIサーバ。
Go 1.25.7 / Gin / GORM (PostgreSQL) 製。モジュール名は `github.com/vsrecorder/core-apiserver`。
APIのベースパスは `/api/v1beta`、待ち受けポートは `8914`。

## コマンド

```sh
make test              # go mod tidy && make lint-tx && UTC と Asia/Tokyo の両方で go test -v -cover -race ./...
make lint-tx           # infrastructure の書き込みが dbFromContext を経由しているかの検査
make run               # go run cmd/core-apiserver/main.go
make build             # go build -o /dev/null ...（バイナリは作らないコンパイル確認）
make mockgen           # domain/repository・usecase のインタフェースからモックを再生成
make integration-test  # 使い捨てPostgresを起動しdb/schema.sqlを適用してTestIntegration*を実行
```

単体で実行する場合:

```sh
go test ./internal/controller/                          # パッケージ単位
go test -run TestRecordController ./internal/controller/ # トップレベルのテスト関数単位
go test -run 'TestRecordController/Create' ./internal/controller/  # シナリオ単位
go test -run 'TestRecordController/Create/正常系' ./internal/controller/  # サブテスト単位
```

- CI (`.github/workflows/`) は `go test -v -cover -race ./...` を **UTC と Asia/Tokyo の
  両方** で実行する（理由は「テストで時刻を作るときの Location」を参照）。
  `make integration-test` はDockerが必要なためCIでは動かない。ローカルで回すこと。
- リポジトリ層でSQLやテーブル・カラム名を変更したときは、sqlmockのテストが通っても
  スキーマとの不整合は検出できないため `make integration-test` も実行する。

## アーキテクチャ

クリーンアーキテクチャ。依存の方向は `controller → usecase → domain ← infrastructure`。
`domain/repository` のインタフェースを `infrastructure` が実装し、`cmd/core-apiserver/main.go`
が全ての依存を手で組み立てて注入する（DIコンテナは使わない）。

リソース（record / deck / match / user ...）ごとに、各層に同名のファイルが並ぶ構成になっている。
1リソースを追える最短ルートは、`record` を各層で横断して読むこと。

```
cmd/core-apiserver/main.go   # 全リソースのDI組み立て + RegisterRoute + graceful shutdown
internal/
  controller/                # ルーティングとハンドラ（リソースごとに1ファイル）
    auth/authentication/     # JWT検証。Required / Optional の2種類のミドルウェア
    auth/authorization/      # 所有者チェック等。repositoryを直接引いてuid一致を検証する
    validation/              # クエリ/ボディの検証 → helperでgin.Contextへ格納
    helper/                  # gin.Contextの型付きSet/Getと、クエリのパース関数
    dto/                     # リクエスト/レスポンスのJSON構造体
    presenter/               # entity → dto の変換
    apierror/                # HTTPステータスを内包した応答用エラー
  usecase/                   # ビジネスロジック。entityを返す
  domain/entity/             # ドメインエンティティ（GORMタグを持たない）
  domain/repository/         # リポジトリインタフェース
  domain/apperror/           # 層をまたぐセンチネルエラー
  infrastructure/            # リポジトリ実装（GORM / S3 / 外部サイトのスクレイピング）
    model/                   # GORMモデル（DBスキーマに対応）
  httpclient/                # タイムアウト付きの共用HTTPクライアント
  logging/                   # 全レイヤー共通のログ土台（ContextHandler・各種ヘルパー）
  ratelimit/                 # インメモリの固定ウィンドウ制限（単一プロセス内のみ）
  mock/                      # mockgen生成物。手で編集しない
db/schema.sql                # DBスキーマの正（マイグレーションツール・AutoMigrateは使わない）
adr/                         # 設計判断の記録と、アルゴリズム仕様書
config/crontab               # cmd配下のバッチの定期実行定義（ホストOSのcron）
```

### リクエストの流れ

`RegisterRoute` でミドルウェアを次の順に並べるのが全リソース共通の型:

```
authentication → authorization → validation → ハンドラ
```

- 各ミドルウェアは検証結果を `helper.SetXxx(ctx, ...)` でgin.Contextへ入れ、ハンドラは
  `helper.GetXxx(ctx)` で受け取る。ハンドラ側でクエリやボディを直接読まない。
- ミドルウェアはエラー時に `apierror.ErrXxx.JSON(ctx)`（内部で `ctx.Abort()`）を呼んで
  打ち切る。`ctx.Next()` は明示的に呼ばない実装になっている。
- ハンドラはusecaseを呼び、`presenter.NewXxxResponse(...)` でdtoへ変換して返す。
- `authentication.OptionalAuthenticationMiddleware` はトークン無しを許容し `uid` を空文字にする。
  `/records` の GET のように「未認証なら公開一覧、認証済みなら自分の一覧」を返すエンドポイントは、
  ハンドラを2つ登録して各々が `helper.GetUID(ctx)` の空/非空で自分の担当かを判定している。

### エラーの扱い

- `domain/apperror`: 「何が起きたか」を表すセンチネルエラー。層をまたいで `errors.Is` で判定する。
- `controller/apierror`: 「どのHTTPステータスで返すか」。ハンドラでのステータス直書きはしない。
- infrastructure層は `wrapError()` で `gorm.ErrRecordNotFound` → `apperror.ErrRecordNotFound`
  に変換する。上位層にgormの型を漏らさない。
- 新しいエラー種別を足すときは apperror にセンチネルを定義し、対応するHTTPステータスを
  apierror 側にコメント付きで用意する。

### ログ

ログは `log/slog` のJSON出力。`internal/logger.go` の `InitLogger` が
`logging.ContextHandler` でラップしたハンドラを作り、`main` が `slog.SetDefault` で
既定のロガーとして設定する。各層は `*slog.Logger` をDIで受け取らず、この既定ロガーを
使う（コンストラクタのシグネチャを変えずに全層でログを出せるようにするため）。

`request_id` / `uid` は context に載せ、`ContextHandler` が全レコードへ自動付与する。
そのため**ログを出すときは必ず ctx を渡す**（`ErrorContext` など `*Context` 系を使う）。

- `request_id`: `internal.RequestIDMiddleware` が `c.Request` の context へ載せる。
- `uid`: `helper.SetUID` が `c.Request` の context へ載せる。
- コントローラは `context.Background()` ではなく **`ctx.Request.Context()`** を下層へ渡す。
  これを忘れると request_id が伝播せず、層をまたいでリクエストを追えなくなる。

各層のログの出し方:

| 層 | 出し方 | レベル |
| --- | --- | --- |
| controller | `apierror.ErrXxx.JSON(ctx, err)` が自動でログ出力する | 5xx=Error / 4xx=Warn |
| usecase | `logError(ctx, err)`（継続する場合は `logWarn(ctx, err)`） | Error / Warn |
| infrastructure | `logError(ctx, err)` | Error |

- エラー応答はすべて `apierror.Error.JSON` を通るため、コントローラ層は個別にログを
  書かなくてよい。原因エラーがある場合は第2引数へ渡す（省略可）。
- `logError` / `logWarn` は呼び出し元のメソッド名(`operation`)とソース位置を runtime から
  解決するので、呼び出し側でメッセージや関数名を書く必要はない。
- 「対象が存在しない」(`apperror.ErrRecordNotFound` / `gorm.ErrRecordNotFound`) は障害では
  ないため `logError` が自動で Debug へ落とす。Error レベルを調査が必要なものだけに保つ。
- 上記に加えて文脈固有の属性を出したい場合のみ、明示的に `slog` を呼ぶ
  （例: `infrastructure/deck_asset.go` の `deck_code` / `request_url`）。
  その際 `request_id` / `uid` は `ContextHandler` が付けるため**重複して指定しない**。

`cmd/core-apiserver` の出力はすべてJSONに揃えてある。壊さないこと:

- `main()` は最初に `InitLogger` + `slog.SetDefault` を実行する。設定不備やDB接続失敗など
  最も見たい起動時エラーもJSONで出すため、`godotenv.Load()` より前に置いている。
  `log` パッケージ（`log.Printf` など）は使わない。
- panicは `gin.Recovery()` ではなく `internal.RecoveryMiddleware` で捕捉する。
  gin側はスタックを独自形式のテキストで出すため、書き出し先をnilにして黙らせ、
  slogでJSONとして出し直している。応答はgin同様ボディ無しの500。
- GORMのロガーは `logger.Silent`（`infrastructure/postgres`）。SQLログを出したくなった場合も
  GORM既定の形式ではなくslog経由にすること。

### entity と model

`domain/entity`（ドメイン表現）と `infrastructure/model`（GORMタグ付きのDB表現）は別物で、
infrastructure層が相互に詰め替える。フィールドの並びやコンストラクタの引数順は両者で
一致していないことがあるため、詰め替え箇所は引数の位置ではなく名前で確認すること。

### 横断的なusecase

`BadgeEvaluation` / `DesignationEvaluation` / `EnvironmentBadgeEvaluation` は、Record・Match・
Deck・User の各usecaseへ注入され、書き込み処理の中でバッジ・称号・通知を評価する。
記録や対戦の作成・削除パスに手を入れるときは、これらの評価呼び出しの有無を必ず確認する。

バッジには2種類ある。オンボーディング系は書き込み時に `user_badges` へ永続化し、
マイルストーン系はシーズンごとに再獲得できるため一覧取得時にライブ集計する。

### シーズン・期間・週

- 「今シーズン」は `championship_series` テーブルが正。`usecase/season.go` の
  `CurrentSeasonLabel` / `PeriodDateRange` を使い、年やシーズン境界をハードコードしない。
  season識別子は `championship_series.id` から接頭辞 `series_` を除いた文字列（例 `"2026"`）。
- 期間は `from_date`（0時始まり）〜 `to_date` の翌日0時（exclusive上限）の半開区間で扱う。
  environment / season / standard_regulation が複数指定された場合は期間の交差を取る。
- **`standard_regulation_id` と `regulation_id` は別物**。前者は『H・I・J』などのマークの
  組み合わせとその適用期間（`standard_regulations`）で、統計APIでは期間の絞り込みに使う。
  後者は記録のレギュレーション区分（`regulations` = スタンダード / エクストラ / 殿堂）で、
  期間とは直交する絞り込み。統計APIでは未指定＝全レギュレーション、`records.regulation_id`
  と突き合わせる。webapp は既定でスタンダード（`regulation_id=1`）を送る。
- 週は月曜始まり（`usecase/week.go`）。週次デッキ使用率の `week` クエリは月曜日の `YYYY-MM-DD`。
- ユーザー統計API（`/users/:id/stats` / `deck_usage` / `opponent_deck_usage`）の `week` は
  週内の任意日 `YYYY-MM-DD`（`weekRange` が月曜へ正規化する）。期間指定の優先順は
  **week > year_month > season**。environment / standard_regulation との交差は他と同じ。

### 書き込みの整合性

1つの操作で複数テーブルに書くときは、**`repository.TransactionManager` で囲む**。囲まないと
片方だけコミットされ、「記録は保存されたのにタグが無い」「バッジは付与されたのに通知が無い」
といった食い違いが残る。バッジの付与と獲得通知のように、片方が欠けると**次回以降その処理を
スキップしてしまう**組み合わせは特に注意する（付与済みと判定され、通知が二度と作られない）。

**infrastructure の書き込みは必ず `dbFromContext(ctx, i.db)` を経由する。** `i.db` を直接
使うと、usecase 側でトランザクションに包んでも参加せず、上記の食い違いが起きる。読み取りは
対象外。`make lint-tx`（`make test` から呼ばれる）が機械的に検査する。

**本体を保存したあとの付随処理（バッジ評価・ストリーク再計算・通知・外部通信）の失敗で、
作成・更新そのものを失敗させない。** ここでエラーを返すと、保存できているのにクライアントには
「失敗」と見えるため、ユーザーが作り直して同じデータが二重に登録される（IDはサーバ採番なので
重複を防げない）。失敗はログに残し、回復は `cmd/repair-streaks` と `cmd/backfill-notifications`
に委ねる。records / matches / decks / deck_codes の作成・更新はこの方針で揃えてある。

### トランザクション

`repository.TransactionManager.Do(ctx, fn)` を使う。実装は `*gorm.DB` をcontextへ埋め込み、
各リポジトリは `dbFromContext(ctx, i.db)` で「トランザクション中ならtx、なければ自前のdb」を
選ぶ。リポジトリに新しいメソッドを足すときも `dbFromContext` 経由にすること
（既存メソッドには `i.db` を直接使うものが残っているが、新規は揃える）。

### 外部通信

外部サイト（ポケモンカード公式・Tonamel）へのアクセスは必ず `internal/httpclient` を経由する。
`http.DefaultClient` / `http.Get` はタイムアウトが無く、外部サイトの遅延がAPIサーバの停止に
波及するため使わない。取得先URLはテストでhttptestサーバへ差し替えられるよう、定数ではなく
パッケージ変数にしてある（`tonamelEventBaseURL` など）。同様にS3クライアントも
インタフェース越しに扱い、実S3へ繋がずにテストできるようにしている。

## テストの規約

- サブテスト名は日本語で `正常系_...` / `異常系_...` を使う。
- controller / usecase 層: `internal/mock/mock_repository`・`mock_usecase` のmockgen生成モックと
  `httptest` を使う。`domain/repository` や `usecase` のインタフェースを変更したら
  **`make mockgen` を実行してモックを再生成する**（Makefileのmockgenターゲットへ新しい
  ソースの行を追加するのも忘れないこと）。
- infrastructure層: `setupSqlmockDB(t)`（`sqlmock_helper_test.go`）でsqlmock接続のgorm.DBを作る。
  GORMが更新する `updated_at` / `deleted_at` は `AnyTime{}` マッチャで受ける。
- 実DBに対するスモークテストは `internal/infrastructure/integration_test.go` に `TestIntegration*`
  として書く。`VSRECORDER_TEST_DATABASE_URL` 未設定時は自動でスキップされる。
- usecase 層のテストから**別の usecase を差し替えるときは `mock_usecase` を import できない**
  （`mock_usecase` が `usecase` を import しているため import cycle になる）。`stubBadgeEvaluation` /
  `stubPushNotifier`（`push_notifier_stub_test.go`）のように同パッケージ内の手書きスタブを使う。
- 現在時刻に依存するロジックは `timeNow` パッケージ変数を経由する。テストからは
  `overrideTimeNow(t, 固定時刻)`（usecase）で差し替える。パッケージ変数を書き換えるため、
  これを使うテストは並列実行しない。
- JWTが必要なテストは `internal/testutil` の `GenerateJWTSecret` / `GenerateJWT` を使う。

### テストで時刻を作るときの Location

CIのrunnerは **UTC**、開発機と本番は **JST** で動く。テストの中で `time.Local` を使うと
同じコードが環境で別の結果になり、片方でしか落ちないテストができる。実際に、JSONを往復する
値を `time.Local` で作ったテストが手元（JST）では通るのにCI（UTC）で落ちたことがある
（`time.Parse` はオフセットがローカルと一致するときだけ `Local` を返すため、復元後の
Location が環境によって変わる）。

- **JSONやDBを往復する時刻は `time.UTC` で固定する。** リクエストの突き合わせや
  `require.Equal` での構造体比較は Location まで見る。
- **タイムゾーン差そのものを検証したいときは `time.FixedZone` で明示的に差を作る。**
  `time.Local` を使うと、UTCのCIでは差が消えて検証にならない。
- 日付の集合を組み立てて内部で完結する比較だけなら `time.Local` でもよい。

`make test` は UTC と Asia/Tokyo の両方で回るので、手元でこの種のズレに気づける。
CI（`.github/workflows/`）も同じく両方で実行する。

## リソースを追加・変更するときに触るファイル

1. `db/schema.sql` にテーブル定義（マイグレーションツールは無いので手で反映する）
2. `internal/infrastructure/model/xxx.go` — GORMモデル
3. `internal/domain/entity/xxx.go` — エンティティ
4. `internal/domain/repository/xxx.go` — リポジトリインタフェース
5. `internal/infrastructure/xxx.go` — 実装（+ `xxx_test.go` はsqlmock）
6. `internal/usecase/xxx.go` — ユースケース（インタフェース + 実装 + `XxxParam`）
7. `internal/controller/dto/xxx.go`・`presenter/xxx.go`・`validation/xxx.go`・
   必要なら `auth/authorization/xxx.go`・`helper/key.go` にSet/Get
8. `internal/controller/xxx.go` — `XxxPath` 定数、`RegisterRoute`、ハンドラ
9. `Makefile` のmockgenターゲットに行を追加 → `make mockgen`
10. `cmd/core-apiserver/main.go` に `controller.NewXxx(...).RegisterRoute(relativePath)` を追加

## cmd配下のバッチ

APIサーバ本体はdistrolessコンテナで動くためコンテナ内でバッチは実行できない。ホスト上で
`go build` したバイナリを叩く（`config/crontab` 参照）。バッチを追加・変更するときの決まり:

- `main.go` 冒頭のパッケージコメントに、目的・判定基準・冪等性・使い方（実行例）を書く。
  各バッチの仕様はここが正。
- 書き込みを伴うバッチは `-dry-run`（**デフォルト `true`**）と、対象を絞る `-user-id` を持たせる。
- 調査・確認ツールは既定で読み取り専用にし、差異検出時に終了コード1を返す `-exit-code` を持たせる。
- `.env` を `godotenv.Load()` で読むため、作業ディレクトリに依存する。cronからは必ず `cd` させる。

## 設計判断の記録

仕様の背景・アルゴリズムは `adr/` に置く。`adr/*.md` は「なぜそう決めたか」（ADR）と
「どう動くか」（アルゴリズム仕様書）が混在しているので、既存の粒度に合わせて追記する。
`adr/security-review-2026-07-18.md` には対処済みのセキュリティ指摘とその理由がまとまっており、
認証・入力処理・タイムアウト周りを触る前に目を通すこと。

## 変更時に壊してはいけない不変条件

セキュリティ精査で入れた防御なので、外さないこと（`adr/security-review-2026-07-18.md`）:

- `VSRECORDER_JWT_SECRET` は32文字以上必須。未設定・短すぎる場合は起動時にfail fastする。
  JWT検証側でも空鍵を拒否する二層防御になっている。
- JWTは `exp` 必須（`jwt.WithExpirationRequired()`）、issuerは `vsrecorder-webapp`、HMAC署名のみ許可。
- HTTPサーバのタイムアウト（ReadHeader 10s / Read 30s / Write 60s / Idle 120s）と
  `BodySizeLimitMiddleware`（1MiB）は全ルートに効かせる。
- `r.SetTrustedProxies(nil)` とCORSのオリジン許可リストは明示指定を維持する。
- `USERS_PLAYERS_LINKING_ENABLED=false` はプレイヤーID連携のキルスイッチ。未設定または
  `false` 以外なら有効という判定（`!= "false"`）を変えない。

### 退会したユーザのデータを残さない

退会（`usecase.User.Delete`）は、そのユーザに紐づくデータを**すべて**削除する。対象は
「`user_id` を持つテーブル」と「それらへFKで繋がる中間テーブル」の全部で、論理削除
（`deleted_at`）を持つテーブルは論理削除、持たないテーブルは行ごと物理削除する。
唯一の例外は `matches.opponents_user_id` で、これは他のユーザが作った対戦記録の中で
対戦相手として参照されているもの。他人のデータなので消さない。

**ユーザに紐づくテーブルを追加したら、次の3箇所を必ず揃えて更新すること。**

1. そのテーブルのリポジトリに `DeleteByUserId` を足し、`usecase.User.Delete` から呼ぶ
2. `internal/usecase/user_test.go` の `expectDeleteAllUserData` に足す
   （呼び忘れるとgomockの未消化EXPECTで落ちる）
3. `cmd/check-deleted-users-data` の `specs` と、`main_test.go` の `TestSpecs` の一覧に足す

中間テーブルは付与先（deck/record/match など）のリポジトリ側で、親を論理削除する**前に**
消す。GORMがサブクエリへ `deleted_at IS NULL` を付けるため、親を先に消すと子が消し残る。

## コーディングスタイル

- コメント・ドキュメント・テスト名は日本語。コメントには「何をしているか」ではなく
  **なぜそうしたか**（採用理由、避けたい事象）を書く既存の慣習に合わせる。
- ログは `log/slog`。リクエストログは `internal/middleware.go` が `request_id` 付きで出力する。
  各層でのログの出し方は「[ログ](#ログ)」を参照。ログメッセージ本文だけは英語で書く
  （コメント・ドキュメントは日本語という他の規約とは異なる、既存の慣習）。
- IDはULID（`generateId()`）。ユーザーIDだけはFirebase Authenticationのuidをそのまま使う。
- 秘密情報は `.env`（gitignore済み）。`.env.sample` に変数を追加したら説明コメントも書く。
