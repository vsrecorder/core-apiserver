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
	var deckJoinDeckCodes []*model.DeckJoinDeckCode

	tx := i.db.Table(
		"decks",
	).Select(`
		decks.id AS deck_id,
		decks.created_at AS deck_created_at,
		decks.updated_at AS deck_updated_at,
		decks.deleted_at AS deck_deleted_at,
		decks.archived_at AS deck_archived_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
		decks.code AS deck_code,
		decks.private_code_flg AS deck_private_code_flg,
		decks.private_flg AS deck_private_flg,
		deck_codes.id AS deck_code_id,
		deck_codes.created_at AS deck_code_created_at,
		deck_codes.updated_at AS deck_code_updated_at,
		deck_codes.deleted_at AS deck_code_deleted_at,
		deck_codes.user_id AS deck_code_user_id,
		deck_codes.deck_id AS deck_code_deck_id,
		deck_codes.code AS deck_code_code,
		deck_codes.private_code_flg AS deck_code_private_code_flg,
		deck_codes.memo AS deck_code_memo
	`,
	).Joins(`
		LEFT JOIN (
			SELECT DISTINCT ON (deck_id)
				id,
				created_at,
				updated_at,
				deleted_at,
				user_id,
				deck_id,
				code,
				private_code_flg,
				memo
			FROM deck_codes
			ORDER BY deck_id, created_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
	`,
	).Where(
		"decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL",
	).Order(
		"decks.created_at DESC",
	).Limit(
		limit,
	).Offset(
		offset,
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		ret = append(ret, entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
			djdc.DeckCode,
			djdc.DeckPrivateCodeFlg,
			djdc.DeckPrivateFlg,
			entity.NewDeckCode(
				djdc.DeckCodeID,
				djdc.DeckCodeCreatedAt,
				djdc.DeckCodeUserId,
				djdc.DeckCodeDeckId,
				djdc.DeckCodeCode,
				djdc.DeckCodePrivateCodeFlg,
				djdc.DeckCodeMemo,
			),
		))
	}

	return ret, nil
}

