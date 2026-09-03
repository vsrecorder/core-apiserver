package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type ChampionsleagueResult struct {
	db *gorm.DB
}

func NewChampionsleagueResult(
	db *gorm.DB,
) repository.ChampionsleagueResultInterface {
	return &ChampionsleagueResult{db}
}

// championsleague_results は入賞者1人につき1レコードを持つため、イベント単位の情報は
// 入賞者の人数分だけ重複する。DISTINCT でイベント単位に畳んでから返す。
type championsleagueResultEventRow struct {
	ChampionsleagueScheduleId string
	OfficialEventId           uint
	LeagueType                uint
	EventDate                 time.Time
}

func (i *ChampionsleagueResult) FindEvents(
	ctx context.Context,
) ([]*entity.ChampionsleagueResultEvent, error) {
	var rows []*championsleagueResultEventRow

	if tx := i.db.Model(&model.ChampionsleagueResult{}).
		Distinct("championsleague_schedule_id", "official_event_id", "league_type", "event_date").
		Order("event_date DESC, league_type DESC, official_event_id ASC").
		Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := []*entity.ChampionsleagueResultEvent{}
	for _, row := range rows {
		ret = append(ret, entity.NewChampionsleagueResultEvent(
			row.ChampionsleagueScheduleId,
			row.OfficialEventId,
			row.LeagueType,
			row.EventDate,
		))
	}

	return ret, nil
}

func (i *ChampionsleagueResult) FindByChampionsleagueScheduleId(
	ctx context.Context,
	leagueType uint,
	championsleagueScheduleId string,
) ([]*entity.ChampionsleagueResult, error) {
	var models []*model.ChampionsleagueResult

	query := i.db.Where("championsleague_schedule_id = ?", championsleagueScheduleId)

	if leagueType != 0 {
		query = query.Where("league_type = ?", leagueType)
	}

	// リーグ区分は降順。マスター(4)が最も読まれるため先頭に来るようにする。
	// 入賞者の並びはシティリーグと違い point を持たないため rank を使う。
	if tx := query.
		Order("event_date DESC, league_type DESC, official_event_id ASC, rank ASC, player_id ASC").
		Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(models) == 0 {
		return nil, apperror.ErrRecordNotFound
	}

	// OfficialEventId 別にまとめる。上の ORDER BY でイベントが固まって並ぶが、
	// 大会は数イベントしか無いのでマップで受けても順序さえ保てば問題にならない。
	var oeList []uint
	oeMap := make(map[uint][]*model.ChampionsleagueResult)
	for _, m := range models {
		if len(oeMap[m.OfficialEventId]) == 0 {
			oeList = append(oeList, m.OfficialEventId)
		}
		oeMap[m.OfficialEventId] = append(oeMap[m.OfficialEventId], m)
	}

	ret := []*entity.ChampionsleagueResult{}
	for _, officialEventId := range oeList {
		championsleagueResults := oeMap[officialEventId]

		var eventResults []*entity.ChampionsleagueEventResult
		for _, cr := range championsleagueResults {
			eventResults = append(eventResults, entity.NewChampionsleagueEventResult(
				cr.PlayerId,
				cr.PlayerName,
				cr.Rank,
				cr.DeckCode,
			))
		}

		ret = append(ret, entity.NewChampionsleagueResult(
			championsleagueResults[0].ChampionsleagueScheduleId,
			championsleagueResults[0].OfficialEventId,
			championsleagueResults[0].LeagueType,
			championsleagueResults[0].EventDate,
			eventResults,
		))
	}

	return ret, nil
}
