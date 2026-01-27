package infrastructure

import (
	"context"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type CityleagueResult struct {
	db *gorm.DB
}

func NewCityleagueResult(
	db *gorm.DB,
) repository.CityleagueResultInterface {
	return &CityleagueResult{db}
}

func (i *CityleagueResult) FindByOfficialEventId(
	ctx context.Context,
	officialEventId string,
) (*entity.CityleagueResult, error) {
	var models []*model.CityleagueResult
	if tx := i.db.Where("official_event_id = ?", officialEventId).Order("point DESC, player_id ASC").First(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var eventResults []*entity.EventResult
	for _, model := range models {
		eventResults = append(eventResults, entity.NewEventResult(
			model.PlayerId,
			model.PlayerName,
			model.Rank,
			model.Point,
			model.DeckCode,
		))
	}

	ret := entity.NewCityleagueResult(
		models[0].CityleagueScheduleId,
		models[0].OfficialEventId,
		models[0].LeagueType,
		models[0].EventDate,
		eventResults,
	)

	return ret, nil
}

func (i *CityleagueResult) FindByCityleagueScheduleId(
	ctx context.Context,
	leagueType uint,
	cityleagueScheduleId string,
) ([]*entity.CityleagueResult, error) {
	var models []*model.CityleagueResult
	if tx := i.db.Where("league_type = ? AND cityleague_id = ?", leagueType, cityleagueScheduleId).Order("event_date DESC, league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	// OfficialEventId別にCityleagueResultをまとめる
	var oeList []uint
	oeMap := make(map[uint][]*model.CityleagueResult)
	for _, model := range models {
		if len(oeMap[model.OfficialEventId]) == 0 {
			oeList = append(oeList, model.OfficialEventId)
		}
		oeMap[model.OfficialEventId] = append(oeMap[model.OfficialEventId], model)
	}

	var ret []*entity.CityleagueResult
	for _, officialEventId := range oeList {
		cityleagueResults := oeMap[officialEventId]
		var eventResults []*entity.EventResult
		for _, cr := range cityleagueResults {
			eventResults = append(eventResults, entity.NewEventResult(
				cr.PlayerId,
				cr.PlayerName,
				cr.Rank,
				cr.Point,
				cr.DeckCode,
			))
		}

		ret = append(ret, entity.NewCityleagueResult(
			cityleagueResults[0].CityleagueScheduleId,
			cityleagueResults[0].OfficialEventId,
			cityleagueResults[0].LeagueType,
			cityleagueResults[0].EventDate,
			eventResults,
		))
	}

	return ret, nil
}

func (i *CityleagueResult) FindByDate(
	ctx context.Context,
	leagueType uint,
	date time.Time,
) ([]*entity.CityleagueResult, error) {
	var models []*model.CityleagueResult
	if leagueType == 0 {
		if tx := i.db.Where("event_date = ?", date).Order("league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("league_type = ? AND event_date = ?", leagueType, date).Order("league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	// OfficialEventId別にCityleagueResultをまとめる
	var oeList []uint
	oeMap := make(map[uint][]*model.CityleagueResult)
	for _, model := range models {
		if len(oeMap[model.OfficialEventId]) == 0 {
			oeList = append(oeList, model.OfficialEventId)
		}
		oeMap[model.OfficialEventId] = append(oeMap[model.OfficialEventId], model)
	}

	var ret []*entity.CityleagueResult
	for _, officialEventId := range oeList {
		cityleagueResults := oeMap[officialEventId]
		var eventResults []*entity.EventResult
		for _, cr := range cityleagueResults {
			eventResults = append(eventResults, entity.NewEventResult(
				cr.PlayerId,
				cr.PlayerName,
				cr.Rank,
				cr.Point,
				cr.DeckCode,
			))
		}

		ret = append(ret, entity.NewCityleagueResult(
			cityleagueResults[0].CityleagueScheduleId,
			cityleagueResults[0].OfficialEventId,
			cityleagueResults[0].LeagueType,
			cityleagueResults[0].EventDate,
			eventResults,
		))
	}

	return ret, nil
}

func (i *CityleagueResult) FindByTerm(
	ctx context.Context,
	leagueType uint,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.CityleagueResult, error) {
	var models []*model.CityleagueResult
	if leagueType == 0 {
		if tx := i.db.Where("event_date >= ? AND event_date <= ?", fromDate, toDate).Order("event_date DESC, league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("league_type = ? AND event_date >= ? AND event_date <= ?", leagueType, fromDate, toDate).Order("event_date DESC, league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	// OfficialEventId別にCityleagueResultをまとめる
	var oeList []uint
	oeMap := make(map[uint][]*model.CityleagueResult)
	for _, model := range models {
		if len(oeMap[model.OfficialEventId]) == 0 {
			oeList = append(oeList, model.OfficialEventId)
		}
		oeMap[model.OfficialEventId] = append(oeMap[model.OfficialEventId], model)
	}

	var ret []*entity.CityleagueResult
	for _, officialEventId := range oeList {
		cityleagueResults := oeMap[officialEventId]
		var eventResults []*entity.EventResult
		for _, cr := range cityleagueResults {
			eventResults = append(eventResults, entity.NewEventResult(
				cr.PlayerId,
				cr.PlayerName,
				cr.Rank,
				cr.Point,
				cr.DeckCode,
			))
		}

		ret = append(ret, entity.NewCityleagueResult(
			cityleagueResults[0].CityleagueScheduleId,
			cityleagueResults[0].OfficialEventId,
			cityleagueResults[0].LeagueType,
			cityleagueResults[0].EventDate,
			eventResults,
		))
	}

	return ret, nil
}
