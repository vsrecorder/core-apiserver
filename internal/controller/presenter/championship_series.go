package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewChampionshipSeriesGetResponse(
	championshipSeriesList []*entity.ChampionshipSeries,
) []*dto.ChampionshipSeriesResponse {
	ret := []*dto.ChampionshipSeriesResponse{}

	for _, championshipSeries := range championshipSeriesList {
		fromDate := time.Date(championshipSeries.FromDate.Year(), championshipSeries.FromDate.Month(), championshipSeries.FromDate.Day(), 0, 0, 0, 0, time.Local)
		toDate := time.Date(championshipSeries.ToDate.Year(), championshipSeries.ToDate.Month(), championshipSeries.ToDate.Day(), 0, 0, 0, 0, time.Local)

		ret = append(ret, &dto.ChampionshipSeriesResponse{
			ID:       championshipSeries.ID,
			Title:    championshipSeries.Title,
			FromDate: fromDate,
			ToDate:   toDate,
		})
	}

	return ret
}

func NewChampionshipSeriesGetByIdResponse(
	championshipSeries *entity.ChampionshipSeries,
) *dto.ChampionshipSeriesResponse {
	fromDate := time.Date(championshipSeries.FromDate.Year(), championshipSeries.FromDate.Month(), championshipSeries.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(championshipSeries.ToDate.Year(), championshipSeries.ToDate.Month(), championshipSeries.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.ChampionshipSeriesResponse{
		ID:       championshipSeries.ID,
		Title:    championshipSeries.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}

func NewChampionshipSeriesGetByDateResponse(
	championshipSeries *entity.ChampionshipSeries,
) *dto.ChampionshipSeriesResponse {
	fromDate := time.Date(championshipSeries.FromDate.Year(), championshipSeries.FromDate.Month(), championshipSeries.FromDate.Day(), 0, 0, 0, 0, time.Local)
	toDate := time.Date(championshipSeries.ToDate.Year(), championshipSeries.ToDate.Month(), championshipSeries.ToDate.Day(), 0, 0, 0, 0, time.Local)

	return &dto.ChampionshipSeriesResponse{
		ID:       championshipSeries.ID,
		Title:    championshipSeries.Title,
		FromDate: fromDate,
		ToDate:   toDate,
	}
}
