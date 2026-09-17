package dto

// OpponentDeckCandidateResponse は相手デッキの入力候補1件。
// 対戦結果(MatchResponse)と同じ項目名にしてあり、webapp は既存の候補生成
// (opponents_deck_info とスプライトの position 1 / 2)をそのまま流用できる。
type OpponentDeckCandidateResponse struct {
	OpponentsDeckInfo string                   `json:"opponents_deck_info"`
	PokemonSprites    []*PokemonSpriteResponse `json:"pokemon_sprites"`
	// Count は全ユーザーの対戦結果にこの組み合わせが現れた回数(多い順に並ぶ)。
	Count int `json:"count"`
}

type OpponentDeckCandidatesGetResponse struct {
	Limit int                              `json:"limit"`
	Data  []*OpponentDeckCandidateResponse `json:"data"`
}
