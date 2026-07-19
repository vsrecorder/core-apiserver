package model

import (
	"time"
)

type UserStreak struct {
	UserId              string `gorm:"primaryKey"`
	CurrentWeeks        int
	LongestWeeks        int
	FreezeUsedCount     int
	FreezeRegenProgress int
	LastRecordedWeek    time.Time
	UpdatedAt           time.Time
}
