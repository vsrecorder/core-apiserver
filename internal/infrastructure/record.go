package infrastructure

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Record struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewRecord(
	db *gorm.DB,
	logger *slog.Logger,
) repository.RecordInterface {
	return &Record{db, logger}
}

// newRecordEntity は records の1行をエンティティへ変換する。
// 付与タグ(Tags)は別テーブルのため、ここでは詰めない。
func newRecordEntity(m *model.Record) *entity.Record {
	ret := entity.NewRecord(
		m.ID,
		m.CreatedAt,
		m.OfficialEventId,
		m.TonamelEventId,
		m.FriendId,
		m.UnofficialEventId,
		m.UserId,
		m.DeckId,
		m.DeckCodeId,
		m.EventDate,
		m.PrivateFlg,
		m.IgnoreStatsFlg,
		m.RegulationId,
		m.TCGMeisterURL,
		m.Memo,
	)
	ret.DeckRegisteredAt = m.DeckRegisteredAt

	return ret
}

// newRecordEntitiesWithTags は複数行をエンティティへ変換し、付与タグを1クエリで
// まとめて詰める。記録一覧は1ページで数十件返るため、記録ごとにタグを引くと
// そのままN+1になる(対戦結果の findTagsByMatchIds と同じ方針)。
func (i *Record) newRecordEntitiesWithTags(
	ctx context.Context,
	models []*model.Record,
) ([]*entity.Record, error) {
	if len(models) == 0 {
		return nil, nil
	}

	recordIds := make([]string, 0, len(models))
	for _, m := range models {
		recordIds = append(recordIds, m.ID)
	}

	tagsByRecordId, err := findTagsByRecordIds(ctx, i.db, recordIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	entities := make([]*entity.Record, 0, len(models))
	for _, m := range models {
		record := newRecordEntity(m)
		record.Tags = tagsByRecordId[m.ID]
		entities = append(entities, record)
	}

	return entities, nil
}

func (i *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	var model model.Record

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	ret := newRecordEntity(&model)

	tagsByRecordId, err := findTagsByRecordIds(ctx, i.db, []string{id})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}
	ret.Tags = tagsByRecordId[id]

	return ret, nil
}

