package infrastructure

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type OfficialEvent struct {
	db *gorm.DB
}

func NewOfficialEvent(
	db *gorm.DB,
) repository.OfficialEventInterface {
	return &OfficialEvent{db}
}

func (i *OfficialEvent) Find(
	ctx context.Context,
	typeId uint,
	leagueType uint,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.OfficialEvent, error) {
	var events []*model.OfficialEvent

	if typeId == 0 { // 大会の種類に指定がない場合
		if 1 <= leagueType && leagueType <= 4 { // リーグの種類に指定がある場合
			leagueTitle := ""

			switch leagueType {
			case 1:
				leagueTitle = "オープン"
			case 2:
				leagueTitle = "ジュニア"
			case 3:
				leagueTitle = "シニア"
			case 4:
				leagueTitle = "マスター"
			}

			tx := i.db.Table(
				"official_events",
			).Select(`
				official_events.id AS id,
				official_events.title AS title,
				official_events.address AS address,
				official_events.venue AS venue,
				official_events.date AS date,
				official_events.started_at AS started_at,
				official_events.ended_at AS ended_at,
				official_events.type_id AS type_id,
				official_events.type_name AS type_name,
				official_events.league_title AS league_title,
				official_events.regulation_title AS regulation_title,
				official_events.csp_flg AS csp_flg,
				official_events.capacity AS capacity,
				official_events.shop_id AS shop_id,
				official_events.shop_name AS shop_name,
				prefectures.id AS prefecture_id ,
				prefectures.name AS prefecture_name,
				environments.id AS environment_id,
				environments.title AS environment_title,
				standard_regulations.id AS standard_regulation_id,
				standard_regulations.marks AS standard_regulation_marks
			`,
			).Joins(
				"LEFT JOIN shops ON shops.id = official_events.shop_id",
			).Joins(
				"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
			).Joins(
				"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
			).Joins(
				"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
			).Where(
				"league_title = ? AND date BETWEEN ? AND ?", leagueTitle, startDate, endDate,
			).Order(
				"started_at ASC",
			).Find(&events)

			if tx.Error != nil {
				logError(ctx, tx.Error)
				return nil, tx.Error
			}
		} else {
			tx := i.db.Table(
				"official_events",
			).Select(`
				official_events.id AS id,
				official_events.title AS title,
				official_events.address AS address,
				official_events.venue AS venue,
				official_events.date AS date,
				official_events.started_at AS started_at,
				official_events.ended_at AS ended_at,
				official_events.type_id AS type_id,
				official_events.type_name AS type_name,
				official_events.league_title AS league_title,
				official_events.regulation_title AS regulation_title,
				official_events.csp_flg AS csp_flg,
				official_events.capacity AS capacity,
				official_events.shop_id AS shop_id,
				official_events.shop_name AS shop_name,
				prefectures.id AS prefecture_id ,
				prefectures.name AS prefecture_name,
				environments.id AS environment_id,
				environments.title AS environment_title,
				standard_regulations.id AS standard_regulation_id,
				standard_regulations.marks AS standard_regulation_marks
			`,
			).Joins(
				"LEFT JOIN shops ON shops.id = official_events.shop_id",
			).Joins(
				"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
			).Joins(
				"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
			).Joins(
				"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
			).Where(
				"date BETWEEN ? AND ?", startDate, endDate,
			).Order(
				"started_at ASC",
			).Find(&events)

			if tx.Error != nil {
				logError(ctx, tx.Error)
				return nil, tx.Error
			}
		}
	} else { // 大型大会(typeId: 1) / シティ(typeId: 2) / トレリ(typeId: 3) / ジムイベント(typeId: 4) / オーガナイザーイベント(typeId: 6) / その他(typeId: 7)の場合
		if 1 <= leagueType && leagueType <= 4 { // リーグの種類に指定がある場合
			leagueTitle := ""

			switch leagueType {
			case 1:
				leagueTitle = "オープン"
			case 2:
				leagueTitle = "ジュニア"
			case 3:
				leagueTitle = "シニア"
			case 4:
				leagueTitle = "マスター"
			}

			tx := i.db.Table(
				"official_events",
			).Select(`
				official_events.id AS id,
				official_events.title AS title,
				official_events.address AS address,
				official_events.venue AS venue,
				official_events.date AS date,
				official_events.started_at AS started_at,
				official_events.ended_at AS ended_at,
				official_events.type_id AS type_id,
				official_events.type_name AS type_name,
				official_events.league_title AS league_title,
				official_events.regulation_title AS regulation_title,
				official_events.csp_flg AS csp_flg,
				official_events.capacity AS capacity,
				official_events.shop_id AS shop_id,
				official_events.shop_name AS shop_name,
				prefectures.id AS prefecture_id ,
				prefectures.name AS prefecture_name,
				environments.id AS environment_id,
				environments.title AS environment_title,
				standard_regulations.id AS standard_regulation_id,
				standard_regulations.marks AS standard_regulation_marks
			`,
			).Joins(
				"LEFT JOIN shops ON shops.id = official_events.shop_id",
			).Joins(
				"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
			).Joins(
				"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
			).Joins(
				"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
			).Where(
				"type_id = ? AND league_title = ? AND date BETWEEN ? AND ?", typeId, leagueTitle, startDate, endDate,
			).Order(
				"started_at ASC",
			).Find(&events)

			if tx.Error != nil {
				logError(ctx, tx.Error)
				return nil, tx.Error
			}
		} else {
			tx := i.db.Table(
				"official_events",
			).Select(`
				official_events.id AS id,
				official_events.title AS title,
				official_events.address AS address,
				official_events.venue AS venue,
				official_events.date AS date,
				official_events.started_at AS started_at,
				official_events.ended_at AS ended_at,
				official_events.type_id AS type_id,
				official_events.type_name AS type_name,
				official_events.league_title AS league_title,
				official_events.regulation_title AS regulation_title,
				official_events.csp_flg AS csp_flg,
				official_events.capacity AS capacity,
				official_events.shop_id AS shop_id,
				official_events.shop_name AS shop_name,
				prefectures.id AS prefecture_id ,
				prefectures.name AS prefecture_name,
				environments.id AS environment_id,
				environments.title AS environment_title,
				standard_regulations.id AS standard_regulation_id,
				standard_regulations.marks AS standard_regulation_marks
			`,
			).Joins(
				"LEFT JOIN shops ON shops.id = official_events.shop_id",
			).Joins(
				"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
			).Joins(
				"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
			).Joins(
				"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
			).Where(
				"type_id = ? AND date BETWEEN ? AND ?", typeId, startDate, endDate,
			).Order(
				"started_at ASC",
			).Find(&events)

			if tx.Error != nil {
				logError(ctx, tx.Error)
				return nil, tx.Error
			}
		}
	}

	var entities []*entity.OfficialEvent
	for _, event := range events {
		entities = append(
			entities,
			entity.NewOfficialEvent(
				event.ID,
				event.Title,
				event.Address,
				event.Venue,
				event.Date,
				event.StartedAt,
				event.EndedAt,
				event.TypeId,
				event.TypeName,
				event.LeagueTitle,
				event.RegulationTitle,
				event.CSPFlg,
				event.Capacity,
				event.ShopId,
				event.ShopName,
				event.PrefectureId,
				event.PrefectureName,
				event.EnvironmentId,
				event.EnvironmentTitle,
				event.StandardRegulationId,
				event.StandardRegulationMarks,
			),
		)
	}

	return entities, nil
}

