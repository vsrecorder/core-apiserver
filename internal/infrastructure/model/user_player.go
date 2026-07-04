package model

import (
	"time"

	"gorm.io/gorm"
)

type UserPlayer struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserId    string
	PlayerId  string
}

// TableName は実テーブル名を明示する。GORMのデフォルト命名規則では
// "UserPlayer" は "user_players" と推測されてしまい、実際のテーブル名
// "users_players" と一致しないため、明示的に上書きする。
func (UserPlayer) TableName() string {
	return "users_players"
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
