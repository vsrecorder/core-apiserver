package presenter

import (
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

func NewCityleagueResultGetByDateResponse(
	leagueType uint,
	date time.Time,
	count int,
	cityleagueResults []*entity.CityleagueResult,
) *dto.CityleagueResultGetByDateResponse {
	eventResults := []*dto.EventResultResponse{}

	for _, cityleagueResult := range cityleagueResults {
		results := []*dto.ResultResponse{}

		for _, result := range cityleagueResult.EventResults {
			results = append(results, &dto.ResultResponse{
				PlayerId:   result.PlayerId,
				PlayerName: result.PlayerName,
				Rank:       result.Rank,
				Point:      result.Point,
				DeckCode:   result.DeckCode,
			})
		}

		eventResults = append(eventResults, &dto.EventResultResponse{
			CityleagueScheduleId: cityleagueResult.CityleagueScheduleId,
			OfficialEventId:      cityleagueResult.OfficialEventId,
			LeagueType:           cityleagueResult.LeagueType,
			Date:                 time.Date(cityleagueResult.EventDate.Year(), cityleagueResult.EventDate.Month(), cityleagueResult.EventDate.Day(), 0, 0, 0, 0, time.Local),
			Results:              results,
		})
	}

	return &dto.CityleagueResultGetByDateResponse{
		CityleagueResultGetResponse: dto.CityleagueResultGetResponse{
			LeagueType:   leagueType,
			FromDate:     date.In(time.Local),
			ToDate:       date.In(time.Local),
			Count:        count,
			EventResults: eventResults,
		},
	}
}

func NewCityleagueResultGetByTermResponse(
	leagueType uint,
	fromDate time.Time,
	toDate time.Time,
	count int,
	cityleagueResults []*entity.CityleagueResult,
) *dto.CityleagueResultGetByDateResponse {
	eventResults := []*dto.EventResultResponse{}

	for _, cityleagueResult := range cityleagueResults {
		results := []*dto.ResultResponse{}

		for _, result := range cityleagueResult.EventResults {
			results = append(results, &dto.ResultResponse{
				PlayerId:   result.PlayerId,
				PlayerName: result.PlayerName,
				Rank:       result.Rank,
				Point:      result.Point,
				DeckCode:   result.DeckCode,
			})
		}

		eventResults = append(eventResults, &dto.EventResultResponse{
			CityleagueScheduleId: cityleagueResult.CityleagueScheduleId,
			OfficialEventId:      cityleagueResult.OfficialEventId,
			LeagueType:           cityleagueResult.LeagueType,
			Date:                 time.Date(cityleagueResult.EventDate.Year(), cityleagueResult.EventDate.Month(), cityleagueResult.EventDate.Day(), 0, 0, 0, 0, time.Local),
			Results:              results,
		})
	}

	return &dto.CityleagueResultGetByDateResponse{
		CityleagueResultGetResponse: dto.CityleagueResultGetResponse{
			LeagueType:   leagueType,
			FromDate:     fromDate.In(time.Local),
			ToDate:       toDate.In(time.Local),
			Count:        count,
			EventResults: eventResults,
		},
	}
}
