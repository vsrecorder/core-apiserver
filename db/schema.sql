CREATE TABLE prefectures (
    id        SMALLINT NOT NULL PRIMARY KEY,
    name      VARCHAR(255) DEFAULT NULL,
    name_kana VARCHAR(255) DEFAULT NULL
);

INSERT INTO prefectures VALUES
(0,'不明','フメイ'),
(1,'北海道','ホッカイドウ'),
(2,'青森県','アオモリケン'),
(3,'岩手県','イワテケン'),
(4,'宮城県','ミヤギケン'),
(5,'秋田県','アキタケン'),
(6,'山形県','ヤマガタケン'),
(7,'福島県','フクシマケン'),
(8,'茨城県','イバラキケン'),
(9,'栃木県','トチギケン'),
(10,'群馬県','グンマケン'),
(11,'埼玉県','サイタマケン'),
(12,'千葉県','チバケン'),
(13,'東京都','トウキョウト'),
(14,'神奈川県','カナガワケン'),
(15,'新潟県','ニイガタケン'),
(16,'富山県','トヤマケン'),
(17,'石川県','イシカワケン'),
(18,'福井県','フクイケン'),
(19,'山梨県','ヤマナシケン'),
(20,'長野県','ナガノケン'),
(21,'岐阜県','ギフケン'),
(22,'静岡県','シズオカケン'),
(23,'愛知県','アイチケン'),
(24,'三重県','ミエケン'),
(25,'滋賀県','シガケン'),
(26,'京都府','キョウトフ'),
(27,'大阪府','オオサカフ'),
(28,'兵庫県','ヒョウゴケン'),
(29,'奈良県','ナラケン'),
(30,'和歌山県','ワカヤマケン'),
(31,'鳥取県','トットリケン'),
(32,'島根県','シマネケン'),
(33,'岡山県','オカヤマケン'),
(34,'広島県','ヒロシマケン'),
(35,'山口県','ヤマグチケン'),
(36,'徳島県','トクシマケン'),
(37,'香川県','カガワケン'),
(38,'愛媛県','エヒメケン'),
(39,'高知県','コウチケン'),
(40,'福岡県','フクオカケン'),
(41,'佐賀県','サガケン'),
(42,'長崎県','ナガサキケン'),
(43,'熊本県','クマモトケン'),
(44,'大分県','オオイタケン'),
(45,'宮崎県','ミヤザキケン'),
(46,'鹿児島県','カゴシマケン'),
(47,'沖縄県','オキナワケン');

CREATE TABLE shops (
    id                            INT NOT NULL PRIMARY KEY,
    name                          VARCHAR(255) NOT NULL,
    term                          SMALLINT NOT NULL,
    zip_code                      VARCHAR(8) DEFAULT NULL,
    prefecture_id                 SMALLINT NOT NULL,
    address                       VARCHAR(255) DEFAULT NULL,
    tel                           VARCHAR(32) DEFAULT NULL,
    access                        TEXT DEFAULT NULL,
    business_hours                VARCHAR(255) DEFAULT NULL,
    url                           VARCHAR(255) DEFAULT NULL,
    geo_coding                    VARCHAR(63) DEFAULT NULL,
    FOREIGN KEY (prefecture_id)   REFERENCES prefectures (id)
);

INSERT INTO shops VALUES(
    0,
    '株式会社ポケモン',
    0,
    NULL,
    0,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
);

CREATE TABLE official_events (
    id                      INT NOT NULL PRIMARY KEY,
    title                   VARCHAR(255) NOT NULL,
    address                 VARCHAR(255) NOT NULL,
    venue                   VARCHAR(255) DEFAULT NULL,
    date                    DATE NOT NULL,
    started_at              TIMESTAMP DEFAULT NULL,
    ended_at                TIMESTAMP DEFAULT NULL,
    deck_count              VARCHAR(255) DEFAULT NULL,
    type_id                 INT DEFAULT NULL,
    type_name               VARCHAR(255) DEFAULT NULL,
    csp_flg                 BOOLEAN DEFAULT NULL,
    league_id               INT DEFAULT NULL,
    league_title            VARCHAR(255) DEFAULT NULL,
    regulation_id           INT DEFAULT NULL,
    regulation_title        VARCHAR(255) DEFAULT NULL,
    capacity                INT DEFAULT NULL,
    attr_id                 INT DEFAULT NULL,
    shop_id                 INT  DEFAULT NULL,
    shop_name               VARCHAR(255) DEFAULT NULL,
    FOREIGN KEY (shop_id)   REFERENCES shops (id)
);

-- Myジムのイベント一覧(OfficialEvent.FindByShopIds)は「登録店舗の、今日から2週間」を引く。
-- official_events は100万行を超えており、索引が無いと毎回この全件を走査する。
-- ホームのパネルは全ログインユーザが開くたびに叩くため、走査量がそのまま
-- 共有バッファを流し続けることになる(本番はDBとアプリが同一VMに同居している)。
--
-- 実測(2026-09-01・約113万行): 77.6ms / 42,032ブロック読み込み → 4.6ms / 10ブロック。
-- shop_id を先頭に置くのは、等値(IN)で絞ってから date の範囲を辿るため。
CREATE INDEX idx_official_events_shop_id_date ON official_events (shop_id, date);



-- Tonamel の大会情報(タイトル・説明・画像)を保持するキャッシュテーブル。
-- id は records.tonamel_event_id と同じ Tonamel の大会ID。
--
-- これらは tonamel.com の大会ページをスクレイピングして得るが、一括取得APIが無いため
-- 大会ごとに1リクエストかかる。カレンダー表示のたびに参照中の全大会を取り直すと、
-- 記録数に比例して外部サイトへのリクエストが増える(N+1)。大会情報はほぼ不変で
-- 全ユーザー共通なので、記録作成時に1度だけ取得してここへ保存し、以降は参照しない。
-- 既存記録ぶんは cmd/backfill-tonamel-events でまとめて投入する。
CREATE TABLE tonamel_events (
    id          VARCHAR(8) NOT NULL PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    image       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL
);



CREATE TABLE unofficial_events (
    id         VARCHAR(26)  NOT NULL PRIMARY KEY,
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NOT NULL,
    deleted_at TIMESTAMP    DEFAULT NULL,
    user_id    VARCHAR(32)  NOT NULL,
    title      VARCHAR(255) NOT NULL,
    date       DATE         NOT NULL
);

CREATE INDEX idx_unofficial_events_deleted_at ON unofficial_events(deleted_at);

CREATE TABLE decks (
    id               VARCHAR(26) PRIMARY KEY,
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL,
    deleted_at       TIMESTAMP DEFAULT NULL,
    archived_at      TIMESTAMP DEFAULT NULL,
    user_id          VARCHAR(32) NOT NULL,
    name             VARCHAR(32) NOT NULL,
    private_flg      BOOLEAN DEFAULT NULL
);

CREATE INDEX idx_decks_created_at ON decks(created_at);
CREATE INDEX idx_decks_deleted_at ON decks(deleted_at);
-- ユーザー単位の集計(きずなLv.・デッキ使用率統計)は必ず user_id で絞る。
-- 索引が無いとサービス全体の行数に比例して遅くなる。
CREATE INDEX idx_decks_user_id ON decks(user_id);

-- お気に入りのデッキ。1行が「あるユーザがあるデッキをお気に入りにしている」ことを表す。
-- decks の列ではなく別テーブルにしてあるのは、1ユーザが持てる件数を将来増やせるようにするため。
-- 現在の上限はアプリ側(usecase.MaxFavoriteDecksPerUser)が持ち、ここでは制約しない。
-- created_at はお気に入りにした日時で、一覧の並び順(古い順)にも使う。
CREATE TABLE user_favorite_decks (
    user_id     VARCHAR(32) NOT NULL,
    deck_id     VARCHAR(26) NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    -- 同じデッキを重複してお気に入りにできないことは主キーで担保する
    PRIMARY KEY (user_id, deck_id),
    FOREIGN KEY (deck_id) REFERENCES decks (id)
);

-- デッキ削除時に、そのデッキのお気に入りをまとめて消すために引く。
-- 主キーの先頭は user_id のため、deck_id 単独では主キー索引が使えない。
CREATE INDEX idx_user_favorite_decks_deck_id ON user_favorite_decks(deck_id);

CREATE TABLE deck_codes (
    id                    VARCHAR(26) PRIMARY KEY, 
    created_at            TIMESTAMP NOT NULL,
    updated_at            TIMESTAMP NOT NULL,
    deleted_at            TIMESTAMP DEFAULT NULL,
    user_id               VARCHAR(32) NOT NULL,
    deck_id               VARCHAR(26) NOT NULL,
    code                  VARCHAR(21) DEFAULT NULL,
    private_code_flg      BOOLEAN DEFAULT NULL,
    memo                  TEXT,
    FOREIGN KEY (deck_id) REFERENCES decks (id)
);

CREATE INDEX idx_deck_codes_created_at ON deck_codes(created_at);
CREATE INDEX idx_deck_codes_deleted_at ON deck_codes(deleted_at);
CREATE INDEX idx_deck_codes_user_id ON deck_codes(user_id);
-- デッキ一覧は「デッキごとに最新のデッキコード1件」を DISTINCT ON で引く(deck.go)。
-- 索引が無いと LIMIT 20 の一覧でも deck_codes 全件をソートしてから結合するため、
-- サービス全体のデッキコード数に比例して遅くなる。実測で 1176ms → 2ms(10万件時)。
-- DISTINCT ON / ORDER BY の並びと一致させる必要があるので、created_at・updated_at の
-- DESC まで含めた複合索引にする(deck_id 単独の索引ではソートを省けず効果が無い)。
CREATE INDEX idx_deck_codes_deck_id_created_at ON deck_codes(deck_id, created_at DESC, updated_at DESC);

