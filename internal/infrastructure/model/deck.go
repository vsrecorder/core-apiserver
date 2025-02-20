package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Deck struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	ArchivedAt     sql.NullTime
	UserId         string
	Name           string
	Code           string
	PrivateCodeFlg bool
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt sql.NullTime,
	userId string,
	name string,
	code string,
	privateCodeFlg bool,
) *Deck {
	return &Deck{
		ID:             id,
		CreatedAt:      createdAt,
		ArchivedAt:     archivedAt,
		UserId:         userId,
		Name:           name,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
	}
}
