# core-apiserver

ポケモンカードの対戦記録を管理するWebサービス **バトレコ (vsrecorder)** のコア機能を提供するAPIサーバです。

公式サイトで作成したデッキコードを使ったデッキ登録、ジムバトル・トレーナーズリーグ・シティリーグといった公式イベントに紐づく対戦記録の作成、使用デッキ／対戦相手デッキや勝敗の記録などの機能を提供します。

## 主な機能

- **対戦記録の管理** — 対戦記録 (Record)、マッチ (Match)、ゲーム (Game) の作成・編集
- **デッキ管理** — デッキ登録、公式デッキコードによるデッキ情報の取得
- **イベント連携** — 公式イベント / 非公式イベント / Tonamelイベントとの紐付け
- **統計情報** — ユーザー統計、デッキ使用率、対戦相手デッキ使用率、週次デッキ使用率など
- **バッジ・実績** — バッジ付与、環境バッジ、連勝 (Streak) 記録、称号 (Designation) の評価
- **プレイヤー連携** — バトレコユーザーIDとポケモンカードゲーム プレイヤーズクラブIDの紐付け（キルスイッチ付き）
- **通知** — ユーザー向け通知の管理

## 技術スタック

| 項目             | 内容                                                     |
| ---------------- | -------------------------------------------------------- |
| 言語             | Go 1.25.7                                                |
| Webフレームワーク | [Gin](https://github.com/gin-gonic/gin)                  |
| ORM              | [GORM](https://gorm.io/) (PostgreSQL ドライバ)           |
| データベース     | PostgreSQL                                               |
| 認証             | JWT (HMAC署名) / Bearerトークン                          |
| オブジェクトストレージ | AWS S3 (aws-sdk-go-v2)                              |
| ID生成           | UUID / ULID                                              |
| テスト           | testify, go-sqlmock, uber-go/mock (mockgen)              |
| コンテナ         | Docker (distroless イメージ)                             |

## アーキテクチャ

クリーンアーキテクチャ（レイヤードアーキテクチャ）を採用しています。

```
cmd/
  core-apiserver/      # APIサーバのエントリポイント (main.go)
  backfill-*/          # データバックフィル用のバッチ
  sync-pokemon-avatars/, repair-streaks/  # 運用バッチ

internal/
  controller/          # HTTPハンドラ、ルーティング、認証/認可、DTO、バリデーション
  usecase/             # ビジネスロジック（ユースケース）
  domain/              # エンティティ、リポジトリインタフェース、ドメインエラー
  infrastructure/      # リポジトリ実装、DBモデル、PostgreSQL接続
  ratelimit/           # レートリミット
  mock/                # mockgen による自動生成モック
  testutil/            # テストユーティリティ

db/schema.sql          # データベーススキーマ
adr/                   # アーキテクチャ・デシジョン・レコード
```

依存の方向は `controller → usecase → domain ← infrastructure` となっており、`domain` 層で定義したリポジトリインタフェースを `infrastructure` 層が実装します。

## API

- ベースパス: `/api/v1beta`
- ポート: `8914`

主なリソースエンドポイント（`/api/v1beta` 以下）:

| パス                     | 内容                       |
| ------------------------ | -------------------------- |
| `/users`                 | ユーザー                   |
| `/records`               | 対戦記録                   |
| `/matches`               | マッチ                     |
| `/decks`, `/deckcodes`   | デッキ / デッキコード      |
| `/official_events`       | 公式イベント               |
| `/unofficial_events`     | 非公式イベント             |
| `/tonamel_events`        | Tonamelイベント            |
| `/stats`                 | ユーザー統計               |
| `/deck_usage`, `/opponent_deck_usage`, `/weekly_usage` | デッキ使用率統計 |
| `/kizuna`                | デッキごとのきずなLv.      |
| `/badges`, `/environment_badges` | バッジ / 環境バッジ |
| `/streak`                | 連勝記録                   |
| `/designations`          | 称号                       |
| `/notifications`         | 通知                       |
| `/usersplayers`          | プレイヤーズクラブID連携   |
| `/championship_series`, `/cityleague_schedules`, `/cityleague_results`, `/standard_regulations`, `/environments` | マスタ／参照系 |

認証が必要なエンドポイントは `Authorization: Bearer <JWT>` ヘッダを要求します。

## バッチ処理 (cmd)

`cmd/` 以下には、APIサーバ本体 (`core-apiserver`) とは別に、運用・データ整備のために単体で実行するコマンドラインプログラムを配置しています。用途に応じて次の3種類に分かれます。

各バッチの詳細な仕様・判定基準・冪等性は、それぞれの `main.go` 冒頭のパッケージコメントに記載しています。

### 初期投入バッチ（一回限り）

機能導入前から既に達成条件を満たしていた既存ユーザーに対し、データを遡って補完するためのバッチです。導入時に一度だけ実行することを想定しています。いずれも `-dry-run`（デフォルト `true`。書き込みせず差分のみ確認）と `-user-id`（特定ユーザーのみ対象）フラグを持ちます。

| コマンド | 説明 |
| -------- | ---- |
| [`backfill-user-badges`](cmd/backfill-user-badges/) | オンボーディング系バッジ（はじめの一歩: signup / first_deck / first_record / first_match）を、実際の達成日時を計算して `user_badges` へ遡って付与します。API処理内でリアルタイム付与される仕様のため、導入前の既存ユーザーには付与されていない欠落分を補完します。通知は作成しません。 |
| [`backfill-user-environment-badges`](cmd/backfill-user-environment-badges/) | 環境バッジ（対戦環境ごとの初回対戦バッジ）を、対戦の基準日時から環境を判定し `user_environment_badges` へ遡って付与します。判定基準の変更後に再実行して達成日時を更新し直せるよう、既存行は上書きします。通知は作成しません。 |
| [`backfill-notifications`](cmd/backfill-notifications/) | 通知機能の導入前から達成済みだったバッジ・称号・ランク・環境バッジの実績を、「既読済みの通知履歴」として `notifications` へ遡って作成します。永続化済みの実績は実際の達成日時を、ライブ集計する実績は日付を遡って走査した到達日を通知日時に使います。誤って複数回実行しても通知が重複しないよう冪等性を持たせています。 |

### 運用バッチ

サービス運用のなかで定期実行、あるいは必要に応じて実行するバッチです。

| コマンド | 説明 |
| -------- | ---- |
| [`sync-pokemon-avatars`](cmd/sync-pokemon-avatars/) | 公式サイト（プレイヤーズクラブ）のアバター一覧API から `avatarList` を取得し、`pokemon_avatars` テーブルへ upsert します。新規アバターの追加やタイトル・画像URLの変更に追随するため、定期実行を想定しています。 |
| [`repair-streaks`](cmd/repair-streaks/) | 何らかの理由で `user_streaks` が現存の `records` と食い違った場合に、`records` の日付からゼロから週次ストリーク状態を再計算し、行ごと上書きして復旧します。`-dry-run` / `-user-id` フラグを持ちます。 |

### 調査・確認ツール

データの整合性を突合・検算するためのツールです。既定では読み取り専用で、DBやFirebaseに書き込みを行いません。差異検出時に終了コード1を返す `-exit-code` フラグを持ち、CI・定期実行での監視に利用できます。

| コマンド | 説明 |
| -------- | ---- |
| [`check-firebase-users`](cmd/check-firebase-users/) | Firebase Authentication 上のユーザーと、DB (`users`) の有効なユーザー (`deleted_at IS NULL`) を突合し、差異（Firebaseのみ存在＝退会済み／登録未完了、DBのみ存在）を検出します。読み取り専用で一切書き込みません。 |
| [`check-deleted-users-data`](cmd/check-deleted-users-data/) | 退会したユーザー (`deleted_at IS NOT NULL`) が作成したデータが残っていないかを検算し、削除漏れ (NG) / 未対応 (WARN) / 参照 (INFO) に分類して表示します。既定は確認のみで、`-delete` を指定したときに限り退会処理と揃えた方法で削除します。 |

## セットアップ

### 前提

- Go 1.25.7 以上
- PostgreSQL
- Docker / Docker Compose（コンテナ実行の場合）

### 環境変数

`.env.sample` をコピーして `.env` を作成し、値を設定します。

```sh
cp .env.sample .env
```

| 変数名                          | 説明                                                      |
| ------------------------------- | --------------------------------------------------------- |
| `VSRECORDER_JWT_SECRET`         | JWT署名に使用するシークレット                             |
| `DB_HOSTNAME` / `DB_PORT`       | PostgreSQL のホスト / ポート                              |
| `DB_USER_NAME` / `DB_USER_PASSWORD` | PostgreSQL の接続ユーザー / パスワード               |
| `DB_NAME`                       | データベース名                                            |
| `AWS_REGION`                    | AWS リージョン                                            |
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | AWS 認証情報 (S3用)                         |
| `USERS_PLAYERS_LINKING_ENABLED` | プレイヤーID連携機能のキルスイッチ。`false` で機能停止（未設定または `false` 以外で有効） |

### 起動

```sh
# ローカル実行
make run

# ビルド
make build

# テスト（カバレッジ・レースディテクタ付き）
make test

# モック生成
make mockgen
```

### Docker

```sh
# イメージのビルド & プッシュ
make image

# 起動 / 停止
make up
make down

# デプロイ（pull して更新）
make deploy

# ログ確認
make log
```

## テスト

```sh
make test   # go test -v -cover -race ./...
```

リポジトリ層は go-sqlmock、ユースケース／コントローラ層は mockgen で生成したモックを用いてテストします。

## ライセンス

本リポジトリのライセンスについてはプロジェクト管理者に確認してください。
