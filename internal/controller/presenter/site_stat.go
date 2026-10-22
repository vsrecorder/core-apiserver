package presenter

import (
	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewSiteStatResponse(
	stat *entity.SiteStat,
) *dto.SiteStatResponse {
	return &dto.SiteStatResponse{
		UserCount:     stat.UserCount,
		MatchCount:    stat.MatchCount,
		DeckCodeCount: stat.DeckCodeCount,
	}
}
