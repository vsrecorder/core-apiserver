-- タグ機能 + プリセットタグ(ACE SPEC): 稼働中DBへ追加適用するマイグレーション
--
-- db/schema.sql を正とする運用のため、本ファイルは既存の稼働DBに tags 関連の差分だけを
-- 適用する。すべて IF NOT EXISTS で冪等にしてあり、次のいずれの状態でも、また複数回
-- 実行しても安全:
--   - タグ機能が未導入(tags テーブルが無い)
--   - タグ機能は導入済みだが preset_flg だけ無い(第1弾のみ適用済み)
--
-- 適用例:
--   docker exec -i <db-container> psql -U vsrecorder -d vsrecorder < db/migration_add_tags.sql

BEGIN;

-- タグマスタ(ユーザーごとのタグ名前空間 + 全ユーザー共通のプリセット)
CREATE TABLE IF NOT EXISTS tags (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    name        VARCHAR(32) NOT NULL,
    color       VARCHAR(7)  DEFAULT NULL,
    -- preset_flg=true は運営が用意する全ユーザー共通のプリセットタグ(例: ACE SPEC)。
    preset_flg  BOOLEAN NOT NULL DEFAULT false
);

-- タグ機能を先に導入済み(preset_flg 無し)の環境向け。
-- 未導入環境では上の CREATE で既に preset_flg を持つため、この ALTER は no-op になる。
ALTER TABLE tags ADD COLUMN IF NOT EXISTS preset_flg BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_tags_created_at ON tags(created_at);
CREATE INDEX IF NOT EXISTS idx_tags_deleted_at ON tags(deleted_at);
CREATE INDEX IF NOT EXISTS idx_tags_user_id    ON tags(user_id);
-- 同一ユーザー内でタグ名は一意。プリセットは user_id='' を共有するため、プリセット名の重複も防ぐ。
CREATE UNIQUE INDEX IF NOT EXISTS unique_tags_user_id_name ON tags (user_id, name) WHERE deleted_at IS NULL;
-- プリセット一覧(GET /tags/presets)の取得用。
CREATE INDEX IF NOT EXISTS idx_tags_preset ON tags(name) WHERE preset_flg = true AND deleted_at IS NULL;

-- デッキ ⇔ タグ。position は付与した順(1始まり、表示は position 昇順で1が先頭)。
CREATE TABLE IF NOT EXISTS deck_tags (
    deck_id  VARCHAR(26) NOT NULL,
    tag_id   VARCHAR(26) NOT NULL,
    position SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (deck_id, tag_id),
    FOREIGN KEY (deck_id) REFERENCES decks(id),
    FOREIGN KEY (tag_id)  REFERENCES tags(id)
);
CREATE INDEX IF NOT EXISTS idx_deck_tags_tag_id ON deck_tags(tag_id);

-- デッキコード(バージョン) ⇔ タグ
CREATE TABLE IF NOT EXISTS deck_code_tags (
    deck_code_id  VARCHAR(26) NOT NULL,
    tag_id        VARCHAR(26) NOT NULL,
    position      SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (deck_code_id, tag_id),
    FOREIGN KEY (deck_code_id) REFERENCES deck_codes(id),
    FOREIGN KEY (tag_id)       REFERENCES tags(id)
);
CREATE INDEX IF NOT EXISTS idx_deck_code_tags_tag_id ON deck_code_tags(tag_id);

-- 対戦結果(match) ⇔ タグ
CREATE TABLE IF NOT EXISTS match_tags (
    match_id  VARCHAR(26) NOT NULL,
    tag_id    VARCHAR(26) NOT NULL,
    position  SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (match_id, tag_id),
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (tag_id)   REFERENCES tags(id)
);
CREATE INDEX IF NOT EXISTS idx_match_tags_tag_id ON match_tags(tag_id);

-- 中間テーブルを先に作成済み(position 無し)の環境向け。未作成環境では上の CREATE で
-- 既に position を持つため、この ALTER は no-op になる。
ALTER TABLE deck_tags      ADD COLUMN IF NOT EXISTS position SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE deck_code_tags ADD COLUMN IF NOT EXISTS position SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE match_tags     ADD COLUMN IF NOT EXISTS position SMALLINT NOT NULL DEFAULT 1;

-- position を先に DEFAULT 0 で導入済みの環境向けに、既定値も1始まりへ揃える。
ALTER TABLE deck_tags      ALTER COLUMN position SET DEFAULT 1;
ALTER TABLE deck_code_tags ALTER COLUMN position SET DEFAULT 1;
ALTER TABLE match_tags     ALTER COLUMN position SET DEFAULT 1;

-- 既存行の position を「オーナー単位で1始まりの連番(1..N)」へ振り直す。
-- 旧実装(0始まり)や DEFAULT 0 で入った行を1始まりへ補正する。並びは現在の表示順
-- (position 昇順、同値はタグ作成日時の降順)を保つので、表示上の順序は変わらない。
-- ROW_NUMBER は常に 1..N を返すため、何度実行しても結果は同じ(冪等)。
UPDATE deck_tags dt
SET position = r.rn
FROM (
    SELECT x.deck_id, x.tag_id,
           ROW_NUMBER() OVER (
               PARTITION BY x.deck_id
               ORDER BY x.position ASC, t.created_at DESC
           ) AS rn
    FROM deck_tags x
    JOIN tags t ON t.id = x.tag_id
) r
WHERE dt.deck_id = r.deck_id AND dt.tag_id = r.tag_id AND dt.position <> r.rn;

UPDATE deck_code_tags dct
SET position = r.rn
FROM (
    SELECT x.deck_code_id, x.tag_id,
           ROW_NUMBER() OVER (
               PARTITION BY x.deck_code_id
               ORDER BY x.position ASC, t.created_at DESC
           ) AS rn
    FROM deck_code_tags x
    JOIN tags t ON t.id = x.tag_id
) r
WHERE dct.deck_code_id = r.deck_code_id AND dct.tag_id = r.tag_id AND dct.position <> r.rn;

UPDATE match_tags mt
SET position = r.rn
FROM (
    SELECT x.match_id, x.tag_id,
           ROW_NUMBER() OVER (
               PARTITION BY x.match_id
               ORDER BY x.position ASC, t.created_at DESC
           ) AS rn
    FROM match_tags x
    JOIN tags t ON t.id = x.tag_id
) r
WHERE mt.match_id = r.match_id AND mt.tag_id = r.tag_id AND mt.position <> r.rn;

GRANT SELECT ON tags           TO grafana;
GRANT SELECT ON deck_tags      TO grafana;
GRANT SELECT ON deck_code_tags TO grafana;
GRANT SELECT ON match_tags     TO grafana;

COMMIT;