-- タグマスタ。ユーザーごとにタグの名前空間を持つ(あるユーザーの「アグロ」と
-- 別ユーザーの「アグロ」は別レコード)。付与先(デッキ/デッキコード/記録/対戦結果)は
-- エンティティごとの中間テーブルで表す。単一のポリモーフィック中間テーブルにしないのは、
-- 参照先IDの型が混在してFK制約を張れず、既存の *_pokemon_sprites の規約に反するため。
CREATE TABLE tags (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    name        VARCHAR(32) NOT NULL,
    color       VARCHAR(7)  DEFAULT NULL,  -- '#RRGGBB' 任意。UI表示用
    -- preset_flg=true は運営が用意する全ユーザー共通の「プリセットタグ」
    -- (例: ACE SPECカード)。プリセットは user_id='' を持ち特定ユーザーに属さない。
    -- 誰でも自分のデッキ/デッキコードに付与できるが、編集・削除はできない
    -- (投入は cmd/backfill-acespec-tags が担う)。
    preset_flg  BOOLEAN NOT NULL DEFAULT false,
    -- プリセットの群。付与先ごとに出し分けるために使う
    -- ('acespec'=ACE SPECカード / 'placement'=大会順位)。
    -- ACE SPECはデッキ・デッキコード・対戦結果に、順位は記録に出す。
    -- ユーザー個別タグ(preset_flg=false)は常に空文字。
    preset_category VARCHAR(16) NOT NULL DEFAULT '',
    -- '#RRGGBB' 任意。color の上に乗せる文字色で、配色を指定したいプリセット用。
    -- 空なら表示側が color の明るさから白/黒を選ぶ(webapp の utils/tagColor.ts)。
    -- 大会順位のプリセットだけは、参照元のシティリーグ入賞バッジと寸分違わず揃えたいので
    -- 自動判定に任せず明示する。
    text_color      VARCHAR(7)  NOT NULL DEFAULT ''
);

CREATE INDEX idx_tags_created_at ON tags(created_at);
CREATE INDEX idx_tags_deleted_at ON tags(deleted_at);
CREATE INDEX idx_tags_user_id    ON tags(user_id);
-- 同一ユーザー内でタグ名は一意。解除済み(deleted_at IS NOT NULL)は対象外にして、
-- 同名のタグを消して作り直せるようにする。
-- プリセットは user_id='' を共有するため、この索引がプリセット名の重複も一意に防ぐ。
CREATE UNIQUE INDEX unique_tags_user_id_name ON tags (user_id, name) WHERE deleted_at IS NULL;
-- プリセットタグ一覧(GET /tags/presets)の取得用。群で絞り、id 昇順(=作成順)で少数を引く。
CREATE INDEX idx_tags_preset ON tags(preset_category, id) WHERE preset_flg = true AND deleted_at IS NULL;

