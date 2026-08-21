package infrastructure

import (
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type DeckCode struct {
	db *gorm.DB
}

func NewDeckCode(
	db *gorm.DB,
) repository.DeckCodeInterface {
	return &DeckCode{db}
}

func (i *DeckCode) FindById(
	ctx context.Context,
	id string,
) (*entity.DeckCode, error) {
	var deckcode *model.DeckCode

	if tx := i.db.Where("id = ?", id).First(&deckcode); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
	}

	tagsByDeckCodeId, err := findTagsByDeckCodeIds(ctx, i.db, []string{deckcode.ID})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	e := entity.NewDeckCode(
		deckcode.ID,
		deckcode.CreatedAt,
		deckcode.UserId,
		deckcode.DeckId,
		deckcode.Code,
		deckcode.PrivateCodeFlg,
		deckcode.Memo,
	)
	e.Tags = tagsByDeckCodeId[deckcode.ID]

	return e, nil
}

func (i *DeckCode) FindByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCode, error) {
	var deckcodes []*model.DeckCode

	if tx := i.db.Where("deck_id = ? ", deckId).Order("created_at DESC, updated_at DESC").Find(&deckcodes); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	deckCodeIds := make([]string, 0, len(deckcodes))
	for _, deckcode := range deckcodes {
		deckCodeIds = append(deckCodeIds, deckcode.ID)
	}

	tagsByDeckCodeId, err := findTagsByDeckCodeIds(ctx, i.db, deckCodeIds)
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var entities []*entity.DeckCode
	for _, deckcode := range deckcodes {
		e := entity.NewDeckCode(
			deckcode.ID,
			deckcode.CreatedAt,
			deckcode.UserId,
			deckcode.DeckId,
			deckcode.Code,
			deckcode.PrivateCodeFlg,
			deckcode.Memo,
		)
		e.Tags = tagsByDeckCodeId[deckcode.ID]
		entities = append(entities, e)
	}

	return entities, nil
}

func (i *DeckCode) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("user_id = ?", uid).Delete(&model.DeckCode{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}

func (i *DeckCode) Save(
	ctx context.Context,
	entity *entity.DeckCode,
) error {
	deckcode := model.NewDeckCode(
		entity.ID,
		entity.CreatedAt,
		entity.UserId,
		entity.DeckId,
		entity.Code,
		entity.PrivateCodeFlg,
		entity.Memo,
	)

	return i.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(deckcode).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

func (i *DeckCode) Delete(
	ctx context.Context,
	id string,
) error {
	if tx := dbFromContext(ctx, i.db).Where("id = ?", id).Delete(&model.DeckCode{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return nil
}
