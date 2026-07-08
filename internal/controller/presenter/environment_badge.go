package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func NewUserEnvironmentBadgesResponse(
	userId string,
	views []*usecase.EnvironmentBadgeView,
) *dto.UserEnvironmentBadgesResponse {
	badges := make([]*dto.EnvironmentBadgeResponse, 0, len(views))
	for _, view := range views {
		var achievedAt *time.Time
		if !view.AchievedAt.IsZero() {
			achievedAt = &view.AchievedAt
		}

		badges = append(badges, &dto.EnvironmentBadgeResponse{
			EnvironmentId: view.Environment.ID,
			Title:         view.Environment.Title,
			FromDate:      view.Environment.FromDate,
			ToDate:        view.Environment.ToDate,
			Achieved:      view.Achieved,
			AchievedAt:    achievedAt,
		})
	}

	return &dto.UserEnvironmentBadgesResponse{
		UserId: userId,
		Badges: badges,
	}
}
