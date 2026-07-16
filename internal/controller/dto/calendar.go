package dto

import "time"

// CalendarRecordResponse は記録と、それに紐づく対戦結果。
type CalendarRecordResponse struct {
	ID                string                   `json:"id"`
	CreatedAt         time.Time                `json:"created_at"`
	OfficialEventId   uint                     `json:"official_event_id"`
	TonamelEventId    string                   `json:"tonamel_event_id"`
	UnofficialEventId string                   `json:"unofficial_event_id"`
	DeckId            string                   `json:"deck_id"`
	DeckCodeId        string                   `json:"deck_code_id"`
	Matches           []*CalendarMatchResponse `json:"matches"`
}

// CalendarMatchResponse はカレンダー表示に必要な対戦結果の情報。
type CalendarMatchResponse struct {
	ID                string                   `json:"id"`
	CreatedAt         time.Time                `json:"created_at"`
	OpponentsDeckInfo string                   `json:"opponents_deck_info"`
	DefaultVictoryFlg bool                     `json:"default_victory_flg"`
	DefaultDefeatFlg  bool                     `json:"default_defeat_flg"`
	VictoryFlg        bool                     `json:"victory_flg"`
	Memo              string                   `json:"memo"`
	Games             []*CalendarGameResponse  `json:"games"`
	PokemonSprites    []*PokemonSpriteResponse `json:"pokemon_sprites"`
}

// CalendarGameResponse はカレンダー表示に必要な対局の情報。
type CalendarGameResponse struct {
	GoFirst             bool `json:"go_first"`
	YourPrizeCards      uint `json:"your_prize_cards"`
	OpponentsPrizeCards uint `json:"opponents_prize_cards"`
}

// CalendarDeckResponse はデッキと、それに紐づくデッキコード(バージョン)。
type CalendarDeckResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// ArchivedAt はアーカイブされていない場合 null。
	ArchivedAt     *time.Time                  `json:"archived_at"`
	Name           string                      `json:"name"`
	PokemonSprites []*PokemonSpriteResponse    `json:"pokemon_sprites"`
	DeckCodes      []*CalendarDeckCodeResponse `json:"deck_codes"`
}

// CalendarDeckCodeResponse はカレンダー表示に必要なデッキコードの情報。
type CalendarDeckCodeResponse struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Code      string    `json:"code"`
}

// CalendarTonamelEventResponse はカレンダー表示に必要なTonamelイベントの情報。
type CalendarTonamelEventResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// CalendarUnofficialEventResponse はカレンダー表示に必要な自由形式イベントの情報。
type CalendarUnofficialEventResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// CalendarGetByUserIdResponse は活動ログのカレンダーを組み立てるのに必要な全データ。
//
// 記録・デッキから参照されるイベント情報まで含めて一度に返すことで、
// 呼び出し側が記録1件ごとに追加のAPIを呼ばずに済むようにしている。
// 表示上の色やラベルは含めない(UIの関心事のため呼び出し側で決める)。
type CalendarGetByUserIdResponse struct {
	Records          []*CalendarRecordResponse          `json:"records"`
	Decks            []*CalendarDeckResponse            `json:"decks"`
	OfficialEvents   []*OfficialEventResponse           `json:"official_events"`
	TonamelEvents    []*CalendarTonamelEventResponse    `json:"tonamel_events"`
	UnofficialEvents []*CalendarUnofficialEventResponse `json:"unofficial_events"`
}
