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

---

## 5. 追補: プリセットタグ(全ユーザー共通) — 例: ACE SPEC

ユーザー個別タグに加え、運営が用意する**全ユーザー共通のプリセットタグ**を導入する。第一弾は ACE SPEC カード(1デッキ1枚の特別なカード)を、どのデッキ/デッキコードにも付けられる共通タグとして事前に用意する。

### D5. プリセットは `tags` に `preset_flg` を足して同居させる

新テーブルを作らず、`tags` に `preset_flg BOOLEAN NOT NULL DEFAULT false` を追加して表現する。プリセットは `preset_flg=true` かつ `user_id=''`(特定ユーザーに属さない)。

- 中間テーブル(`deck_tags` / `deck_code_tags`)はそのまま流用できる(`tag_id` が指す先がユーザータグかプリセットかを問わない)。別テーブルにすると join 側も二重になるため避ける。
- 一意制約 `unique_tags_user_id_name (user_id, name) WHERE deleted_at IS NULL` は、プリセットが `user_id=''` を共有するため**プリセット名の重複もそのまま一意に防ぐ**。

### 権限・付与ルール

- **付与**: 誰でも自分のデッキ/デッキコードにプリセットを付けられる。付与検証は `FindByIdsAndUserId` を **`FindAttachableByIds`**(`WHERE id IN ? AND (user_id = ? OR preset_flg = true)`)に置き換える。
- **編集/削除**: ユーザーはプリセットを編集・削除できない。既存の所有者チェック(`uid == tag.UserId`)が `user_id=''` により自然に 403 を返すため、追加実装は不要。
- **一覧**: `GET /tags` は自分のタグのみ(プリセット除外)。プリセットは `GET /tags/presets` で別に返し、フロントは別セクション表示。レスポンス `TagResponse` に `preset_flg` を追加。

### 投入は cards テーブルを情報源にしたバックフィルで行う

ACE SPEC かどうかの判定情報源は `cards.card_name` に付く `(ACE SPEC)` の目印(`deckcard-api` の `fetch_acespec_ids` と同規約)。`cmd/backfill-acespec-tags` が `cards` から名前を取得し、目印を除いた名前でプリセットタグ(`preset_flg=true` / `user_id=''` / 色は ACE SPEC カードのマゼンタ調 `#FF007F`。全 ACE SPEC 共通の1色)を作る。既存プリセットの色定義を変えた場合も、再実行で色だけ更新される。

タグの表示は、色を持つタグ(＝プリセット)は**背景をその色にして名前を白の太字**で描く。色を持たないユーザータグは既定の見た目のまま。白文字の可読性のため、色は白とのコントラストが取れる濃さのマゼンタにしている。

- **対象は現行スタンダードのレギュレーションマークのみ**: 既定で `regulation_mark = 'H'` に絞る(`-regulation-mark` で変更可)。旧マークで刷られた同名の再録を除外し、現行の ACE SPEC だけをプリセットにする。レギュレーション更新時は `-regulation-mark=I` のように指定して再実行する。
- **冪等**: 既存プリセットと名前で突き合わせ、未登録分だけ追加。新カードが増えたら再実行で差分投入。
- 記憶ベースのハードコード一覧にせず、実データを情報源にすることで正確・自動追随にする。

### 拡張性

将来 ACE SPEC 以外のプリセット群(例: スタジアム、特定の道具)を足す場合も同じ枠組みで足せる。群の区別が必要になったら `preset_category` 列の追加を検討する(現状は ACE SPEC のみのため未導入)。

---

## 6. 追補: 対戦結果(match)へのタグ付与

D4 の拡張性設計どおり、対戦結果(match)にもタグを付けられるようにした。デッキ/デッキコードと同じ枠組みで、追加は「中間テーブル1つ + `tagLinkTable` 1行 + 薄い配線」で済んだ。

- **スキーマ**: `match_tags(match_id, tag_id)` を `deck_tags` と同じ規約(FK制約・ソフトデリート列なし)で追加。
- **infra**: `matchTagLink = tagLinkTable{"match_tags", "match_id"}` を1つ足し、`ReplaceMatchTags` / `findTagsByMatchIds` を薄く生やすだけ(差分同期・バッチ読み出しの本体は共通ヘルパを流用)。読み出しは match の全 Find(FindById / FindByRecordId / FindByUserId / FindLatest)でスプライトと同様にバッチロード。
- **usecase/controller**: Match の Create/Update で所有権チェック済みタグを `ReplaceMatchTags` で同期。`MatchRequest` に `tag_ids`、`MatchResponse` に `tags` を追加。
- **タグ削除の連鎖**: `Tag.Delete` は `deck_tags` / `deck_code_tags` に加え `match_tags` の行も物理削除する(タグを消すと付与先すべてから外れる、を維持)。
- **frontend**: 対戦結果の作成/編集フォームにタグ付与(たたんだアコーディオン)を追加し、対戦一覧の各行にタグを表示。

