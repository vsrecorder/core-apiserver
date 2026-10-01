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