-- 大会順位のプリセットタグ。記録(record)へ付けて「優勝した記録」を後から辿れるようにする。
-- ACE SPEC と違い実データから導けない固定のマスタなので、regulations 等と同じくここで投入する。
-- id は生成順に単調増加するULID(FindPresets が id 昇順で引くため、この並びが表示順になる)。
--
-- 名前の絵文字はシティリーグ入賞バッジ(webapp の utils/cityleagueRank.ts)と揃える。
-- 上位3つだけメダルを付け、ベスト8以下は付けない。同じ「優勝」がサービス内で
-- 別の見え方にならないようにするため、絵文字は表示側で足さず名前そのものに含める。
-- 配色はシティリーグ入賞バッジ(webapp の utils/cityleagueRank.ts の cityleagueRankBadgeClass)と
-- 背景・文字色とも同じ値。Tailwind v4 のパレットを sRGB に落としたもので、順に
-- amber-400+amber-950 / zinc-300+zinc-800 / orange-400+orange-950 / blue-500+白 / emerald-500+白。
-- ベスト32 は入賞バッジ側に対応が無いため、上の5色と混ざらない violet-500+白 を当てている。
INSERT INTO tags (id, created_at, updated_at, user_id, name, color, preset_flg, preset_category, text_color) VALUES
    ('01M11HEQG0XAGJ76V8SSGJ8E19', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥇 優勝',   '#FFB900', true, 'placement', '#461901'),
    ('01M11HEQG0XAGJ76V8ST8Z633W', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥈 準優勝', '#D4D4D8', true, 'placement', '#27272A'),
    ('01M11HEQG0XAGJ76V8SY7H9KWQ', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', '🥉 ベスト4', '#FF8904', true, 'placement', '#441306'),
    ('01M11HEQG0XAGJ76V8SZQA07ZQ', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト8',    '#2B7FFF', true, 'placement', '#FFFFFF'),
    ('01M11HEQG0XAGJ76V8T3CC63GA', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト16',   '#00BC7D', true, 'placement', '#FFFFFF'),
    ('01M11HEQG0XAGJ76V8T3J6KC2X', '2026-08-27 12:00:00', '2026-08-27 12:00:00', '', 'ベスト32',   '#8E51FF', true, 'placement', '#FFFFFF');

-- デッキ ⇔ タグ。中間テーブルはソフトデリート列を持たず、関連の解除は行の物理削除で表す
-- (deck_pokemon_sprites と同じ規約)。
CREATE TABLE deck_tags (
    deck_id  VARCHAR(26) NOT NULL,
    tag_id   VARCHAR(26) NOT NULL,
    -- position は付与した順(1始まり)。表示は position 昇順(1が先頭)。ReplaceDeckTags が採番する。
    position SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (deck_id, tag_id),
    FOREIGN KEY (deck_id) REFERENCES decks(id),
    FOREIGN KEY (tag_id)  REFERENCES tags(id)
);

-- タグ削除時に、そのタグの関連をまとめて消すために引く。
-- 主キーの先頭は deck_id のため、tag_id 単独では主キー索引が使えない。
CREATE INDEX idx_deck_tags_tag_id ON deck_tags(tag_id);

-- デッキコード(バージョン) ⇔ タグ。
CREATE TABLE deck_code_tags (
    deck_code_id  VARCHAR(26) NOT NULL,
    tag_id        VARCHAR(26) NOT NULL,
    -- position は付与した順(1始まり)。表示は position 昇順(1が先頭)。ReplaceDeckCodeTags が採番する。
    position      SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (deck_code_id, tag_id),
    FOREIGN KEY (deck_code_id) REFERENCES deck_codes(id),
    FOREIGN KEY (tag_id)       REFERENCES tags(id)
);

CREATE INDEX idx_deck_code_tags_tag_id ON deck_code_tags(tag_id);

-- 対戦結果(match) ⇔ タグの中間テーブルは、FK参照先の matches を定義した後で作成する
-- (matches の CREATE TABLE 直後、下の方に定義してある)。

-- 対戦記録のレギュレーション(使用可能カードの範囲)。記録作成時にユーザが選ぶ。
-- 既存の standard_regulations は「スタンダードで使えるレギュレーションマークの
-- 組み合わせと、その適用期間」を表す別物なので混同しないこと。
CREATE TABLE regulations (
    id   SMALLINT NOT NULL PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

INSERT INTO regulations VALUES (1,'スタンダード');
INSERT INTO regulations VALUES (2,'エクストラ');
INSERT INTO regulations VALUES (3,'殿堂');
-- 上のいずれにも当てはまらない対戦(独自ルールの自主大会など)を記録するための受け皿。
INSERT INTO regulations VALUES (4,'その他');

CREATE TABLE records (
    id                        VARCHAR(26) PRIMARY KEY,
    created_at                TIMESTAMP NOT NULL,
    updated_at                TIMESTAMP NOT NULL,
    deleted_at                TIMESTAMP DEFAULT NULL,
    -- deck_id/deck_code_idが未設定→設定ありに変わった日時。称号判定のasOf集計
    -- (CountRecordsAsOfByUserId)で「デッキ後付け登録」を正しく判定するために使う。
    deck_registered_at        TIMESTAMP DEFAULT NULL,
    official_event_id         INT DEFAULT NULL,
    tonamel_event_id          VARCHAR(8) DEFAULT NULL,
    friend_id                 VARCHAR(32) DEFAULT NULL,
    unofficial_event_id       VARCHAR(26) DEFAULT NULL,
    user_id                   VARCHAR(32) NOT NULL,
    deck_id                   VARCHAR(26) DEFAULT NULL,
    deck_code_id              VARCHAR(26) DEFAULT NULL,
    event_date                DATE DEFAULT NULL,
    private_flg               BOOLEAN DEFAULT NULL,
    ignore_stats_flg          BOOLEAN NOT NULL DEFAULT false,
    -- レギュレーション。既存の記録と、未指定でPOSTしてくる旧クライアントは
    -- スタンダード(1)として扱うため NOT NULL DEFAULT 1 にしてある。
    regulation_id             SMALLINT NOT NULL DEFAULT 1,
    tcg_meister_url           TEXT,
    memo                      TEXT,
    FOREIGN KEY (regulation_id) REFERENCES regulations (id)
);












CREATE INDEX idx_records_created_at ON records(created_at);
CREATE INDEX idx_records_deleted_at ON records(deleted_at);
CREATE INDEX idx_records_user_id ON records(user_id);

-- 記録(record) ⇔ タグ。deck_tags / match_tags と同じ規約
-- (中間テーブルはソフトデリート列を持たず、関連の解除は行の物理削除で表す)。
-- FK参照先の records を定義した後に作成する必要があるため、ここに置く。
CREATE TABLE record_tags (
    record_id VARCHAR(26) NOT NULL,
    tag_id    VARCHAR(26) NOT NULL,
    -- position は付与した順(1始まり)。表示は position 昇順(1が先頭)。ReplaceRecordTags が採番する。
    position  SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (record_id, tag_id),
    FOREIGN KEY (record_id) REFERENCES records(id),
    FOREIGN KEY (tag_id)    REFERENCES tags(id)
);

CREATE INDEX idx_record_tags_tag_id ON record_tags(tag_id);

-- 称号判定(designation_stats.go)の高速化用インデックス。records には user_id 索引しか無く、
-- official_event_id / event_date が未索引だったため、以下が records の全件走査になっていた:
--   1. cityleague_results と同じ official_event_id を持つ本人の記録の存在確認
--      (ベテラン〜名人の内部条件、および GetRankStats のユーザー横断集計での records 結合)。
--      → (official_event_id, user_id) の複合索引で直接シークにする。official_event_id は
--        Tonamel/記入形式の記録では NULL のため、部分索引で公式イベントの記録のみを対象にする
--        (対象クエリは常に official_event_id = ? で NOT NULL が含意されるため部分索引を利用できる)。
--   2. シーズン期間(event_date 範囲)でのユーザー横断集計(GetRankStats)。
--      → event_date の索引で範囲スキャンにする。
CREATE INDEX IF NOT EXISTS idx_records_official_event_id_user_id ON records (official_event_id, user_id) WHERE official_event_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_records_event_date ON records (event_date);

CREATE TABLE matches (
    id                        VARCHAR(26) PRIMARY KEY,
    created_at                TIMESTAMP NOT NULL,
    updated_at                TIMESTAMP NOT NULL,
    deleted_at                TIMESTAMP DEFAULT NULL,
    record_id                 VARCHAR(26) NOT NULL,
    deck_id                   VARCHAR(26) DEFAULT NULL,
    deck_code_id              VARCHAR(26) DEFAULT NULL,
    user_id                   VARCHAR(32) NOT NULL,
    opponents_user_id         VARCHAR(32) DEFAULT NULL,
    bo3_flg                   BOOLEAN NOT NULL,
    group_match_flg           BOOLEAN NOT NULL DEFAULT false,
    qualifying_round_flg      BOOLEAN NOT NULL,
    final_tournament_flg      BOOLEAN NOT NULL,
    default_victory_flg       BOOLEAN NOT NULL DEFAULT false,
    default_defeat_flg        BOOLEAN NOT NULL DEFAULT false,
    victory_flg               BOOLEAN NOT NULL,
    draw_flg                  BOOLEAN NOT NULL DEFAULT false,
    group_match_victory_flg   BOOLEAN NOT NULL DEFAULT false,
    opponents_deck_info       VARCHAR(63) DEFAULT NULL,
    memo                      TEXT,
    position                  INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (record_id)   REFERENCES records (id),
    -- 「勝ち かつ 引き分け」という矛盾した状態を禁止する
    CONSTRAINT matches_result_chk CHECK (NOT (victory_flg AND draw_flg)),
    -- 両者引き分け(ダブルドロー)を設定できるのはBO3のみ(BO1/チーム戦は勝ち負けのみ)
    CONSTRAINT matches_draw_bo3_chk CHECK (NOT draw_flg OR bo3_flg)
);

-- matches は集計の中心にあるが索引が無く、ユーザー単位の集計でも全件走査になっていた。
-- user_id は絞り込み用、record_id は records との結合用。
CREATE INDEX idx_matches_user_id ON matches(user_id);
CREATE INDEX idx_matches_record_id ON matches(record_id);

-- 対戦結果(match) ⇔ タグ。deck_tags / deck_code_tags と同じ規約
-- (中間テーブルはソフトデリート列を持たず、関連の解除は行の物理削除で表す)。
-- FK参照先の matches を定義した後に作成する必要があるため、ここに置く。
CREATE TABLE match_tags (
    match_id  VARCHAR(26) NOT NULL,
    tag_id    VARCHAR(26) NOT NULL,
    -- position は付与した順(1始まり)。表示は position 昇順(1が先頭)。ReplaceMatchTags が採番する。
    position  SMALLINT NOT NULL DEFAULT 1,
    PRIMARY KEY (match_id, tag_id),
    FOREIGN KEY (match_id) REFERENCES matches(id),
    FOREIGN KEY (tag_id)   REFERENCES tags(id)
);

CREATE INDEX idx_match_tags_tag_id ON match_tags(tag_id);

CREATE TABLE games (
    id                       VARCHAR(26) PRIMARY KEY,
    created_at               TIMESTAMP NOT NULL,
    updated_at               TIMESTAMP NOT NULL,
    deleted_at               TIMESTAMP DEFAULT NULL,
    match_id                 VARCHAR(26) NOT NULL,
    user_id                  VARCHAR(32) NOT NULL,
    go_first                 BOOLEAN DEFAULT NULL,
    winning_flg              BOOLEAN DEFAULT NULL,
    your_prize_cards         SMALLINT DEFAULT NULL,
    opponents_prize_cards    SMALLINT DEFAULT NULL,
    memo                     TEXT,
    FOREIGN KEY (match_id)   REFERENCES matches (id)
);

-- games は勝率・先攻/後攻率の集計で matches に必ず結合されるが、PostgreSQL は
-- 外部キーに索引を自動作成しないため、1ユーザーぶんの集計でも games 全件走査になっていた。
-- 実測で 863ms → 4.7ms(対局30万件時)。差はサービス全体の対局数に比例して開く。
CREATE INDEX idx_games_match_id ON games(match_id);

CREATE TABLE users (
    id          VARCHAR(32) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    name        VARCHAR(63) DEFAULT NULL,
    image_url   VARCHAR(255) DEFAULT NULL,
    -- 退会後、紐づくデータを cmd/purge-deleted-user-data で物理削除した日時。
    --
    -- 行そのものは消さずに残す(消すと usecase.User.Create の IsWithdrawn による
    -- 「退会済みUIDでの再登録拒否」が効かなくなり、check-deleted-users-data も
    -- users との結合で検出しているため消し残しを追えなくなる)。そのうえで
    -- 「退会しただけ」と「データはもう存在しない」を区別するための列。
    --
    -- 退会ユーザは全ユーザのごく一部で、絞り込みも一覧取得のあとGo側で行うため索引は張らない。
    -- 列の位置は本番の ALTER TABLE ... ADD COLUMN で付く位置(末尾)に合わせてある。
    purged_at   TIMESTAMP DEFAULT NULL
);

CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);





CREATE TABLE users_players (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    player_id   VARCHAR(16) NOT NULL
);

CREATE INDEX idx_users_players_created_at ON users_players(created_at);
CREATE INDEX idx_users_players_deleted_at ON users_players(deleted_at);

-- 有効な紐付け(deleted_at IS NULL)は1ユーザーにつき1件のみ。
--
-- player_id 側は一意にしない(同じ player_id を複数のユーザーが登録できる)。
-- 所有権の確認を行わない方針としたため、重複を禁止すると「先に登録した人が正しい
-- 持ち主とは限らない」状態で正当な利用者を締め出してしまうことになるため。
-- player_id は cityleague_results との結合に使うので索引自体は残す。
CREATE UNIQUE INDEX unique_users_players_user_id ON users_players (user_id)   WHERE deleted_at IS NULL;
CREATE INDEX idx_users_players_player_id         ON users_players (player_id) WHERE deleted_at IS NULL;





CREATE TABLE environments (
    id         VARCHAR(8) PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    from_date  DATE NOT NULL,
    to_date    DATE NOT NULL
);

INSERT INTO environments VALUES ('m6a','30th CELEBRATION','2026-09-16','2026-11-26');
INSERT INTO environments VALUES ('m6','ストームエメラルダ','2026-07-31','2026-09-15');
INSERT INTO environments VALUES ('m5','アビスアイ','2026-05-22','2026-07-30');
INSERT INTO environments VALUES ('m4','ニンジャスピナー','2026-03-13','2026-05-21');
INSERT INTO environments VALUES ('m3','ムニキスゼロ','2026-01-23','2026-03-12');
INSERT INTO environments VALUES ('mc','スタートデッキ100 バトルコレクション','2025-12-19','2026-01-22');
INSERT INTO environments VALUES ('m2a','MEGAドリームex','2025-11-28','2025-12-18');
INSERT INTO environments VALUES ('m2','インフェルノX','2025-09-26','2025-11-27');
INSERT INTO environments VALUES ('m1','メガブレイブ/メガシンフォニア','2025-08-01','2025-09-25');
INSERT INTO environments VALUES ('sv11','ブラックボルト/ホワイトフレア','2025-06-06','2025-07-31');
INSERT INTO environments VALUES ('sv10','ロケット団の栄光','2025-04-18','2025-06-05');
INSERT INTO environments VALUES ('sv9a','熱風のアリーナ','2025-03-14','2025-04-17');
INSERT INTO environments VALUES ('sv9','バトルパートナーズ','2025-01-24','2025-03-13');
INSERT INTO environments VALUES ('sv8a','テラスタルフェスex','2024-12-06','2025-01-23');
INSERT INTO environments VALUES ('sv8','超電ブレイカー','2024-10-18','2024-12-05');
INSERT INTO environments VALUES ('sv7a','楽園ドラゴーナ','2024-09-13','2024-10-17');
INSERT INTO environments VALUES ('sv7','ステラミラクル','2024-07-19','2024-09-12');
INSERT INTO environments VALUES ('sv6a','ナイトワンダラー','2024-06-07','2024-07-18');
INSERT INTO environments VALUES ('sv6','変幻の仮面','2024-04-26','2024-06-06');
INSERT INTO environments VALUES ('sv5a','クリムゾンヘイズ','2024-03-22','2024-04-25');
INSERT INTO environments VALUES ('sv5','ワイルドフォース/サイバージャッジ','2024-01-26','2024-03-21');
INSERT INTO environments VALUES ('sv4a','シャイニートレジャーex','2023-12-01','2024-01-25');
INSERT INTO environments VALUES ('sv4','古代の咆哮/未来の一閃','2023-10-27','2023-11-30');
INSERT INTO environments VALUES ('sv3a','レイジングサーフ','2023-09-22','2023-10-26');
INSERT INTO environments VALUES ('sv3','黒炎の支配者','2023-07-28','2023-09-21');
INSERT INTO environments VALUES ('sv2a','ポケモンカード151','2023-06-16','2023-07-27');
INSERT INTO environments VALUES ('sv2','スノーハザード/クレイバースト','2023-04-14','2023-06-15');
INSERT INTO environments VALUES ('sv1a','トリプレットビート','2023-03-10','2023-04-13');
INSERT INTO environments VALUES ('sv1','スカーレットex/バイオレットex','2023-01-20','2023-03-09');
INSERT INTO environments VALUES ('s12a', 'VSTARユニバース','2022-12-02','2023-01-19');
INSERT INTO environments VALUES ('s12', 'パラダイムトリガー','2022-10-21','2022-12-01');
INSERT INTO environments VALUES ('s11a', '白熱のアルカナ','2022-09-02','2022-10-20');




CREATE TABLE championship_series (
    id          VARCHAR(11) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    from_date   DATE NOT NULL,
    to_date     DATE NOT NULL
);

INSERT INTO championship_series VALUES ('series_2027','チャンピオンシップシリーズ2027','2026-09-01','2027-08-31');
INSERT INTO championship_series VALUES ('series_2026','チャンピオンシップシリーズ2026','2025-09-01','2026-08-31');
INSERT INTO championship_series VALUES ('series_2025','チャンピオンシップシリーズ2025','2024-09-01','2025-08-31');
INSERT INTO championship_series VALUES ('series_2024','チャンピオンシップシリーズ2024','2023-09-01','2024-08-31');
INSERT INTO championship_series VALUES ('series_2023','チャンピオンシップシリーズ2023','2022-09-01','2023-08-31');




CREATE TABLE cityleague_schedules (
    id          VARCHAR(6) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    from_date   DATE NOT NULL,
    to_date     DATE NOT NULL
);

INSERT INTO cityleague_schedules VALUES ('2027s1','シティリーグ2027 シーズン1','2026-09-26','2026-11-15');

INSERT INTO cityleague_schedules VALUES ('2026s4','シティリーグ2026 シーズン4','2026-03-14','2026-05-06');
INSERT INTO cityleague_schedules VALUES ('2026s3','シティリーグ2026 シーズン3','2026-01-10','2026-03-08');
INSERT INTO cityleague_schedules VALUES ('2026s2','シティリーグ2026 シーズン2','2025-11-08','2026-01-04');
INSERT INTO cityleague_schedules VALUES ('2026s1','シティリーグ2026 シーズン1','2025-09-06','2025-11-03');

INSERT INTO cityleague_schedules VALUES ('2025s4','シティリーグ2025 シーズン4','2025-03-15','2025-05-11');
INSERT INTO cityleague_schedules VALUES ('2025s3','シティリーグ2025 シーズン3','2025-01-11','2025-03-09');
INSERT INTO cityleague_schedules VALUES ('2025s2','シティリーグ2025 シーズン2','2024-11-09','2025-01-05');
INSERT INTO cityleague_schedules VALUES ('2025s1','シティリーグ2025 シーズン1','2024-09-07','2024-11-04');

INSERT INTO cityleague_schedules VALUES ('2024s4','シティリーグ2024 シーズン4','2024-03-30','2024-05-06');
INSERT INTO cityleague_schedules VALUES ('2024s3','シティリーグ2024 シーズン3','2024-02-10','2024-03-20');
INSERT INTO cityleague_schedules VALUES ('2024s2','シティリーグ2024 シーズン2','2023-12-02','2024-02-04');
INSERT INTO cityleague_schedules VALUES ('2024s1','シティリーグ2024 シーズン1','2023-09-30','2023-11-26');

INSERT INTO cityleague_schedules VALUES ('2023s4','シティリーグ2023 シーズン4','2023-04-08','2023-05-28');
INSERT INTO cityleague_schedules VALUES ('2023s3','シティリーグ2023 シーズン3','2023-02-04','2023-03-26');
INSERT INTO cityleague_schedules VALUES ('2023s2','シティリーグ2023 シーズン2','2022-12-03','2023-01-15');
INSERT INTO cityleague_schedules VALUES ('2023s1','シティリーグ2023 シーズン1','2022-10-08','2022-11-27');





CREATE TABLE cityleague_results (
    cityleague_schedule_id               VARCHAR(6) NOT NULL,
    official_event_id                    INT NOT NULL,
    league_type                          INT NOT NULL,
    event_date                           DATE DEFAULT NULL,
    player_id                            VARCHAR(10) NOT NULL,
    player_name                          VARCHAR(255) NOT NULL,
    rank                                 SMALLINT NOT NULL,
    point                                SMALLINT NOT NULL,
    deck_code                            VARCHAR(21) NOT NULL,
    FOREIGN KEY (cityleague_schedule_id) REFERENCES cityleague_schedules (id),
    FOREIGN KEY (official_event_id)      REFERENCES official_events (id)
);

CREATE UNIQUE INDEX cityleague_results_unique ON cityleague_results (cityleague_schedule_id, official_event_id, player_id);

-- 称号判定(designation_stats.go)はプレイヤーIDで入賞・決勝進出を何度も引くが、
-- 上の複合索引は player_id が先頭ではないため使えず、毎回全件走査になっていた。
-- 実測で 4.98ms → 0.19ms(6.9万件時)。この表は毎シーズン増え続ける。
CREATE INDEX idx_cityleague_results_player_id ON cityleague_results (player_id);

-- 称号ランク分布(GetRankStats)のユーザー横断集計は、シーズン期間(event_date 範囲)で
-- cityleague_results を絞り込み、そこを起点に users_players・records と結合する。event_date が
-- 未索引だと結果側から駆動できず、悪い結合順序で全件走査になっていた。実測(records 20万件・
-- cityleague_results 6万件・8000ユーザー)で ExistsCityLeagueResultGroupByUserId が
-- 約26秒 → 0.4秒。この表は毎シーズン増え続けるため索引で駆動できるようにする。
CREATE INDEX IF NOT EXISTS idx_cityleague_results_event_date ON cityleague_results (event_date);





CREATE TABLE championsleague_schedules (
    id          VARCHAR(63) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    from_date   DATE NOT NULL,
    to_date     DATE NOT NULL
);

INSERT INTO championsleague_schedules VALUES ('cl2027_aichi',    'チャンピオンズリーグ2027 愛知','2027-05-07','2027-05-09');
INSERT INTO championsleague_schedules VALUES ('cl2027_osaka',    'チャンピオンズリーグ2027 大阪','2027-04-02','2027-04-04');
INSERT INTO championsleague_schedules VALUES ('cl2027_miyagi',   'チャンピオンズリーグ2027 宮城','2027-02-12','2027-02-14');
INSERT INTO championsleague_schedules VALUES ('cl2027_chiba',    'チャンピオンズリーグ2027 千葉','2026-11-22','2026-11-23');
INSERT INTO championsleague_schedules VALUES ('cl2027_yokohama', 'チャンピオンズリーグ2027 横浜','2026-09-20','2026-09-22');

INSERT INTO championsleague_schedules VALUES ('pjcs2026',        'ポケモンジャパンチャンピオンシップス2026','2026-06-06','2026-06-07');
INSERT INTO championsleague_schedules VALUES ('cl2026_aichi_may','チャンピオンズリーグ2026 愛知 May',     '2026-05-09','2026-05-10');
INSERT INTO championsleague_schedules VALUES ('cl2026_osaka',    'チャンピオンズリーグ2026 大阪',         '2026-03-28','2026-03-29');
INSERT INTO championsleague_schedules VALUES ('cl2026_fukuoka',  'チャンピオンズリーグ2026 福岡',         '2026-02-21','2026-02-22');
INSERT INTO championsleague_schedules VALUES ('cl2026_aichi_dec','チャンピオンズリーグ2026 愛知 Dec.',    '2025-12-06','2025-12-07');
INSERT INTO championsleague_schedules VALUES ('cl2026_yokohama', 'チャンピオンズリーグ2026 横浜',         '2025-09-20','2025-09-21');

INSERT INTO championsleague_schedules VALUES ('pjcs2025',      'ポケモンジャパンチャンピオンシップス2025','2025-06-21','2025-06-22');
INSERT INTO championsleague_schedules VALUES ('cl2025_aichi',  'チャンピオンズリーグ2025 愛知 ',        '2025-05-03','2025-05-04');
INSERT INTO championsleague_schedules VALUES ('cl2025_miyagi', 'チャンピオンズリーグ2025 宮城',         '2025-04-05','2025-04-06');
INSERT INTO championsleague_schedules VALUES ('cl2025_fukuoka','チャンピオンズリーグ2025 福岡',         '2025-02-15','2025-02-16');
INSERT INTO championsleague_schedules VALUES ('cl2025_osaka',  'チャンピオンズリーグ2025 大阪',         '2024-12-21','2024-12-22');
INSERT INTO championsleague_schedules VALUES ('cl2025_tokyo',  'チャンピオンズリーグ2025 東京',         '2024-09-22','2024-09-23');

INSERT INTO championsleague_schedules VALUES ('pjcs2024',        'ポケモンジャパンチャンピオンシップス2024','2024-06-01','2024-06-02');
INSERT INTO championsleague_schedules VALUES ('cl2024_sapporo',  'チャンピオンズリーグ2024 札幌',         '2024-05-03','2024-05-04');
INSERT INTO championsleague_schedules VALUES ('cl2024_aichi',    'チャンピオンズリーグ2024 愛知',         '2024-04-13','2024-04-14');
INSERT INTO championsleague_schedules VALUES ('cl2024_fukuoka',  'チャンピオンズリーグ2024 福岡',         '2024-02-17','2024-02-18');
INSERT INTO championsleague_schedules VALUES ('cl2024_kyoto',    'チャンピオンズリーグ2024 京都',         '2023-12-23','2023-12-24');
INSERT INTO championsleague_schedules VALUES ('cl2024_yokohama', 'チャンピオンズリーグ2024 横浜',         '2023-09-09','2023-09-10');

INSERT INTO championsleague_schedules VALUES ('pjcs2023',        'ポケモンジャパンチャンピオンシップス2023','2023-06-10','2023-06-11');
INSERT INTO championsleague_schedules VALUES ('cl2023_niigata',  'チャンピオンズリーグ2023 新潟',         '2023-05-06','2023-05-07');
INSERT INTO championsleague_schedules VALUES ('cl2023_miyagi',   'チャンピオンズリーグ2023 宮城',         '2023-04-01','2023-04-02');
INSERT INTO championsleague_schedules VALUES ('cl2023_aichi',    'チャンピオンズリーグ2023 愛知',         '2023-02-25','2023-02-26');
INSERT INTO championsleague_schedules VALUES ('cl2023_kyoto',    'チャンピオンズリーグ2023 京都',         '2022-12-10','2022-12-11');
INSERT INTO championsleague_schedules VALUES ('cl2023_yokohama', 'チャンピオンズリーグ2023 横浜',         '2022-09-17','2022-09-18');






CREATE TABLE championsleague_results (
    championsleague_schedule_id               VARCHAR(63) NOT NULL,
    official_event_id                         INT NOT NULL,
    league_type                               INT NOT NULL,
    event_date                                DATE DEFAULT NULL,
    player_id                                 VARCHAR(10) NOT NULL,
    player_name                               VARCHAR(255) NOT NULL,
    rank                                      SMALLINT NOT NULL,
    deck_code                                 VARCHAR(21) NOT NULL,
    FOREIGN KEY (championsleague_schedule_id) REFERENCES championsleague_schedules (id),
    FOREIGN KEY (official_event_id)           REFERENCES official_events (id)
);

CREATE UNIQUE INDEX championsleague_results_unique ON public.championsleague_results USING btree (championsleague_schedule_id, official_event_id, player_id);




CREATE TABLE standard_regulations (
    id         VARCHAR(9) PRIMARY KEY,
    marks      VARCHAR(17) NOT NULL,
    from_date  DATE NOT NULL,
    to_date    DATE NOT NULL
);

INSERT INTO standard_regulations VALUES ('ABC', 'A・B・C', '2018-09-01', '2019-11-28');
INSERT INTO standard_regulations VALUES ('BCD', 'B・C・D', '2019-11-29', '2020-12-03');
INSERT INTO standard_regulations VALUES ('BCDE', 'B・C・D・E', '2020-12-04','2021-01-21');
INSERT INTO standard_regulations VALUES ('CDE', 'C・D・E', '2021-01-22','2021-12-16');
INSERT INTO standard_regulations VALUES ('CDEF', 'C・D・E・F', '2021-12-17','2022-01-13');
INSERT INTO standard_regulations VALUES ('DEF', 'D・E・F', '2022-01-14','2023-01-19');
INSERT INTO standard_regulations VALUES ('EFG', 'E・F・G', '2023-01-20','2024-01-25');
INSERT INTO standard_regulations VALUES ('FGH', 'F・G・H', '2024-01-26','2025-01-23');
INSERT INTO standard_regulations VALUES ('GHI', 'G・H・I', '2025-01-24','2025-12-18');
INSERT INTO standard_regulations VALUES ('GHIJ', 'G・H・I・J', '2025-12-19','2026-01-22');
INSERT INTO standard_regulations VALUES ('HIJ', 'H・I・J', '2026-01-23','2027-01-21');





CREATE TABLE cards (
    id                  INT NOT NULL PRIMARY KEY,
    collection_code     VARCHAR(255) NOT NULL,
    card_name           VARCHAR(512) NOT NULL,
    card_category       SMALLINT NOT NULL,
    card_sub_category   SMALLINT NOT NULL,
    rare_code           SMALLINT NOT NULL,
    card_image_filename VARCHAR(512) NOT NULL,
    publish_status      SMALLINT NOT NULL,
    block_code          VARCHAR(32) NOT NULL,
    group_id            SMALLINT NOT NULL,
    pokemon_level       SMALLINT NOT NULL,
    pokemon_hp          SMALLINT NOT NULL,
    pokemon_type        SMALLINT NOT NULL,
    run_away_cost       VARCHAR(16) NOT NULL,
    evolution_number    SMALLINT NOT NULL,
    great_pokemon_code  SMALLINT NOT NULL,
    regulation          VARCHAR(32) NOT NULL,
    regulation_mark     VARCHAR(32) NOT NULL
);



CREATE TABLE pokemon_cards (
    id                  INT NOT NULL PRIMARY KEY,
    card_name           VARCHAR(512) NOT NULL,
    ability             VARCHAR(512) NOT NULL,
    attack              VARCHAR(512) NOT NULL
);



CREATE TABLE pokemon_sprites (
    id      VARCHAR(128) PRIMARY KEY,
    name    VARCHAR(256) NOT NULL
);



CREATE TABLE match_pokemon_sprites (
    match_id VARCHAR(26) NOT NULL,
    position SMALLINT NOT NULL CHECK (position > 0),
    pokemon_sprite_id VARCHAR(128) NOT NULL,
    PRIMARY KEY (match_id, position),
    FOREIGN KEY (match_id)          REFERENCES matches(id),
    FOREIGN KEY (pokemon_sprite_id) REFERENCES pokemon_sprites(id)
);



CREATE TABLE deck_pokemon_sprites (
    deck_id  VARCHAR(26) NOT NULL,
    position SMALLINT NOT NULL CHECK (position > 0),
    pokemon_sprite_id VARCHAR(128) NOT NULL,
    PRIMARY KEY (deck_id, position),
    FOREIGN KEY (deck_id)           REFERENCES decks(id),
    FOREIGN KEY (pokemon_sprite_id) REFERENCES pokemon_sprites(id)
);



-- デッキ名エイリアス辞書。スプライト未設定のマッチ/デッキを、
-- デッキ名(自由入力)から代表スプライトへ解決するために使う(週次デッキ使用率の集計)。
-- alias は人間可読のまま格納してよい(集計時に NormalizeDeckName で正規化して突合する)。
-- 1エイリアスにつき代表スプライトは最大2体(position 1/2、UIの表示スロットに対応)。
-- 推測で1体しか特定できないと実スプライト2体の変種と指紋が分裂するため、
-- 有力アーキタイプには代表2体をここで定義して分裂を緩和する。
-- エイリアスの追加・修正は INSERT/UPDATE のみで、次リクエストから反映される(デプロイ不要)。
-- なお pokemon_sprites.name(正式名)は集計時に自動で突合対象へ取り込まれるため、
-- ここには略称・通称のみ登録すればよい。同名エイリアスはこの辞書が正式名より優先される。
CREATE TABLE deck_name_aliases (
    alias             VARCHAR(256) NOT NULL,
    position          SMALLINT NOT NULL CHECK (position > 0),
    pokemon_sprite_id VARCHAR(128) NOT NULL,
    -- source は登録元。
    --   'manual': 人が登録したエントリ。自動生成バッチは読むだけで書き換えない。
    --   'auto'  : cmd/generate-deck-name-aliases が共起マイニングで生成したエントリ。
    --             バッチ実行のたびに source='auto' の行だけ全削除→再生成する(冪等)。
    -- 突合時は manual が auto・正式名より優先される(同一正規化キーは先勝ち)。
    source            VARCHAR(16) NOT NULL DEFAULT 'manual',
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (alias, position),
    FOREIGN KEY (pokemon_sprite_id) REFERENCES pokemon_sprites(id)
);

-- 自動生成の全削除→再生成は source で絞るため索引を張る。
CREATE INDEX idx_deck_name_aliases_source ON deck_name_aliases(source);






CREATE TABLE user_streaks (
    user_id               VARCHAR(32) PRIMARY KEY,
    current_weeks         INT NOT NULL DEFAULT 0,
    longest_weeks         INT NOT NULL DEFAULT 0,
    freeze_used_count     INT NOT NULL DEFAULT 0,
    freeze_regen_progress INT NOT NULL DEFAULT 0,  -- フリーズ枠回復に向けたクリーン連続週数(streakFreezeRegenWeeks 週ごとに1枠回復)
    last_recorded_week    DATE NOT NULL,
    updated_at            TIMESTAMP NOT NULL
);



-- エンゲージメント計測: 「見る」利用の日次シグナル (USER_DAILY_ACTIVITIES_PLAN.md)
--
-- records/decks の「作成」しか活動として測れていないため、戦績を見返しに来ただけの
-- ユーザーが活動として計上されない。ログイン済みユーザーの1日の利用を
-- (user_id, date, category) 単位で1行 upsert し、記録経験者を
--   「記録あり / 見返しのみ / 訪問のみ / 不在」の4層に分解できるようにする。
--
-- category の採番ルール:
--   'visit'  = その日サイトを開いた (全ページ共通。必ず送られる)
--   'review' = その日自分の戦績を見返した (対象ルートは USER_DAILY_ACTIVITIES_PLAN.md §2)
--   新しいカテゴリを追加するときはアプリ側の定義
--   (entity.UserDailyActivityCategories / webapp の CATEGORY_RULES) に足すだけでよく、
--   このテーブルの変更は不要。意図的に CHECK 制約を付けていないのはそのため
--   (制約を付けるとカテゴリ追加のたびに本番のマイグレーションが必要になる)。
--   一度使ったカテゴリ名は意味を変えない・使い回さない(過去データの解釈が壊れるため)。
--
-- signal_count はクライアント側で日次1回に間引いて送るため「回数」ではなく実質フラグ
-- (1。複数端末・localStorageクリア時のみ増える)。「その日開いたか」の判定は
-- カテゴリを問わず「行の存在」で行い、値の大小には意味を持たせない。
CREATE TABLE user_daily_activities (
    user_id      VARCHAR(32) NOT NULL,
    date         DATE        NOT NULL,           -- JST基準の日付 (アプリを開いた日)
    category     VARCHAR(32) NOT NULL,           -- 'visit' / 'review' / ...
    signal_count INT         NOT NULL DEFAULT 1, -- その日そのカテゴリで受け取ったシグナル数
    updated_at   TIMESTAMP   NOT NULL,           -- 最終シグナル時刻(JST)。通知→来訪の遅延測定に使う
    PRIMARY KEY (user_id, date, category)
);

-- 集計は「直近N日 × カテゴリ」で走査するため、date を先頭に置いたインデックスを別に持つ
-- (PKの先頭は user_id なので日付範囲の絞り込みには効かない)。
CREATE INDEX idx_user_daily_activities_date_category ON user_daily_activities (date, category);



-- 施策D: 記録ストリーク・実績バッジ (MOTIVATION.md 施策D / BADGE_STREAK_PLAN.md)
--
-- badge_definitions.id の採番ルール:
--   フォーマットは "{category}-{カテゴリ内2桁連番}" (例: onboarding-01, milestone-01)。
--   連番は「カテゴリ内で採番した順」であり表示順ではない。表示順は created_at で決める。
--   一度発番したidは変更・使い回ししない(廃止したバッジのidも欠番のまま残す。週次ストリーク
--   系の streak-01〜04 は廃止済み)。廃止のしかたはカテゴリで異なる。user_badges へ永続化
--   される系(オンボーディング系)は行を消すと badge_definition_id のFKが切れるため、削除せず
--   available_to に終了日を設定する。一覧取得時にライブ集計する系(マイルストーン系)は
--   user_badges に行を持たないため、定義行ごと削除してよい。
--   新しいカテゴリを追加する際は、そのカテゴリ用のプレフィックスを新設して 01 から採番する。
CREATE TABLE badge_definitions (
    id             VARCHAR(26) PRIMARY KEY,
    code           VARCHAR(64) NOT NULL,
    category       VARCHAR(32) NOT NULL,
    name           VARCHAR(64) NOT NULL,
    description    VARCHAR(256) NOT NULL,
    icon_key       VARCHAR(64) DEFAULT NULL,
    criteria_type  VARCHAR(32) NOT NULL,
    criteria_value INT NOT NULL DEFAULT 0,
    available_from DATE DEFAULT NULL,
    available_to   DATE DEFAULT NULL,
    created_at     TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX idx_badge_definitions_code ON badge_definitions(code);


-- badge_definitions フェーズ1シード: オンボーディング系(onboarding-xx)
INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('onboarding-00', 'signup',           'onboarding', 'ユーザ登録', 'バトレコのユーザになった',     'user',   'signup',       1, now(), now()),
('onboarding-01', 'first_deck',       'onboarding', '初デッキ',  '初めてデッキを登録した',       'deck',   'deck_count',   1, now(), now()),
('onboarding-02', 'first_record',     'onboarding', '初記録',   '初めて記録を作成した',         'record', 'record_count', 1, now(), now()),
('onboarding-03', 'first_match',      'onboarding', '初対戦',   '初めて対戦結果を追加した',     'trophy', 'match_count',  1, now(), now());


-- badge_definitions フェーズ1シード: マイルストーン系(milestone-〇〇-xx)
INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-record-01', 'record_count_3',  'milestone', '駆け出しユーザー', '記録数が3に到達した',  'medal', 'record_count', 3, now(), now()),
('milestone-record-02', 'record_count_15', 'milestone', '常連ユーザー',    '記録数が15に到達した',  'medal', 'record_count', 15, now(), now()),
('milestone-record-03', 'record_count_30', 'milestone', 'ベテランユーザー', '記録数が30に到達した',  'medal', 'record_count', 30, now(), now()),
('milestone-record-04', 'record_count_50', 'milestone', 'マスターユーザー', '記録数が50に到達した', 'medal', 'record_count', 50, now(), now());

INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-deck-01', 'deck_count_3',  'milestone', '駆け出しビルダー',  'デッキコード数が3に到達した',  'medal', 'deck_code_count', 3,  now(), now()),
('milestone-deck-02', 'deck_count_15', 'milestone', '常連ビルダー',     'デッキコード数が15に到達した', 'medal', 'deck_code_count', 15, now(), now()),
('milestone-deck-03', 'deck_count_30', 'milestone', 'ベテランビルダー',  'デッキコード数が30に到達した', 'medal', 'deck_code_count', 30, now(), now()),
('milestone-deck-04', 'deck_count_50', 'milestone', 'マスタービルダー',  'デッキコード数が50に到達した', 'medal', 'deck_code_count', 50, now(), now());

INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-match-01', 'match_count_10',  'milestone', '駆け出しバトラー',  '対戦数が10に到達した',  'medal', 'match_count', 10,  now(), now()),
('milestone-match-02', 'match_count_50',  'milestone', '常連バトラー',      '対戦数が50に到達した', 'medal', 'match_count', 50, now(), now()),
('milestone-match-03', 'match_count_100', 'milestone', 'ベテランバトラー',  '対戦数が100に到達した', 'medal', 'match_count', 100, now(), now()),
('milestone-match-04', 'match_count_150', 'milestone', 'マスターバトラー',  '対戦数が150に到達した', 'medal', 'match_count', 150, now(), now());



-- user_badges は badge_definitions を外部キー参照するため、必ず badge_definitions を作成した
-- 後に定義する(このファイルを先頭から流して新しいDBを構築できるようにするため)。
CREATE TABLE user_badges (
    id                    VARCHAR(26) PRIMARY KEY,
    created_at            TIMESTAMP NOT NULL,
    user_id               VARCHAR(32) NOT NULL,
    badge_definition_id   VARCHAR(26) NOT NULL,
    record_id             VARCHAR(26) DEFAULT NULL,
    achieved_at           TIMESTAMP NOT NULL,
    FOREIGN KEY (badge_definition_id) REFERENCES badge_definitions (id)
);
CREATE UNIQUE INDEX idx_user_badges_user_id_badge_definition_id ON user_badges(user_id, badge_definition_id);
CREATE INDEX idx_user_badges_user_id ON user_badges(user_id);



-- 環境バッジ: ユーザーがその環境(environments)で初めて対戦結果を追加したことを表す実績。
-- badge_definitions/user_badges とは別の独立した仕組み(名前・期間は environments を唯一の
-- 正とするため、badge_definitions へのコピーは持たない)。
CREATE TABLE user_environment_badges (
    user_id         VARCHAR(32) NOT NULL,
    environment_id  VARCHAR(8)  NOT NULL REFERENCES environments(id),
    record_id       VARCHAR(26) DEFAULT NULL,
    notification_id VARCHAR(26) DEFAULT NULL, -- 紐づくnotifications.id。バックフィル再実行時に、新規作成ではなく既存通知の上書きを行うための参照。
    achieved_at     TIMESTAMP NOT NULL,
    created_at      TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, environment_id)
);
CREATE INDEX idx_user_environment_badges_user_id ON user_environment_badges(user_id);



-- 称号(designation): ユーザーの通算成長を表す一本道のランク。バッジと異なり、
-- ユーザーごとの到達実績を永続化するテーブルは持たず、指定シーズンの集計値から
-- 都度ライブ判定する(過去シーズンの実績を永続的に保持しないため、シーズンを
-- 切り替えるとその期間の状態がそのまま表示される)。
-- criteria_type = 'unimplemented' のティアはまだ判定ロジックが無いため、実装が追加されるまで
-- 絶対に達成されない(=「準備中」)。
CREATE TABLE designations (
    id             VARCHAR(26) PRIMARY KEY,
    tier           INT NOT NULL,
    code           VARCHAR(64) NOT NULL,
    emoji          VARCHAR(8) NOT NULL,
    name           VARCHAR(64) NOT NULL,
    description    VARCHAR(256) NOT NULL,
    criteria_type  VARCHAR(32) NOT NULL,
    criteria_value INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX idx_designations_tier ON designations(tier);
CREATE UNIQUE INDEX idx_designations_code ON designations(code);


INSERT INTO designations (id, tier, code, emoji, name, description, criteria_type, criteria_value, created_at, updated_at) VALUES
('designation-01', 1,  'beginner',     '🌱', '駆け出し',   '公式イベント/Tonamel/記入形式、いずれかの記録において使用したデッキを指定のうえで作成し、対戦結果を追加した', 'record', 1, now(), now()),
('designation-02', 2,  'novice',       '🔰', '見習い',     '称号：【🌱 駆け出し】を持っており、公式イベント/Tonamel/記入形式、いずれかの記録を3つ以上作成した', 'record', 3, now(), now()),
('designation-03', 3,  'independent',  '👍', '一人前',     '称号：【🔰 見習い】を持っており、トレーナーズリーグかシティリーグの記録を作成した', 'official_league_record', 1, now(), now()),
('designation-04', 4,  'regular',      '🎫', 'レギュラー',  '称号：【👍 一人前】を持っており、前シーズンに引き続き今シーズンでもシティリーグの記録をしているか、今シーズンでシティリーグの記録を2つ以上作成した', 'official_city_league_record', 1, now(), now()),
('designation-05', 5,  'veteran',      '💪', 'ベテラン',   '称号:【🎫 レギュラー】を持っており、連携したプレイヤーズクラブのプレイヤーIDで今シーズン1回以上、シティリーグで入賞した', 'official_city_league_placement', 1, now(), now()),
('designation-06', 6,  'expert',       '🎖️', '熟練',     '称号:【💪 ベテラン】を持っており、連携したプレイヤーズクラブのプレイヤーIDで今シーズン1回以上、シティリーグで決勝トーナメントに進出した', 'official_city_league_playoff', 1, now(), now()),
('designation-07', 7,  'master',       '🏆', '達人',       '称号:【🎖️ 熟練】を持っており、連携したプレイヤーズクラブのプレイヤーIDで今シーズン、シティリーグで優勝した', 'official_city_league_champion', 1, now(), now()),
('designation-08', 8,  'grandmaster',  '👑', '名人',       '称号:【🏆 達人】を持っており、連携したプレイヤーズクラブのプレイヤーIDで今シーズンのシティリーグ全4大会で入賞し、うち1回以上優勝した', 'official_city_league_grandmaster', 1, now(), now()),
('designation-09', 9,  'legend',       '💎', 'レジェンド', '準備中', 'unimplemented', 0, now(), now()),
('designation-10', 10, 'hall_of_fame', '🏛️', '殿堂入り',   '準備中', 'unimplemented', 0, now(), now());




-- 称号の定義変更(達人・名人の達成条件など)は、上の INSERT に最終値を反映してある。
-- 稼働DBへの反映は cmd/ のバックフィルや一度きりの UPDATE で行い、schema.sql には残さない
-- (schema.sql は新規構築用の正で、稼働DBへは流さない)。



-- 公式サイト(プレイヤーズクラブ)のアバター一覧(avatar_search API)を
-- cmd/sync-avatars バッチで定期的に同期して保持するマスタテーブル。
-- id は公式サイトの avatar_id をそのまま使う。
CREATE TABLE pokemon_avatars (
    id         INT NOT NULL PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    image_url  VARCHAR(255) NOT NULL,
    detail     VARCHAR(255) DEFAULT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);





CREATE TABLE player_rankings (
    ranking_date        DATE NOT NULL,
    league_id           INT NOT NULL,
    player_id           VARCHAR(10) NOT NULL,
    nickname            VARCHAR(255) NOT NULL,
    current_ranking     INT NOT NULL,
    prefecture_name     VARCHAR(255) NOT NULL,
    champion_ship_point INT NOT NULL,
    public_flg          BOOLEAN NOT NULL,
    champion_flg        BOOLEAN NOT NULL,
    avatar_image        VARCHAR(255) NOT NULL,
    PRIMARY KEY (ranking_date, league_id, player_id)
);



-- 通知(バッジ獲得等をユーザーに知らせるアプリ内通知)
CREATE TABLE notifications (
    id          VARCHAR(26) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    user_id     VARCHAR(32) NOT NULL,
    category    VARCHAR(32) NOT NULL, -- 'badge'/'designation'/'rank'/'streak'/'weekly_report'/'env_news'/'reminder'
    title       VARCHAR(128) NOT NULL,
    body        VARCHAR(256) NOT NULL,
    link_url    VARCHAR(256) NOT NULL DEFAULT '',
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    read_at     TIMESTAMP DEFAULT NULL
);

CREATE INDEX idx_notifications_user_id_created_at ON notifications (user_id, created_at DESC);

-- 施策B-1: Web Push 購読(B1_B2_PUSH_NOTIFICATION_PLAN.md §5.2)。端末ごとに1行。1ユーザーが複数端末を持ちうる。
CREATE TABLE push_subscriptions (
    id              VARCHAR(26) PRIMARY KEY,
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL,
    revoked_at      TIMESTAMP DEFAULT NULL,  -- 解除・失効(404/410・連続失敗)した時刻
    user_id         VARCHAR(32) NOT NULL,
    endpoint        TEXT NOT NULL,           -- プッシュサービスのURL。実質のデバイス識別子
    p256dh          VARCHAR(255) NOT NULL,   -- payload暗号化の公開鍵
    auth            VARCHAR(255) NOT NULL,   -- payload暗号化の認証シークレット
    platform        VARCHAR(32) NOT NULL DEFAULT '', -- 'ios-pwa'/'android'/'desktop'
    failure_count   INT NOT NULL DEFAULT 0,  -- 連続失敗回数。成功で0に戻る
    last_success_at TIMESTAMP DEFAULT NULL
);

-- 同一端末が再購読したときに行を増やさない(endpointが実質のデバイス識別子)
CREATE UNIQUE INDEX idx_push_subscriptions_endpoint ON push_subscriptions (endpoint);
-- 配信対象の抽出は「生きている購読」だけを見るため部分インデックスにする
CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions (user_id) WHERE revoked_at IS NULL;

-- 施策B-1: 配信ログ。「許諾率 × 到達率 × 反応率」の分解を測るために持つ。
-- notifications とは別テーブルにする。notifications は「通知の実体」、こちらは
-- 「配達というチャネル固有の事象」で、1通知が複数端末へ配達されうるため 1:N になる。
CREATE TABLE push_deliveries (
    id              VARCHAR(26) PRIMARY KEY,
    created_at      TIMESTAMP NOT NULL,
    user_id         VARCHAR(32) NOT NULL,
    subscription_id VARCHAR(26) NOT NULL,
    notification_id VARCHAR(26) NOT NULL DEFAULT '',
    campaign        VARCHAR(32) NOT NULL,    -- 'streak_nudge' / 'weekly_report' / 'env_news' / 'weekend_reminder'
    status          VARCHAR(16) NOT NULL,    -- 'pending'(送出前) / 'sent' / 'failed' / 'expired'
    status_code     INT NOT NULL DEFAULT 0,  -- プッシュサービスのHTTPステータス
    delivered_at    TIMESTAMP DEFAULT NULL,  -- 端末のSWがpushを受け取った時刻
    clicked_at      TIMESTAMP DEFAULT NULL   -- 通知がタップされた時刻
);

CREATE INDEX idx_push_deliveries_created_at_campaign ON push_deliveries (created_at, campaign);
CREATE INDEX idx_push_deliveries_user_id_created_at ON push_deliveries (user_id, created_at DESC);





-- 施策0-4: 流入元(アトリビューション)。登録の瞬間に1行だけ作られ、以後は増えない
-- (utm-attribution-plan.md §3.4)。「どのX投稿が、W0を通過して定着する登録を連れてきたか」を
-- 継続的に見るために持つ。
--
-- 値の出どころは webapp の proxy.ts が着地時に発行する first-party Cookie(vsr_attr)で、
-- クライアント由来のため改ざんできる。厳密な監査には使えない計測用の値である。
-- utm_* は誰でも任意の値を付けられるので、正規化・allowlist・長さ切り詰めを
-- proxy とサーバ(entity.NormalizeAcquisition*)の双方で通してから保存する。
--
-- users への FK は張らない。users を参照する子テーブル(users_players / user_streaks)に
-- FK を張らないのが本リポジトリの作法。
CREATE TABLE user_acquisitions (
    user_id            VARCHAR(32) PRIMARY KEY,
    source             VARCHAR(32)  DEFAULT NULL,  -- 'x' 等のチャネル
    medium             VARCHAR(32)  DEFAULT NULL,  -- 'social' / 'referral'(リファラ推定)
    campaign           VARCHAR(64)  DEFAULT NULL,  -- 投稿タイプ。未知の値は '(other)' に丸まる
    content            VARCHAR(64)  DEFAULT NULL,  -- 投稿日+連番(例 '20260831a')
    referrer           VARCHAR(255) DEFAULT NULL,  -- ホスト名のみ(パスには検索語が含まれうる)
    landing_path       VARCHAR(255) DEFAULT NULL,  -- 着地ページ('/' / '/decks' / '/records/quick')
    landing_at         TIMESTAMP    DEFAULT NULL,  -- 着地時刻。登録との差が「遅延コンバージョン」
    source_inferred_flg BOOLEAN NOT NULL DEFAULT FALSE, -- source が utm_source ではなくリファラからの推定
    survey_answer      VARCHAR(32)  DEFAULT NULL,  -- 登録時アンケート「どこで知ったか」。'x' / 'friend' / 'search' / 'other'(S4)
    created_at         TIMESTAMP    NOT NULL,
    updated_at         TIMESTAMP    NOT NULL
);

-- 流入元別の集計は campaign 単位の GROUP BY が主。登録数が増えても数百行規模なので
-- インデックスは持たせない(user_id の主キーのみ)。


-- Myジム。ユーザが「よく行く店舗」として登録した shops への参照。
-- ホームのMyジムパネルが、ここに登録された店舗の公式イベント(official_events)を引く。
--
-- 現在の上限はアプリ側(usecase.MaxUserGymsPerUser)が持ち、ここでは制約しない。
-- 上限に達した状態での追加はエラーにする(お気に入りデッキのような自動の押し出しはしない)。
-- ユーザが3枠を明示的に選ぶものなので、黙って古いものが外れる方が驚きが大きいため。
-- 解除は行の削除で表すため、論理削除(deleted_at)は持たない。
-- created_at は登録した日時で、一覧の並び順(古い順)にも使う。
--
-- users への FK は張らない。users を参照する子テーブル(users_players / user_streaks)に
-- FK を張らないのが本リポジトリの作法。
CREATE TABLE user_gyms (
    user_id    VARCHAR(32) NOT NULL,
    shop_id    INT         NOT NULL,
    created_at TIMESTAMP   NOT NULL,
    -- 同じ店舗を重複して登録できないことは主キーで担保する
    PRIMARY KEY (user_id, shop_id),
    FOREIGN KEY (shop_id) REFERENCES shops (id)
);


GRANT SELECT ON shops                   TO grafana;
GRANT SELECT ON official_events         TO grafana;
GRANT SELECT ON unofficial_events       TO grafana;

GRANT SELECT ON pokemon_sprites         TO grafana;
GRANT SELECT ON pokemon_avatars         TO grafana;
GRANT SELECT ON tonamel_events          TO grafana;

GRANT SELECT ON users                   TO grafana;
GRANT SELECT ON users_players           TO grafana;
GRANT SELECT ON player_rankings         TO grafana;

GRANT SELECT ON records                 TO grafana;
GRANT SELECT ON matches                 TO grafana;
GRANT SELECT ON match_pokemon_sprites   TO grafana;
GRANT SELECT ON games                   TO grafana;

GRANT SELECT ON decks                   TO grafana;
GRANT SELECT ON deck_codes              TO grafana;
GRANT SELECT ON deck_pokemon_sprites    TO grafana;
GRANT SELECT ON deck_name_aliases       TO grafana;

GRANT SELECT ON tags                    TO grafana;
GRANT SELECT ON deck_tags               TO grafana;
GRANT SELECT ON deck_code_tags          TO grafana;
GRANT SELECT ON match_tags              TO grafana;
GRANT SELECT ON record_tags             TO grafana;

GRANT SELECT ON championship_series     TO grafana;
GRANT SELECT ON standard_regulations    TO grafana;
GRANT SELECT ON regulations             TO grafana;
GRANT SELECT ON environments            TO grafana;

GRANT SELECT ON cityleague_schedules    TO grafana;
GRANT SELECT ON cityleague_results      TO grafana;

GRANT SELECT ON cards                   TO grafana;
GRANT SELECT ON pokemon_cards           TO grafana;

GRANT SELECT ON badge_definitions       TO grafana;
GRANT SELECT ON user_badges             TO grafana;
GRANT SELECT ON user_streaks            TO grafana;
GRANT SELECT ON user_daily_activities   TO grafana;

GRANT SELECT ON user_environment_badges TO grafana;

GRANT SELECT ON designations            TO grafana;

GRANT SELECT ON notifications           TO grafana;
GRANT SELECT ON push_subscriptions      TO grafana;
GRANT SELECT ON push_deliveries         TO grafana;

GRANT SELECT ON user_acquisitions       TO grafana;
GRANT SELECT ON user_gyms               TO grafana;


-- みんなの公開デッキ(公開したデッキコードの投稿)。1行が「あるデッキコード(バージョン)を
-- タイムラインへ公開したこと」を表す。
--
-- 取り下げは unpublished_at で表し、行は残す(公開し直しの間隔制限と、いいね数などの
-- 実績を追えるようにするため)。公開し直しは別の行(別ID)として作るので、
-- 「公開中の投稿は1コードにつき1件」は部分一意索引で担保する。
-- hidden_at は運営の非表示で、APIからは書き込まない(psql で入れる)。
-- ace_spec_card_id / ace_spec_card_name / ace_spec_image_url は公開時に deckcard-api で判定した
-- ACE SPEC。表示にも使う(一覧でカードごとに acespec API を引かない)。入っていなければ空文字。
-- いいね数は deck_code_post_likes を、登録された数は deck_code_post_imports を数えて出す
-- (非正規化した列は持たない。取り消し・退会時の削除・取り下げ時の一括削除で数が狂わないようにするため)。
-- 論理削除(deleted_at)は持たない。退会時はいいねごと物理削除する。
CREATE TABLE deck_code_posts (
    id                 VARCHAR(26) PRIMARY KEY,
    created_at         TIMESTAMP NOT NULL,
    updated_at         TIMESTAMP NOT NULL,
    user_id            VARCHAR(32) NOT NULL,
    deck_id            VARCHAR(26) NOT NULL,
    deck_code_id       VARCHAR(26) NOT NULL,
    published_at       TIMESTAMP NOT NULL,
    unpublished_at     TIMESTAMP DEFAULT NULL,
    hidden_at          TIMESTAMP DEFAULT NULL,
    ace_spec_card_id   VARCHAR(16) NOT NULL DEFAULT '',
    ace_spec_card_name VARCHAR(64) NOT NULL DEFAULT '',
    ace_spec_image_url VARCHAR(256) NOT NULL DEFAULT '',
    FOREIGN KEY (deck_id)      REFERENCES decks (id),
    FOREIGN KEY (deck_code_id) REFERENCES deck_codes (id)
);

-- 取り下げていない投稿は1コードにつき1件(運営が非表示にした投稿も枠を占有する。
-- 非表示のまま公開し直せないようにするため)。
CREATE UNIQUE INDEX idx_deck_code_posts_active_deck_code_id
    ON deck_code_posts (deck_code_id) WHERE unpublished_at IS NULL;
-- タイムライン(公開中を新しい順)。環境の期間で published_at を範囲指定するため、
-- 公開中に絞った部分索引にする。
CREATE INDEX idx_deck_code_posts_active_published_at
    ON deck_code_posts (published_at DESC) WHERE unpublished_at IS NULL AND hidden_at IS NULL;
-- 投稿者ページと退会時の削除で引く。
CREATE INDEX idx_deck_code_posts_user_id ON deck_code_posts (user_id);
-- デッキのアーカイブ・削除に連動した取り下げで引く。
CREATE INDEX idx_deck_code_posts_deck_id ON deck_code_posts (deck_id);

-- 投稿へのいいね。1ユーザ1投稿につき1件で、取り消しは行の物理削除で表す。
CREATE TABLE deck_code_post_likes (
    post_id     VARCHAR(26) NOT NULL,
    user_id     VARCHAR(32) NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    PRIMARY KEY (post_id, user_id),
    FOREIGN KEY (post_id) REFERENCES deck_code_posts (id)
);

-- 退会時に、そのユーザが押したいいねをまとめて消すために引く。
CREATE INDEX idx_deck_code_post_likes_user_id ON deck_code_post_likes (user_id);

-- 投稿の「取り込む」(デッキ登録)の利用記録。投稿カードに「N人が登録」として出すほか、
-- 運営の指標(どの投稿が取り込まれたか)にも使う。
-- 同じ人が何度押しても1回として数えるため (post_id, user_id) を主キーにする
-- (押した回数を数えると、繰り返し押すだけで指標を水増しできる)。取り下げても残す。
CREATE TABLE deck_code_post_imports (
    post_id     VARCHAR(26) NOT NULL,
    user_id     VARCHAR(32) NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    PRIMARY KEY (post_id, user_id),
    FOREIGN KEY (post_id) REFERENCES deck_code_posts (id)
);

-- 退会時に、そのユーザの取り込み記録をまとめて消すために引く。
CREATE INDEX idx_deck_code_post_imports_user_id ON deck_code_post_imports (user_id);
-- 人気順(直近7日間のいいね数)の集計と、日次のまとめ通知(期間内のいいねを投稿ごとに
-- 集計)で引く。post_id を含めて索引だけで数えられるようにする。
CREATE INDEX idx_deck_code_post_likes_created_at ON deck_code_post_likes (created_at, post_id);

-- Grafana の読み取り権限(他のテーブルの GRANT は上にまとめてあるが、これらはテーブル定義の後に置く)
GRANT SELECT ON deck_code_posts         TO grafana;
GRANT SELECT ON deck_code_post_likes    TO grafana;
GRANT SELECT ON deck_code_post_imports  TO grafana;
