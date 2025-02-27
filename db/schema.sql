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

CREATE TABLE decks_new (
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
