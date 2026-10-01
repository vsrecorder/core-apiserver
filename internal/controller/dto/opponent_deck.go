package dto

// OpponentDeckResponse は自分の対戦結果に付けた相手デッキの組み合わせ1件。
// 項目名は対戦結果(MatchResponse)と揃えてある。
type OpponentDeckResponse struct {
	OpponentsDeckInfo string                   `json:"opponents_deck_info"`
	PokemonSprites    []*PokemonSpriteResponse `json:"pokemon_sprites"`
	// Count はこの組み合わせの対戦の数(多い順に並ぶ)。
	Count int `json:"count"`
	// LastEventDate はこの組み合わせと最後に対戦した記録の開催日(YYYY-MM-DD)。
	LastEventDate string `json:"last_event_date"`
}

type OpponentDecksGetResponse struct {
	Data []*OpponentDeckResponse `json:"data"`
}

// OpponentDeckMatchGameResponse は対局 1 本の先攻・後攻と勝敗。
// 勝敗の項目名は対局(GameResponse)と揃えて winnging_flg にしている(webapp が同じ型で扱えるように)。
type OpponentDeckMatchGameResponse struct {
	GoFirst    bool `json:"go_first"`
	WinningFlg bool `json:"winnging_flg"`
}

// OpponentDeckMatchResponse は相手デッキの組み合わせに当てはまる対戦 1 件と、その記録の見出し。
type OpponentDeckMatchResponse struct {
	ID       string `json:"id"`
	RecordId string `json:"record_id"`
	// EventDate は記録の開催日(YYYY-MM-DD)。未設定の記録は空文字。
	EventDate string `json:"event_date"`
	// EventType は "official" / "tonamel" / "unofficial"。どれでもなければ空文字。
	EventType            string                           `json:"event_type"`
	EventTitle           string                           `json:"event_title"`
	DeckName             string                           `json:"deck_name"`
	BO3Flg               bool                             `json:"bo3_flg"`
	GroupMatchFlg        bool                             `json:"group_match_flg"`
	GroupMatchVictoryFlg bool                             `json:"group_match_victory_flg"`
	DefaultVictoryFlg    bool                             `json:"default_victory_flg"`
	DefaultDefeatFlg     bool                             `json:"default_defeat_flg"`
	VictoryFlg           bool                             `json:"victory_flg"`
	DrawFlg              bool                             `json:"draw_flg"`
	Games                []*OpponentDeckMatchGameResponse `json:"games"`
}

type OpponentDeckMatchesGetResponse struct {
	Data []*OpponentDeckMatchResponse `json:"data"`
}

// OpponentDeckSpecRequest は相手デッキの表記とスプライトの指定。
type OpponentDeckSpecRequest struct {
	OpponentsDeckInfo string                  `json:"opponents_deck_info"`
	PokemonSprites    []*PokemonSpriteRequest `json:"pokemon_sprites"`
}

// OpponentDeckReplaceRequest は相手デッキの一括置き換え。from と同じ組み合わせの対戦を to にする。
type OpponentDeckReplaceRequest struct {
	From *OpponentDeckSpecRequest `json:"from"`
	To   *OpponentDeckSpecRequest `json:"to"`
}

type OpponentDeckReplaceResponse struct {
	// UpdatedCount は置き換えた対戦の数。
	UpdatedCount int `json:"updated_count"`
}
