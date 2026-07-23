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
	FreezeRegenRemainingWeeks int `json:"freeze_regen_remaining_weeks"`
	// FreezeRegenWeeks は、フリーズ枠が1つ回復するのに必要なクリーン継続週数(回復周期)。
	// ツールチップ等で「何週続けるごとに1つ復活するか」を案内するために返す。
	FreezeRegenWeeks int       `json:"freeze_regen_weeks"`
	LastRecordedWeek time.Time `json:"last_recorded_week,omitempty"`
}
