package entity

import (
	"time"
)

type UserStreak struct {
	UserId          string
	CurrentWeeks    int
	LongestWeeks    int
	FreezeUsedCount int
	// FreezeRegenProgress はフリーズ枠回復に向けて連続でクリーン記録(フリーズを使わず
	// 前週から途切れずに記録)した週数。streakFreezeRegenWeeks に達するごとに使用済み
	// フリーズ枠を1つ戻して0にリセットする。ストリークが途切れる・フリーズを消費すると0に戻る。
	FreezeRegenProgress int
	LastRecordedWeek    time.Time
	UpdatedAt           time.Time
}

func NewUserStreak(
	userId string,
	currentWeeks int,
	longestWeeks int,
	freezeUsedCount int,
	freezeRegenProgress int,
	lastRecordedWeek time.Time,
	updatedAt time.Time,
) *UserStreak {
	return &UserStreak{
		UserId:              userId,
		CurrentWeeks:        currentWeeks,
		LongestWeeks:        longestWeeks,
		FreezeUsedCount:     freezeUsedCount,
		FreezeRegenProgress: freezeRegenProgress,
		LastRecordedWeek:    lastRecordedWeek,
		UpdatedAt:           updatedAt,
	}
}
