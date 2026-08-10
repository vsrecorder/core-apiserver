package entity

import "time"

// Tag はユーザーが任意に付けられるラベル。ユーザーごとに名前空間を持ち、
// デッキやデッキコード(将来は記録・対戦結果)に付与できる。
// 付与先との関連はエンティティごとの中間テーブル(deck_tags など)で表す。
type Tag struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	UserId    string
	Name      string
	Color     string // '#RRGGBB' 形式。未設定は空文字。
	// PresetFlg=true は運営が用意する全ユーザー共通のプリセットタグ(例: ACE SPECカード)。
	// プリセットは特定ユーザーに属さない(UserId は空文字)。誰でも付与できるが編集・削除は不可。
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