func (i *OfficialEvent) FindById(
	ctx context.Context,
	id uint,
) (*entity.OfficialEvent, error) {
	var event model.OfficialEvent

	tx := i.db.Table(
		"official_events",
	).Select(`
		official_events.id AS id,
		official_events.title AS title,
		official_events.address AS address,
		official_events.venue AS venue,
		official_events.date AS date,
		official_events.started_at AS started_at,
		official_events.ended_at AS ended_at,
		official_events.type_id AS type_id,
		official_events.type_name AS type_name,
		official_events.league_title AS league_title,
		official_events.regulation_title AS regulation_title,
		official_events.csp_flg AS csp_flg,
		official_events.capacity AS capacity,
		official_events.shop_id AS shop_id,
		official_events.shop_name AS shop_name,
		prefectures.id AS prefecture_id ,
		prefectures.name AS prefecture_name,
		environments.id AS environment_id,
		environments.title AS environment_title,
		standard_regulations.id AS standard_regulation_id,
		standard_regulations.marks AS standard_regulation_marks
	`,
	).Joins(
		"LEFT JOIN shops ON shops.id = official_events.shop_id",
	).Joins(
		"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	).Joins(
		"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
	).Joins(
		"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
	).Where(
		"official_events.id = ?", id,
	).Scan(&event)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return entity.NewOfficialEvent(
		event.ID,
		event.Title,
		event.Address,
		event.Venue,
		event.Date,
		event.StartedAt,
		event.EndedAt,
		event.TypeId,
		event.TypeName,
		event.LeagueTitle,
		event.RegulationTitle,
		event.CSPFlg,
		event.Capacity,
		event.ShopId,
		event.ShopName,
		event.PrefectureId,
		event.PrefectureName,
		event.EnvironmentId,
		event.EnvironmentTitle,
		event.StandardRegulationId,
		event.StandardRegulationMarks,
	), nil
}

