-- 大会結果のプレイヤー名を players へ切り出す(1/2: テーブルの作成と名前の移し替え)。
--
-- cityleague_results・championsleague_results は入賞1件ごとに player_name を持っていた。
-- 名前を players(1人1行)へ移し、結果テーブルは player_id だけを持つようにする(db/schema.sql の
-- players のコメント参照)。名前は最新の1つだけを持つ。同じIDで名前が複数ある人は、開催日が
-- いちばん新しい大会の名前を使う(同じ日なら official_event_id の大きい方)。
--
-- このファイルでやること:
--   1. players を作る(既にあれば何もしない)
--   2. 両方の結果テーブルの player_name から players を作る
--   3. 結果テーブルの player_name を NULL 可にする
--      (新しい取り込みバッチは player_name を書かないため。列自体は 2/2 で消す)
--
-- 何度流してもよい。2回目以降は「players にある名前より新しい大会の名前」だけを反映する。
-- 取り込みバッチを入れ替えるまでのあいだに古いバッチが取り込んだ結果は player_name にしか
-- 名前が無いので、バッチを入れ替えたあとにもう一度流して拾う。
--
-- 適用の順序:
--   1. このファイルを流す(古い core-apiserver・取り込みバッチはそのまま動く)
--   2. core-apiserver をデプロイする(名前を players から引くようになる)
--   3. 取り込みバッチ(import-cityleague-result-job / import-cityleague-result-bat /
--      import-championsleague-result-bat)をデプロイする(名前を players へ書くようになる)
--   4. このファイルをもう一度流す(1〜3 のあいだに古いバッチが取り込んだ名前を拾う)
--   5. db/backfill/2026-11-23-drop-player-name.sql で player_name 列を消す
--
-- 実行: psql -v ON_ERROR_STOP=1 -f db/backfill/2026-11-23-players.sql

BEGIN;

CREATE TABLE IF NOT EXISTS players (
    id                 VARCHAR(10) PRIMARY KEY,
    name               VARCHAR(255) NOT NULL,
    latest_event_date  DATE NOT NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 開催日が無い行(event_date IS NULL)は、名前の候補としてはいちばん古い扱いにする。
-- 開催日のある大会が1件でもあればそちらの名前を使い、無ければその行の名前を使う。
INSERT INTO players (id, name, latest_event_date)
SELECT DISTINCT ON (r.player_id)
       r.player_id,
       r.player_name,
       COALESCE(r.event_date, DATE '1970-01-01')
FROM (
    SELECT player_id, player_name, event_date, official_event_id
    FROM cityleague_results
    WHERE player_name IS NOT NULL
    UNION ALL
    SELECT player_id, player_name, event_date, official_event_id
    FROM championsleague_results
    WHERE player_name IS NOT NULL
) AS r
ORDER BY r.player_id, r.event_date DESC NULLS LAST, r.official_event_id DESC
ON CONFLICT (id) DO UPDATE
    SET name              = EXCLUDED.name,
        latest_event_date = EXCLUDED.latest_event_date,
        updated_at        = CURRENT_TIMESTAMP
    -- 新しいバッチが同じ日の大会から書いた名前を、古い列の名前で上書きしないよう「より新しい」に限る
    WHERE players.latest_event_date < EXCLUDED.latest_event_date;

ALTER TABLE cityleague_results      ALTER COLUMN player_name DROP NOT NULL;
ALTER TABLE championsleague_results ALTER COLUMN player_name DROP NOT NULL;

-- 確認: 結果テーブルに出てくるプレイヤーIDが、すべて players にあること(どちらも 0 になる)
SELECT count(DISTINCT player_id) AS cityleague_players_without_name
FROM cityleague_results r
WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.id = r.player_id);

SELECT count(DISTINCT player_id) AS championsleague_players_without_name
FROM championsleague_results r
WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.id = r.player_id);

COMMIT;

ANALYZE players;
