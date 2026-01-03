package model

import (
	"time"

	"gorm.io/gorm"
)

type Match struct {
	ID                 string `gorm:"primaryKey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
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
}

type MatchJoinGame struct {
	MatchID                 string
	MatchCreatedAt          time.Time
	MatchUpdatedAt          time.Time
	MatchDeletedAt          gorm.DeletedAt
	MatchRecordId           string
	MatchDeckId             string
	MatchDeckCodeId         string
	MatchUserId             string
	MatchOpponentsUserId    string
	MatchBO3Flg             bool
	MatchQualifyingRoundFlg bool
	MatchFinalTournamentFlg bool
	MatchDefaultVictoryFlg  bool
	MatchDefaultDefeatFlg   bool
	MatchVictoryFlg         bool
	MatchOpponentsDeckInfo  string
	MatchMemo               string
	GameID                  string
	GameCreatedAt           time.Time
	GameUpdatedAt           time.Time
	GameDeletedAt           gorm.DeletedAt
	GameMatchId             string
	GameUserId              string
	GameGoFirst             bool
	GameWinningFlg          bool
	GameYourPrizeCards      uint
	GameOpponentsPrizeCards uint
	GameMemo                string
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
	}
}
