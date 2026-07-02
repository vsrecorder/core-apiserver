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
		return nil, wrapError(tx.Error)
	}

	entity := entity.NewDeckCode(
		deckcode.ID,
		deckcode.CreatedAt,
		deckcode.UserId,
		deckcode.DeckId,
		deckcode.Code,
		deckcode.PrivateCodeFlg,
		deckcode.Memo,
	)

	return entity, nil
}

func (i *DeckCode) FindByDeckId(
	ctx context.Context,
	deckId string,
) ([]*entity.DeckCode, error) {
	var deckcodes []*model.DeckCode

	if tx := i.db.Where("deck_id = ? ", deckId).Order("created_at DESC, updated_at DESC").Find(&deckcodes); tx.Error != nil {
		return nil, tx.Error
	}

	var entities []*entity.DeckCode
	for _, deckcode := range deckcodes {
		entity := entity.NewDeckCode(
			deckcode.ID,
			deckcode.CreatedAt,
			deckcode.UserId,
			deckcode.DeckId,
			deckcode.Code,
			deckcode.PrivateCodeFlg,
			deckcode.Memo,
		)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (i *DeckCode) FindIdsByUserId(
	ctx context.Context,
	uid string,
) ([]string, error) {
	var ids []string

	if tx := dbFromContext(ctx, i.db).Model(&model.DeckCode{}).Where("user_id = ?", uid).Pluck("id", &ids); tx.Error != nil {
		return nil, tx.Error
	}

	return ids, nil
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
		return tx.Error
	}

	return nil
}
