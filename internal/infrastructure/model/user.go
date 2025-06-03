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
