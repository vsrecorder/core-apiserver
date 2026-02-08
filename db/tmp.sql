


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