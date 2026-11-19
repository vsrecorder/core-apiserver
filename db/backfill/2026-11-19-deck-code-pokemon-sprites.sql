-- デッキのスプライトを、そのデッキの既存バージョン(デッキコード)すべてへ設定する。
--
-- バージョンごとにスプライトを持てるようにした(deck_code_pokemon_sprites)が、それ以前に作られた
-- バージョンには行が無く「未設定」になっている。未設定のバージョンはデッキのスプライトで
-- 表示されるため見た目は変わらないが、デッキのスプライトを後から変えると過去のバージョン
-- (とそれを使った記録)の表示まで一緒に変わってしまう。いまのデッキのスプライトを各バージョンに
-- 書き込んでおき、以後はバージョンごとに独立して持つようにする。
--
-- 対象: スプライトが設定されているデッキの、スプライト未設定のバージョン(削除済みは除く)。
--   - すでにスプライトを設定してあるバージョンは上書きしない
--   - デッキにスプライトが無ければ何もしない(未設定のまま)
--   - 何度流しても結果は同じ(2回目以降は 0 行)
--
-- 適用の順序: db/schema.sql の deck_code_pokemon_sprites を作成したあと、
--             新しい core-apiserver をデプロイする前後どちらでもよい。
--
-- 実行: psql -v ON_ERROR_STOP=1 -f db/backfill/2026-11-19-deck-code-pokemon-sprites.sql

BEGIN;

INSERT INTO deck_code_pokemon_sprites (deck_code_id, position, pokemon_sprite_id)
SELECT dc.id, dps.position, dps.pokemon_sprite_id
FROM deck_codes dc
JOIN deck_pokemon_sprites dps ON dps.deck_id = dc.deck_id
WHERE dc.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM deck_code_pokemon_sprites existing WHERE existing.deck_code_id = dc.id
  );

-- 確認: スプライトのあるデッキに、未設定のバージョンが残っていないこと(0 になる)
SELECT count(*) AS remaining_unset_versions
FROM deck_codes dc
WHERE dc.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM deck_pokemon_sprites dps WHERE dps.deck_id = dc.deck_id)
  AND NOT EXISTS (
      SELECT 1 FROM deck_code_pokemon_sprites existing WHERE existing.deck_code_id = dc.id
  );

COMMIT;
