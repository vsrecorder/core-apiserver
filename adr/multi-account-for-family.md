# 複数アカウント機能(親子での利用)

## ステータス

提案中 (Proposed) — 2026-07-13

## Context

保護者(親)が、自分のログインアカウントから**子どものアカウントを作成・管理し、ワンタップで切り替えて対戦記録をつけられる**ようにしたい。

想定シーン: 親子でジムバトルに参加し、親が自分のスマホで自分の戦績と子の戦績を両方つける。子はスマホを持っておらず、メールアドレスも Google アカウントも持っていない。

### 要件(本ADRのスコープ)

- **R1.** 親は自分のアカウントから子アカウントを作成できる(メールアドレス・パスワード不要)。
- **R2.** 親はヘッダーのUIから操作対象アカウントを切り替えられる。切替後は、記録・デッキ・統計・バッジ・通知のすべてが「その子のもの」として振る舞う。
- **R3.** 子アカウントは**独立したユーザー**として扱われる。子は自分の公式プレイヤーIDを紐付けられ、自分のバッジ・称号・連続記録・シティリーグ結果を持つ。
- **R4.** 子が成長して自分のスマホを持ったとき、**データを引き継いだまま**自分の認証手段(Google / メール)でログインできるようになる(独立)。
- **R5.** 親以外は子アカウントを操作できない。委任は明示的なリンクによってのみ成立する。

### 非スコープ(将来の別ADR)

- 親のダッシュボードでの親子横断戦績表示 / 通知の集約表示
- 独立済みの子と親を「閲覧のみ」で相互リンクする家族機能
- 3人以上のアカウント間での記録の共同編集

### 既存コードベースの前提(踏襲する規約)

- **認証はwebappがトークン発行者。** Firebase Auth の ID トークンを next-auth の Credentials プロバイダが検証([webapp/src/app/auth.ts](webapp/src/app/auth.ts))、以降 BFF(`src/app/api/*/route.ts`)が **HS256 の短命JWT `{iss: "vsrecorder-webapp", uid}`(10s)** を発行し、core-apiserver の `RequiredAuthenticationMiddleware` が検証して `helper.SetUID(ctx, uid)` する([authentication.go](core-apiserver/internal/controller/auth/authentication/authentication.go))。**JWT のクレームを拡張すれば、委任情報をバックエンドまで運べる。**
- **`users.id` は Firebase UID(`VARCHAR(32)`)。** 全ドメインテーブルが `user_id VARCHAR(32)` で所有者を持つ(`decks` / `deck_codes` / `records` / `matches` / `games` / `unofficial_events` / `user_badges` / `user_streaks` / `user_environment_badges` / `notifications` / `users_players`)。
- **認可は「`helper.GetUID(ctx)` と対象エンティティの `UserId` の一致」で行う**(`internal/controller/auth/authorization/*.go` の全ミドルウェア)。
- **Firebase ユーザー作成の失敗時ロールバックは既に webapp が行う規約。** `auth.ts` の新規登録フローは「core-apiserver への登録が失敗したら `firebaseAdmin.auth().deleteUser()` で巻き戻す」を実装済み。子アカウント作成もこの形を踏襲する。
- ID は ULID(`VARCHAR(26)`)、ソフトデリート(`deleted_at`)、複数テーブル書き込みは `TransactionManager.Do`。

---

## Decision

### D1. 「サブプロフィール」ではなく「リンクドアカウント + 代理操作(delegation)」を採用する

子アカウントは `users` テーブルの**独立した1行**として作る。親子関係は新テーブル `user_links` で表現し、親は「子として振る舞う」形で操作する。

```
   ┌────────────────────────────┐
   │ users (親)                  │  ← Firebase に認証手段あり(Google等)
   │  id = <firebase uid A>      │
   └────────────┬───────────────┘
                │ user_links
                │ (owner_user_id = A, member_user_id = B, role='managed')
                ▼
   ┌────────────────────────────┐
   │ users (子)                  │  ← Firebase ユーザーは存在するが認証手段なし
   │  id = <firebase uid B>      │     (= 自力ではログインできない「管理アカウント」)
   │  managed_flg = true         │
   └────────────┬───────────────┘
                │
                ▼  子は普通のユーザーなので、既存機能がそのまま動く
     records / decks / matches / user_badges / user_streaks /
     users_players(自分の公式プレイヤーID) / notifications ...
```

