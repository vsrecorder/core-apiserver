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

	spritesByDeckCodeId, err := findDeckCodePokemonSpritesByDeckCodeIds(ctx, i.db, []string{deckcode.ID})
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
	e.PokemonSprites = spritesByDeckCodeId[deckcode.ID]

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

	spritesByDeckCodeId, err := findDeckCodePokemonSpritesByDeckCodeIds(ctx, i.db, deckCodeIds)
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
		e.PokemonSprites = spritesByDeckCodeId[deckcode.ID]
		entities = append(entities, e)
	}

	return entities, nil
}

func (i *DeckCode) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		// 付与タグ(中間テーブル)を先に物理削除する。deck_codes は論理削除のため、
		// 先に消すとサブクエリが deleted_at IS NULL で0件になり、タグの行が消し残る。
		if tx := tx.Where(
			"deck_code_id IN (?)",
			tx.Model(&model.DeckCode{}).Select("id").Where("user_id = ?", uid),
		).Delete(&model.DeckCodeTag{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		// スプライト(中間テーブル)もタグと同じ理由で、deck_codes より先に物理削除する。
		if tx := tx.Where(
			"deck_code_id IN (?)",
			tx.Model(&model.DeckCode{}).Select("id").Where("user_id = ?", uid),
		).Delete(&model.DeckCodePokemonSprite{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		if tx := tx.Where("user_id = ?", uid).Delete(&model.DeckCode{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		return nil
	})
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

	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(deckcode).Error; err != nil {
			logError(ctx, err)
			return err
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}

// ReplacePokemonSprites は deckCodeId のスプライトを「全削除 → 再INSERT」で置き換える。
// ctx にトランザクションがあればそれを使う(本体の Save と同じトランザクションにまとめるため)。
func (i *DeckCode) ReplacePokemonSprites(
	ctx context.Context,
	deckCodeId string,
	sprites []*entity.PokemonSprite,
) error {
	return dbFromContext(ctx, i.db).Transaction(func(tx *gorm.DB) error {
		if tx := tx.Where("deck_code_id = ?", deckCodeId).Delete(&model.DeckCodePokemonSprite{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		for idx, sprite := range sprites {
			// position が指定されていればスロットとして使う。
			// 未指定(0)は配列インデックスから採番する(deck_pokemon_sprites と同じ規約)。
			position := sprite.Position
			if position == 0 {
				position = uint(idx + 1)
			}

			if err := tx.Create(model.NewDeckCodePokemonSprite(deckCodeId, position, sprite.ID)).Error; err != nil {
				logError(ctx, err)
				// 存在しないスプライトIDは外部キーで弾かれる。クライアント起因なので 400 で返せるよう変換する。
				return wrapForeignKeyViolation(err)
			}
		}

		return nil
	})
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
