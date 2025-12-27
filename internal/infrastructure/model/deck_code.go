package model

import (
	"time"

	"gorm.io/gorm"
)

type DeckCode struct {
	ID             string `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	UserId         string
	DeckId         string
	Code           string
	PrivateCodeFlg bool
}

func NewDeckCode(
	id string,
	createdAt time.Time,
	userId string,
	deckId string,
	code string,
	privateCodeFlg bool,
) *DeckCode {
	return &DeckCode{
		ID:             id,
		CreatedAt:      createdAt,
		UserId:         userId,
		DeckId:         deckId,
		Code:           code,
		PrivateCodeFlg: privateCodeFlg,
	}
}
