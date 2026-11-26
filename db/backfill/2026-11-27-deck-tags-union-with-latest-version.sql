-- デッキのタグと最新バージョンのタグを、消さずに足し合わせて揃える(和集合)。
--
-- デッキのタグと最新バージョンのタグは、書き込みのたびに双方向で揃えるようにした
-- (usecase/deck_tag_sync.go)。それ以前のデータには食い違いがあり、そのままだと次の保存で
-- 片側のタグが黙って消える(例: デッキにだけ ACE SPEC が付いたデッキで、最新バージョンのメモを
-- 直すとデッキの ACE SPEC が消える)。2026-11-27 時点の本番で 1,343 デッキ中 107 デッキが該当。
-- 大半は、最初のバージョンを作る画面がタグを空で始めていたため、デッキにだけタグが残ったものと見ている。
--
-- ユーザーが付けたタグを消さないよう、相手側にだけあるタグを足すだけにする:
--   - 足す先の既存のタグは消さず、並びも変えない。足すタグは既存の後ろに、相手側での並び順で続ける
--   - ACE SPEC・大会順位のプリセットは同じ群から1つしか付けられない。足す先に同じ群の
--     プリセットが既にあれば、そのプリセットは足さない(そのデッキは揃わないまま残る)
--   - 足した結果がタグ数の上限(20)を超える分は足さない
--   - 削除されたデッキ・バージョン・タグは対象外。過去のバージョンには触れない
--   - 何度流しても結果は同じ(2回目以降は 0 行)
--
-- 適用の順序: 新しい core-apiserver(タグの双方向同期)をデプロイしたあと。
--             先に流すと、デプロイまでのあいだの保存で再び食い違いが生まれ得る。
--
-- 実行: psql -v ON_ERROR_STOP=1 -f db/backfill/2026-11-27-deck-tags-union-with-latest-version.sql

BEGIN;

-- 足す前に、足す内容の内訳を出しておく(足す先・タグの行数・デッキ数)
WITH latest AS (
    -- デッキごとの最新バージョン(API の latest_deck_code と同じ決め方)。削除済みのデッキ・バージョンは除く
    SELECT DISTINCT ON (dc.deck_id) dc.id AS deck_code_id, dc.deck_id, d.user_id
    FROM deck_codes dc
    JOIN decks d ON d.id = dc.deck_id AND d.deleted_at IS NULL
    WHERE dc.deleted_at IS NULL
    ORDER BY dc.deck_id, dc.created_at DESC, dc.updated_at DESC
),
cur AS (
    -- デッキと最新バージョンに付いている有効な(削除されていない)タグ
    SELECT 'deck' AS side, l.deck_id, l.deck_code_id, l.user_id, x.tag_id, x.position,
           t.created_at AS tag_created_at, t.preset_flg, t.preset_category
    FROM latest l
    JOIN deck_tags x ON x.deck_id = l.deck_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
    UNION ALL
    SELECT 'version', l.deck_id, l.deck_code_id, l.user_id, x.tag_id, x.position,
           t.created_at, t.preset_flg, t.preset_category
    FROM latest l
    JOIN deck_code_tags x ON x.deck_code_id = l.deck_code_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
),
missing AS (
    -- 相手側にだけあるタグ(= 足す候補)。target は足す先
    SELECT CASE src.side WHEN 'deck' THEN 'version' ELSE 'deck' END AS target, src.*,
           -- 足す先に同じ排他群(ACE SPEC・大会順位は1つだけ)のプリセットが既にある
           (src.preset_flg AND src.preset_category IN ('acespec', 'placement') AND EXISTS (
               SELECT 1 FROM cur dst
               WHERE dst.deck_id = src.deck_id AND dst.side <> src.side
                 AND dst.preset_flg AND dst.preset_category = src.preset_category
           )) AS exclusive_conflict
    FROM cur src
    WHERE NOT EXISTS (
        SELECT 1 FROM cur dst
        WHERE dst.deck_id = src.deck_id AND dst.side <> src.side AND dst.tag_id = src.tag_id
    )
),
numbered AS (
    -- 足す順は相手側での並び順。足す先の既存タグの後ろへ続けて採番する
    SELECT m.*,
           row_number() OVER (
               PARTITION BY m.target, m.deck_id
               ORDER BY m.position, m.tag_created_at DESC
           ) AS n
    FROM missing m
    WHERE NOT m.exclusive_conflict
),
plan AS (
    SELECT nb.target, nb.deck_id, nb.deck_code_id, nb.user_id, nb.tag_id, nb.preset_flg,
           (mp.max_pos + nb.n)::smallint AS position,
           ac.active + nb.n > 20 AS over_limit
    FROM numbered nb
    CROSS JOIN LATERAL (
        -- 足す先の position の最大(削除済みタグの行も含めて数え、番号を重ねない)
        SELECT COALESCE(max(p.position), 0) AS max_pos FROM (
            SELECT position FROM deck_tags WHERE nb.target = 'deck' AND deck_id = nb.deck_id
            UNION ALL
            SELECT position FROM deck_code_tags WHERE nb.target = 'version' AND deck_code_id = nb.deck_code_id
        ) p
    ) mp
    CROSS JOIN LATERAL (
        -- 足す先に付いている有効なタグの数(上限 20 の判定に使う)
        SELECT count(*) AS active FROM cur c WHERE c.deck_id = nb.deck_id AND c.side = nb.target
    ) ac
)
SELECT target, count(*) FILTER (WHERE NOT over_limit) AS tags_to_add,
       count(DISTINCT deck_id) FILTER (WHERE NOT over_limit) AS decks,
       count(*) FILTER (WHERE over_limit) AS skipped_over_limit