func (i *Deck) FindAll(
	ctx context.Context,
	uid string,
) ([]*entity.Deck, error) {
	var deckJoinDeckCodes []*model.DeckJoinDeckCode

	tx := i.db.Table(
		"decks",
	).Select(`
			decks.id AS deck_id,
			decks.created_at AS deck_created_at,
			decks.updated_at AS deck_updated_at,
			decks.deleted_at AS deck_deleted_at,
			decks.archived_at AS deck_archived_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
			decks.code AS deck_code,
			decks.private_code_flg AS deck_private_code_flg,
			decks.private_flg AS deck_private_flg,
			deck_codes.id AS deck_code_id,
			deck_codes.created_at AS deck_code_created_at,
			deck_codes.updated_at AS deck_code_updated_at,
			deck_codes.deleted_at AS deck_code_deleted_at,
			deck_codes.user_id AS deck_code_user_id,
			deck_codes.deck_id AS deck_code_deck_id,
			deck_codes.code AS deck_code_code,
			deck_codes.private_code_flg AS deck_code_private_code_flg,
			deck_codes.memo AS deck_code_memo
		`,
	).Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (deck_id)
					id,
					created_at,
					updated_at,
					deleted_at,
					user_id,
					deck_id,
					code,
					private_code_flg,
					memo
				FROM deck_codes
				WHERE user_id = ?
				ORDER BY deck_id, created_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
		`, uid,
	).Where(
		"decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", uid,
	).Order(
		"decks.created_at DESC",
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		ret = append(ret, entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
			djdc.DeckCode,
			djdc.DeckPrivateCodeFlg,
			djdc.DeckPrivateFlg,
			entity.NewDeckCode(
				djdc.DeckCodeID,
				djdc.DeckCodeCreatedAt,
				djdc.DeckCodeUserId,
				djdc.DeckCodeDeckId,
				djdc.DeckCodeCode,
				djdc.DeckCodePrivateCodeFlg,
				djdc.DeckCodeMemo,
			),
		))
	}

	return ret, nil
}

func (i *Deck) FindOnCursor(
	ctx context.Context,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	var deckJoinDeckCodes []*model.DeckJoinDeckCode

	tx := i.db.Table(
		"decks",
	).Select(`
		decks.id AS deck_id,
		decks.created_at AS deck_created_at,
		decks.updated_at AS deck_updated_at,
		decks.deleted_at AS deck_deleted_at,
		decks.archived_at AS deck_archived_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
		decks.code AS deck_code,
		decks.private_code_flg AS deck_private_code_flg,
		decks.private_flg AS deck_private_flg,
		deck_codes.id AS deck_code_id,
		deck_codes.created_at AS deck_code_created_at,
		deck_codes.updated_at AS deck_code_updated_at,
		deck_codes.deleted_at AS deck_code_deleted_at,
		deck_codes.user_id AS deck_code_user_id,
		deck_codes.deck_id AS deck_code_deck_id,
		deck_codes.code AS deck_code_code,
		deck_codes.private_code_flg AS deck_code_private_code_flg,
		deck_codes.memo AS deck_code_memo
	`,
	).Joins(`
		LEFT JOIN (
			SELECT DISTINCT ON (deck_id)
				id,
				created_at,
				updated_at,
				deleted_at,
				user_id,
				deck_id,
				code,
				private_code_flg,
				memo
			FROM deck_codes
			ORDER BY deck_id, created_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
	`,
	).Where(
		"decks.created_at < ? AND decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", cursor,
	).Order(
		"decks.created_at DESC",
	).Limit(
		limit,
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		ret = append(ret, entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
			djdc.DeckCode,
			djdc.DeckPrivateCodeFlg,
			djdc.DeckPrivateFlg,
			entity.NewDeckCode(
				djdc.DeckCodeID,
				djdc.DeckCodeCreatedAt,
				djdc.DeckCodeUserId,
				djdc.DeckCodeDeckId,
				djdc.DeckCodeCode,
				djdc.DeckCodePrivateCodeFlg,
				djdc.DeckCodeMemo,
			),
		))
	}

	return ret, nil
}

func (i *Deck) FindById(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// idの存在確認
	if tx := i.db.Where("id = ?", id).First(&model.Deck{}); tx.Error != nil {
		return nil, tx.Error
	}

	var deckJoinDeckCodes *model.DeckJoinDeckCode

	tx := i.db.Table(
		"decks",
	).Select(`
		decks.id AS deck_id,
		decks.created_at AS deck_created_at,
		decks.updated_at AS deck_updated_at,
		decks.deleted_at AS deck_deleted_at,
		decks.archived_at AS deck_archived_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
		decks.code AS deck_code,
		decks.private_code_flg AS deck_private_code_flg,
		decks.private_flg AS deck_private_flg,
		deck_codes.id AS deck_code_id,
		deck_codes.created_at AS deck_code_created_at,
		deck_codes.updated_at AS deck_code_updated_at,
		deck_codes.deleted_at AS deck_code_deleted_at,
		deck_codes.user_id AS deck_code_user_id,
		deck_codes.deck_id AS deck_code_deck_id,
		deck_codes.code AS deck_code_code,
		deck_codes.private_code_flg AS deck_code_private_code_flg,
		deck_codes.memo AS deck_code_memo
	`,
	).Joins(`
		LEFT JOIN (
			SELECT DISTINCT ON (deck_id)
				id,
				created_at,
				updated_at,
				deleted_at,
				user_id,
				deck_id,
				code,
				private_code_flg,
				memo
			FROM deck_codes
			WHERE deck_id = ?
			ORDER BY deck_id, created_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
	`, id,
	).Where(
		"decks.id = ? AND decks.deleted_at IS NULL", id,
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		return nil, tx.Error
	}

	ret := entity.NewDeck(
		deckJoinDeckCodes.DeckID,
		deckJoinDeckCodes.DeckCreatedAt,
		deckJoinDeckCodes.DeckArchivedAt.Time,
		deckJoinDeckCodes.DeckUserId,
		deckJoinDeckCodes.DeckName,
		deckJoinDeckCodes.DeckCode,
		deckJoinDeckCodes.DeckPrivateCodeFlg,
		deckJoinDeckCodes.DeckPrivateFlg,
		entity.NewDeckCode(
			deckJoinDeckCodes.DeckCodeID,
			deckJoinDeckCodes.DeckCodeCreatedAt,
			deckJoinDeckCodes.DeckCodeUserId,
			deckJoinDeckCodes.DeckCodeDeckId,
			deckJoinDeckCodes.DeckCodeCode,
			deckJoinDeckCodes.DeckCodePrivateCodeFlg,
			deckJoinDeckCodes.DeckCodeMemo,
		),
	)

	return ret, nil
}

func (i *Deck) FindByUserId(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	offset int,
) ([]*entity.Deck, error) {
	var deckJoinDeckCodes []*model.DeckJoinDeckCode

	if archivedFlg {
		tx := i.db.Table(
			"decks",
		).Select(`
			decks.id AS deck_id,
			decks.created_at AS deck_created_at,
			decks.updated_at AS deck_updated_at,
			decks.deleted_at AS deck_deleted_at,
			decks.archived_at AS deck_archived_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
			decks.code AS deck_code,
			decks.private_code_flg AS deck_private_code_flg,
			decks.private_flg AS deck_private_flg,
			deck_codes.id AS deck_code_id,
			deck_codes.created_at AS deck_code_created_at,
			deck_codes.updated_at AS deck_code_updated_at,
			deck_codes.deleted_at AS deck_code_deleted_at,
			deck_codes.user_id AS deck_code_user_id,
			deck_codes.deck_id AS deck_code_deck_id,
			deck_codes.code AS deck_code_code,
			deck_codes.private_code_flg AS deck_code_private_code_flg,
			deck_codes.memo AS deck_code_memo
		`,
		).Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (deck_id)
					id,
					created_at,
					updated_at,
					deleted_at,
					user_id,
					deck_id,
					code,
					private_code_flg,
					memo
				FROM deck_codes
				WHERE user_id = ?
				ORDER BY deck_id, created_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
		`, uid,
		).Where(
			"decks.user_id = ? AND decks.archived_at IS NOT NULL AND decks.deleted_at IS NULL", uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Offset(
			offset,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		tx := i.db.Table(
			"decks",
		).Select(`
			decks.id AS deck_id,
			decks.created_at AS deck_created_at,
			decks.updated_at AS deck_updated_at,
			decks.deleted_at AS deck_deleted_at,
			decks.archived_at AS deck_archived_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
			decks.code AS deck_code,
			decks.private_code_flg AS deck_private_code_flg,
			decks.private_flg AS deck_private_flg,
			deck_codes.id AS deck_code_id,
			deck_codes.created_at AS deck_code_created_at,
			deck_codes.updated_at AS deck_code_updated_at,
			deck_codes.deleted_at AS deck_code_deleted_at,
			deck_codes.user_id AS deck_code_user_id,
			deck_codes.deck_id AS deck_code_deck_id,
			deck_codes.code AS deck_code_code,
			deck_codes.private_code_flg AS deck_code_private_code_flg,
			deck_codes.memo AS deck_code_memo
		`,
		).Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (deck_id)
					id,
					created_at,
					updated_at,
					deleted_at,
					user_id,
					deck_id,
					code,
					private_code_flg,
					memo
				FROM deck_codes
				WHERE user_id = ?
				ORDER BY deck_id, created_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
		`, uid,
		).Where(
			"decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Offset(
			offset,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			return nil, tx.Error
		}
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		ret = append(ret, entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
			djdc.DeckCode,
			djdc.DeckPrivateCodeFlg,
			djdc.DeckPrivateFlg,
			entity.NewDeckCode(
				djdc.DeckCodeID,
				djdc.DeckCodeCreatedAt,
				djdc.DeckCodeUserId,
				djdc.DeckCodeDeckId,
				djdc.DeckCodeCode,
				djdc.DeckCodePrivateCodeFlg,
				djdc.DeckCodeMemo,
			),
		))
	}

	return ret, nil
}

func (i *Deck) FindByUserIdOnCursor(
	ctx context.Context,
	uid string,
	archivedFlg bool,
	limit int,
	cursor time.Time,
) ([]*entity.Deck, error) {
	var deckJoinDeckCodes []*model.DeckJoinDeckCode

	if archivedFlg {
		tx := i.db.Table(
			"decks",
		).Select(`
			decks.id AS deck_id,
			decks.created_at AS deck_created_at,
			decks.updated_at AS deck_updated_at,
			decks.deleted_at AS deck_deleted_at,
			decks.archived_at AS deck_archived_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
			decks.code AS deck_code,
			decks.private_code_flg AS deck_private_code_flg,
			decks.private_flg AS deck_private_flg,
			deck_codes.id AS deck_code_id,
			deck_codes.created_at AS deck_code_created_at,
			deck_codes.updated_at AS deck_code_updated_at,
			deck_codes.deleted_at AS deck_code_deleted_at,
			deck_codes.user_id AS deck_code_user_id,
			deck_codes.deck_id AS deck_code_deck_id,
			deck_codes.code AS deck_code_code,
			deck_codes.private_code_flg AS deck_code_private_code_flg,
			deck_codes.memo AS deck_code_memo
		`,
		).Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (deck_id)
					id,
					created_at,
					updated_at,
					deleted_at,
					user_id,
					deck_id,
					code,
					private_code_flg,
					memo
				FROM deck_codes
				WHERE user_id = ?
			) AS deck_codes ON decks.id = deck_codes.deck_id
		`, uid,
		).Where(
			"decks.created_at < ? AND decks.user_id = ? AND decks.archived_at IS NOT NULL AND decks.deleted_at IS NULL", cursor, uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			return nil, tx.Error
		}
	} else {
		tx := i.db.Table(
			"decks",
		).Select(`
			decks.id AS deck_id,
			decks.created_at AS deck_created_at,
			decks.updated_at AS deck_updated_at,
			decks.deleted_at AS deck_deleted_at,
			decks.archived_at AS deck_archived_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
			decks.code AS deck_code,
			decks.private_code_flg AS deck_private_code_flg,
			decks.private_flg AS deck_private_flg,
			deck_codes.id AS deck_code_id,
			deck_codes.created_at AS deck_code_created_at,
			deck_codes.updated_at AS deck_code_updated_at,
			deck_codes.deleted_at AS deck_code_deleted_at,
			deck_codes.user_id AS deck_code_user_id,
			deck_codes.deck_id AS deck_code_deck_id,
			deck_codes.code AS deck_code_code,
			deck_codes.private_code_flg AS deck_code_private_code_flg,
			deck_codes.memo AS deck_code_memo
		`,
		).Joins(`
			LEFT JOIN (
				SELECT DISTINCT ON (deck_id)
					id,
					created_at,
					updated_at,
					deleted_at,
					user_id,
					deck_id,
					code,
					private_code_flg,
					memo
				FROM deck_codes
				WHERE user_id = ?
			) AS deck_codes ON decks.id = deck_codes.deck_id
		`, uid,
		).Where(
			"decks.created_at < ? AND decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", cursor, uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			return nil, tx.Error
		}
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		ret = append(ret, entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
			djdc.DeckCode,
			djdc.DeckPrivateCodeFlg,
			djdc.DeckPrivateFlg,
			entity.NewDeckCode(
				djdc.DeckCodeID,
				djdc.DeckCodeCreatedAt,
				djdc.DeckCodeUserId,
				djdc.DeckCodeDeckId,
				djdc.DeckCodeCode,
				djdc.DeckCodePrivateCodeFlg,
				djdc.DeckCodeMemo,
			),
		))
	}

	return ret, nil
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

	deck := model.NewDeck(
		entity.ID,
		entity.CreatedAt,
		archivedAt,
		entity.UserId,
		entity.Name,
		entity.Code,
		entity.PrivateCodeFlg,
		entity.PrivateFlg,
	)

	if entity.LatestDeckCode != nil {
		deckcode := model.NewDeckCode(
			entity.LatestDeckCode.ID,
			entity.LatestDeckCode.CreatedAt,
			entity.LatestDeckCode.UserId,
			entity.LatestDeckCode.DeckId,
			entity.LatestDeckCode.Code,
			entity.LatestDeckCode.PrivateCodeFlg,
			entity.LatestDeckCode.Memo,
		)

		return i.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Save(deck).Error; err != nil {
				return err
			}

			if err := tx.Save(deckcode).Error; err != nil {
				return err
			}

			return nil
		}, &sql.TxOptions{Isolation: sql.LevelDefault})
	} else {
		if tx := i.db.Save(deck); tx.Error != nil {
			return tx.Error
		}
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
