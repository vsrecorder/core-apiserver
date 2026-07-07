package model

import (
	"time"
)

type Notification struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	UserId    string
	Category  string
	Title     string
	Body      string
	LinkUrl   string
	IsRead    bool
	ReadAt    *time.Time
}
