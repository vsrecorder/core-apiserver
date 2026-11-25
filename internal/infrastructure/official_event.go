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

type OfficialEvent struct {
	db *gorm.DB
}

func NewOfficialEvent(
	db *gorm.DB,
) repository.OfficialEventInterface {
	return &OfficialEvent{db}
}

// officialEventSelect は公式イベントを引く全メソッドで共通のSELECT句。
const officialEventSelect = `
	official_events.id AS id,
	official_events.title AS title,
	official_events.address AS address,
	official_events.venue AS venue,
	official_events.date AS date,
	official_events.started_at AS started_at,
	official_events.ended_at AS ended_at,
	official_events.type_id AS type_id,
	official_events.type_name AS type_name,
	official_events.league_id AS league_id,
	official_events.league_title AS league_title,
	official_events.regulation_title AS regulation_title,
	official_events.csp_flg AS csp_flg,
	official_events.capacity AS capacity,
	official_events.shop_id AS shop_id,
	official_events.shop_name AS shop_name,
	prefectures.id AS prefecture_id,
	prefectures.name AS prefecture_name,
	environments.id AS environment_id,
	environments.title AS environment_title,
	standard_regulations.id AS standard_regulation_id,
	standard_regulations.marks AS standard_regulation_marks
`

// officialEventJoins は公式イベントを引く全メソッドで共通のJOIN句。
//
// 環境(environments)は開催日から引くのが基本だが、大型大会(チャンピオンズリーグ・PJCS)は
// 開催日時点で発売済みの最新弾がカードプールに入らないことがあり、開催日から引くと実際に
// 対戦する環境とズレる。official_event_environments に例外登録があればそちらを優先する
// (判定は usecase.ResolveEnvironmentForOfficialEvent と揃えてある)。
//
// 例外の有無をORで分岐させず相関サブクエリにしているのは、official_events が100万行を
// 超えており、ORのJOIN条件だとプランが崩れてFindByShopIds(ホームのパネルが全ログイン
// ユーザ分叩く)の走査量が戻ってしまうため。environments は数十行しかないので、
// サブクエリ自体のコストは無視できる。
var officialEventJoins = []string{
	"LEFT JOIN shops ON shops.id = official_events.shop_id",
	"LEFT JOIN prefectures ON prefectures.id = shops.prefecture_id",
	"LEFT JOIN official_event_environments AS oee ON oee.official_event_id = official_events.id",
	`LEFT JOIN environments ON environments.id = COALESCE(
		oee.environment_id,
		(SELECT e.id FROM environments AS e WHERE e.from_date <= official_events.date AND e.to_date >= official_events.date)
	)`,
	"LEFT JOIN standard_regulations ON standard_regulations.to_date >= official_events.date AND standard_regulations.from_date <= official_events.date",
}

// officialEventBaseQuery は公式イベントを引くクエリの共通部分(SELECT・JOIN)を組み立てる。
// 絞り込みと並び順だけを各メソッドが足す。
func officialEventBaseQuery(db *gorm.DB) *gorm.DB {
	tx := db.Table("official_events").Select(officialEventSelect)

	for _, join := range officialEventJoins {
		tx = tx.Joins(join)
	}

	return tx
}

// officialEventsFindLimit は Find が1度に返す公式イベントの最大件数。
//
// このエンドポイントはログインなしで呼べ、期間と種類は呼び出し側が決める。ジムイベントは
// 1週間で5,000件を超え、種類を絞らない1週間(約6,300件)を返すだけでプロセスのメモリが
// 約30MB増える(2026-11-25 本番で実測)。件数に上限が無いと、広い期間を1回指定されるだけで
// コンテナのメモリ上限(128MB)を超えて落ちうる。
//
// 正規の利用で最も多いのは、記録作成画面が引く「種類を絞らない1日ぶん」の約1,300件と、
// シティリーグ90日ぶんの約640件(同日実測)。その数倍の余裕を取る。超えた場合は一部だけを
// 返さずエラーにする(取り込みバッチが欠けた一覧を正しいものとして扱わないようにするため)。
const officialEventsFindLimit = 5000

func (i *OfficialEvent) Find(
	ctx context.Context,
	typeId uint,
	leagueType uint,
	startDate time.Time,
	endDate time.Time,
) ([]*entity.OfficialEvent, error) {
	var events []*model.OfficialEvent

	tx := officialEventBaseQuery(i.db)

	// リーグの種類は 1〜4 のときだけ絞り込む。範囲外は指定なしとして扱う
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

	// 大会の種類: 大型大会(1) / シティ(2) / トレリ(3) / ジムイベント(4) /
	// オーガナイザーイベント(6) / その他(7)。0 は指定なし
	switch {
	case typeId == 0 && leagueTitle != "":
		tx = tx.Where("league_title = ? AND date BETWEEN ? AND ?", leagueTitle, startDate, endDate)
	case typeId == 0:
		tx = tx.Where("date BETWEEN ? AND ?", startDate, endDate)
	case leagueTitle != "":
		tx = tx.Where(
			"type_id = ? AND league_title = ? AND date BETWEEN ? AND ?", typeId, leagueTitle, startDate, endDate,
		)
	default:
		tx = tx.Where("type_id = ? AND date BETWEEN ? AND ?", typeId, startDate, endDate)
	}

	// 上限を1件だけ超えて引き、超えていたら「多すぎる」と分かるようにする(理由は officialEventsFindLimit)
	tx = tx.Order("started_at ASC").Limit(officialEventsFindLimit + 1).Find(&events)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(events) > officialEventsFindLimit {
		return nil, apperror.ErrTooManyResults
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
				event.LeagueId,
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

	tx := officialEventBaseQuery(i.db).Where(
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
		event.LeagueId,
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

	tx := officialEventBaseQuery(dbFromContext(ctx, i.db)).Where(
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
				event.LeagueId,
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
