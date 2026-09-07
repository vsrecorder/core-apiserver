package dto

import "time"

type MatchRequest struct {
	RecordId             string                  `json:"record_id"`
	DeckId               string                  `json:"deck_id"`
	DeckCodeId           string                  `json:"deck_code_id"`
	OpponentsUserId      string                  `json:"opponentes_user_id"`
	BO3Flg               bool                    `json:"bo3_flg"`
	GroupMatchFlg        bool                    `json:"group_match_flg"`
	QualifyingRoundFlg   bool                    `json:"qualifying_round_flg"`
	FinalTournamentFlg   bool                    `json:"final_tournament_flg"`
	DefaultVictoryFlg    bool                    `json:"default_victory_flg"`
	DefaultDefeatFlg     bool                    `json:"default_defeat_flg"`
	VictoryFlg           bool                    `json:"victory_flg"`
	DrawFlg              bool                    `json:"draw_flg"`
	GroupMatchVictoryFlg bool                    `json:"group_match_victory_flg"`
	OpponentsDeckInfo    string                  `json:"opponents_deck_info"`
	Memo                 string                  `json:"memo"`
	Games                []*GameRequest          `json:"games"`
	PokemonSprites       []*PokemonSpriteRequest `json:"pokemon_sprites"`
	TagIds               []string                `json:"tag_ids"`
}

type MatchCreateRequest struct {
	MatchRequest
}

type MatchUpdateRequest struct {
	MatchRequest
}

type MatchResponse struct {
	ID                   string                   `json:"id"`
	CreatedAt            time.Time                `json:"created_at"`
	RecordId             string                   `json:"record_id"`
	DeckId               string                   `json:"deck_id"`
	DeckCodeId           string                   `json:"deck_code_id"`
	UserId               string                   `json:"user_id"`
	OpponentsUserId      string                   `json:"opponents_user_id"`
	BO3Flg               bool                     `json:"bo3_flg"`
	GroupMatchFlg        bool                     `json:"group_match_flg"`
	QualifyingRoundFlg   bool                     `json:"qualifying_round_flg"`
	FinalTournamentFlg   bool                     `json:"final_tournament_flg"`
	DefaultVictoryFlg    bool                     `json:"default_victory_flg"`
	DefaultDefeatFlg     bool                     `json:"default_defeat_flg"`
	VictoryFlg           bool                     `json:"victory_flg"`
	DrawFlg              bool                     `json:"draw_flg"`
	GroupMatchVictoryFlg bool                     `json:"group_match_victory_flg"`
	OpponentsDeckInfo    string                   `json:"opponents_deck_info"`
	Memo                 string                   `json:"memo"`
	Games                []*GameResponse          `json:"games"`
	PokemonSprites       []*PokemonSpriteResponse `json:"pokemon_sprites"`
	Tags                 []*TagResponse           `json:"tags"`
}

type MatchGetByIdResponse struct {
	MatchResponse
}

type MatchGetByRecordIdResponse struct {
	MatchResponse
}

type MatchCreateResponse struct {
	MatchResponse
}

type MatchUpdateResponse struct {
	MatchResponse
}

type MatchOrderItem struct {
	Id                 string `json:"id"`
	QualifyingRoundFlg bool   `json:"qualifying_round_flg"`
	FinalTournamentFlg bool   `json:"final_tournament_flg"`
}

type MatchReorderRequest struct {
	Matches []*MatchOrderItem `json:"matches"`
}

// MatchSummaryResponse は1つの記録に紐づく対戦の集計。
// 記録一覧のカードが使うのは勝敗数とチーム戦/BO3の有無だけのため、
// 対戦一覧(MatchResponse)ではなくこの形で返す。
type MatchSummaryResponse struct {
	RecordId      string `json:"record_id"`
	Total         int    `json:"total"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	Draws         int    `json:"draws"`
	HasGroupMatch bool   `json:"has_group_match"`
	HasBo3        bool   `json:"has_bo3"`
}

// MatchGetSummariesResponse は GET /matches/summary の応答。
// 後から件数や打ち切りの情報を足せるよう、配列を直に返さずオブジェクトで包む。
type MatchGetSummariesResponse struct {
	Summaries []*MatchSummaryResponse `json:"summaries"`
}
