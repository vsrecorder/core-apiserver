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

CREATE TABLE decks (
    id               VARCHAR(26) PRIMARY KEY,
    created_at       TIMESTAMP NOT NULL,
    updated_at       TIMESTAMP NOT NULL,
    deleted_at       TIMESTAMP DEFAULT NULL,
    archived_at      TIMESTAMP DEFAULT NULL,
    user_id          VARCHAR(32) NOT NULL,
    name             VARCHAR(32) NOT NULL,
    code             VARCHAR(21) DEFAULT NULL,
    private_code_flg BOOLEAN DEFAULT NULL
);

CREATE TABLE records (
    id                                VARCHAR(26) PRIMARY KEY,
    created_at                        TIMESTAMP NOT NULL,
    updated_at                        TIMESTAMP NOT NULL,
    deleted_at                        TIMESTAMP DEFAULT NULL,
    official_event_id                 INT DEFAULT NULL,
    tonamel_event_id                  VARCHAR(8) DEFAULT NULL,
    friend_id                         VARCHAR(32) DEFAULT NULL,
    user_id                           VARCHAR(32) NOT NULL,
    deck_id                           VARCHAR(26) DEFAULT NULL,
    private_flg                       BOOLEAN DEFAULT NULL,
    tcg_meister_url                   TEXT,
    memo                              TEXT
);

CREATE TABLE matches (
    id                        VARCHAR(26) PRIMARY KEY,
    created_at                TIMESTAMP NOT NULL,
    updated_at                TIMESTAMP NOT NULL,
    deleted_at                TIMESTAMP DEFAULT NULL,
    record_id                 VARCHAR(26) NOT NULL,
    deck_id                   VARCHAR(26) DEFAULT NULL,
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

CREATE TABLE player_users (
    created_at  TIMESTAMP NOT NULL,
    updated_at  TIMESTAMP NOT NULL,
    deleted_at  TIMESTAMP DEFAULT NULL,
    player_id   VARCHAR(16) NOT NULL,
    user_id     VARCHAR(32) NOT NULL,
);

CREATE UNIQUE INDEX unique_player_users ON player_users (player_id, user_id) WHERE deleted_at IS NULL;

CREATE TABLE official_event_results (
);

CREATE TABLE environments (
    id         VARCHAR(8) PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    from_date  DATE NOT NULL,
    to_date    DATE NOT NULL
);

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
    id         VARCHAR(6) PRIMARY KEY,
    title      VARCHAR(255) NOT NULL,
    from_date  DATE NOT NULL,
    to_date    DATE NOT NULL
);

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
    cityleague_schedule_id VARCHAR(6) NOT NULL,
    official_event_id      INT NOT NULL,
    league_type            INT NOT NULL,
    event_date             DATE DEFAULT NULL,
    player_id              VARCHAR(10) NOT NULL,
    player_name            VARCHAR(255) NOT NULL,
    rank                   SMALLINT NOT NULL,
    point                  SMALLINT NOT NULL,
    deck_code              VARCHAR(21) NOT NULL,
    FOREIGN KEY (cityleague_schedule_id) REFERENCES cityleague_schedules (id),
    FOREIGN KEY (official_event_id)      REFERENCES official_events (id)
);

CREATE UNIQUE INDEX cityleague_results_unique ON cityleague_results (cityleague_schedule_id, official_event_id, player_id);




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



GRANT SELECT ON decks TO grafana;
GRANT SELECT ON records TO grafana;
GRANT SELECT ON matches TO grafana;
GRANT SELECT ON games TO grafana;
GRANT SELECT ON users TO grafana;
GRANT SELECT ON environments TO grafana;
GRANT SELECT ON cityleague_schedules TO grafana;
GRANT SELECT ON cityleague_results TO grafana;
GRANT SELECT ON standard_regulations TO grafana;
GRANT SELECT ON cards TO grafana;
GRANT SELECT ON pokemon_cards TO grafana;





GRANT SELECT ON <table_name> TO grafana;

DROP INDEX <index_name>;
DROP TABLE <table_name>;

ALTER TABLE products RENAME COLUMN product_no TO product_number;


