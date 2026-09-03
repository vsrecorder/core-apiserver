package presenter

import (
	"fmt"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/controller/dto"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

// dateOnlyInLocal は DATE カラム由来の時刻を、ローカルの0時に揃える。
// GORM は DATE を UTC 0時で返すため、そのまま返すと JST の webapp では前日になる。
func dateOnlyInLocal(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
}

func NewChampionsleagueResultGetEventsResponse(
	count int,
	championsleagueResultEvents []*entity.ChampionsleagueResultEvent,
) *dto.ChampionsleagueResultGetEventsResponse {
	events := []*dto.ChampionsleagueResultEventResponse{}

	for _, championsleagueResultEvent := range championsleagueResultEvents {
		events = append(events, &dto.ChampionsleagueResultEventResponse{
			ChampionsleagueScheduleId: championsleagueResultEvent.ChampionsleagueScheduleId,
			OfficialEventId:           championsleagueResultEvent.OfficialEventId,
			LeagueType:                championsleagueResultEvent.LeagueType,
			Date:                      dateOnlyInLocal(championsleagueResultEvent.EventDate),
		})
	}

	return &dto.ChampionsleagueResultGetEventsResponse{
		Count:  count,
		Events: events,
	}
}

func NewChampionsleagueResultGetByChampionsleagueScheduleIdResponse(
	championsleagueScheduleId string,
	leagueType uint,
	count int,
	championsleagueResults []*entity.ChampionsleagueResult,
) *dto.ChampionsleagueResultGetByChampionsleagueScheduleIdResponse {
	eventResults := []*dto.ChampionsleagueEventResultResponse{}

	for _, championsleagueResult := range championsleagueResults {
		results := []*dto.ChampionsleagueResultResponse{}

		for _, result := range championsleagueResult.EventResults {
			results = append(results, &dto.ChampionsleagueResultResponse{
				PlayerId:   result.PlayerId,
				PlayerName: result.PlayerName,
				Rank:       result.Rank,
				DeckCode:   result.DeckCode,
			})
		}

		eventResults = append(eventResults, &dto.ChampionsleagueEventResultResponse{
			ChampionsleagueScheduleId: championsleagueResult.ChampionsleagueScheduleId,
			OfficialEventId:           championsleagueResult.OfficialEventId,
			LeagueType:                championsleagueResult.LeagueType,
			Date:                      dateOnlyInLocal(championsleagueResult.EventDate),
			EventDetailResultURL:      fmt.Sprintf("https://players.pokemon-card.com/event/detail/%d/result", championsleagueResult.OfficialEventId),
			Results:                   results,
		})
	}

	return &dto.ChampionsleagueResultGetByChampionsleagueScheduleIdResponse{
		ChampionsleagueScheduleId: championsleagueScheduleId,
		LeagueType:                leagueType,
		Count:                     count,
		EventResults:              eventResults,
	}
}
