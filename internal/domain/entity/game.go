package entity

import (
	"time"
)

type Game struct {
	ID                  string
	CreatedAt           time.Time
	MatchId             string
	UserId              string
	GoFirst             bool
	WinningFlg          bool
	YourPrizeCards      uint
	OpponentsPrizeCards uint
	Memo                string
}

func NewGame(
	id string,
	createdAt time.Time,
	matchId string,
	userId string,
	goFirst bool,
	winningFlg bool,
	yourPrizeCards uint,
	opponentsPrizeCards uint,
	memo string,
) *Game {
	return &Game{
		ID:                  id,
		CreatedAt:           createdAt,
		MatchId:             matchId,
		UserId:              userId,
		GoFirst:             goFirst,
		WinningFlg:          winningFlg,
		YourPrizeCards:      yourPrizeCards,
		OpponentsPrizeCards: opponentsPrizeCards,
		Memo:                memo,
	}
}