記録(record)へ広げる場合も、残る `recordTagLink` を1つ足すだけで同様に実装できる。

---

## 7. 追補: 表示順の確定(付与順・プリセットのcard id順)と表示崩れ対策

運用しながら「並び順」に関する要件が固まったため、以下を確定した。

### D6. タグの表示は「付与した順」にする(中間テーブルに `position`)

当初は `tags.created_at` で並べていたが、これはタグの作成日順であって付与順ではない。中間テーブル(`deck_tags` / `deck_code_tags` / `match_tags`)に **`position SMALLINT`** を持たせ、付与順(=リクエストの `tag_ids` の並び)で採番して、読み出しは `position` 昇順で返す。

- **1始まり**: `ReplaceXxxTags` は `tag_ids` の並びどおり `position` を **1, 2, 3…** と採番する(最初に付与したタグが `position=1`、以降昇順)。列の `DEFAULT` も 1。
- **読み出し**: `findTagsByOwnerIds` は `ORDER BY <中間テーブル>.position ASC, tags.created_at DESC`(同値タイブレークのみ created_at)。
- **既存データの補正**: 稼働DBへ適用した `db/migration_add_tags.sql`(適用完了につき削除済み。以降は `db/schema.sql` が正)で、オーナー単位で `position` を 1..N へ振り直す `UPDATE`(`ROW_NUMBER()`)を流した。表示順(position昇順・created_at降順)と同順で振り直すので見た目は不変、かつ冪等。
- **注意点(採番の入力順)**: `FindAttachableByIds` は `WHERE id IN (...)` で **戻り順が不定**。usecase 側で付与順を保つため、共通ヘルパ `orderAttachableTagsByIds` で `tag_ids` の並びに整列し直してから `ReplaceXxxTags` に渡す(付与不可ID・重複も除去)。ここを戻り順のまま採番すると付与順にならない。

### D7. プリセット(ACE SPEC)は card id 昇順で並べる

プリセットは名前順(五十音/アルファベット順)よりも、収録・登場順に近い **card id 昇順**が自然。専用の並び順カラムは設けず、ULIDの生成順で表現する。

- **backfill**: 取得を `... GROUP BY card_name ORDER BY MIN(id) ASC`(同名再録は最小 card id で代表)にし、ULID発番を **`ulid.Monotonic`(単調増加)** にする。これで「作成順 = card id 昇順」が id の昇順に一致する。
- **読み出し**: `FindPresets` は `ORDER BY id ASC`。
- **限界と代替**: 既存プリセットの id は振り直されない(冪等な差分投入のため)。後から**小さい card id のカードを取りこぼし補充**した場合だけ末尾に付いて厳密な順序から外れる。厳密性が要るようになったら、明示的な並び順カラム(`preset_order`)を足して backfill が既存分も in-place で `UPDATE` する方式に切り替える(非破壊)。

### D8. `match_tags` は FK 参照先(`matches`)の後に定義する

`schema.sql` は上から順に流すため、`match_tags` の `FOREIGN KEY (match_id) REFERENCES matches(id)` を満たすには `matches` の `CREATE TABLE` より後に置く必要がある(当初 `deck_tags` の並びで前方に置いてしまい、`make integration-test` で `relation "match_tags" does not exist` になった)。この制約は `schema.sql` を頭から流す場合のもので、稼働DBへ適用した `db/migration_add_tags.sql`(削除済み)は `matches` が既存の前提だったため順序非依存だった。

### D9. 対戦一覧のタグ表示は折り返さず横スクロール

対戦一覧の行(HeroUI `Button`, `h-10` 固定 + `overflow-hidden`)にタグを**別行**で足すと、行数増加で相手デッキ名などが見切れる。タグを先攻/後攻・サイド数などの情報チップと**同じ1行**に置き、その行を `flex-nowrap + overflow-x-auto`(`[&>*]:shrink-0`)で**折り返さず横スクロール**にした。行の高さは 2 行のまま変わらず、タグは何個でも横スクロールで辿れる。

- **編集の変更検知**: 付与順が表示に効くため、`hasChanges` は集合一致ではなく**並び順も見る**比較にする(match/deck の更新モーダル)。同じ集合でも並べ替えれば更新できる。

### テスト

