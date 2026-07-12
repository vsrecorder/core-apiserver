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
| `/badges`, `/environment_badges` | バッジ / 環境バッジ |
| `/streak`                | 連勝記録                   |
| `/designations`          | 称号                       |
| `/notifications`         | 通知                       |
| `/usersplayers`          | プレイヤーズクラブID連携   |
| `/championship_series`, `/cityleague_schedules`, `/cityleague_results`, `/standard_regulations`, `/environments` | マスタ／参照系 |

認証が必要なエンドポイントは `Authorization: Bearer <JWT>` ヘッダを要求します。

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
