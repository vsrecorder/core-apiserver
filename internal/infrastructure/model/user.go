package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string
	ImageURL  string

	// PurgedAt は退会後に紐づくデータを物理削除した日時(cmd/purge-deleted-user-data が入れる)。
	// APIからは書き込まないため読み取り専用(`->`)にしている。GORMのSaveは全カラムを更新するので、
	// タグが無いとプロフィール更新のたびに NULL で上書きされてしまう。
	PurgedAt *time.Time `gorm:"->"`
}

func NewUser(
	id string,
	createdAt time.Time,
	name string,
	imageURL string,
) *User {
	return &User{
		ID:        id,
		CreatedAt: createdAt,
		Name:      name,
		ImageURL:  imageURL,
	}
}