- `orderAttachableTagsByIds`(付与順整列・付与不可除去・重複1回化)の純粋ユニットテスト。
- 統合テスト: `position` の実値が 1 始まりで付与順に並ぶこと、`FindPresets` が id 昇順で返ること、backfill の `fetchAceSpecCardNames` が最小 card id 昇順で返すこと、`cleanAceSpecNames` の目印除去・重複除去。`make integration-test` は `./cmd/backfill-acespec-tags/` も対象に含める。

---

## 8. 追補: 記録(record)へのタグ付与と、プリセットの群(`preset_category`)

記録にもタグを付けられるようにした。D4 の想定どおり中間テーブル1つと `tagLinkTable` 1行で済んだが、**プリセットの出し分け**という新しい論点が出たので、ここに残す。

### D10. 記録に見せるプリセットは ACE SPEC ではなく「大会順位」

プリセット第一弾の ACE SPEC は「デッキに入れているカード」を表すラベルで、デッキ・デッキコード・対戦結果には意味があるが、**記録(＝1イベントぶんの参加記録)には関係しない**。記録に欲しいのは「優勝」「ベスト4」といった**その日の結果**である。

そこで、5節「拡張性」で先送りしていた `preset_category` 列を追加し、プリセットを群に分けて付与先ごとに出し分ける。

| 群 | 中身 | 出す場所 |
| --- | --- | --- |
| `acespec` | ACE SPECカード名 | デッキ / デッキコード / 対戦結果 |
| `placement` | 🥇 優勝 / 🥈 準優勝 / 🥉 ベスト4 / ベスト8 / ベスト16 / ベスト32 | 記録 |

- **スキーマ**: `tags.preset_category VARCHAR(16) NOT NULL DEFAULT ''`。ユーザー個別タグは常に空文字。プリセット索引も `tags(preset_category, id) WHERE preset_flg` に張り替えた(群で絞って id 昇順に引くため)。
- **API**: `GET /tags/presets?category=acespec|placement`。未指定・未定義の値は「群で絞らない(全プリセット)」に丸める(`ParseQueryEventType` と同じ方針)。既存クライアントは指定なしで従来どおり動く。
- **frontend**: `useTags(presetCategory)` が群ごとに別URLでプリセットを取り(SWRのキャッシュも群ごとに分かれる)、`TagSelector` の `presetCategory` プロップで見出しも切り替える(「ACE SPEC・プリセット」/「大会順位・プリセット」)。記録の付与UIだけ `placement` を渡す。

### D11. 大会順位のプリセットは `schema.sql` に固定ULIDで置く

ACE SPEC は実データ(`cards`)から導けるのでバックフィルバッチが投入するが、大会順位は**実データから導けない固定のマスタ**で、`regulations` / `environments` と同じ性質。バッチを1つ増やさず `schema.sql` の `INSERT` に置く。

- id は `ulid.Monotonic` で生成順に単調増加させた固定ULIDを埋め込む。`FindPresets` が id 昇順で引くので、**この並びがそのまま表示順(優勝→ベスト32)**になる(D7と同じ規約)。
- **見た目はシティリーグ入賞バッジ(webapp `utils/cityleagueRank.ts`)に揃える。** 同じ「優勝」がサービス内で別物に見えないようにするため、名前のメダル絵文字も背景色もそこから持ってくる。
  - **メダル絵文字はタグ名そのものに含める**(`🥇 優勝`)。上位3つだけ付けてベスト8以下は付けないのは `RANK_LABELS` と同一。表示側で名前から絵文字を導く実装(プリセット名のハードコード対応表)にすると、タグ表示の共通コンポーネント(`TagChips` / `TagSelector`)すべてに分岐が要る。名前に含めておけば付与先を問わず1つの見え方に揃う。検索は部分一致(`includes`)なので「優勝」と打てば従来どおりヒットする。
  - **配色は `cityleagueRankBadgeClass` と同じ値**(Tailwind v4 のパレットを sRGB へ落としたもの)。背景・文字色の組で、順に amber-400+amber-950 `#FFB900`/`#461901` / zinc-300+zinc-800 `#D4D4D8`/`#27272A` / orange-400+orange-950 `#FF8904`/`#441306` / blue-500+白 `#2B7FFF`/`#FFFFFF` / emerald-500+白 `#00BC7D`/`#FFFFFF`。ベスト32 は入賞バッジ側に対応が無いので、上の5色と混ざらない violet-500+白 `#8E51FF`/`#FFFFFF` を当てている。これで大会順位のプリセットは6件とも配色を自分で持ち、この群では自動判定に落ちない。

### D13. タグの文字色は `text_color`(任意) + 背景からの自動判定

