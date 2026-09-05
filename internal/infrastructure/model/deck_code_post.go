package model

import "time"

// DeckCodePost は deck_code_posts テーブル(みんなの公開デッキへの投稿)。
// 取り下げは unpublished_at で表し、行は残す(公開し直しの間隔制限に使うため)。
// 論理削除(deleted_at)は持たない。退会時は物理削除する。
// いいね数は deck_code_post_likes を数えて出すため、ここには持たない。
type DeckCodePost struct {
	ID          string `gorm:"primaryKey"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	UserId      string
	DeckId      string
	DeckCodeId  string
	PublishedAt time.Time
	// UnpublishedAt / HiddenAt は NULL を「未設定」として扱うためポインタにする。
	UnpublishedAt   *time.Time
	HiddenAt        *time.Time
	AceSpecCardId   string
	AceSpecCardName string
	AceSpecImageURL string
}

func NewDeckCodePost(
	id string,
	createdAt time.Time,
	updatedAt time.Time,
	userId string,
	deckId string,
	deckCodeId string,
	publishedAt time.Time,
	unpublishedAt *time.Time,
	hiddenAt *time.Time,
	aceSpecCardId string,
	aceSpecCardName string,
	aceSpecImageURL string,
) *DeckCodePost {
	return &DeckCodePost{
		ID:              id,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		UserId:          userId,
		DeckId:          deckId,
		DeckCodeId:      deckCodeId,
		PublishedAt:     publishedAt,
		UnpublishedAt:   unpublishedAt,
		HiddenAt:        hiddenAt,
		AceSpecCardId:   aceSpecCardId,
		AceSpecCardName: aceSpecCardName,
		AceSpecImageURL: aceSpecImageURL,
	}
}

// DeckCodePostImport は deck_code_post_imports テーブル。(投稿, ユーザ) につき1行で、取り込みを1回として数える。
type DeckCodePostImport struct {
	PostId    string `gorm:"primaryKey"`
	UserId    string `gorm:"primaryKey"`
	CreatedAt time.Time
}

// DeckCodePostLike は deck_code_post_likes テーブル。取り消しは行の物理削除で表す。
type DeckCodePostLike struct {
	PostId    string `gorm:"primaryKey"`
	UserId    string `gorm:"primaryKey"`
	CreatedAt time.Time
}
