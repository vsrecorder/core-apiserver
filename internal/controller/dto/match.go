package dto

import "time"

type MatchRequest struct {
	RecordId           string         `json:"record_id"`
	DeckId             string         `json:"deck_id"`
	OpponentsUserId    string         `json:"opponentes_user_id"`
	BO3Flg             bool           `json:"bo3_flg"`
	QualifyingRoundFlg bool           `json:"qualifying_round_flg"`
	FinalTournamentFlg bool           `json:"final_tournament_flg"`
	DefaultVictoryFlg  bool           `json:"default_victory_flg"`
	DefaultDefeatFlg   bool           `json:"default_defeat_flg"`
	VictoryFlg         bool           `json:"victory_flg"`
	OpponentsDeckInfo  string         `json:"opponents_deck_info"`
	Memo               string         `json:"memo"`
	Games              []*GameRequest `json:"games"`
}

type MatchCreateRequest struct {
	MatchRequest
}

type MatchUpdateRequest struct {
	MatchRequest
}

type MatchResponse struct {
	ID                 string          `json:"id"`
	CreatedAt          time.Time       `json:"created_at"`
	RecordId           string          `json:"record_id"`
	DeckId             string          `json:"deck_id"`
	UserId             string          `json:"user_id"`
	OpponentsUserId    string          `json:"opponents_user_id"`
	BO3Flg             bool            `json:"bo3_flg"`
	QualifyingRoundFlg bool            `json:"qualifying_round_flg"`
	FinalTournamentFlg bool            `json:"final_tournament_flg"`
	DefaultVictoryFlg  bool            `json:"default_victory_flg"`
	DefaultDefeatFlg   bool            `json:"default_defeat_flg"`
	VictoryFlg         bool            `json:"victory_flg"`
	OpponentsDeckInfo  string          `json:"opponents_deck_info"`
	Memo               string          `json:"memo"`
	Games              []*GameResponse `json:"games"`
}

type MatchGetByIdResponse struct {
	MatchResponse
}

type MatchGetByRecordIdResponse struct {
	RecordId string           `json:"record_id"`
	DeckId   string           `json:"deck_id"`
	UserId   string           `json:"user_id"`
	Matches  []*MatchResponse `json:"matches"`
}

type MatchCreateResponse struct {
	MatchResponse
}

type MatchUpdateResponse struct {
	MatchResponse
}
