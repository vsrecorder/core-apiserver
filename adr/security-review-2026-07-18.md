# セキュリティ精査レポート — core-apiserver

- **対象**: vsrecorder/core-apiserver (main)
- **作成日**: 2026-07-18
- **範囲**: 認証・認可・入力処理・DoS耐性・情報漏洩（Goファイル 417）

## サマリ

| 区分 | 件数 | 内訳 |
| --- | --- | --- |
| 修正済み | 7 | 高 1 / 中 5 / 低 1 |
| 未対応（低リスク・要判断） | 3 | sslmode / 鍵共有 / レートリミット |
| 精査して問題なし | 9 | SQLi・IDOR・認証・情報漏洩 ほか |

検証: `go build ./...` / `go vet ./...` / `go test -count=1 ./...` すべて通過。挙動は一時テストで直接確認後に削除。

---

## 修正済み（7件）

### 1. [高] JWT秘密鍵が未設定でも起動し、空鍵で偽造トークンを受理

環境変数 `VSRECORDER_JWT_SECRET` 未設定時に空鍵 `[]byte("")` で署名検証が行われ、任意の `uid` を名乗るトークンを誰でも偽造できた（テストで再現確認）。認可層は正しく `uid` 一致を見ているが、その `uid` 自体が偽造可能なため防御にならない。

- **対処**: 起動時に長さ32文字以上を検証し fail fast。加えて認証・チャレンジ双方のパーサ自身でも空鍵を拒否（呼び出し順に依存しない二層防御）。
- **箇所**: `cmd/core-apiserver/main.go`, `internal/controller/auth/authentication/authentication.go`, `internal/usecase/user_player_challenge.go`

### 2. [中] JWTの exp クレームが必須化されていない

有効期限を持たないトークンが無期限に有効だった。発行側の変更やトークン漏洩時に失効が効かない。

- **対処**: `jwt.WithExpirationRequired()` を認証・チャレンジ両方に追加。exp なしは 401。
- **箇所**: `authentication.go`, `user_player_challenge.go`

### 3. [中] HTTPサーバにタイムアウトがなく Slowloris に無防備

ヘッダ／ボディを小刻みに送り続ける接続でコネクションを占有され続ける可能性があった。

- **対処**: ReadHeader 10s / Read 30s / Write 60s / Idle 120s を設定。
- **箇所**: `cmd/core-apiserver/main.go`

### 4. [中] リクエストボディのサイズ上限がない

`ShouldBindJSON` がボディ全体をメモリに読むため、巨大JSONでメモリを圧迫できた。

- **対処**: `BodySizeLimitMiddleware`（1MiB）を全ルートに適用。2MiBボディが 400 になることを確認。
- **箇所**: `internal/middleware.go`, `main.go`

### 5. [中] 外向きHTTP呼び出しにタイムアウトがない

`http.Get` / `http.PostForm` は `DefaultClient`（タイムアウト無し）を使用。外部サイト（ポケカ公式・Tonamel）の遅延で goroutine が滞留する恐れがあった。

- **対処**: `internal/httpclient` を新設し10秒タイムアウトを付与、該当5箇所を全置換。
- **箇所**: `internal/httpclient/`, `deck_asset.go`, `tonamel_event.go`, `player_account.go`, `validation/util.go`

### 6. [中] 文字列長のバリデーションがない

上限のない `TEXT` 列（各種 memo, tcg_meister_url）に無制限の書き込みが可能だった。

- **対処**: DBスキーマに対応した定数を定義し user / deck / deck_code / record / match / unofficial_event に追加。判定は `utf8.RuneCountInString` で行い、日本語（VARCHAR は文字数制限）を誤って弾かないことをテストで確認。
- **箇所**: `validation/util.go` ほか

### 7. [低] ImageURL のスキームを検証していない

`GET /users/:id` は認証不要で誰でも取得できるため、`javascript:` や `data:` を保存すると描画側の実装次第で XSS に繋がり得た。

- **対処**: `isValidImageURL` で `https` かつホスト有りに限定。webapp が実際に送る値（デフォルトアイコン / CDNアップロード結果、いずれも https）が通り、攻撃ベクタ8種が 400 になることを確認。ホスト許可リストは既存データ・CDN設定への結合を避けて不採用。
- **箇所**: `validation/user.go`, `validation/util.go`

---

## 未対応（低リスク・運用判断が必要）（3件）

### A. [低 / 構成次第] DB接続が sslmode=disable（平文・証明書未検証）

