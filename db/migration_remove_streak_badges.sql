-- 週次ストリークバッジ(badge_definitions の streak-01〜04)の廃止に伴う既存データの片付け。
-- db/schema.sql からはシードを削除済みなので、稼働中のDBにも同じ状態を反映する。
-- 適用後、このファイルは削除してよい(スキーマの正は db/schema.sql)。
--
-- 対象外(消さないもの):
--   - user_streaks       … StreakPanel の「◯週連続記録中」とフリーズ枠。廃止していない。
--   - 途切れ防止nudgeの通知 … category は同じ 'streak' だが title が異なる
--                            ("連続記録がとぎれそうです")。notify-streak-nudge は継続稼働する。

BEGIN;

-- 1. 「ストリークを継続中です」通知。
--    バッジ定義が無くなり達成条件を確認する手段もバッジ一覧の表示も無くなるため、
--    残しておくと辿れない実績の通知だけが履歴に残ることになる。
DELETE FROM notifications
 WHERE category = 'streak'
   AND title = 'ストリークを継続中です';

-- 2. 週次ストリーク系はシーズンごとのライブ集計で user_badges には永続化しない仕様のため
--    通常は0件だが、過去のバックフィル等で行が残っていた場合に備えて定義より先に消す
--    (user_badges.badge_definition_id が badge_definitions を参照しているため)。
DELETE FROM user_badges
 WHERE badge_definition_id IN ('streak-01', 'streak-02', 'streak-03', 'streak-04');

-- 3. バッジ定義本体。id は欠番のまま残し、再利用しない。
DELETE FROM badge_definitions
 WHERE id IN ('streak-01', 'streak-02', 'streak-03', 'streak-04');

COMMIT;
