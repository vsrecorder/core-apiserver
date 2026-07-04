package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newBadgeDefinitionResponse(def *entity.BadgeDefinition) *dto.BadgeDefinitionResponse {
	return &dto.BadgeDefinitionResponse{
		ID:            def.ID,
		Code:          def.Code,
		Category:      def.Category,
		Name:          def.Name,
		Description:   def.Description,
		IconKey:       def.IconKey,
		CriteriaType:  def.CriteriaType,
		CriteriaValue: def.CriteriaValue,
	}
}

func NewBadgeDefinitionsResponse(
	definitions []*entity.BadgeDefinition,
) *dto.BadgeDefinitionsResponse {
	badges := make([]*dto.BadgeDefinitionResponse, 0, len(definitions))
	for _, def := range definitions {
		badges = append(badges, newBadgeDefinitionResponse(def))
	}

	return &dto.BadgeDefinitionsResponse{
		Badges: badges,
	}
}

func NewUserBadgesResponse(
	userId string,
	season string,
	views []*usecase.UserBadgeView,
) *dto.UserBadgesResponse {
	badges := make([]*dto.UserBadgeResponse, 0, len(views))
	for _, view := range views {
		var achievedAt *time.Time
		if !view.AchievedAt.IsZero() {
			achievedAt = &view.AchievedAt
		}

		badges = append(badges, &dto.UserBadgeResponse{
			BadgeDefinitionResponse: *newBadgeDefinitionResponse(view.Definition),
			Achieved:                view.Achieved,
			AchievedAt:              achievedAt,
			CurrentValue:            view.CurrentValue,
		})
	}

	return &dto.UserBadgesResponse{
		UserId: userId,
		Season: season,
		Badges: badges,
	}
}
