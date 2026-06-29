package model

import (
	"time"

	"gorm.io/gorm"
)

type UnofficialEvent struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	UserId    string
	Title     string
	Date      time.Time `gorm:"type:date"`
}

func NewUnofficialEvent(
	id string,
	userId string,
	title string,
	date time.Time,
) *UnofficialEvent {
	return &UnofficialEvent{
		ID:     id,
		UserId: userId,
		Title:  title,
		Date:   date,
	}
}
