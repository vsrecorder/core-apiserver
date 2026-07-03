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

CREATE TABLE records (
    id                        VARCHAR(26) PRIMARY KEY,
    created_at                TIMESTAMP NOT NULL,
    updated_at                TIMESTAMP NOT NULL,
    deleted_at                TIMESTAMP DEFAULT NULL,
    official_event_id         INT DEFAULT NULL,
    tonamel_event_id          VARCHAR(8) DEFAULT NULL,
    friend_id                 VARCHAR(32) DEFAULT NULL,
    unofficial_event_id       VARCHAR(26) DEFAULT NULL,
    user_id                   VARCHAR(32) NOT NULL,
    deck_id                   VARCHAR(26) DEFAULT NULL,
    deck_code_id              VARCHAR(26) DEFAULT NULL,
    event_date                DATE DEFAULT NULL,
    private_flg               BOOLEAN DEFAULT NULL,
    tcg_meister_url           TEXT,
    memo                      TEXT
);

CREATE INDEX idx_records_created_at ON records(created_at);
CREATE INDEX idx_records_deleted_at ON records(deleted_at);

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
    group_match_victory_flg   BOOLEAN NOT NULL DEFAULT false,
    opponents_deck_info       VARCHAR(63) DEFAULT NULL,
    memo                      TEXT,
    FOREIGN KEY (record_id)   REFERENCES records (id)
);

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

CREATE TABLE users (
    id          VARCHAR(32) PRIMARY KEY,
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    name        VARCHAR(63) DEFAULT NULL,
    image_url   VARCHAR(255) DEFAULT NULL
);

CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

CREATE TABLE users_players (
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    user_id     VARCHAR(32) NOT NULL,
    player_id   VARCHAR(16) NOT NULL,
);

CREATE UNIQUE INDEX unique_player_users ON player_users (player_id, user_id) WHERE deleted_at IS NULL;


CREATE TABLE environments (
    id         VARCHAR(8) PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    from_date  DATE NOT NULL,
    to_date    DATE NOT NULL
);

update environments set to_date = '2026-09-' where id = 'm6';

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



CREATE TABLE cityleague_schedules (
    id          VARCHAR(6) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    from_date   DATE NOT NULL,
    to_date     DATE NOT NULL
);

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





CREATE TABLE championsleague_schedules (
    id          VARCHAR(63) PRIMARY KEY,
    title       VARCHAR(255) NOT NULL,
    from_date   DATE NOT NULL,
    to_date     DATE NOT NULL
);

INSERT INTO championsleague_schedules VALUES ('cl2026_aichi_may','チャンピオンズリーグ2026 愛知 May','2026-05-09','2026-05-10');
INSERT INTO championsleague_schedules VALUES ('cl2026_osaka','チャンピオンズリーグ2026 大阪','2026-03-28','2026-03-29');
INSERT INTO championsleague_schedules VALUES ('cl2026_fukuoka','チャンピオンズリーグ2026 福岡','2026-02-21','2026-02-22');
INSERT INTO championsleague_schedules VALUES ('cl2026_aichi_dec','チャンピオンズリーグ2026 愛知 Dec.','2025-12-06','2025-12-07');
INSERT INTO championsleague_schedules VALUES ('cl2026_yokohama','チャンピオンズリーグ2026 横浜','2025-09-20','2025-09-21');





