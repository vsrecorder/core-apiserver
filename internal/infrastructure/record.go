package infrastructure

import (
	"context"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type Record struct {
	db *gorm.DB
}

func NewRecord(
	db *gorm.DB,
) repository.RecordInterface {
	return &Record{db}
}

func (i *Record) Find(
	ctx context.Context,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("private_flg = ?", false).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
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
			model.UserId,
			model.DeckId,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Record) FindById(
	ctx context.Context,
	id string,
) (*entity.Record, error) {
	var model model.Record

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewRecord(
		model.ID,
		model.CreatedAt,
		model.OfficialEventId,
		model.TonamelEventId,
		model.FriendId,
		model.UserId,
		model.DeckId,
		model.PrivateFlg,
		model.TCGMeisterURL,
		model.Memo,
	)

	return entity, nil
}

func (i *Record) FindByUserId(
	ctx context.Context,
	uid string,
	limit int,
	offset int,
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("user_id = ?", uid).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
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
			model.UserId,
			model.DeckId,
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

	if tx := i.db.Where("official_event_id = ? AND private_flg = ?", officialEventId, false).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
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
			model.UserId,
			model.DeckId,
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

	if tx := i.db.Where("tonamel_event_id = ? AND private_flg = ?", tonamelEventId, false).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
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
			model.UserId,
			model.DeckId,
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
) ([]*entity.Record, error) {
	var models []*model.Record

	if tx := i.db.Where("deck_id = ?", deckId).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
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
			model.UserId,
			model.DeckId,
			model.PrivateFlg,
			model.TCGMeisterURL,
			model.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
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
		entity.PrivateFlg,
		entity.TCGMeisterURL,
		entity.Memo,
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
	if tx := i.db.Where("id = ?", id).Delete(&model.Record{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}