現状「本番DBはAPIサーバと同一ホスト」との確認により、信頼できない経路を通らないため据え置き。**将来DBを外部（RDS等）へ移す場合は低リスクではなくなる。**

- **推奨**: 外部化時に `DB_SSLMODE` を環境変数化し `require` 以上（マネージドDBは `verify-full`）へ。
- **箇所**: `infrastructure/postgres/postgres.go`

### B. [低] 認証トークンとチャレンジトークンが同一鍵を共有

取り違えは両者の issuer 検証で成立せず、実害は小さい。本質的な論点は HS256（対称鍵）ゆえ「検証できる者は発行もできる」点で、鍵が webapp と APIサーバの2箇所に存在する。

- **推奨**: 中長期で RS256/ES256 へ移行すれば APIサーバは公開鍵のみ保持で足りる。webapp との協調が必要なため優先度は低い。
- **箇所**: `authentication.go`, `user_player_challenge.go`

### C. [低] レートリミットがインメモリ実装

プレイヤーID列挙を抑止する固定ウィンドウ方式。**単一インスタンス前提で機能**しており、現構成（単一コンテナ）では有効。スケールアウトすると実効上限が台数倍に緩む点が暗黙の前提になっている。

- **推奨**: 多インスタンス運用に移る際に Redis 等の共有ストアへ。それまでは現状維持で妥当。
- **箇所**: `internal/ratelimit/ratelimit.go`, `validation/user_player.go`

---

## 精査して問題なしと確認（9件）

- **SQLインジェクション**: 生SQL（Raw/Exec）はゼロ。全て GORM のクエリビルダ経由。`record.go` のカーソル条件も固定文字列＋プレースホルダのみで、ユーザー入力は全てパラメータバインド。
- **認可 / IDOR**: リソース単位でミドルウェアを一貫適用。notification は `WHERE id = ? AND user_id = ?` でクエリレベルの所有権チェック、deck / record の公開一覧は `private_flg = false` で非公開を除外。他人のリソースへの到達手段は確認されず。
- **変更系エンドポイントの認証**: 全コントローラの POST / PUT / PATCH / DELETE に `RequiredAuthenticationMiddleware` が付与されていることを確認。認証漏れなし。
- **エラー詳細のクライアント漏洩**: 内部エラーは `slog` でログ出力のみ。クライアントへは `apierror` の定型メッセージを返し、`err.Error()` やスタックを応答に含めない。
- **認証バイパス（Abort漏れ）**: `apierror.JSON()` が `ctx.Abort()` を呼ぶため、認証・認可失敗後に後続ハンドラへ進まない。
- **CORS 設定**: オリジンは明示的な許可リスト、`AllowCredentials: false`、`SetTrustedProxies(nil)`。ワイルドカード無し。
- **バッチ用 cmd ツール群**: backfill / repair / check 系はいずれも `-dry-run` がデフォルト true のワンショット CLI。HTTPサーバは起動せず、外部公開面を持たない。
- **チャレンジトークンの束縛**: プレイヤーID所有権チャレンジは `uid` と `player_id` をトークンに束縛し、登録時に照合。総当たりはランダム指定アバターへの実変更が必要で成立しない。
- **コンテナ / シークレット管理**: distroless の nonroot（65532）で実行。`.env` は gitignore・dockerignore の両方に登録済み。リポジトリに秘密情報の追跡なし。

---

## 運用上の申し送り

### デプロイ前の確認事項（今回の修正の副作用）

起動時検証により、`VSRECORDER_JWT_SECRET` が本番環境に正しく渡っていないとサーバが起動しなくなる（意図的な fail fast）。docker-compose 実行時のシェル環境に変数が渡っているか要確認。現行値 43 文字は条件を満たす。

### ImageURL の既存データ

入口を塞いだのみで、既存データに不正値があれば残る。気になる場合は以下で確認可（ゼロ件なら対応不要）。

```sql
SELECT id, image_url FROM users WHERE image_url NOT LIKE 'https://%';
```

### 未追跡の差分

テストファイル5件（`internal/usecase/deck_test.go` ほか、`/* */` のコメントアウト解除等）は本精査での変更ではない。エディタ保存等による意図的な編集と見られるため未変更のまま。内容の確認を推奨。

---

## 検証ログ

一時テストで挙動を直接確認後に削除:

- 空鍵偽造トークン → 401
- exp 無しトークン → 401
- 正当なトークン → 200
- 2MiB ボディ → 400
- 日本語 63 文字 → 200 / 64 文字 → 400
- ImageURL 攻撃ベクタ8種 → 400

`go build ./...` / `go vet ./...` / `go test -count=1 ./...` すべて通過。
