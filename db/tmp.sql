

CREATE TABLE records (
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
    qualifying_round_flg      BOOLEAN NOT NULL,
    final_tournament_flg      BOOLEAN NOT NULL,
    default_victory_flg       BOOLEAN NOT NULL DEFAULT false,
    default_defeat_flg        BOOLEAN NOT NULL DEFAULT false,
    victory_flg               BOOLEAN NOT NULL,
    opponents_deck_info       VARCHAR(63) DEFAULT NULL,
    memo                      TEXT,
    FOREIGN KEY (record_id)   REFERENCES records (id)
);

CREATE TABLE decks (
    id             VARCHAR(26) PRIMARY KEY,
    created_at     TIMESTAMP NOT NULL,
    updated_at     TIMESTAMP NOT NULL,
    deleted_at     TIMESTAMP DEFAULT NULL,
    archived_at    TIMESTAMP DEFAULT NULL,
    user_id        VARCHAR(32) NOT NULL,
    name           VARCHAR(32) NOT NULL,
    private_flg    BOOLEAN DEFAULT NULL
);



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

-- 古いテーブルを削除
DROP TABLE decks;

-- リネーム
ALTER TABLE new_decks RENAME TO decks;

COMMIT;


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

ALTER TABLE deck_codes ADD COLUMN memo TEXT; 