CREATE TABLE championsleague_results (
    championsleague_schedule_id               VARCHAR(6) NOT NULL,
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





TRUNCATE TABLE user_badges, user_streaks, badge_definitions, designations, user_designations;





-- 施策D: 記録ストリーク・実績バッジ (MOTIVATION.md 施策D / BADGE_STREAK_PLAN.md)
--
-- badge_definitions.id の採番ルール:
--   フォーマットは "{category}-{カテゴリ内2桁連番}" (例: onboarding-01, milestone-01, streak-01)。
--   連番は「カテゴリ内で採番した順」であり表示順ではない。表示順は created_at で決める。
--   一度発番したidは変更・使い回ししない。バッジを廃止する場合も削除せず available_to に
--   終了日を設定して無効化する(user_badges.badge_definition_id のFK切れを防ぐため)。
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


CREATE TABLE user_streaks (
    user_id             VARCHAR(32) PRIMARY KEY,
    current_weeks       INT NOT NULL DEFAULT 0,
    longest_weeks       INT NOT NULL DEFAULT 0,
    freeze_used_count   INT NOT NULL DEFAULT 0,
    last_recorded_week  DATE NOT NULL,
    updated_at          TIMESTAMP NOT NULL
);


-- 称号(designation): ユーザーの通算成長を表す一本道のランク。バッジと異なり、
-- 現在の最高到達ティアのみを user_designations に保持する(過去のティアは自動的に内包される)。
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


CREATE TABLE user_designations (
    user_id        VARCHAR(32) PRIMARY KEY,
    designation_id VARCHAR(26) NOT NULL REFERENCES designations(id),
    achieved_at    TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP NOT NULL
);



-- badge_definitions フェーズ1シード: オンボーディング系(onboarding-xx)
-- onboarding-00 はデッキ登録等より前に達成される起点のため、連番の先頭として 00 を割り当てる
INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('onboarding-00', 'signup',          'onboarding', 'ユーザ登録', 'バトレコのユーザになった',     'user',   'signup',       1, now(), now()),
('onboarding-01', 'first_deck',       'onboarding', '初デッキ', '初めてデッキを登録した',       'deck',   'deck_count',   1, now(), now()),
('onboarding-02', 'first_record',     'onboarding', '初記録',   '初めて記録を作成した',         'record', 'record_count', 1, now(), now()),
('onboarding-03', 'first_match',      'onboarding', '初対戦',   '初めて対戦結果を追加した',     'trophy', 'match_count',  1, now(), now());

-- badge_definitions フェーズ1シード: マイルストーン系(milestone-〇〇-xx)
INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-record-01', 'record_count_3',  'milestone', '駆け出しユーザー',  '記録数が3に到達した',  'medal', 'record_count', 3, now(), now()),
('milestone-record-02', 'record_count_15', 'milestone', '常連ユーザー',     '記録数が15に到達した',  'medal', 'record_count', 15, now(), now()),
('milestone-record-03', 'record_count_30', 'milestone', 'ベテランユーザー',  '記録数が30に到達した',  'medal', 'record_count', 30, now(), now()),
('milestone-record-04', 'record_count_50', 'milestone', 'マスターユーザー',  '記録数が50に到達した', 'medal', 'record_count', 50, now(), now());

INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-deck-01', 'deck_count_3',  'milestone', '駆け出しビルダー',  'デッキコード数が3に到達した',  'medal', 'deck_count', 3,  now(), now()),
('milestone-deck-02', 'deck_count_15', 'milestone', '常連ビルダー',     'デッキコード数が15に到達した', 'medal', 'deck_count', 15, now(), now()),
('milestone-deck-03', 'deck_count_30', 'milestone', 'ベテランビルダー',  'デッキコード数が30に到達した', 'medal', 'deck_count', 30, now(), now()),
('milestone-deck-04', 'deck_count_50', 'milestone', 'マスタービルダー',  'デッキコード数が50に到達した', 'medal', 'deck_count', 50, now(), now());

INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('milestone-match-01', 'match_count_10',  'milestone', '駆け出しバトラー',  '対戦数が10に到達した',  'medal', 'match_count', 10,  now(), now()),
('milestone-match-02', 'match_count_50',  'milestone', '常連バトラー',      '対戦数が50に到達した', 'medal', 'match_count', 50, now(), now()),
('milestone-match-03', 'match_count_100', 'milestone', 'ベテランバトラー',  '対戦数が100に到達した', 'medal', 'match_count', 100, now(), now()),
('milestone-match-04', 'match_count_150', 'milestone', 'マスターバトラー',  '対戦数が150に到達した', 'medal', 'match_count', 150, now(), now());

-- badge_definitions フェーズ1シード: 週次ストリーク系(streak-xx)
INSERT INTO badge_definitions (id, code, category, name, description, icon_key, criteria_type, criteria_value, created_at, updated_at) VALUES
('streak-01', 'streak_week_3',  'streak', '週次記録3週連続',  '3週連続で対戦を記録した',  'flame', 'streak_weeks', 3,  now(), now()),
('streak-02', 'streak_week_7',  'streak', '週次記録7週連続',  '7週連続で対戦を記録した',  'flame', 'streak_weeks', 7,  now(), now()),
('streak-03', 'streak_week_15', 'streak', '週次記録15週連続', '15週連続で対戦を記録した', 'flame', 'streak_weeks', 15, now(), now()),
('streak-04', 'streak_week_30', 'streak', '週次記録30週連続', '30週連続で対戦を記録した', 'flame', 'streak_weeks', 30, now(), now());



INSERT INTO designations (id, tier, code, emoji, name, description, criteria_type, criteria_value, created_at, updated_at) VALUES
('designation-01', 1,  'beginner',     '🌱', '駆け出し',   'ジムバトルの記録を作成した', 'official_gym_battle_record', 1, now(), now()),
('designation-02', 2,  'novice',       '🔰', '見習い',     '称号：【🌱 駆け出し】を持っており、ジムバトルの記録を5つ以上作成した', 'official_gym_battle_record', 5, now(), now()),
('designation-03', 3,  'independent',  '👍', '一人前',     '称号：【🔰 見習い】を持っており、トレーナーズリーグかシティリーグの記録を作成している', 'official_league_record', 1, now(), now()),
('designation-04', 4,  'regular',      '🎫', '常連',       '称号：【👍 一人前】を持っており、前シーズンに引き続き、今シーズンでもシティリーグの記録を作成している', 'official_city_league_record', 1, now(), now()),
('designation-05', 5,  'veteran',      '💪', 'ベテラン',   '（準備中）称号：【🎫 常連】を持っており、今シーズン1回以上、シティリーグで入賞している', 'unimplemented', 0, now(), now()),
('designation-06', 6,  'expert',       '🎖️', '熟練者',     '（準備中）称号：【💪 ベテラン】を持っており、今シーズン1回以上、シティリーグで決勝トーナメントしている', 'unimplemented', 0, now(), now()),
('designation-07', 7,  'master',       '🏆', '達人',       '準備中', 'unimplemented', 0, now(), now()),
('designation-08', 8,  'grandmaster',  '👑', '名人',       '準備中', 'unimplemented', 0, now(), now()),
('designation-09', 9,  'legend',       '💎', 'レジェンド', '準備中', 'unimplemented', 0, now(), now()),
('designation-10', 10, 'hall_of_fame', '🏛️', '殿堂入り',   '準備中', 'unimplemented', 0, now(), now());



UPDATE designations SET description = '称号：【🔰 見習い】を持っており、トレーナーズリーグかシティリーグの記録を作成している'                WHERE id = 'designation-03';
UPDATE designations SET description = '称号：【👍 一人前】を持っており、前シーズンに引き続き、今シーズンでもシティリーグの記録を作成している'   WHERE id = 'designation-04';
UPDATE designations SET description = '（準備中）称号：【🎫 常連】を持っており、今シーズン1回以上、シティリーグで入賞している'               WHERE id = 'designation-05';
UPDATE designations SET description = '（準備中）称号：【💪 ベテラン】を持っており、今シーズン1回以上、シティリーグで決勝トーナメントしている'  WHERE id = 'designation-06';

UPDATE designations SET criteria_value = 1 WHERE id = 'designation-03';
UPDATE designations SET criteria_value = 1 WHERE id = 'designation-04';
UPDATE designations SET criteria_value = 0 WHERE id = 'designation-05';
UPDATE designations SET criteria_value = 0 WHERE id = 'designation-06';



GRANT SELECT ON cards                 TO grafana;
GRANT SELECT ON cityleague_schedules  TO grafana;
GRANT SELECT ON cityleague_results    TO grafana;
GRANT SELECT ON deck_codes            TO grafana;
GRANT SELECT ON deck_pokemon_sprites  TO grafana;
GRANT SELECT ON decks                 TO grafana;
GRANT SELECT ON environments          TO grafana;
GRANT SELECT ON games                 TO grafana;
GRANT SELECT ON match_pokemon_sprites TO grafana;
GRANT SELECT ON matches               TO grafana;
GRANT SELECT ON official_events       TO grafana;
GRANT SELECT ON pokemon_cards         TO grafana;
GRANT SELECT ON pokemon_sprites       TO grafana;
GRANT SELECT ON records               TO grafana;
GRANT SELECT ON shops                 TO grafana;
GRANT SELECT ON standard_regulations  TO grafana;
GRANT SELECT ON users                 TO grafana;

GRANT SELECT ON badge_definitions     TO grafana;
GRANT SELECT ON user_badges           TO grafana;
GRANT SELECT ON user_streaks          TO grafana;
GRANT SELECT ON designations          TO grafana;
GRANT SELECT ON user_designations     TO grafana;






GRANT SELECT ON <table_name> TO grafana;

DROP INDEX <index_name>;
DROP TABLE <table_name>;

ALTER TABLE products RENAME COLUMN product_no TO product_number;








BEGIN;

-- 新テーブルを作成
CREATE TABLE new_decks (
    id               VARCHAR(26) PRIMARY KEY,
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL,
    deleted_at       TIMESTAMP DEFAULT NULL,
    archived_at      TIMESTAMP DEFAULT NULL,
    user_id          VARCHAR(32) NOT NULL,
    name             VARCHAR(32) NOT NULL,
    code             VARCHAR(21) DEFAULT NULL,
    private_code_flg BOOLEAN DEFAULT NULL,
    private_flg      BOOLEAN DEFAULT NULL
);

-- データコピー
INSERT INTO new_decks (id, created_at, updated_at, deleted_at, archived_at, user_id, name, code, private_code_flg, private_flg)
SELECT id, created_at, updated_at, deleted_at, archived_at, user_id, name, code, private_code_flg, private_code_flg
FROM decks;

-- リネーム
ALTER TABLE decks RENAME TO old_decks;
ALTER TABLE new_decks RENAME TO decks;

COMMIT;




COMMIT;

-- 古いテーブルを削除
DROP TABLE old_decks;

BEGIN;











BEGIN;

-- 新テーブルを作成
CREATE TABLE new_records (
    id                        VARCHAR(26) PRIMARY KEY,
    created_at                TIMESTAMP NOT NULL,
    updated_at                TIMESTAMP NOT NULL,
    deleted_at                TIMESTAMP DEFAULT NULL,
    official_event_id         INT DEFAULT NULL,
    tonamel_event_id          VARCHAR(8) DEFAULT NULL,
    friend_id                 VARCHAR(32) DEFAULT NULL,
    user_id                   VARCHAR(32) NOT NULL,
    deck_id                   VARCHAR(26) DEFAULT NULL,
    deck_code_id              VARCHAR(26) DEFAULT NULL,
    private_flg               BOOLEAN DEFAULT NULL,
    tcg_meister_url           TEXT,
    memo                      TEXT
);

-- データコピー
INSERT INTO new_records (id, created_at, updated_at, deleted_at, official_event_id, tonamel_event_id, friend_id, user_id, deck_id, private_flg, tcg_meister_url, memo)
SELECT id, created_at, updated_at, deleted_at, official_event_id, tonamel_event_id, friend_id, user_id, deck_id, private_flg, tcg_meister_url, memo
FROM records;

-- リネーム
ALTER TABLE records RENAME TO old_records;
ALTER TABLE new_records RENAME TO records;



-- 新テーブルを作成
CREATE TABLE new_matches (
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
    qualifying_round_flg      BOOLEAN NOT NULL,
    final_tournament_flg      BOOLEAN NOT NULL,
    default_victory_flg       BOOLEAN NOT NULL DEFAULT false,
    default_defeat_flg        BOOLEAN NOT NULL DEFAULT false,
    victory_flg               BOOLEAN NOT NULL,
    opponents_deck_info       VARCHAR(63) DEFAULT NULL,
    memo                      TEXT,
    FOREIGN KEY (record_id)   REFERENCES records (id)
);

-- データコピー
INSERT INTO new_matches (id, created_at, updated_at, deleted_at, record_id, deck_id, user_id, opponents_user_id, bo3_flg, qualifying_round_flg, final_tournament_flg, default_victory_flg, default_defeat_flg, victory_flg, opponents_deck_info, memo)
SELECT id, created_at, updated_at, deleted_at, record_id, deck_id, user_id, opponents_user_id, bo3_flg, qualifying_round_flg, final_tournament_flg, default_victory_flg, default_defeat_flg, victory_flg, opponents_deck_info, memo
FROM matches;

-- リネーム
ALTER TABLE matches RENAME TO old_matches;
ALTER TABLE new_matches RENAME TO matches;



-- 新テーブルを作成
CREATE TABLE new_games (
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

-- データコピー
INSERT INTO new_games (id, created_at, updated_at, deleted_at, match_id, user_id, go_first, winning_flg, your_prize_cards, opponents_prize_cards, memo)
SELECT id, created_at, updated_at, deleted_at, match_id, user_id, go_first, winning_flg, your_prize_cards, opponents_prize_cards, memo
FROM games;

-- リネーム
ALTER TABLE games RENAME TO old_games;
ALTER TABLE new_games RENAME TO games;

COMMIT;




BEGIN;

-- 古いテーブルを削除
DROP TABLE old_games;
DROP TABLE old_matches;
DROP TABLE old_records;

COMMIT;




BEGIN;

GRANT SELECT ON records TO grafana;
GRANT SELECT ON matches TO grafana;
GRANT SELECT ON games TO grafana;

COMMIT;







/*
DROP TABLE games;
DROP TABLE matches;
DROP TABLE records;

ALTER TABLE old_games RENAME TO games;
ALTER TABLE old_matches RENAME TO matches;
ALTER TABLE old_records RENAME TO records;
*/


