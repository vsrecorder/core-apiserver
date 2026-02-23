package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

type Deck struct {
	ID         string `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
	ArchivedAt sql.NullTime
	UserId     string
	Name       string
	PrivateFlg bool
}

type DeckJoinDeckCode struct {
	DeckID                 string
	DeckCreatedAt          time.Time
	DeckUpdatedAt          time.Time
	DeckDeletedAt          gorm.DeletedAt
	DeckArchivedAt         sql.NullTime
	DeckUserId             string
	DeckName               string
	DeckPrivateFlg         bool
	DeckCodeID             string
	DeckCodeCreatedAt      time.Time
	DeckCodeUpdatedAt      time.Time
	DeckCodeDeletedAt      gorm.DeletedAt
	DeckCodeUserId         string
	DeckCodeDeckId         string
	DeckCodeCode           string
	DeckCodePrivateCodeFlg bool
	DeckCodeMemo           string
}

func NewDeck(
	id string,
	createdAt time.Time,
	archivedAt sql.NullTime,
	userId string,
	name string,
	privateFlg bool,
) *Deck {
	return &Deck{
		ID:         id,
		CreatedAt:  createdAt,
		ArchivedAt: archivedAt,
		UserId:     userId,
		Name:       name,
		PrivateFlg: privateFlg,
	}
}
