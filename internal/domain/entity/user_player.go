package entity

import "time"

type UserPlayer struct {
	ID        string
	CreatedAt time.Time
	UserId    string
	PlayerId  string
}

func NewUserPlayer(
	id string,
	createdAt time.Time,
	userId string,
	playerId string,
) *UserPlayer {
	return &UserPlayer{
		ID:        id,
		CreatedAt: createdAt,
		UserId:    userId,
		PlayerId:  playerId,
	}
}

// LockedUntil は紐付けの変更が可能になる日時(紐付けから1ヶ月後)を返す。
func (e *UserPlayer) LockedUntil() time.Time {
	return e.CreatedAt.AddDate(0, 1, 0)
}
