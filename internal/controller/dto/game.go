package dto

import "time"

type GameRequest struct {
	GoFirst             bool   `json:"go_first"`
	WinningFlg          bool   `json:"winnging_flg"`
	YourPrizeCards      uint   `json:"your_prize_cards"`
	OpponentsPrizeCards uint   `json:"opponents_prize_cards"`
	Memo                string `json:"memo"`
}

type GameResponse struct {
	ID                  string    `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	MatchId             string    `json:"match_id"`
	UserId              string    `json:"user_id"`
	GoFirst             bool      `json:"go_first"`
	WinningFlg          bool      `json:"winnging_flg"`
	YourPrizeCards      uint      `json:"your_prize_cards"`
	OpponentsPrizeCards uint      `json:"opponents_prize_cards"`
	Memo                string    `json:"memo"`
}
