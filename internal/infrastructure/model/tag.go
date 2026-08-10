package model

import (
	"time"

	"gorm.io/gorm"
)

// Tag は tags テーブル(タグマスタ)。ユーザーごとの名前空間を持つ。
type Tag struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserId    string
	Name      string
	Color     string
	PresetFlg bool
}

func NewTag(
	id string,
	createdAt time.Time,
	updatedAt time.Time,
	userId string,
	name string,
	color string,
	presetFlg bool,
) *Tag {
	return &Tag{
		ID:        id,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		UserId:    userId,
		Name:      name,
		Color:     color,
		PresetFlg: presetFlg,
	}
}

// DeckTag は deck_tags 中間テーブル。関連の解除は行の物理削除で表すため
// 論理削除(DeletedAt)は持たない(deck_pokemon_sprites と同じ規約)。
type DeckTag struct {
	DeckId string `gorm:"primaryKey"`
	TagId  string `gorm:"primaryKey"`
}

func NewDeckTag(
	deckId string,
	tagId string,
) *DeckTag {
	return &DeckTag{
		DeckId: deckId,
		TagId:  tagId,
	}
}

// DeckCodeTag は deck_code_tags 中間テーブル。
type DeckCodeTag struct {
	DeckCodeId string `gorm:"primaryKey"`
	TagId      string `gorm:"primaryKey"`
}

func NewDeckCodeTag(
	deckCodeId string,
	tagId string,
) *DeckCodeTag {
	return &DeckCodeTag{
		DeckCodeId: deckCodeId,
		TagId:      tagId,
	}
}

// MatchTag は match_tags 中間テーブル(対戦結果 ⇔ タグ)。
type MatchTag struct {
	MatchId string `gorm:"primaryKey"`
	TagId   string `gorm:"primaryKey"`
}

func NewMatchTag(
	matchId string,
	tagId string,
) *MatchTag {
	return &MatchTag{
		MatchId: matchId,
		TagId:   tagId,
	}
}
