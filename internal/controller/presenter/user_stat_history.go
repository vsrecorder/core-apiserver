package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewUserStatHistoryResponse(
	userId string,
	period string,
	season string,
	history []*entity.UserStatMonthly,
) *dto.UserStatHistoryResponse {
	items := make([]dto.UserStatHistoryItem, 0, len(history))
	for _, h := range history {
		items = append(items, dto.UserStatHistoryItem{
			YearMonth:    h.YearMonth,
			TotalMatches: h.TotalMatches,
			Wins:         h.Wins,
			Losses:       h.Losses,
			WinRate:      h.WinRate,
		})
	}
	return &dto.UserStatHistoryResponse{
		UserId:  userId,
		Period:  period,
		Season:  season,
		History: items,
	}
}
