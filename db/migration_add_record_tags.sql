-- 記録(record)へのタグ付与を稼働中DBへ適用するマイグレーション。
-- db/schema.sql は新規構築用の正で、稼働DBへは流せない(既存テーブルがあるため)。
-- このファイルは適用が済んだら削除する(既存の migration_add_tags.sql と同じ運用)。
--
--   psql -U vsrecorder -d vsrecorder -f db/migration_add_record_tags.sql
--
-- 冪等: 何度流しても同じ結果になるよう IF NOT EXISTS / ON CONFLICT を使う。

BEGIN;

-- 1. プリセットタグの群(ACE SPEC / 大会順位)を区別する列を足す。
--    既存のプリセットは全て ACE SPEC なので 'acespec' に寄せる。
ALTER TABLE tags ADD COLUMN IF NOT EXISTS preset_category VARCHAR(16) NOT NULL DEFAULT '';

-- プリセットが配色(背景色に乗せる文字色)を指定するための列。
-- 空なら表示側が背景の明るさから白/黒を選ぶので、既存タグの見た目は変わらない。
ALTER TABLE tags ADD COLUMN IF NOT EXISTS text_color VARCHAR(7) NOT NULL DEFAULT '';

UPDATE tags
   SET preset_category = 'acespec'
 WHERE preset_flg = true
   AND preset_category = '';

-- プリセット一覧は群で絞って id 昇順に引くため、索引も張り替える。
DROP INDEX IF EXISTS idx_tags_preset;
CREATE INDEX idx_tags_preset ON tags(preset_category, id) WHERE preset_flg = true AND deleted_at IS NULL;

-- 2. 大会順位のプリセットタグを投入する。
--    id は schema.sql と同じ固定ULID(生成順に単調増加。この並びが表示順になる)。
--    名前の絵文字と配色はシティリーグ入賞バッジ(webapp の utils/cityleagueRank.ts)と揃える。
--    既存行があれば名前・配色を上書きする。投入後に定義(絵文字や色)を直しても、
--    このファイルを流し直せば付与済みのタグへそのまま反映される(id は変えない)。
INSERT INTO tags (id, created_at, updated_at, user_id, name, color, preset_flg, preset_category, text_color) VALUES
    ('01M11HEQG0XAGJ76V8SSGJ8E19', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥇 優勝',   '#FFB900', true, 'placement', '#461901'),
    ('01M11HEQG0XAGJ76V8ST8Z633W', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥈 準優勝', '#D4D4D8', true, 'placement', '#27272A'),
    ('01M11HEQG0XAGJ76V8SY7H9KWQ', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥉 ベスト4', '#FF8904', true, 'placement', '#441306'),
    ('01M11HEQG0XAGJ76V8SZQA07ZQ', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト8',    '#2B7FFF', true, 'placement', '#FFFFFF'),
    ('01M11HEQG0XAGJ76V8T3CC63GA', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト16',   '#00BC7D', true, 'placement', '#FFFFFF'),
    ('01M11HEQG0XAGJ76V8T3J6KC2X', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト32',   '#8E51FF', true, 'placement', '#FFFFFF')
ON CONFLICT (id) DO UPDATE SET
    name            = EXCLUDED.name,
    color           = EXCLUDED.color,
    preset_flg      = EXCLUDED.preset_flg,
    preset_category = EXCLUDED.preset_category,
    text_color      = EXCLUDED.text_color,
    updated_at      = EXCLUDED.updated_at;

-- 3. 記録 ⇔ タグの中間テーブル。deck_tags / match_tags と同じ規約。
CREATE TABLE IF NOT EXISTS record_tags (
    record_id VARCHAR(26) NOT NULL,
    tag_id    VARCHAR(26) NOT NULL,
    position  SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (record_id, tag_id),
    FOREIGN KEY (record_id) REFERENCES records(id),
    FOREIGN KEY (tag_id)    REFERENCES tags(id)
);

CREATE INDEX IF NOT EXISTS idx_record_tags_tag_id ON record_tags(tag_id);

GRANT SELECT ON record_tags TO grafana;

COMMIT;
