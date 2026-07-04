# バトレコユーザーIDとポケモンカードゲーム プレイヤーズクラブIDの紐付け機能

## Context

バトレコ（本サービス）のユーザーアカウントと、公式のポケモンカードゲーム プレイヤーズクラブが発行する `player_id` を紐付けたい。将来的に公式サイト側のプレイヤー情報（ニックネーム等）と対戦記録を連携させるための土台となる機能。

紐付けは「なりすまし・誤登録防止」のため一度行うと1ヶ月は変更できない仕様とする。そのため誤ったIDを紐付けてしまった際の対応が論点になるが、ユーザーとの相談の結果、**追加のコードは実装せず、運用者（開発者自身）が手動でDBを修正する運用**とすることで合意した（本サービスは小規模な個人運用サービスのため、管理用エンドポイントを新設するコストに見合わない）。

ユーザーが提示した元のSQLには以下の不備があったため、併せて修正する:

1. `users_players` テーブル定義の末尾に余分なカンマがあり、SQLとして不正（[schema.sql:222-228](core-apiserver/db/schema.sql#L222-L228)）。
2. `id` (PRIMARY KEY) が無く、他の全テーブルの命名規則（`id VARCHAR(n) PRIMARY KEY`）と不整合。
3. `CREATE UNIQUE INDEX ... ON player_users (...)` が存在しないテーブル名 `player_users` を参照している（正しくは `users_players`）。
4. 複合ユニークインデックス `(player_id, user_id)` だけでは、1人のユーザーが複数の `player_id` を同時に紐付けたり、1つの `player_id` を複数ユーザーが同時に紐付けたりできてしまう。本来は「ユーザー1人につき有効な紐付けは1件」「`player_id` 1つにつき有効な紐付けは1件」という1:1関係にすべき。

`player_id` の実在確認は、既存の `deck_codes` バリデーション（`checkDeckCode` が `pokemon-card.com` にリクエストして実在確認する）と同じ考え方で、以下の外部APIを叩いて確認する:

```
curl -s -X POST -d 'player_id=XXXXXXXXXXXXXXXX' "https://players.pokemon-card.com/get_player_account_other"
```

成功時レスポンス例:

```json
{
  "code": 200,
  "player": {
    "player_id": "...",
    "nickname": "...",
    "avatar_image": "...",
    "current_league": "...",
    "prefecture": "..."
  }
}
```

実装は既存の `deck_codes` 一式（entity/model/repository/usecase/controller/validation/dto/presenter、Go: gin+gorm のレイヤードアーキテクチャ）と、フロントエンドの `UpdateNameModal.tsx` / `/api/users/[id]/route.ts`（Next.js BFF + next-auth + JWTでバックエンドへ中継）のパターンを踏襲する。

---

## 1. スキーマ修正 (core-apiserver/db/schema.sql)

[schema.sql:222-230](core-apiserver/db/schema.sql#L222-L230) を以下に置き換える:

```sql
CREATE TABLE users_players (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    player_id   VARCHAR(16) NOT NULL
);

CREATE INDEX idx_users_players_created_at ON users_players(created_at);
CREATE INDEX idx_users_players_deleted_at ON users_players(deleted_at);

-- 有効な紐付け(deleted_at IS NULL)は user_id / player_id それぞれについて1件のみ
CREATE UNIQUE INDEX unique_users_players_user_id   ON users_players (user_id)   WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX unique_users_players_player_id ON users_players (player_id) WHERE deleted_at IS NULL;
```

`id` は `deck_codes` と同じ `VARCHAR(26)`(ULID) とする。GRANT一覧（[schema.sql:603-619](core-apiserver/db/schema.sql#L603-L619)付近）にも `GRANT SELECT ON users_players TO grafana;` を追加し既存テーブルと揃える。

「変更不可の1ヶ月ロック」はDB制約ではなく、後述のusecase層で `既存の有効な行のcreated_at + 1ヶ月` を見て判定する（`deck_codes`のPUTが所有者チェックをauthorizationミドルウェアで行うのと同様、ビジネスルールはアプリ層に置く）。

---

## 2. バックエンド (core-apiserver) — `deck_codes` 一式をテンプレートに新規追加

### 2.1 ドメイン層

- `internal/domain/entity/user_player.go`: `UserPlayer{ID, CreatedAt, UserId, PlayerId}` + `NewUserPlayer(...)` + `LockedUntil() time.Time`（`CreatedAt.AddDate(0, 1, 0)` を返す）。
- `internal/domain/repository/user_player.go`: `UserPlayerInterface`
  - `FindByUserId(ctx, userId string) (*entity.UserPlayer, error)`（有効な行が無ければ `apperror.ErrRecordNotFound`）
  - `ExistsActiveByPlayerId(ctx, playerId string) (bool, error)`
  - `Save(ctx, *entity.UserPlayer) error`
  - `Delete(ctx, id string) error`（soft delete）
- `internal/domain/apperror/apperror.go` に汎用センチネルを追加:
  - `ErrLocked = errors.New("locked")` （一定期間操作できない場合。HTTP 409想定）
  - 「別ユーザーに使用中」は既存の `ErrAlreadyExists` を再利用する。

### 2.2 インフラ層

- `internal/infrastructure/model/user_player.go`: `deck_codes`と同じ形（`gorm:"primaryKey"` / `gorm.DeletedAt`）。
- `internal/infrastructure/user_player.go`: `deck_code.go` を踏襲し、`dbFromContext(ctx, i.db)` 経由でクエリを実行（`TransactionManager.Do` 経由の呼び出しに対応するため）。`ExistsActiveByPlayerId` は `Where("player_id = ?", id).Count(...)`。

### 2.3 ユースケース層 (`internal/usecase/user_player.go`)

`Create` が「新規紐付け」と「ロック解除後の変更（=旧レコードをsoft delete＋新規作成）」の両方を担う（PUT/Updateは作らない — 変更は実質的に作成のやり直しとして表現し、DeleteエンドポイントもUIから呼ばれる想定が無いため実装しない）。

```go
func (u *UserPlayer) Create(ctx, param *UserPlayerCreateParam) (*entity.UserPlayer, error) {
    existing, err := u.repository.FindByUserId(ctx, param.UserId)
    if err != nil && !errors.Is(err, apperror.ErrRecordNotFound) {
        return nil, err
    }
    if existing != nil && time.Now().Local().Before(existing.LockedUntil()) {
        return nil, apperror.ErrLocked
    }

    if inUse, err := u.repository.ExistsActiveByPlayerId(ctx, param.PlayerId); err != nil {
        return nil, err
    } else if inUse {
        return nil, apperror.ErrAlreadyExists
    }

    id, err := generateId()
    ...
    userPlayer := entity.NewUserPlayer(id, time.Now().Local(), param.UserId, param.PlayerId)

    err = u.transactionManager.Do(ctx, func(ctx context.Context) error {
        if existing != nil {
            if err := u.repository.Delete(ctx, existing.ID); err != nil {
                return err
            }
        }
        return u.repository.Save(ctx, userPlayer)
    })
    if err != nil {
        return nil, err
    }
    return userPlayer, nil
}

func (u *UserPlayer) FindByUserId(ctx, userId string) (*entity.UserPlayer, error) { ... } // GET用、そのまま返す
```

`repository.TransactionManager`（[internal/domain/repository/transaction.go](core-apiserver/internal/domain/repository/transaction.go)）をコンストラクタで受け取る。

### 2.4 コントローラ層

- `internal/controller/dto/user_player.go`:
  ```go
  type UserPlayerCreateRequest struct { PlayerId string `json:"player_id"` }
  type UserPlayerResponse struct {
      ID          string    `json:"id"`
      CreatedAt   time.Time `json:"created_at"`
      UserId      string    `json:"user_id"`
      PlayerId    string    `json:"player_id"`
      LockedUntil time.Time `json:"locked_until"`
  }
  type UserPlayerGetResponse struct{ UserPlayerResponse }
  type UserPlayerCreateResponse struct{ UserPlayerResponse }
  ```
- `internal/controller/presenter/user_player.go`: `entity.UserPlayer` → 上記DTO（`LockedUntil()`をそのまま詰める）。`deck_code.go` presenterと同じ形。
- `internal/controller/validation/user_player.go`:
  - `UserPlayerCreateMiddleware()`: `ShouldBindJSON` → `PlayerId` が空 or 16文字超なら `apierror.ErrBadRequest` → `checkPlayerId(ctx, req.PlayerId)` を呼び外部API実在確認 → `helper.SetUserPlayerCreateRequest(ctx, req)`。
  - `internal/controller/validation/util.go` に `checkPlayerId` を追加。`checkDeckCode`（[util.go:26](core-apiserver/internal/controller/validation/util.go#L26)）と同型で、`https://players.pokemon-card.com/get_player_account_other` に `player_id` を `PostForm`。HTTPステータス異常はステータス別に既存の `apierror`（`ErrServiceUnavailable`/`ErrGatewayTimeout`等）へマップ。200でもレスポンスJSONの `code` が200以外、または `player` が空ならプレイヤーID不正として `apierror.ErrBadRequest.JSON(ctx)`。
- `internal/controller/helper` に `GetUserPlayerCreateRequest`/`SetUserPlayerCreateRequest` を追加（`key.go`の既存ヘルパーと同じ生成パターン）。
- `internal/controller/user_player.go`:
  ```go
  const UserPlayersPath = "/usersplayers"
  ```

  - `RegisterRoute`: `r := c.router.Group(relativePath + UserPlayersPath)`
    - `GET  ""` : `authentication.RequiredAuthenticationMiddleware()` → `uid := helper.GetUID(ctx)` → `usecase.FindByUserId` → 404なら`apierror.ErrNotFound`、成功時 `presenter.NewUserPlayerGetResponse`
    - `POST ""`: `authentication.RequiredAuthenticationMiddleware()` + `validation.UserPlayerCreateMiddleware()` → `uid := helper.GetUID(ctx)`（URLパラメータではなくJWTのuidのみを信頼し、他人の紐付けを操作できないようにする） → `usecase.Create` → エラーハンドリング:
      - `errors.Is(err, apperror.ErrLocked)` → 新規 `apierror.ErrUserPlayerLocked`（409, メッセージ「紐付けから1ヶ月間は変更できません」）
      - `errors.Is(err, apperror.ErrAlreadyExists)` → 新規 `apierror.ErrPlayerIdAlreadyLinked`（409, 「このプレイヤーIDは既に別のアカウントで使用されています」）
      - それ以外 → `ErrInternalServerError`
  - `internal/controller/apierror/apierror.go` に上記2つの `Error` 定数を追加（既存の `ErrDeckCodeHasRecords` 等と同じ並びに追加）。

### 2.5 DI登録

[cmd/core-apiserver/main.go:213-220](core-apiserver/cmd/core-apiserver/main.go#L213-L220) の `controller.NewDeckCode(...)` の並びに倣い、`controller.NewUserPlayer(r, infrastructure.NewUserPlayer(db), usecase.NewUserPlayer(infrastructure.NewUserPlayer(db), infrastructure.NewTransactionManager(db))).RegisterRoute(relativePath)` を追加。

---

## 3. フロントエンド (webapp)

### 3.1 型定義: `src/app/types/user_player.ts`

`types/user.ts` と同じ命名規則で定義:

```ts
export type UserPlayerType = {
  id: string;
  created_at: string;
  user_id: string;
  player_id: string;
  locked_until: string;
};
export type UserPlayerGetResponseType = UserPlayerType;
export type UserPlayerCreateRequestType = { player_id: string };
export type UserPlayerCreateResponseType = UserPlayerType;
```

### 3.2 BFF route: `src/app/api/userplayers/route.ts`

`src/app/api/users/[id]/route.ts` と同じ構成（`auth()` でセッション確認 → `makeToken(session.user.id)` でJWT発行 → バックエンド `https://${domain}/api/v1beta/usersplayers` へ中継）。

- `GET`: セッション必須、404はそのまま透過（未紐付け状態として扱う）。
- `POST`: セッション必須、bodyをそのまま中継。バックエンドの409（ロック中・重複）はそのままステータスとメッセージを透過し、フロント側でエラーメッセージとして表示する。

### 3.3 UI

- `src/app/components/organisms/User/PlayerLinkCard.tsx`（新規、`UserIdentityCard.tsx`と同格のカード）: 現在の紐付け状態を表示。
  - 未紐付け: 「プレイヤーIDを登録」ボタン → `LinkPlayerIdModal` を開く。
  - 紐付け済み & ロック中: `player_id` とロック解除日（`locked_until`）を表示し、`WithdrawModal.tsx` の警告UI（`LuTriangleAlert` + 文言）を参考に「変更は次の変更可能日以降に行えます」旨を表示。変更ボタンは非活性。
  - 紐付け済み & ロック解除済み: 「プレイヤーIDを変更」ボタンで再度モーダルを開ける。
- `src/app/components/organisms/User/Modal/LinkPlayerIdModal.tsx`（新規）: `UpdateNameModal.tsx` のtoast/送信パターンを踏襲。
  - `Input` で `player_id` を入力（16文字以内）。
  - 送信時: `POST /api/userplayers` → 成功: 成功トースト＋`onLinked`コールバックで親のstate更新。失敗: `res.status === 409` の場合はレスポンスの `message` をそのままトースト表示（ロック中/重複のメッセージがバックエンドから来る）、それ以外は既存パターン通り汎用エラートースト。
  - モーダル内にも「一度登録すると1ヶ月間は変更できません」の注意書きを表示（送信前に必ず目に入るようにする）。
- `src/app/components/templates/User.tsx`（[User.tsx:30-35](webapp/src/app/components/templates/User.tsx#L30-L35)）に `PlayerLinkCard` を追加。

---

## 4. 誤紐付け時の対応（コード実装なし）

合意内容: 1ヶ月ロックは仕様通り維持し、ユーザーからの申告があった場合は**運用者がDBを直接修正する**。追加のAPI/認証機構は作らない。手順の目安（ドキュメント化はせず、対応時にその場で実行する想定）:

1. 対象ユーザーの `users_players` の有効行（`deleted_at IS NULL`）を確認。
2. その行を `UPDATE users_players SET deleted_at = now(), updated_at = now() WHERE id = '...'` でsoft delete。
3. 正しい `player_id` で `INSERT INTO users_players (id, created_at, updated_at, user_id, player_id) VALUES (...)` を実行（`id`はULID採番）。

アプリ側のロック判定は「有効行のcreated_at」のみを見るため、この手動操作をすれば当該ユーザーは即座に新しい1ヶ月ロックのもとで正しい状態に戻る。

---

## 5. 検証方法

- バックエンド: 該当ハンドラのユニットテスト（`deck_code_test.go`等の既存テスト構成を参考に）で、①初回紐付け成功、②ロック期間中の再紐付けが409、③他ユーザーが使用中の`player_id`が409、④1ヶ月経過後は再紐付けが成功し旧行がsoft deleteされること、を確認。
- 外部API実在確認は実際に `curl` で疎通確認したうえで、Goのテストは実サイトを叩かず HTTPクライアントをモック/httptestで差し替える（`checkDeckCode`のテストがあれば同じ手法を確認して踏襲）。
- `go build ./...` / 既存の `go test ./...` を実行し、既存機能に影響がないことを確認。
- フロントエンド: `npm run dev` で `/users` ページを開き、未紐付け→紐付け→ロック表示→（DBのcreated_atを1ヶ月以上前に書き換えて）再紐付け可能、の一連を手動確認。`npm run lint` / `npm run build`(型チェック) を実行。
