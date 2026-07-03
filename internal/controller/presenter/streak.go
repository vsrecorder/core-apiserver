package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func NewUserStreakResponse(
	streak *entity.UserStreak,
) *dto.UserStreakResponse {
	return &dto.UserStreakResponse{
		UserId:           streak.UserId,
		CurrentWeeks:     streak.CurrentWeeks,
		LongestWeeks:     streak.LongestWeeks,
		FreezeUsedCount:  streak.FreezeUsedCount,
		MaxFreezeCount:   usecase.StreakMaxFreezeCount,
		LastRecordedWeek: streak.LastRecordedWeek,
	}
}
