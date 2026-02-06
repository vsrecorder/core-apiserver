package entity

import (
	"time"
)

type Match struct {
	ID                 string
	CreatedAt          time.Time
	RecordId           string
	DeckId             string
	DeckCodeId         string
	UserId             string
	OpponentsUserId    string
	BO3Flg             bool
	QualifyingRoundFlg bool
	FinalTournamentFlg bool
	DefaultVictoryFlg  bool
	DefaultDefeatFlg   bool
	VictoryFlg         bool
	OpponentsDeckInfo  string
	Memo               string
	Games              []*Game
}

func NewMatch(
	id string,
	createdAt time.Time,
	recordId string,
	deckId string,
	deckCodeId string,
	userId string,
	opponentsUserId string,
	bo3Flg bool,
	qualifyingRoundFlg bool,
	finalTournamentFlg bool,
	defaultVictoryFlg bool,
	defaultDefeatFlg bool,
	victoryFlg bool,
	opponentsDeckInfo string,
	memo string,
	games []*Game,
) *Match {
	return &Match{
		ID:                 id,
		CreatedAt:          createdAt,
		RecordId:           recordId,
		DeckId:             deckId,
		DeckCodeId:         deckCodeId,
		UserId:             userId,
		OpponentsUserId:    opponentsUserId,
		BO3Flg:             bo3Flg,
		QualifyingRoundFlg: qualifyingRoundFlg,
		FinalTournamentFlg: finalTournamentFlg,
		DefaultVictoryFlg:  defaultVictoryFlg,
		DefaultDefeatFlg:   defaultDefeatFlg,
		VictoryFlg:         victoryFlg,
		OpponentsDeckInfo:  opponentsDeckInfo,
		Memo:               memo,
		Games:              games,
	}
}
