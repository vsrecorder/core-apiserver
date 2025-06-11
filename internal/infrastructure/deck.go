package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
	"gorm.io/gorm"
)

type Deck struct {
	db *gorm.DB
}

func NewDeck(
	db *gorm.DB,
) repository.DeckInterface {
	return &Deck{db}
}

func (i *Deck) Find(
	ctx context.Context,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	var models []*model.Deck

	if tx := i.db.Where("private_code_flg = false AND code IS NOT NULL AND archived_at IS NULL").Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Deck
	for _, model := range models {
		entity := entity.NewDeck(
			model.ID,
			model.CreatedAt,
			model.ArchivedAt.Time,
			model.UserId,
			model.Name,
			model.Code,
			model.PrivateCodeFlg,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Deck) FindOnCursor(
	ctx context.Context,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	var models []*model.Deck

	if tx := i.db.Where("created_at < ? AND private_code_flg = false AND code IS NOT NULL AND archived_at IS NULL", cursor).Limit(limit).Order("created_at DESC").Find(&models); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.Deck
	for _, model := range models {
		entity := entity.NewDeck(
			model.ID,
			model.CreatedAt,
			model.ArchivedAt.Time,
			model.UserId,
			model.Name,
			model.Code,
			model.PrivateCodeFlg,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Deck) FindById(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	var model *model.Deck

	if tx := i.db.Where("id = ?", id).First(&model); tx.Error != nil {
		return nil, tx.Error
	}

	entity := entity.NewDeck(
		model.ID,
		model.CreatedAt,
		model.ArchivedAt.Time,
		model.UserId,
		model.Name,
		model.Code,
		model.PrivateCodeFlg,
	)

	return entity, nil
}

func (i *Deck) FindByUserId(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	var models []*model.Deck

	if archivedFlg {
		if tx := i.db.Where("user_id = ? AND archived_at IS NOT NULL", uid).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("user_id = ? AND archived_at IS NULL", uid).Limit(limit).Offset(offset).Order("created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	var entities []*entity.Deck
	for _, model := range models {
		entity := entity.NewDeck(
			model.ID,
			model.CreatedAt,
			model.ArchivedAt.Time,
			model.UserId,
			model.Name,
			model.Code,
			model.PrivateCodeFlg,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Deck) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	var models []*model.Deck

	if archivedFlg {
		if tx := i.db.Where("created_at < ? AND user_id = ? AND archived_at IS NOT NULL", cursor, uid).Limit(limit).Order("created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		if tx := i.db.Where("created_at < ? AND user_id = ? AND archived_at IS NULL", cursor, uid).Limit(limit).Order("created_at DESC").Find(&models); tx.Error != nil {
			return nil, tx.Error
		}
	}

	var entities []*entity.Deck
	for _, model := range models {
		entity := entity.NewDeck(
			model.ID,
			model.CreatedAt,
			model.ArchivedAt.Time,
			model.UserId,
			model.Name,
			model.Code,
			model.PrivateCodeFlg,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *Deck) Save(
	ctx context.Context,
	entity *entity.Deck,
) error {
	archivedAt := sql.NullTime{}
	archivedAt.Time = entity.ArchivedAt

	if entity.ArchivedAt.IsZero() {
		archivedAt.Valid = false
	} else {
		archivedAt.Valid = true
	}

	model := model.NewDeck(
		entity.ID,
		entity.CreatedAt,
		archivedAt,
		entity.UserId,
		entity.Name,
		entity.Code,
		entity.PrivateCodeFlg,
	)

	if tx := i.db.Save(model); tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (i *Deck) Delete(
	ctx context.Context,
	id string,
) error {
	if tx := i.db.Where("id = ?", id).Delete(&model.Deck{}); tx.Error != nil {
		return tx.Error
	}

	return nil
}
