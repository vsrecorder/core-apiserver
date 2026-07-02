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

func (i *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	var model model.Record

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewRecord(
		model.ID,
		model.CreatedAt,
		model.OfficialEventId,
		model.TonamelEventId,
		model.FriendId,
		model.UnofficialEventId,
		model.UserId,
		model.DeckId,
		model.DeckCodeId,
		model.EventDate,
		model.PrivateFlg,
		model.TCGMeisterURL,
		model.Memo,
	)

	return entity, nil
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
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("private_flg = false").Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("user_id = ?", uid).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Record) FindByOfficialEventId(
	ctx context.Context,
	officialEventId uint,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("official_event_id = ? AND private_flg = ?", officialEventId, false).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Record) FindByTonamelEventId(
	ctx context.Context,
	tonamelEventId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("tonamel_event_id = ? AND private_flg = ?", tonamelEventId, false).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
			return nil, tx.Error
		}
	case "tonamel":
		if tx := i.db.Where("tonamel_event_id != '' AND deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	case "unofficial":
		if tx := i.db.Where("unofficial_event_id != '' AND deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	default:
		if tx := i.db.Where("deck_id = ?", deckId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Record) FindByDeckCodeId(
	ctx context.Context,
	deckCodeId string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("deck_code_id = ?", deckCodeId).Limit(limit).Offset(offset).Order("event_date DESC NULLS LAST, created_at DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Record
	for _, model := range models {
		entity := entity.NewRecord(
			model.ID,
			model.CreatedAt,
			model.OfficialEventId,
			model.TonamelEventId,
			model.FriendId,
			model.UnofficialEventId,
			model.UserId,
			model.DeckId,
			model.DeckCodeId,
			model.EventDate,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Record) FindIdsByUserId(
	ctx context.Context,
	uid string,
) ([]string, error) {
	var ids []string

	if tx := dbFromContext(ctx, i.db).Model(&model.Record{}).Where("user_id = ?", uid).Pluck("id", &ids); tx.Error != nil {
		return nil, tx.Error
	}

	return ids, nil
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
		entity.TCGMeisterURL,
		entity.Memo,
		entity.EventDate,
		entity.UnofficialEventId,
	)

	if tx := i.db.Save(model); tx.Error != nil {
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
		return wrapError(tx.Error)
	}

	var matches []*model.Match
	if tx := db.Where("record_id = ?", id).Order("created_at ASC").Find(&matches); tx.Error != nil {
		return tx.Error
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, match := range matches {
			if tx := tx.Where("match_id = ?", match.ID).Delete(&model.Game{}); tx.Error != nil {
				return tx.Error
			}
		}

		if tx := tx.Where("record_id = ?", id).Delete(&model.Match{}); tx.Error != nil {
			return tx.Error
		}

		if tx := tx.Where("id = ?", id).Delete(&model.Record{}); tx.Error != nil {
			return tx.Error
		}

		// 自由形式イベントを参照していた場合、紐づく unofficial_event も削除する(孤立行を残さない)
		if record.UnofficialEventId != "" {
			if tx := tx.Where("id = ?", record.UnofficialEventId).Delete(&model.UnofficialEvent{}); tx.Error != nil {
				return tx.Error
			}
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}
