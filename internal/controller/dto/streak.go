package dto

import (
	"time"
)

type UserStreakResponse struct {
	UserId          string `json:"user_id"`
	CurrentWeeks    int    `json:"current_weeks"`
	LongestWeeks    int    `json:"longest_weeks"`
	FreezeUsedCount int    `json:"freeze_used_count"`
	MaxFreezeCount  int    `json:"max_freeze_count"`
	// FreezeRegenRemainingWeeks は、使用済みフリーズ枠が1つ回復するまでに必要な残りの
	// クリーン記録週数。回復対象が無い(フリーズ未使用)場合は0。
	FreezeRegenRemainingWeeks int       `json:"freeze_regen_remaining_weeks"`
	LastRecordedWeek          time.Time `json:"last_recorded_week,omitempty"`
}
