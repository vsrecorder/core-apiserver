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

type CityleagueResult struct {
	db *gorm.DB
}

func NewCityleagueResult(
	db *gorm.DB,
) repository.CityleagueResultInterface {
	return &CityleagueResult{db}
}

// cityleague_results は入賞者1人につき1レコードを持つため、イベント単位の情報は
// 同じ値が入賞者の人数分だけ重複する。DISTINCT でイベント単位に畳んでから返す。
type cityleagueResultEventRow struct {
	OfficialEventId uint
	LeagueType      uint
	EventDate       time.Time
}

func (i *CityleagueResult) FindEvents(
	ctx context.Context,
	leagueType uint,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.CityleagueResultEvent, error) {
	query := i.db.Model(&model.CityleagueResult{}).
		Distinct("official_event_id", "league_type", "event_date")

	if leagueType != 0 {
		query = query.Where("league_type = ?", leagueType)
	}

	if !fromDate.IsZero() && !toDate.IsZero() {
		query = query.Where("event_date >= ? AND event_date <= ?", fromDate, toDate)
	}

	var rows []*cityleagueResultEventRow
	if tx := query.Order("event_date DESC, league_type ASC, official_event_id ASC").Scan(&rows); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	ret := []*entity.CityleagueResultEvent{}
	for _, row := range rows {
		ret = append(ret, entity.NewCityleagueResultEvent(
			row.OfficialEventId,
			row.LeagueType,
			row.EventDate,
		))
	}

	return ret, nil
}

// playerCityleagueResultRow は FindByPlayerId の結合結果を受けるスキャン用の行。
// official_events(+shops・prefectures)を結合するため、cityleague_results の
// モデルだけでは受けきれない。
type playerCityleagueResultRow struct {
	CityleagueScheduleId string
	OfficialEventId      uint
	LeagueType           uint
	EventDate            time.Time
	Rank                 uint
	Point                uint
	DeckCode             string
	EventTitle           string
	ShopName             string
	PrefectureName       string
	EnvironmentTitle     string
}

func (i *CityleagueResult) FindByPlayerId(
	ctx context.Context,
	playerId string,
	fromDate time.Time,
	toDate time.Time,
) ([]*entity.PlayerCityleagueResult, error) {
	// 大会名・店舗名・都道府県は official_events 側にしかないが、対象は1プレイヤーの入賞
	// (1シーズンでも数件)に限られるため、結合の追加コストは無視できる。呼び出し側で
	// イベントを引き直すと入賞の件数だけ往復が増えるので、ここで一度に揃える。
	// official_events 側の行が欠けていても入賞自体は表示したいのでLEFT JOINにする。
	query := i.db.Table("cityleague_results").
		Select(
			"cityleague_results.cityleague_schedule_id AS cityleague_schedule_id,"+
				"cityleague_results.official_event_id AS official_event_id,"+
				"cityleague_results.league_type AS league_type,"+
				"cityleague_results.event_date AS event_date,"+
				"cityleague_results.rank AS rank,"+
				"cityleague_results.point AS point,"+
				"cityleague_results.deck_code AS deck_code,"+
				"official_events.title AS event_title,"+
				"official_events.shop_name AS shop_name,"+
				"prefectures.name AS prefecture_name,"+
				"environments.title AS environment_title",
		).
		Joins(
			"LEFT JOIN official_events ON official_events.id = cityleague_results.official_event_id",
		).
		Joins(
			"LEFT JOIN shops ON shops.id = official_events.shop_id",
		).
		Joins(
			"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
		).
		// 対戦環境は開催日が属する期間で引く(official_event.go の結合と同じ考え方)。
		// official_events.date ではなく cityleague_results.event_date を基準にするのは、
		// official_events 側の行が欠けていても環境名は出したいため。
		Joins(
			"LEFT JOIN environments ON environments.from_date <= cityleague_results.event_date AND environments.to_date >= cityleague_results.event_date",
		).
		Where("cityleague_results.player_id = ?", playerId)

	// シーズン期間は [fromDate, toDate) の半開区間(usecase/season.go の取り決め)。
	if !fromDate.IsZero() {
		query = query.Where("cityleague_results.event_date >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("cityleague_results.event_date < ?", toDate)
	}

	var rows []*playerCityleagueResultRow
	if tx := query.Order(
		"cityleague_results.event_date DESC, cityleague_results.rank ASC, cityleague_results.official_event_id ASC",
	).Scan(&rows); tx.Error != nil {
		return nil, tx.Error
	}

	ret := []*entity.PlayerCityleagueResult{}
	for _, row := range rows {
		ret = append(ret, entity.NewPlayerCityleagueResult(
			row.CityleagueScheduleId,
			row.OfficialEventId,
			row.LeagueType,
			row.EventDate,
			row.Rank,
			row.Point,
			row.DeckCode,
			row.EventTitle,
			row.ShopName,
			row.PrefectureName,
			row.EnvironmentTitle,
		))
	}

	return ret, nil
}

func (i *CityleagueResult) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
) (*entity.CityleagueResult, error) {
	var models []*model.CityleagueResult
	if tx := i.db.Where("official_event_id = ?", officialEventId).Order("point DESC, player_id ASC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
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

	if len(models) == 0 {
		return nil, apperror.ErrRecordNotFound
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
		logError(ctx, tx.Error)
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

		if len(cityleagueResults) == 0 {
			continue
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
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("league_type = ? AND event_date = ?", leagueType, date).Order("league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
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

		if len(cityleagueResults) == 0 {
			continue
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
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("league_type = ? AND event_date >= ? AND event_date <= ?", leagueType, fromDate, toDate).Order("event_date DESC, league_type ASC, official_event_id ASC, point DESC, player_id ASC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
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

		if len(cityleagueResults) == 0 {
			continue
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