入賞バッジは順位ごとに文字色も手で選んでいる(`bg-amber-400 text-amber-950` など)が、タグは `tags.color` を1つ持つだけで、描く側は名前を白の太字で乗せていた。金 `#FFB900`(白とのコントラスト比 1.72)・銀 `#D4D4D8`(1.48)・銅 `#FF8904`(2.38)はいずれも明るく、そのままでは名前が潰れる。

**背景から自動判定するだけでは足りない。** 自動判定は「読める色」を選ぶので、白とのコントラスト比が 2.47 しかない emerald-500(ベスト16)には暗い文字を当ててしまい、白を使っている入賞バッジと見た目が食い違う。同じ「ベスト16」がサービス内で2通りに見えるのは避けたいので、**配色を決め切りたいタグは自分で文字色を持てる**ようにした。

- `tags.text_color VARCHAR(7) NOT NULL DEFAULT ''` を追加。APIからは設定できず、投入時(`schema.sql` / backfill)にだけ決まる。大会順位のプリセットだけが値を持ち、他は空。
- 空のときは描画側(`utils/tagColor.ts` の `tagTextColor`)が背景の相対輝度から選ぶ。判定は「**白とのコントラスト比が 3:1 を下回るときだけ暗い文字(zinc-900 `#18181B`)に落とす**」。「白と黒でコントラストが高い方」を採る一般的なやり方にしていないのは、既存タグの見た目を変えないため(ACE SPEC `#FF007F` は白 3.78 / 黒 5.56 で、高い方を採ると黒に変わってしまう)。この線を挟んで最も近い色でも 2.47 と 3.76 で開きがあり、後からユーザーが色を付けても判定がぶれない。
- **文字色はチップ本体(`style`)に置く。** HeroUI の Chip は content スロットが `text-inherit`、closeButton も色クラスを持たないため、本体に色を置けば中身も×も継承する。スロットごとに `text-white` を指定して回っていた従来の書き方より指定箇所が減る。
- 稼働DBへは `db/migration_add_record_tags.sql`(列追加・既存プリセットの `acespec` 寄せ・順位プリセット投入・`record_tags` 作成)を流す。適用後は削除する(4節・7節と同じ運用)。

### D12. 更新APIは「送った集合に置き換える」ので、部分更新でも `tag_ids` を送り直す

記録の更新は全フィールドを受け取るPUTで、`tag_ids` も**送った集合に一致させる**。レギュレーションだけ変えるような部分更新で `tag_ids` を送り忘れると、**付与済みのタグが全部外れる**。

webapp 側は `updateRecordFields`(現在値を詰めてから patch を重ねる共通関数)が `record.tags` から `tag_ids` を復元するようにし、これを経由しない4箇所(イベント情報編集・TCGマイスターURL編集・使用デッキ変更×2)にも同じ復元を入れた。`RecordUpdateRequestType` に必須項目として足してあるので、新しい更新箇所を書いたときは型エラーで気付ける。

### 実装メモ

- **infra**: `recordTagLink = tagLinkTable{"record_tags", "record_id"}` を足し、`ReplaceRecordTags` / `findTagsByRecordIds` を薄く生やした。記録の Find は9メソッドあり同じ変換を書き写していたので、`newRecordEntity` / `newRecordEntitiesWithTags`(タグを1クエリでまとめ読み)へ集約した。0件のときはタグ取得自体を省く。
- **タグ削除の連鎖**: `Tag.Delete` は `record_tags` の行も物理削除する。
- **記録の削除時**: `record_tags` は残す。記録は論理削除で行が生き続けるため(対戦結果と同じ扱い)。
- **記録作成時**: `tag_ids` は受け付けるが、webapp の作成フォームからは常に空で送る。大会順位は結果が出てから付けるもので、作成時に選ばせる意味が薄いため。付与は記録詳細ページ・記録情報モーダルのボード内「タグ」パネル(レギュレーションの直下に置き、レギュレーション・戦績集計と同じく即時反映)から行う。
- **一覧表示**: 記録カードでは、タグを**レギュレーションチップと同じ行**(カード最上段)に置く。行を足すとタグの有無でカードの高さが変わって一覧がガタつくため、既にある固定高(`h-5`)の行へ流し込み、溢れる分は折り返さず横スクロールで辿らせる(D9と同じ判断)。並びは「レギュレーション → タグ → 集計対象外」で、例外を示す印である**「集計対象外」は常に最後**に置く。タグのコンテナだけを縮ませて(`min-w-0` / `flex-grow` なし)、タグが増えても最後の「集計対象外」が押し出されないようにしてある。この行にはカード右上のバッジ(チーム戦/BO3)が `absolute` で重なるため、バッジの有無に応じて行の右側に余白を確保して潜り込みを防いでいる。
