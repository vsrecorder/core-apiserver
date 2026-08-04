package model

import (
	"time"
)

type UserDailyActivity struct {
	UserId      string    `gorm:"primaryKey"`
	Date        time.Time `gorm:"primaryKey;type:date"`
	Category    string    `gorm:"primaryKey"`
	SignalCount int
	UpdatedAt   time.Time
}
