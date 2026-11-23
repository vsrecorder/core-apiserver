-- 大会結果のプレイヤー名を players へ切り出す(2/2: 結果テーブルの player_name 列を消す)。
--
-- db/backfill/2026-11-23-players.sql の「適用の順序」1〜4 が終わってから流す。
-- 古い core-apiserver・取り込みバッチは player_name を読み書きするので、どれか1つでも
-- 残っているうちに流すとそれが失敗する。
--
-- 名前を取りこぼしていないことを先に確かめ、取りこぼしがあれば列を消さずに止める
-- (その場合は 2026-11-23-players.sql をもう一度流してから、このファイルを流し直す)。
--
-- 実行: psql -v ON_ERROR_STOP=1 -f db/backfill/2026-11-23-drop-player-name.sql

BEGIN;

DO $$
DECLARE
    missing integer;
BEGIN
    -- 消すと失われる名前(player_name 列にだけ名前があり、players に行が無い)を数える。
    -- 名前が空の結果(取り込みバッチは空の名前を players に書かない)は失うものが無いので数えない
    SELECT count(*) INTO missing
    FROM (
        SELECT player_id FROM cityleague_results      WHERE player_name <> ''
        UNION
        SELECT player_id FROM championsleague_results WHERE player_name <> ''
    ) AS r
    WHERE NOT EXISTS (SELECT 1 FROM players p WHERE p.id = r.player_id);

    IF missing > 0 THEN
        RAISE EXCEPTION 'players に名前が無いプレイヤーIDが % 件あります。2026-11-23-players.sql を流し直してください', missing;
    END IF;
END
$$;

ALTER TABLE cityleague_results      DROP COLUMN IF EXISTS player_name;
ALTER TABLE championsleague_results DROP COLUMN IF EXISTS player_name;

COMMIT;
