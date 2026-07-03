package entity

import (
	"time"
)

type UserStreak struct {
	UserId           string
	CurrentWeeks     int
	LongestWeeks     int
	FreezeUsedCount  int
	LastRecordedWeek time.Time
	UpdatedAt        time.Time
}

func NewUserStreak(
	userId string,
	currentWeeks int,
	longestWeeks int,
	freezeUsedCount int,
	lastRecordedWeek time.Time,
	updatedAt time.Time,
) *UserStreak {
	return &UserStreak{
		UserId:           userId,
		CurrentWeeks:     currentWeeks,
		LongestWeeks:     longestWeeks,
		FreezeUsedCount:  freezeUsedCount,
		LastRecordedWeek: lastRecordedWeek,
		UpdatedAt:        updatedAt,
	}
}