func (i *Record) Find(
	ctx context.Context,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	switch eventType {
	case "official":
		if tx := i.db.Where("official_event_id != 0 AND private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

// buildCursorCondition は ORDER BY event_date DESC NULLS LAST, created_at DESC に対応した
// カーソルページング用の WHERE 条件と引数を返す。
// cursorEventDate が非ゼロ（event_date あり区間）の場合:
//   - 同日のうち created_at が小さいレコードを含める
//   - event_date IS NULL のレコードは常に含める（全件 dated records の後続にある）
//
// cursorEventDate がゼロ（event_date IS NULL 区間）の場合:
//   - event_date IS NULL かつ created_at < cursorCreatedAt のみ返す
func buildCursorCondition(cursorEventDate, cursorCreatedAt time.Time) (string, []interface{}) {
	if !cursorEventDate.IsZero() {
		return "((event_date < ? AND event_date IS NOT NULL) OR (event_date = ? AND created_at < ?) OR event_date IS NULL)",
			[]interface{}{cursorEventDate, cursorEventDate, cursorCreatedAt}
	}
	return "(event_date IS NULL AND created_at < ?)", []interface{}{cursorCreatedAt}
}

func (i *Record) FindOnCursor(
	ctx context.Context,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	cursorCond, cursorArgs := buildCursorCondition(cursorEventDate, cursorCreatedAt)

	var cond string
	switch eventType {
	case "official":
		cond = "official_event_id != 0 AND " + cursorCond + " AND private_flg = false"
	case "tonamel":
		cond = "tonamel_event_id != '' AND " + cursorCond + " AND private_flg = false"
	case "unofficial":
		cond = "unofficial_event_id != '' AND " + cursorCond + " AND private_flg = false"
	default:
		cond = cursorCond + " AND private_flg = false"
	}

	if tx := i.db.Where(cond, cursorArgs...).Limit(limit).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByUserId(
	ctx context.Context,
	uid string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	switch eventType {
	case "official":
		if tx := i.db.Where("official_event_id != 0 AND user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	cursorCond, cursorArgs := buildCursorCondition(cursorEventDate, cursorCreatedAt)
	uidArgs := append(cursorArgs, uid)

	var cond string
	switch eventType {
	case "official":
		cond = "official_event_id != 0 AND " + cursorCond + " AND user_id = ?"
	case "tonamel":
		cond = "tonamel_event_id != '' AND " + cursorCond + " AND user_id = ?"
	case "unofficial":
		cond = "unofficial_event_id != '' AND " + cursorCond + " AND user_id = ?"
	default:
		cond = cursorCond + " AND user_id = ?"
	}

	if tx := i.db.Where(cond, uidArgs...).Limit(limit).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("official_event_id = ? AND private_flg = ?", officialEventId, false).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByTonamelEventId(
	ctx context.Context,
	tonamelEventId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("tonamel_event_id = ? AND private_flg = ?", tonamelEventId, false).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByDeckId(
	ctx context.Context,
	deckId string,
	limit int,
	offset int,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	switch eventType {
	case "official":
		if tx := i.db.Where("official_event_id != 0 AND deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByDeckIdOnCursor(
	ctx context.Context,
	deckId string,
	limit int,
	cursorEventDate time.Time,
	cursorCreatedAt time.Time,
	eventType string,
) ([]*entity.Record, error) {
	var models []*model.Record

	cursorCond, cursorArgs := buildCursorCondition(cursorEventDate, cursorCreatedAt)
	deckArgs := append([]interface{}{deckId}, cursorArgs...)

	var cond string
	switch eventType {
	case "official":
		cond = "official_event_id != 0 AND deck_id = ? AND " + cursorCond
	case "tonamel":
		cond = "tonamel_event_id != '' AND deck_id = ? AND " + cursorCond
	case "unofficial":
		cond = "unofficial_event_id != '' AND deck_id = ? AND " + cursorCond
	default:
		cond = "deck_id = ? AND " + cursorCond
	}

	if tx := i.db.Where(cond, deckArgs...).Limit(limit).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) FindByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("deck_code_id = ?", deckCodeId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	return i.newRecordEntitiesWithTags(ctx, models)
}

func (i *Record) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	db := dbFromContext(ctx, i.db)

	return db.Transaction(func(tx *gorm.DB) error {
		// 消す順序は「参照する側が先」。records / matches は論理削除のため、先に消すと
		// 以降のサブクエリが deleted_at IS NULL で0件になり、子の行が消し残る。
		//
		// 各サブクエリは毎回 tx.Model(...) から組み立てる。GORM の *gorm.DB は
		// 条件を積んでいくため、同じインスタンスを使い回すと条件が混ざる。
		recordIds := func() *gorm.DB {
			return tx.Model(&model.Record{}).Select("id").Where("user_id = ?", uid)
		}
		// 対戦結果は「自分が作ったもの」と「自分の記録に紐づくもの」の両方を対象にする。
		// 通常この2つは一致するが、片方だけの孤立行が残らないよう両側から拾う。
		matchIds := func() *gorm.DB {
			return tx.Model(&model.Match{}).Select("id").Where(
				"user_id = ? OR record_id IN (?)", uid, recordIds(),
			)
		}

		// 対戦結果に紐づく中間テーブル。論理削除を持たないため物理削除する。
		if tx := tx.Where("match_id IN (?)", matchIds()).Delete(&model.MatchPokemonSprite{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		if tx := tx.Where("match_id IN (?)", matchIds()).Delete(&model.MatchTag{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 記録に紐づく中間テーブル。
		if tx := tx.Where("record_id IN (?)", recordIds()).Delete(&model.RecordTag{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 対局(games)
		if tx := tx.Where(
			"user_id = ? OR match_id IN (?)", uid, matchIds(),
		).Delete(&model.Game{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 対戦結果(matches)
		if tx := tx.Where(
			"user_id = ? OR record_id IN (?)", uid, recordIds(),
		).Delete(&model.Match{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 自由形式イベント(unofficial_events): 記録から参照されているものだけでなく、
		// どの記録からも参照されないまま残った作りかけのものも user_id から消す。
		if tx := tx.Where("user_id = ?", uid).Delete(&model.UnofficialEvent{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 記録(records)
		if tx := tx.Where("user_id = ?", uid).Delete(&model.Record{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

// normalizeRegulationId は未設定(0)のレギュレーションを、DB側の DEFAULT と同じ
// スタンダードへ寄せる。records.regulation_id は regulations へのFK制約付きで
// 0のままでは保存できないため、Save(GORMは全カラムを書く)の前に必ず通す。
// Save の引数名 entity が entity パッケージを隠すため、関数へ切り出している。
func normalizeRegulationId(record *entity.Record) uint {
	return entity.NormalizeRegulationId(record.RegulationId)
}

func (i *Record) Save(
	ctx context.Context,
	entity *entity.Record,
) error {
	model := model.NewRecord(
		entity.ID,
		entity.CreatedAt,
		entity.OfficialEventId,
		entity.TonamelEventId,
		entity.FriendId,
		entity.UserId,
		entity.DeckId,
		entity.DeckCodeId,
		entity.PrivateFlg,
		entity.IgnoreStatsFlg,
		normalizeRegulationId(entity),
		entity.TCGMeisterURL,
		entity.Memo,
		entity.EventDate,
		entity.UnofficialEventId,
	)
	model.DeckRegisteredAt = entity.DeckRegisteredAt

	// 記録本体とタグの付与を1つのトランザクションにまとめられるよう、
	// ctx にトランザクションがあればそれを使う(usecase.Record.Create/Update)。
	if tx := dbFromContext(ctx, i.db).Save(model); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

func (i *Record) Delete(
	ctx context.Context,
	id string,
) error {
	db := dbFromContext(ctx, i.db)

	// 削除対象の record が参照している自由形式イベント(unofficial_event)を把握するため、
	// 先に record を取得しておく
	var record model.Record
	if tx := db.Where("id = ?", id).First(&record); tx.Error != nil {
		logError(ctx, tx.Error)
		return wrapError(tx.Error)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 対局(games)は、マッチを1件ずつ引いてから消すとマッチ数に比例してクエリが増える。
		// 消す対象はサブクエリで指定できるため、マッチ数によらず1文で済ませる。
		// 退会処理(usecase.User.Delete)は記録の数だけこの削除を呼ぶので、
		// ここが記録数×マッチ数に膨らむと1トランザクションの保持時間がそのまま延びる。
		if tx := tx.Where(
			"match_id IN (?)",
			tx.Model(&model.Match{}).Select("id").Where("record_id = ?", id),
		).Delete(&model.Game{}); tx.Error != nil {
			return tx.Error
		}

		if tx := tx.Where("record_id = ?", id).Delete(&model.Match{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		if tx := tx.Where("id = ?", id).Delete(&model.Record{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// 自由形式イベントを参照していた場合、紐づく unofficial_event も削除する(孤立行を残さない)
		if record.UnofficialEventId != "" {
			if tx := tx.Where("id = ?", record.UnofficialEventId).Delete(&model.UnofficialEvent{}); tx.Error != nil {
				logError(ctx, tx.Error)
				return tx.Error
			}
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}