**却下案: サブプロフィール方式(1つの `users` 行が複数プロフィールを持ち、`profile_id` で記録を分ける)**

以下の理由で採用しない。

- **既存スキーマの主キーを壊す。** `user_streaks` は `user_id` が **PRIMARY KEY**、`user_environment_badges` は `(user_id, environment_id)` が **PRIMARY KEY**([schema.sql:550-635](core-apiserver/db/schema.sql#L550-L635))。プロフィール単位で連続記録・環境バッジを持つには主キーの変更が必要で、破壊的なデータ移行になる。
- **公式プレイヤーIDが紐付けられない。** `users_players` は `user_id` / `player_id` の**双方に一意制約**([schema.sql:268-269](core-apiserver/db/schema.sql#L268-L269))。親子はそれぞれ別の公式プレイヤーIDを持つが、1つの `users` 行には1つの `player_id` しか紐付けられない。結果として**子のシティリーグ結果(`cityleague_results` は `player_id` がキー)を取り込めない**。これは R3 を満たせないことを意味し、単独で却下理由になる。
- **全テーブル・全クエリの書き換えになる。** 11テーブルに `profile_id` を追加し、所有者判定を行う全 usecase / repository / 認可ミドルウェアを書き換え、既存データを移行する必要がある。
- **R4(子の独立)にデータ移行が必要。** サブプロフィール方式では、子が独立する際にプロフィール配下の全記録を新しい `users` 行へ付け替える大規模な移行処理が要る。リンクドアカウント方式なら**Firebase 側に認証手段を足してリンクを外すだけ**で、データは1行も動かない。

**リンクドアカウント方式の決定的な利点:** 子は「普通のユーザー」なので、バッジ・称号・連続記録・統計・プレイヤーID・シティリーグ結果・通知といった**既存機能が一切の改修なしにそのまま子アカウントで動く**。実装は「アカウントを作る」「切り替える」「委任を検証する」の3点に集約される。

### D2. 子アカウントは「認証手段を持たない Firebase ユーザー」として作る

`firebaseAdmin.auth().createUser({ displayName })` はメール・パスワードなしでもユーザーを作成でき、UID が払い出される。この UID を `users.id` に使う。

- 認証プロバイダが紐付いていないため、**子は自力ではサインインできない**。操作は必ず親の委任経由になる(R5)。
- `users.id` の形式が既存ユーザーと完全に同一のため、スキーマも既存クエリも変わらない。
- **R4(独立)は Firebase 側の操作だけで完結する。** 子の Firebase ユーザーに `updateUser({ email, password })` もしくは Google プロバイダをリンクすれば、その瞬間から `users.id` はそのままに自力ログインできるようになる。あとは `user_links` をソフトデリートし、`managed_flg` を `false` にするだけ。**アプリケーションデータの移行はゼロ。**

### D3. 委任は JWT の `act`(actor)クレームで運び、**core-apiserver 側で検証する**

BFF が発行する JWT を拡張する:

```jsonc
{
  "iss": "vsrecorder-webapp",
  "uid": "<subject: 実際に操作するアカウント(親自身 or 子)>",
  "act": "<actor: 認証済みの実ログインアカウント(親)>"   // 省略時は uid と同一とみなす
}
```

`uid` に**操作対象**を入れるのが要点。これにより:

- **`internal/controller/auth/authorization/*.go` の全ミドルウェアは無改修で正しく動く。** 認可は「`GetUID(ctx)` == エンティティの `UserId`」という判定のままでよく、`uid` が既に子を指しているため、親は子のリソースに対して正当な所有者として振る舞う。
- **webapp 側も `session.user.id` が操作対象アカウントを返すようにすれば、`session.user.id` を参照している既存48箇所が無改修で子アカウント文脈になる。**

`act != uid` のとき、**`RequiredAuthenticationMiddleware` が `user_links` を引いて委任の正当性を検証する**(存在しなければ 401)。

> **なぜ BFF を信頼せず、バックエンドでも検証するのか。** 現状 core-apiserver は共有秘密鍵で署名された webapp の JWT を全面的に信頼している。しかし JWT 発行箇所は **BFF の27ルートに分散**しており、そのうち1つでも「リクエストボディや検証していないヘッダから `uid` を組み立てる」実装ミスをすると、**任意アカウントへのなりすまし(権限昇格)が成立してしまう**。権限昇格は最も影響が大きいバグクラスであり、検証点は認証ミドルウェアの1箇所に集約する。

**性能への配慮:** 毎リクエストで `user_links` を引くのを避けるため、リンク検証結果は **TTL 60秒の in-memory キャッシュ**に載せる(`(act, uid)` をキーとする)。リンク解除の反映が最大60秒遅れるが、親が自分の子アカウントへの委任を解除するケースのみであり、実害はない。

### D4. 「今どのアカウントで操作しているか」は next-auth のセッショントークンに持つ

- next-auth の JWT に `activeUid` を追加する(初期値は本人の `uid`)。**Cookie やクライアント側の状態ではなくサーバー署名済みトークンに持つ**ため、ユーザーによる改竄ができない。
- 切替は `POST /api/accounts/switch { userId }` → BFF が core-apiserver に委任の正当性を確認 → next-auth の `jwt` コールバック(`trigger: "update"`)で `activeUid` を書き換える。
- `session.user` は `{ id: activeUid, actorId: uid }` を返す。既存コードが参照する `session.user.id` は**操作対象**を指すことになる(D3参照)。

### D5. JWT 発行を共通ヘルパに切り出す

現在 BFF の27ルートに JWT 発行コードがコピペされている。今回すべてに `act` クレームを足す必要があるため、この機会に共通化する:

```ts
// webapp/src/app/utils/backendToken.ts
export async function createBackendToken(): Promise<{ token: string; uid: string } | null> {
  const session = await auth();
  if (!session) return null;
  // uid = 操作対象(activeUid) / act = 実ログインアカウント
  const token = jwt.sign(
    { iss: "vsrecorder-webapp", uid: session.user.id, act: session.user.actorId },
    process.env.VSRECORDER_JWT_SECRET!,
    { algorithm: "HS256", expiresIn: "10s" },
  );
  return { token, uid: session.user.id };
}
```

以降 BFF は `const auth = await createBackendToken(); if (!auth) return 401;` の3行で済む。**27ルートの重複が消えるうえ、「`uid` を組み立てる箇所」が1箇所になり、D3 で懸念した実装ミスの余地そのものが減る。**

---

## 1. スキーマ (core-apiserver/db/schema.sql)

`users` / `users_players` の近くに追加する。

```sql
-- 管理アカウント(自力ログイン不可)かどうかのフラグ
ALTER TABLE users ADD COLUMN managed_flg BOOLEAN NOT NULL DEFAULT false;

-- アカウント間の委任リンク(親 → 子)
CREATE TABLE user_links (
    id              VARCHAR(26) PRIMARY KEY,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    deleted_at      TIMESTAMP DEFAULT NULL,
    owner_user_id   VARCHAR(32) NOT NULL,   -- 親(認証主体・委任元)
    member_user_id  VARCHAR(32) NOT NULL,   -- 子(被代理アカウント)
    role            VARCHAR(16) NOT NULL,   -- 'managed' (将来 'viewer' 等を追加しうる)
    FOREIGN KEY (owner_user_id)  REFERENCES users (id),
    FOREIGN KEY (member_user_id) REFERENCES users (id)
);

CREATE INDEX idx_user_links_owner_user_id ON user_links (owner_user_id);
CREATE INDEX idx_user_links_deleted_at    ON user_links (deleted_at);

-- 子は同時に1人の親にのみ属する(有効な行のみ)。users_players と同じ規約。
CREATE UNIQUE INDEX unique_user_links_member_user_id
    ON user_links (member_user_id) WHERE deleted_at IS NULL;

GRANT SELECT ON user_links TO grafana;
```

- **既存テーブルへの変更は `users.managed_flg` の追加(デフォルト値あり)のみ**で、既存行・既存クエリに影響しない。
- 子アカウント数の上限(**5**)は usecase 層で担保する(乱用防止。スキーマ制約にはしない)。

### マイグレーション

本リポジトリは `db/schema.sql` を正とする運用のため、上記 `ALTER` / `CREATE TABLE` / `CREATE INDEX` / `GRANT` を稼働中DBに適用するSQLを別途流す。既存行はすべて `managed_flg = false`(＝通常アカウント)となり後方互換。

---

## 2. バックエンド (core-apiserver)

### 2.1 ドメイン層

- `internal/domain/entity/user_link.go`

  ```go
  type UserLink struct {
      ID           string
      CreatedAt    time.Time
      UpdatedAt    time.Time
      OwnerUserId  string
      MemberUserId string
      Role         string
  }
  func NewUserLink(id string, createdAt, updatedAt time.Time, ownerUserId, memberUserId, role string) *UserLink
  ```

- `internal/domain/entity/user.go` に `ManagedFlg bool` を追加。
- `internal/domain/repository/user_link.go` — `UserLinkInterface`:
  - `FindByOwnerUserId(ctx, ownerUserId) ([]*entity.UserLink, error)` — 子一覧
  - `FindByMemberUserId(ctx, memberUserId) (*entity.UserLink, error)` — 親の特定
  - `Exists(ctx, ownerUserId, memberUserId string) (bool, error)` — **委任検証用(ホットパス)**
  - `Save(ctx, *entity.UserLink) error` / `Delete(ctx, id string) error`(ソフトデリート)
- `internal/domain/apperror`: 上限超過用に `ErrLinkLimitExceeded` を追加(既存 `ErrAlreadyExists` / `ErrForbidden` は再利用)。

### 2.2 インフラ層

- `internal/infrastructure/model/user_link.go`(gorm モデル)、`internal/infrastructure/user_link.go`(実装)。全クエリは `dbFromContext(ctx, i.db)` 経由。
- `internal/infrastructure/user_link_cache.go` — `UserLinkInterface` をラップし `Exists` の結果のみ TTL 60秒でキャッシュするデコレータ(D3)。認証ミドルウェアにはこのラッパを注入する。

### 2.3 ユースケース層

- `internal/usecase/user_link.go`(新規)
  - `CreateManagedUser(ctx, ownerUserId string, param CreateManagedUserParam) (*entity.User, error)`
    - **`TransactionManager.Do` の中で `users`(`managed_flg = true`)と `user_links` を同時に作る。** `users.id` は webapp が Firebase で払い出した UID を受け取る(自前生成しない)。
    - 事前検証: 親が `managed_flg = true` でないこと(**管理アカウントは子を作れない**)、子の数が上限(5)未満であること。
  - `FindByOwnerUserId(ctx, ownerUserId)` — アカウント切替UI用の一覧。
  - `Unlink(ctx, ownerUserId, linkId string) error` — リンク解除(R4の独立フローで使用)。
- `internal/usecase/user.go` の `Delete` を拡張: 親の退会時、**管理下の子アカウントも同一トランザクションで削除**する(後述の Consequences 参照)。

### 2.4 コントローラ層 / 認証

- **`internal/controller/auth/authentication/authentication.go` を拡張(本ADRの中核)**

  ```go
  type VSRClaims struct {
      jwt.RegisteredClaims
      UID string `json:"uid"`
      Act string `json:"act,omitempty"` // 実ログインアカウント(委任元)
  }

  func RequiredAuthenticationMiddleware(userLinkRepository repository.UserLinkInterface) gin.HandlerFunc {
      return func(ctx *gin.Context) {
          // ... 既存のトークン検証 ...

          // 代理操作の検証: act と uid が異なる場合、委任リンクの存在を必須とする
          if claims.Act != "" && claims.Act != claims.UID {
              ok, err := userLinkRepository.Exists(ctx, claims.Act, claims.UID)
              if err != nil {
                  apierror.ErrInternalServerError.JSON(ctx)
                  return
              }
              if !ok {
                  apierror.ErrUnauthorized.JSON(ctx)
                  return
              }
              helper.SetActorUID(ctx, claims.Act)
          }

          helper.SetUID(ctx, claims.UID) // uid は「操作対象」。既存の認可はこれで正しく動く
      }
  }
  ```

  `OptionalAuthenticationMiddleware` も同様に対応する。`helper` に `SetActorUID` / `GetActorUID` を追加(監査ログ用)。

- **`internal/controller/auth/authorization/*.go` は無改修。** `GetUID(ctx)` が操作対象を返すため、既存の所有者一致判定がそのまま正しい(D3)。
- `internal/controller/user_link.go`(新規): ルート `/account_links`(すべて `RequiredAuthentication`)
  - `GET    /account_links` — 自分が親であるリンク(＝子)一覧
  - `POST   /account_links` — 管理アカウント作成(body: `{ user_id, name, image_url }`。`user_id` は webapp が Firebase で払い出した UID)
  - `DELETE /account_links/:id` — リンク解除(`UserLinkAuthorizationMiddleware` で `owner_user_id == GetUID(ctx)` を検証)
  - **代理操作中(`GetActorUID(ctx) != ""`)は `POST` / `DELETE` を拒否する。** 子アカウントの文脈から家族構成を変更させない。
- `internal/controller/dto/user_link.go` / `presenter/user_link.go` / `validation/`(名前の長さ等)を既存 `user` 一式に倣って追加。
- `internal/ratelimit`: 管理アカウント作成に**親ユーザー単位のレート制限**をかける。
- `cmd/core-apiserver/main.go`: `infrastructure.NewUserLinkCache(infrastructure.NewUserLink(db))` を認証ミドルウェアに注入し、`controller.NewUserLink(...)` を配線。

### 2.5 監査

代理操作で行われた書き込み(`GetActorUID(ctx) != ""`)は、既存のアクセスログに `actor_uid` を含めて記録する。「誰が誰として操作したか」を後から追跡できるようにする。

---

## 3. フロントエンド (webapp)

### 3.1 セッションと委任

- `src/app/auth.ts`
  - `declare module "next-auth"` の `Session` を `{ user: { id: string; actorId: string } }` に拡張。`JWT` に `activeUid?: string` を追加。
  - `jwt` コールバックに `trigger === "update"` の分岐を追加し、切替リクエストで `activeUid` を更新する。**このとき必ずバックエンドに委任の正当性を確認してから書き込む。**
  - `session` コールバック: `session.user = { id: token.activeUid ?? token.uid, actorId: token.uid }`。
- `src/app/utils/backendToken.ts`(新規、D5): JWT 発行の共通ヘルパ。
- **BFF 27ルートを `createBackendToken()` 利用に置き換える**(振る舞いは変わらないが、重複が消え `act` クレームが全ルートに乗る)。
- `src/app/api/accounts/switch/route.ts`(新規): 切替。`POST { userId }` → `userId === session.user.actorId`(＝親自身に戻る)か、`GET /account_links` の結果に含まれることを確認 → セッション更新。

### 3.2 管理アカウント作成フロー

`auth.ts` の既存の新規登録フロー(失敗時 `firebaseAdmin.auth().deleteUser()` でロールバック)と**同じパターン**を踏襲する。

`src/app/api/accounts/route.ts`(`POST`):

1. `auth()` でセッション取得。**代理操作中(`session.user.id !== session.user.actorId`)なら 403。**
2. `firebaseAdmin.auth().createUser({ displayName })` → 子の UID を払い出す(認証手段は付けない)。
3. core-apiserver `POST /account_links`(親の JWT を添えて `{ user_id, name, image_url }`)。
4. **失敗したら `firebaseAdmin.auth().deleteUser(childUid)` で巻き戻す。**

### 3.3 UI

- **アカウント切替**: ヘッダーのアバター([components/organisms/Layout/Header.tsx](webapp/src/app/components/organisms/Layout/Header.tsx))のメニューにアカウント一覧と「子アカウントを追加」を置く。
- **代理操作中の明示(最重要)**: 誤って子アカウントで自分の記録をつけてしまう事故を防ぐため、**切替中は常時見えるインジケータ**を出す。ヘッダーのアバター横に子アカウント名のバッジ、加えて画面上部に細い着色バー(「〇〇として操作中 / 自分に戻る」)を表示する。
- **キャッシュ破棄**: 切替時にクライアントキャッシュ(SWR等)を全破棄し `router.refresh()` する。**アカウント別のデータ混線はこの機能で最も起こりやすいバグ**であり、データ取得フックのキャッシュキーには `session.user.id` を含める。
- **子アカウントのプライバシー既定値**: 子アカウントで作成する記録・デッキは `private_flg` を**デフォルト非公開**にする(親が明示的に公開できる)。子どもの情報は既定で公開しない。
- `src/app/contexts/UserAvatarContext.tsx` を切替に追従させる。

### 3.4 規約・ポリシー

`/terms` `/privacy` に、保護者が子アカウントを作成・管理できること、子アカウントの情報の取り扱い(**メールアドレス等を収集しない**)、親の退会時の子アカウントの扱いを追記する。

---

## 4. 段階的リリース

| フェーズ | 内容 |
| --- | --- |
| **Phase 1(本ADRのMVP)** | `user_links` / 管理アカウント作成 / アカウント切替 / `act` クレームによる委任検証。親が子の記録をつけられる状態にする。 |
| **Phase 2** | 子の独立(R4)。Firebase ユーザーに認証手段を付与 → `user_links` 解除 → `managed_flg = false`。**データ移行なし。** |
| **Phase 3** | 親のダッシュボードでの親子横断表示、通知の集約、独立済みの子との閲覧のみリンク。 |

---

## 5. Consequences

**Pros**

- **子が「普通のユーザー」なので、既存機能(バッジ・称号・連続記録・統計・公式プレイヤーID・シティリーグ結果・通知)が改修なしでそのまま動く。**
- **既存の認可ミドルウェアが1行も変わらない。** `uid` に操作対象を入れる設計(D3)により、変更は認証ミドルウェアの1箇所に閉じる。
- スキーマ変更が新規テーブル1つ + デフォルト値付きカラム1つで、**既存データの移行が不要**。
- R4(子の独立)が Firebase 側の操作だけで完結し、**アプリケーションデータを1行も動かさずに済む**。
- 副産物として、BFF 27ルートに散在する JWT 発行の重複が解消される(D5)。

**Cons / トレードオフ**

- **子アカウントは Firebase ユーザーを消費する。** Firebase Authentication の MAU は「サインインしたユーザー」で計上されるため課金上の影響は小さいが、ユーザー数そのものは増える。
- **委任検証のため認証パスにDBアクセスが1回入る**(`act != uid` のときのみ)。TTL 60秒のキャッシュで緩和するが、リンク解除の反映が最大60秒遅れる。
- **アカウント切替のUXは事故りやすい。** 「子として操作していることに気づかず自分の記録をつける」が最も起きやすい失敗であり、常時可視のインジケータ(3.3)が機能の成否を左右する。
- 親の退会時、管理下の子アカウントも削除される(データ喪失)。退会ダイアログで**子アカウント名を列挙して明示的に同意を取る**こと。将来的には「退会前に子を独立させる」導線(Phase 2)を用意したい。

**未決事項**

- 子アカウント数の上限は 5 とするが、実利用を見て調整する。
- 親の退会時に「子アカウントを削除」ではなく「子を独立させてから退会」を必須にするかは、Phase 2 の完成後に再検討する。
- 親子で対戦した場合の記録(`matches.opponents_user_id` に互いを指定)は現状の仕組みで作れるが、両方のアカウントに記録を自動生成するかは別途検討。
