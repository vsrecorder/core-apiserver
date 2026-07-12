# デッキ・デッキコード(バージョン)へのタグ付け機能

## ステータス

提案中 (Proposed) — 2026-07-12

## Context

デッキおよびデッキコード(＝デッキのバージョン)に、ユーザーが任意の**タグ**を付けられるようにしたい。「アグロ」「コントロール」「シティ用」「調整中」といったラベルで自分のデッキ資産を分類・検索できるようにするのが目的。

将来的には**記録(records)や対戦結果(matches)にも同じ仕組みでタグを付けられる**ようにしたいため、特定エンティティに閉じない拡張性のある設計にする必要がある。これが本ADRの最大の論点。

### 既存コードベースの前提(踏襲する規約)

本機能は以下の既存規約・パターンを踏襲する:

- **多対多は「エンティティ別の中間テーブル + FK制約」で表現する。** 既に `deck_pokemon_sprites` / `match_pokemon_sprites`([schema.sql:511-529](core-apiserver/db/schema.sql#L511-L529))という前例がある。ポリモーフィック(`taggable_type` + `taggable_id` の単一中間テーブル)は採用しない(理由は後述)。
- ID は ULID (`VARCHAR(26)`)。`internal/usecase/util.go` の `generateId()` で生成。
- 各エンティティは `created_at` / `updated_at` / `deleted_at`(`gorm.DeletedAt` によるソフトデリート)を持つ。所有者は `user_id VARCHAR(32)`。
- レイヤードアーキテクチャ(gin + gorm): `entity` → `repository`(interface) → `infrastructure`(model + impl) → `usecase` → `controller`(+ `dto` / `presenter` / `validation`)。
- 複数テーブルへの書き込みは `TransactionManager.Do(ctx, func)` でトランザクション化する。
- **子要素はエンティティの作成/更新リクエストに埋め込んで渡す。** `pokemon_sprites` が `DeckCreateRequest` / `DeckUpdateRequest` に配列で埋め込まれている([dto/deck.go](core-apiserver/internal/controller/dto/deck.go))のと同じ形をタグにも適用する。
- フロントエンド(webapp)は Next.js の BFF(`src/app/api/*/route.ts`)経由でバックエンドに中継し、`src/app/types/*.ts` に型を定義、`src/app/components/**` で描画する。

---

## Decision

### D1. データモデル: 「タグマスタ」+「エンティティ別中間テーブル」

タグの実体を保持する `tags` マスタテーブルを**1つ**用意し、各エンティティとの関連は**エンティティごとの中間テーブル**(`deck_tags` / `deck_code_tags`、将来 `record_tags` / `match_tags`)で表現する。

```
                       ┌─────────────────┐
        ┌──────────────│      tags        │──────────────┐
        │              │ (user_id 単位の  │              │
        │              │  タグ名前空間)   │              │
        │              └─────────────────┘              │
        │                       │                       │
   deck_tags            deck_code_tags        (将来) record_tags /
        │                       │                    match_tags
        ▼                       ▼                       ▼
   ┌─────────┐          ┌──────────────┐        ┌──────────────┐
   │  decks  │          │  deck_codes  │        │ records/…    │
   └─────────┘          └──────────────┘        └──────────────┘
```

**なぜポリモーフィック(単一 `taggables` テーブル)にしないか:**

- ポリモーフィックだと `taggable_id` に対して**外部キー制約を張れない**。本コードベースは全中間テーブルで FK 制約を張る規約(`deck_pokemon_sprites` 等)であり、これを壊す。
- エンティティごとに `VARCHAR(26)`(ULID)と `INT`(公式イベント等)で ID 型が混在しており、単一カラムで参照先を型安全に扱えない。
- タグ付け対象は「デッキ」「デッキコード」「記録」「対戦結果」と**有限かつ低頻度でしか増えない**。テーブルが数個増えるコストより、FK 制約と型安全性のメリットが上回る。
- 中間テーブルの増加による実装コストは、後述の D4「共通ヘルパ」で吸収する。

### D2. `tags` はユーザー単位の名前空間を持つ

タグはユーザーごとに独立(あるユーザーの「アグロ」と別ユーザーの「アグロ」は別レコード)。同一ユーザー内でタグ名は一意とする。

### D3. 付与/解除は親エンティティのリクエストに埋め込む(sprite と同じ)

デッキ/デッキコードの作成・更新リクエストに `tag_ids: string[]` を埋め込み、レスポンスに `tags: TagResponse[]` を含める。タグの新規作成・リネーム・削除は別途 `/tags` エンドポイントで管理する(オートコンプリート候補の取得もここ)。

### D4. 中間テーブルの差分同期を共通ヘルパで抽象化する

拡張性の肝。中間テーブルは増えるが、attach/detach ロジックは1箇所に集約する(D4 詳細は後述)。新しいエンティティへタグを広げる作業は「テーブル追加 + 設定値1つ追加 + 配線」だけになる。

---

## 1. スキーマ (core-apiserver/db/schema.sql)

`deck_pokemon_sprites` / `deck_codes` の近くに以下を追加する。中間テーブルはソフトデリート列を持たず**行の物理削除**で関連を解除する(`deck_pokemon_sprites` と同じ規約)。

```sql
-- タグマスタ (ユーザーごとの名前空間)
CREATE TABLE tags (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    name        VARCHAR(32) NOT NULL,
    color       VARCHAR(7)  DEFAULT NULL   -- '#RRGGBB' 任意。UI表示用
);

CREATE INDEX idx_tags_created_at ON tags(created_at);
CREATE INDEX idx_tags_deleted_at ON tags(deleted_at);
CREATE INDEX idx_tags_user_id    ON tags(user_id);
-- 同一ユーザー内でタグ名は一意 (有効な行のみ)
CREATE UNIQUE INDEX unique_tags_user_id_name ON tags (user_id, name) WHERE deleted_at IS NULL;

-- デッキ ⇔ タグ
CREATE TABLE deck_tags (
    deck_id  VARCHAR(26) NOT NULL,
    tag_id   VARCHAR(26) NOT NULL,
    PRIMARY KEY (deck_id, tag_id),
    FOREIGN KEY (deck_id) REFERENCES decks(id),
    FOREIGN KEY (tag_id)  REFERENCES tags(id)
);
CREATE INDEX idx_deck_tags_tag_id ON deck_tags(tag_id);

-- デッキコード(バージョン) ⇔ タグ
CREATE TABLE deck_code_tags (
    deck_code_id  VARCHAR(26) NOT NULL,
    tag_id        VARCHAR(26) NOT NULL,
    PRIMARY KEY (deck_code_id, tag_id),
    FOREIGN KEY (deck_code_id) REFERENCES deck_codes(id),
    FOREIGN KEY (tag_id)       REFERENCES tags(id)
);
CREATE INDEX idx_deck_code_tags_tag_id ON deck_code_tags(tag_id);
```

GRANT 一覧([schema.sql:731 付近](core-apiserver/db/schema.sql#L731))にも既存テーブルと揃えて追加する:

```sql
GRANT SELECT ON tags           TO grafana;
GRANT SELECT ON deck_tags      TO grafana;
GRANT SELECT ON deck_code_tags TO grafana;
```

> **将来の拡張(記録/対戦結果)** はテーブル追加のみ。`record_tags(record_id, tag_id)` / `match_tags(match_id, tag_id)` を同じ形で足す。

### マイグレーション

本リポジトリは単一 `db/schema.sql` を正とする運用のため、稼働中DBには上記 `CREATE TABLE` / `CREATE INDEX` / `GRANT` を追加適用するマイグレーションSQLを別途流す(新規テーブルのみで既存テーブルへの `ALTER` は無いため後方互換)。

---

## 2. バックエンド (core-apiserver)

`deck_codes` / `deck_pokemon_sprites` 一式をテンプレートに、以下を追加・拡張する。

### 2.1 ドメイン層

- `internal/domain/entity/tag.go`
  ```go
  type Tag struct {
      ID        string
      CreatedAt time.Time
      UpdatedAt time.Time
      UserId    string
      Name      string
      Color     string // 空文字可
  }
  func NewTag(id string, createdAt, updatedAt time.Time, userId, name, color string) *Tag
  ```
- `internal/domain/entity/deck.go` / `deck_code.go` に `Tags []*Tag` フィールドを追加(`PokemonSprites` と並ぶ形)。
- `internal/domain/repository/tag.go` — `TagInterface`:
  - タグマスタ CRUD: `FindByUserId`, `FindById`, `FindByIds(ctx, ids []string)`, `FindByUserIdAndName`, `Save`, `Delete`(soft delete)。
  - 関連取得: `FindByDeckId(ctx, deckId) ([]*entity.Tag, error)`, `FindByDeckCodeId(ctx, deckCodeId) (...)`。
  - 関連同期: `ReplaceDeckTags(ctx, deckId string, tagIds []string) error`, `ReplaceDeckCodeTags(ctx, deckCodeId string, tagIds []string) error`(渡された集合に一致するよう中間テーブルを差分更新)。
- `internal/domain/apperror/apperror.go`: タグ名重複は既存の `ErrAlreadyExists` を再利用。

### 2.2 インフラ層

- `internal/infrastructure/model/tag.go`: `Tag`(gorm、`deck.go` と同形)、`DeckTag{DeckID, TagID}`、`DeckCodeTag{DeckCodeID, TagID}`(複合PK、`gorm:"primaryKey"` を両カラムに)。
- `internal/infrastructure/tag.go`: 実装。全クエリは `dbFromContext(ctx, i.db)` 経由(`TransactionManager.Do` 対応)。

**D4: 中間テーブル差分同期の共通ヘルパ**(拡張性の中核)。関連テーブルごとの attach/detach ロジックを1箇所に集約する:

```go
// 中間テーブルのメタ情報。新しいエンティティはこの値を1つ足すだけ。
type tagLinkTable struct {
    name        string // "deck_tags"
    ownerColumn string // "deck_id"
}

var (
    deckTagLink     = tagLinkTable{"deck_tags", "deck_id"}
    deckCodeTagLink = tagLinkTable{"deck_code_tags", "deck_code_id"}
    // 将来: recordTagLink = tagLinkTable{"record_tags", "record_id"} など
)

// ownerId が持つタグ関連を tagIds の集合に一致させる(差分 INSERT / DELETE)。
func (i *Tag) replaceTags(ctx context.Context, link tagLinkTable, ownerId string, tagIds []string) error {
    db := dbFromContext(ctx, i.db)
    // 1. 現在の関連を取得 → 2. 追加分を INSERT、不要分を DELETE
    // (呼び出し元が TransactionManager.Do 内で呼ぶ想定)
    ...
}

func (i *Tag) ReplaceDeckTags(ctx context.Context, deckId string, tagIds []string) error {
    return i.replaceTags(ctx, deckTagLink, deckId, tagIds)
}
func (i *Tag) ReplaceDeckCodeTags(ctx context.Context, deckCodeId string, tagIds []string) error {
    return i.replaceTags(ctx, deckCodeTagLink, deckCodeId, tagIds)
}
```

これにより、記録/対戦結果へ広げる際は `tagLinkTable` を1つ足して `ReplaceRecordTags` を薄く生やすだけで済む。

### 2.3 ユースケース層

- `internal/usecase/tag.go`(新規): タグマスタの `Create` / `Update`(リネーム) / `Delete` / `FindByUserId`(一覧・候補)。
  - `Create` は `(user_id, name)` の重複を検査(既存があれば `ErrAlreadyExists`、もしくは find-or-create 方針なら既存を返す)。所有者チェックは controller の authorization ミドルウェアで担保。
- `internal/usecase/deck.go` / `deck_code.go` の `Create` / `Update` を拡張:
  - パラメータに `TagIds []string` を追加。
  - `TransactionManager.Do` の中で、本体 `Save` の後に `tagRepository.ReplaceDeckTags(ctx, deck.ID, param.TagIds)` を呼ぶ。
  - **付与しようとしたタグが本人所有か**を検証(`tagRepository.FindByIds` の結果を `user_id` で照合し、他人のタグIDを弾く)。不正なら `ErrForbidden` 相当。
  - 取得系(`FindById` 等)で `deck.Tags` を詰めて返す。

### 2.4 コントローラ層

- `internal/controller/tag.go`(新規): ルート `/tags`
  - `GET  /tags`(自分のタグ一覧 / オートコンプリート、`RequiredAuthentication`)
  - `POST /tags`(作成)
  - `PUT  /tags/:id`(リネーム/色変更、`TagUpdateAuthorizationMiddleware` で所有者チェック)
  - `DELETE /tags/:id`(削除。関連中間テーブル行も併せて物理削除)
- `internal/controller/dto/tag.go`: `TagResponse{ID, Name, Color}`、`TagCreateRequest{Name, Color}`、`TagUpdateRequest{...}`。
- `internal/controller/dto/deck.go` / `deck_code.go`:
  - `DeckCreateRequest` / `DeckUpdateRequest`(および deck_code 側)に `TagIds []string json:"tag_ids"` を追加。
  - `DeckResponse` / `DeckCodeResponse` に `Tags []*TagResponse json:"tags"` を追加。
- `internal/controller/presenter/`: `tag.go` を追加し、`deck.go` / `deck_code.go` の presenter で `Tags` を詰める。
- `internal/controller/validation/`: タグ名の長さ(1〜32)・色(`#RRGGBB` 形式)・`tag_ids` の要素数上限バリデーション。
- `internal/controller/auth/authorization/`: `TagUpdateAuthorizationMiddleware`(`deck` の同種ミドルウェアを踏襲)。
- `cmd/core-apiserver/main.go`: `controller.NewTag(...)` を配線し、`NewDeck` / `NewDeckCode` に `infrastructure.NewTag(db)` を注入。

---

## 3. フロントエンド (webapp)

- `src/app/types/tag.ts`: `TagType { id, name, color }` と各リクエスト/レスポンス型。`deck.ts` / `deck_code.ts` の型に `tags: TagType[]`、作成/更新リクエスト型に `tag_ids: string[]` を追加。
- `src/app/api/tags/route.ts`(+ `[id]/route.ts`): next-auth の JWT を付けてバックエンド `/tags` に中継する BFF。
- `src/app/hooks/useTags.ts`: タグ一覧取得フック(候補表示用)。
- コンポーネント:
  - タグ選択/作成 UI(既存タグから選択 + 新規作成のコンボボックス)を、デッキ作成・編集フォームおよびデッキコード登録フォームに追加。`components/organisms/Deck/**` の既存フォームに組み込む。
  - デッキ一覧カード / デッキ詳細でタグを chip 表示(`color` を反映)。

---

## 4. Consequences

**Pros**
- FK 制約と型安全性を維持したまま、既存の中間テーブル規約(`*_pokemon_sprites`)に完全準拠。
- 記録/対戦結果への拡張が「中間テーブル1つ + `tagLinkTable` 1行 + 薄い配線」で済む(D4)。
- タグ付与の UX は既存の sprite 付与と同じメンタルモデル(親リクエストに埋め込み)で一貫。

**Cons / トレードオフ**
- エンティティが増えるごとに中間テーブルが増える(ポリモーフィックより物理テーブル数は多い)。→ 対象が有限・低頻度で増える前提のため許容。共通ヘルパで実装コストは最小化。
- 「全エンティティ横断でこのタグが付いた対象を一覧する」ようなクエリは中間テーブルを UNION する必要がある。→ 現状の要件(デッキ/デッキコード単位の分類)には不要。将来必要になったらビュー化を検討。

**未決事項 / 今後の検討**
- タグによる検索・フィルタ API(一覧エンドポイントへの `tag_id` クエリ追加)は別ADRで扱う。
- タグ数の上限、色パレットの固定/自由入力の是非は UI 検討時に確定する。