FROM plan GROUP BY target ORDER BY target;

-- 両側への追加は1つの文で行う。同じ文の中の CTE は同じ時点のデータを見るので、
-- 片側へ足した結果がもう片側の計算に混ざらない
WITH latest AS (
    -- デッキごとの最新バージョン(API の latest_deck_code と同じ決め方)。削除済みのデッキ・バージョンは除く
    SELECT DISTINCT ON (dc.deck_id) dc.id AS deck_code_id, dc.deck_id, d.user_id
    FROM deck_codes dc
    JOIN decks d ON d.id = dc.deck_id AND d.deleted_at IS NULL
    WHERE dc.deleted_at IS NULL
    ORDER BY dc.deck_id, dc.created_at DESC, dc.updated_at DESC
),
cur AS (
    -- デッキと最新バージョンに付いている有効な(削除されていない)タグ
    SELECT 'deck' AS side, l.deck_id, l.deck_code_id, l.user_id, x.tag_id, x.position,
           t.created_at AS tag_created_at, t.preset_flg, t.preset_category
    FROM latest l
    JOIN deck_tags x ON x.deck_id = l.deck_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
    UNION ALL
    SELECT 'version', l.deck_id, l.deck_code_id, l.user_id, x.tag_id, x.position,
           t.created_at, t.preset_flg, t.preset_category
    FROM latest l
    JOIN deck_code_tags x ON x.deck_code_id = l.deck_code_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
),
missing AS (
    -- 相手側にだけあるタグ(= 足す候補)。target は足す先
    SELECT CASE src.side WHEN 'deck' THEN 'version' ELSE 'deck' END AS target, src.*,
           -- 足す先に同じ排他群(ACE SPEC・大会順位は1つだけ)のプリセットが既にある
           (src.preset_flg AND src.preset_category IN ('acespec', 'placement') AND EXISTS (
               SELECT 1 FROM cur dst
               WHERE dst.deck_id = src.deck_id AND dst.side <> src.side
                 AND dst.preset_flg AND dst.preset_category = src.preset_category
           )) AS exclusive_conflict
    FROM cur src
    WHERE NOT EXISTS (
        SELECT 1 FROM cur dst
        WHERE dst.deck_id = src.deck_id AND dst.side <> src.side AND dst.tag_id = src.tag_id
    )
),
numbered AS (
    -- 足す順は相手側での並び順。足す先の既存タグの後ろへ続けて採番する
    SELECT m.*,
           row_number() OVER (
               PARTITION BY m.target, m.deck_id
               ORDER BY m.position, m.tag_created_at DESC
           ) AS n
    FROM missing m
    WHERE NOT m.exclusive_conflict
),
plan AS (
    SELECT nb.target, nb.deck_id, nb.deck_code_id, nb.user_id, nb.tag_id, nb.preset_flg,
           (mp.max_pos + nb.n)::smallint AS position,
           ac.active + nb.n > 20 AS over_limit
    FROM numbered nb
    CROSS JOIN LATERAL (
        -- 足す先の position の最大(削除済みタグの行も含めて数え、番号を重ねない)
        SELECT COALESCE(max(p.position), 0) AS max_pos FROM (
            SELECT position FROM deck_tags WHERE nb.target = 'deck' AND deck_id = nb.deck_id
            UNION ALL
            SELECT position FROM deck_code_tags WHERE nb.target = 'version' AND deck_code_id = nb.deck_code_id
        ) p
    ) mp
    CROSS JOIN LATERAL (
        -- 足す先に付いている有効なタグの数(上限 20 の判定に使う)
        SELECT count(*) AS active FROM cur c WHERE c.deck_id = nb.deck_id AND c.side = nb.target
    ) ac
),
added_to_version AS (
    INSERT INTO deck_code_tags (deck_code_id, tag_id, position)
    SELECT deck_code_id, tag_id, position FROM plan
    WHERE target = 'version' AND NOT over_limit
    RETURNING 1
),
added_to_deck AS (
    INSERT INTO deck_tags (deck_id, tag_id, position)
    SELECT deck_id, tag_id, position FROM plan
    WHERE target = 'deck' AND NOT over_limit
    RETURNING 1
)
SELECT (SELECT count(*) FROM added_to_version) AS added_to_version,
       (SELECT count(*) FROM added_to_deck) AS added_to_deck;

-- 確認: まだ揃っていないデッキの数。排他のプリセットが食い違うデッキと、上限で足しきれなかった
-- デッキだけが残る(足す前の内訳の skipped_over_limit と、事前確認の排他の件数で説明がつく数になる)
WITH latest AS (
    SELECT DISTINCT ON (dc.deck_id) dc.id AS deck_code_id, dc.deck_id
    FROM deck_codes dc
    JOIN decks d ON d.id = dc.deck_id AND d.deleted_at IS NULL
    WHERE dc.deleted_at IS NULL
    ORDER BY dc.deck_id, dc.created_at DESC, dc.updated_at DESC
),
cur AS (
    SELECT 'deck' AS side, l.deck_id, x.tag_id
    FROM latest l
    JOIN deck_tags x ON x.deck_id = l.deck_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
    UNION ALL
    SELECT 'version', l.deck_id, x.tag_id
    FROM latest l
    JOIN deck_code_tags x ON x.deck_code_id = l.deck_code_id
    JOIN tags t ON t.id = x.tag_id AND t.deleted_at IS NULL
)
SELECT count(DISTINCT src.deck_id) AS remaining_mismatched_decks
FROM cur src
WHERE NOT EXISTS (
    SELECT 1 FROM cur dst
    WHERE dst.deck_id = src.deck_id AND dst.side <> src.side AND dst.tag_id = src.tag_id
);

COMMIT;