// FindByShopIds は指定した店舗の公式イベントを期間内について開催日時の昇順で返す。
// Myジム(user_gyms)のイベント一覧に使う。
//
// 選択列と結合は Find / FindById と同じで、絞り込みだけが shop_id の集合になる。
func (i *OfficialEvent) FindByShopIds(
	ctx context.Context,
	shopIds []uint,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.OfficialEvent, error) {
	// Myジム未登録のユーザで IN () の全件走査を発生させない
	if len(shopIds) == 0 {
		return []*entity.OfficialEvent{}, nil
	}

	var events []*model.OfficialEvent

	tx := dbFromContext(ctx, i.db).Table(
		"official_events",
	).Select(`
		official_events.id AS id,
		official_events.title AS title,
		official_events.address AS address,
		official_events.venue AS venue,
		official_events.date AS date,
		official_events.started_at AS started_at,
		official_events.ended_at AS ended_at,
		official_events.type_id AS type_id,
		official_events.type_name AS type_name,
		official_events.league_title AS league_title,
		official_events.regulation_title AS regulation_title,
		official_events.csp_flg AS csp_flg,
		official_events.capacity AS capacity,
		official_events.shop_id AS shop_id,
		official_events.shop_name AS shop_name,
		prefectures.id AS prefecture_id ,
		prefectures.name AS prefecture_name,
		environments.id AS environment_id,
		environments.title AS environment_title,
		standard_regulations.id AS standard_regulation_id,
		standard_regulations.marks AS standard_regulation_marks
	`,
	).Joins(
		"LEFT JOIN shops ON shops.id = official_events.shop_id",
	).Joins(
		"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	).Joins(
		"LEFT JOIN environments ON environments.to_date >= official_events.date AND environments.from_date <= official_events.date",
	).Joins(
		"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
	).Where(
		"official_events.shop_id IN ? AND official_events.date BETWEEN ? AND ?", shopIds, startDate, endDate,
	).Order(
		// 同じ日に複数店舗のイベントが並ぶため、開始時刻だけでなく日付と
		// イベントIDまで含めて順序を一意にする(ページングはしないが表示のちらつきを防ぐ)。
		"official_events.date ASC, official_events.started_at ASC, official_events.id ASC",
	).Find(&events)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	entities := make([]*entity.OfficialEvent, 0, len(events))
	for _, event := range events {
		entities = append(
			entities,
			entity.NewOfficialEvent(
				event.ID,
				event.Title,
				event.Address,
				event.Venue,
				event.Date,
				event.StartedAt,
				event.EndedAt,
				event.TypeId,
				event.TypeName,
				event.LeagueTitle,
				event.RegulationTitle,
				event.CSPFlg,
				event.Capacity,
				event.ShopId,
				event.ShopName,
				event.PrefectureId,
				event.PrefectureName,
				event.EnvironmentId,
				event.EnvironmentTitle,
				event.StandardRegulationId,
				event.StandardRegulationMarks,
			),
		)
	}

	return entities, nil
}
