package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/usecase"
)

func newDesignationResponse(def *entity.Designation) *dto.DesignationResponse {
	return &dto.DesignationResponse{
		ID:            def.ID,
		Tier:          def.Tier,
		Code:          def.Code,
		Emoji:         def.Emoji,
		Name:          def.Name,
		Description:   def.Description,
		CriteriaType:  def.CriteriaType,
		CriteriaValue: def.CriteriaValue,
	}
}

func NewDesignationsResponse(
	definitions []*entity.Designation,
) *dto.DesignationsResponse {
	designations := make([]*dto.DesignationResponse, 0, len(definitions))
	for _, def := range definitions {
		designations = append(designations, newDesignationResponse(def))
	}

	return &dto.DesignationsResponse{
		Designations: designations,
	}
}

func NewUserDesignationResponse(
	userId string,
	season string,
	view *usecase.UserDesignationView,
) *dto.UserDesignationResponse {
	var current *dto.DesignationResponse
	if view.Current != nil {
		current = newDesignationResponse(view.Current)
	}

	ladder := make([]*dto.DesignationLadderItemResponse, 0, len(view.Ladder))
	for _, item := range view.Ladder {
		ladder = append(ladder, &dto.DesignationLadderItemResponse{
			DesignationResponse: *newDesignationResponse(item.Designation),
			Achieved:            item.Achieved,
			CurrentValue:        item.CurrentValue,
		})
	}

	return &dto.UserDesignationResponse{
		UserId:  userId,
		Season:  resolvedSeason(season),
		Current: current,
		Ladder:  ladder,
	}
}
