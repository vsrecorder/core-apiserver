package dto

import (
	"time"
)

type UserStreakResponse struct {
	UserId           string    `json:"user_id"`
	CurrentWeeks     int       `json:"current_weeks"`
	LongestWeeks     int       `json:"longest_weeks"`
	FreezeUsedCount  int       `json:"freeze_used_count"`
	MaxFreezeCount   int       `json:"max_freeze_count"`
	LastRecordedWeek time.Time `json:"last_recorded_week,omitempty"`
}
