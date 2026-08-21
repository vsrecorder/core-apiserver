package infrastructure

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"github.com/vsrecorder/core-apiserver/internal/infrastructure/model"
)

type Deck struct {
	db *gorm.DB
}

func NewDeck(
	db *gorm.DB,
) repository.DeckInterface {
	return &Deck{db}
}

// attachLatestDeckCodeTags は、各 deck の LatestDeckCode に付与タグをまとめてロードして載せる。
// deck の読み出しは deck 本体のタグ(findTagsByDeckIds)しか積まないため、レスポンスの
// latest_deck_code.tags を正しく返す(新バージョン作成時のタグ継承などで使う)にはこれを併用する。
func attachLatestDeckCodeTags(ctx context.Context, db *gorm.DB, decks []*entity.Deck) error {
	ids := make([]string, 0, len(decks))
	for _, d := range decks {
		if d != nil && d.LatestDeckCode != nil && d.LatestDeckCode.ID != "" {
			ids = append(ids, d.LatestDeckCode.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	tagsByDeckCodeId, err := findTagsByDeckCodeIds(ctx, db, ids)
	if err != nil {
		logError(ctx, err)
		return err
	}

	for _, d := range decks {
		if d != nil && d.LatestDeckCode != nil {
			d.LatestDeckCode.Tags = tagsByDeckCodeId[d.LatestDeckCode.ID]
		}
	}
	return nil
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
		user_favorite_decks.created_at AS deck_favorited_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
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
			WHERE deleted_at IS NULL
			ORDER BY deck_id, created_at DESC, updated_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
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
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		pokemonSprites := spritesByDeckId[djdc.DeckID]

		deck := entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckFavoritedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
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
			pokemonSprites,
		)
		deck.Tags = tagsByDeckId[djdc.DeckID]
		ret = append(ret, deck)
	}

	// latest_deck_code の付与タグをまとめてロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, ret); err != nil {
		logError(ctx, err)
		return nil, err
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
			user_favorite_decks.created_at AS deck_favorited_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
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
				WHERE user_id = ? AND deleted_at IS NULL
				ORDER BY deck_id, created_at DESC, updated_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
		`, uid,
	).Where(
		"decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", uid,
	).Order(
		// お気に入りのデッキを先頭に置く(記録作成時のデッキ選択で最初に現れるようにする)。
		// PostgreSQL では false < true のため、JOIN が当たった(= IS NULL が false の)
		// お気に入りが先に来る。残りは従来どおり新しい順。
		"user_favorite_decks.created_at IS NULL, decks.created_at DESC",
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		pokemonSprites := spritesByDeckId[djdc.DeckID]

		deck := entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckFavoritedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
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
			pokemonSprites,
		)
		deck.Tags = tagsByDeckId[djdc.DeckID]
		ret = append(ret, deck)
	}

	// latest_deck_code の付与タグをまとめてロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, ret); err != nil {
		logError(ctx, err)
		return nil, err
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
		user_favorite_decks.created_at AS deck_favorited_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
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
			WHERE deleted_at IS NULL
			ORDER BY deck_id, created_at DESC, updated_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
	`,
	).Where(
		"decks.created_at < ? AND decks.private_flg = false AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", cursor,
	).Order(
		"decks.created_at DESC",
	).Limit(
		limit,
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		pokemonSprites := spritesByDeckId[djdc.DeckID]

		deck := entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckFavoritedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
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
			pokemonSprites,
		)
		deck.Tags = tagsByDeckId[djdc.DeckID]
		ret = append(ret, deck)
	}

	// latest_deck_code の付与タグをまとめてロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, ret); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return ret, nil
}

func (i *Deck) FindById(
	ctx context.Context,
	id string,
) (*entity.Deck, error) {
	// idの存在確認
	if tx := i.db.Where("id = ?", id).First(&model.Deck{}); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, wrapError(tx.Error)
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
		user_favorite_decks.created_at AS deck_favorited_at,
		decks.user_id AS deck_user_id,
		decks.name AS deck_name,
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
			WHERE deck_id = ? AND deleted_at IS NULL
			ORDER BY deck_id, created_at DESC, updated_at DESC
		) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
	`, id,
	).Where(
		"decks.id = ? AND decks.deleted_at IS NULL", id,
	).Scan(&deckJoinDeckCodes)

	if tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	var deckPokemonSpriteModels []*model.DeckPokemonSprite
	if tx := i.db.Where("deck_id = ?", deckJoinDeckCodes.DeckID).Order("position ASC").Find(&deckPokemonSpriteModels); tx.Error != nil {
		logError(ctx, tx.Error)
		return nil, tx.Error
	}

	var pokemonSprites []*entity.PokemonSprite
	for _, deckPokemonSpriteModel := range deckPokemonSpriteModels {
		entity := entity.NewPokemonSpriteWithPosition(deckPokemonSpriteModel.PokemonSpriteId, deckPokemonSpriteModel.Position)
		pokemonSprites = append(pokemonSprites, entity)
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, []string{deckJoinDeckCodes.DeckID})
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	ret := entity.NewDeck(
		deckJoinDeckCodes.DeckID,
		deckJoinDeckCodes.DeckCreatedAt,
		deckJoinDeckCodes.DeckArchivedAt.Time,
		deckJoinDeckCodes.DeckFavoritedAt.Time,
		deckJoinDeckCodes.DeckUserId,
		deckJoinDeckCodes.DeckName,
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
		pokemonSprites,
	)
	ret.Tags = tagsByDeckId[deckJoinDeckCodes.DeckID]

	// latest_deck_code の付与タグをロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, []*entity.Deck{ret}); err != nil {
		logError(ctx, err)
		return nil, err
	}

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
			user_favorite_decks.created_at AS deck_favorited_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
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
				WHERE user_id = ? AND deleted_at IS NULL
				ORDER BY deck_id, created_at DESC, updated_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
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
			logError(ctx, tx.Error)
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
			user_favorite_decks.created_at AS deck_favorited_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
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
				WHERE user_id = ? AND deleted_at IS NULL
				ORDER BY deck_id, created_at DESC, updated_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
		`, uid,
		).Where(
			"decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", uid,
		).Order(
			// お気に入りのデッキを一覧の先頭に置く。カーソル取得側(FindByUserIdOnCursor)は
			// 従来の並びのままにしてある。2ページ目以降でも先頭に差し込むと、
			// 本来そのデッキが現れるページとで重複して返ることになるため。
			"user_favorite_decks.created_at IS NULL, decks.created_at DESC",
		).Limit(
			limit,
		).Offset(
			offset,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		pokemonSprites := spritesByDeckId[djdc.DeckID]

		deck := entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckFavoritedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
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
			pokemonSprites,
		)
		deck.Tags = tagsByDeckId[djdc.DeckID]
		ret = append(ret, deck)
	}

	// latest_deck_code の付与タグをまとめてロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, ret); err != nil {
		logError(ctx, err)
		return nil, err
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
			user_favorite_decks.created_at AS deck_favorited_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
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
				WHERE user_id = ? AND deleted_at IS NULL
				ORDER BY deck_id, created_at DESC, updated_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
		`, uid,
		).Where(
			"decks.created_at < ? AND decks.user_id = ? AND decks.archived_at IS NOT NULL AND decks.deleted_at IS NULL", cursor, uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			logError(ctx, tx.Error)
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
			user_favorite_decks.created_at AS deck_favorited_at,
			decks.user_id AS deck_user_id,
			decks.name AS deck_name,
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
				WHERE user_id = ? AND deleted_at IS NULL
				ORDER BY deck_id, created_at DESC, updated_at DESC
			) AS deck_codes ON decks.id = deck_codes.deck_id
			LEFT JOIN user_favorite_decks
				ON user_favorite_decks.deck_id = decks.id
				AND user_favorite_decks.user_id = decks.user_id
		`, uid,
		).Where(
			"decks.created_at < ? AND decks.user_id = ? AND decks.archived_at IS NULL AND decks.deleted_at IS NULL", cursor, uid,
		).Order(
			"decks.created_at DESC",
		).Limit(
			limit,
		).Scan(&deckJoinDeckCodes)

		if tx.Error != nil {
			logError(ctx, tx.Error)
			return nil, tx.Error
		}
	}

	if len(deckJoinDeckCodes) == 0 {
		return []*entity.Deck{}, nil
	}

	spritesByDeckId, err := findDeckPokemonSpritesByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	tagsByDeckId, err := findTagsByDeckIds(ctx, i.db, deckIdsOf(deckJoinDeckCodes))
	if err != nil {
		logError(ctx, err)
		return nil, err
	}

	var ret []*entity.Deck

	for _, djdc := range deckJoinDeckCodes {
		pokemonSprites := spritesByDeckId[djdc.DeckID]

		deck := entity.NewDeck(
			djdc.DeckID,
			djdc.DeckCreatedAt,
			djdc.DeckArchivedAt.Time,
			djdc.DeckFavoritedAt.Time,
			djdc.DeckUserId,
			djdc.DeckName,
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
			pokemonSprites,
		)
		deck.Tags = tagsByDeckId[djdc.DeckID]
		ret = append(ret, deck)
	}

	// latest_deck_code の付与タグをまとめてロードして載せる。
	if err := attachLatestDeckCodeTags(ctx, i.db, ret); err != nil {
		logError(ctx, err)
		return nil, err
	}

	return ret, nil
}

func (i *Deck) DeleteByUserId(
	ctx context.Context,
	uid string,
) error {
	db := dbFromContext(ctx, i.db)

	// アーカイブ済みデッキも含めて全件対象にする
	return db.Transaction(func(tx *gorm.DB) error {
		// デッキコードを先に消す。decks を先に消すとサブクエリが0件になり消し残る。
		// ここで消えるのは「このユーザのデッキに紐づくコード」であり、
		// 作成者が他人のものも含む(デッキが消える以上、残しても参照先が無い)。
		if tx := tx.Where(
			"deck_id IN (?)",
			tx.Model(&model.Deck{}).Select("id").Where("user_id = ?", uid),
		).Delete(&model.DeckCode{}); tx.Error != nil {
			return tx.Error
		}

		// お気に入りも同様に、decks を消す前にまとめて消す。
		// 論理削除の decks と違いこちらは実削除のため、消し残すと
		// 参照先の無い行が残る。
		if tx := tx.Where(
			"deck_id IN (?)",
			tx.Model(&model.Deck{}).Select("id").Where("user_id = ?", uid),
		).Delete(&model.UserFavoriteDeck{}); tx.Error != nil {
			return tx.Error
		}

		if tx := tx.Where("user_id = ?", uid).Delete(&model.Deck{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
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

	// お気に入りは user_favorite_decks で管理しているため、ここでは書き戻さない。
	deck := model.NewDeck(
		entity.ID,
		entity.CreatedAt,
		archivedAt,
		entity.UserId,
		entity.Name,
		entity.PrivateFlg,
	)

	var deckPokemonSpriteModals []*model.DeckPokemonSprite
	for i, pokemonSprite := range entity.PokemonSprites {
		// position が指定されていればスロットとして使う。
		// 未指定(0)は後方互換のため配列インデックスから採番する。
		position := pokemonSprite.Position
		if position == 0 {
			position = uint(i + 1)
		}
		deckPokemonSpriteModals = append(deckPokemonSpriteModals, model.NewDeckPokemonSprite(entity.ID, position, pokemonSprite.ID))
	}

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
			// Deck を先に保存して FK 制約を満たす
			if err := tx.Save(deck).Error; err != nil {
				logError(ctx, err)
				return err
			}

			if tx := tx.Where("deck_id = ?", entity.ID).Delete(&model.DeckPokemonSprite{}); tx.Error != nil {
				logError(ctx, tx.Error)
				return tx.Error
			}

			for _, deckPokemonSpriteModal := range deckPokemonSpriteModals {
				if err := tx.Save(deckPokemonSpriteModal).Error; err != nil {
					logError(ctx, err)
					return err
				}
			}

			// デッキコードのIDが空でない場合は保存する
			if deckcode.ID != "" {
				if err := tx.Save(deckcode).Error; err != nil {
					logError(ctx, err)
					return err
				}
			}

			return nil
		}, &sql.TxOptions{Isolation: sql.LevelDefault})
	} else {
		return i.db.Transaction(func(tx *gorm.DB) error {
			// Deck を先に保存して FK 制約を満たす
			if err := tx.Save(deck).Error; err != nil {
				logError(ctx, err)
				return err
			}

			if tx := tx.Where("deck_id = ?", entity.ID).Delete(&model.DeckPokemonSprite{}); tx.Error != nil {
				logError(ctx, tx.Error)
				return tx.Error
			}

			for _, deckPokemonSpriteModal := range deckPokemonSpriteModals {
				if err := tx.Save(deckPokemonSpriteModal).Error; err != nil {
					logError(ctx, err)
					return err
				}
			}

			return nil
		}, &sql.TxOptions{Isolation: sql.LevelDefault})
	}
}

func (i *Deck) Delete(
	ctx context.Context,
	id string,
) error {
	db := dbFromContext(ctx, i.db)

	var deckCodes []*model.DeckCode
	if tx := db.Where("deck_id = ?", id).Order("created_at ASC").Find(&deckCodes); tx.Error != nil {
		logError(ctx, tx.Error)
		return tx.Error
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, deckCode := range deckCodes {
			if tx := tx.Where("id = ?", deckCode.ID).Delete(&model.DeckCode{}); tx.Error != nil {
				logError(ctx, tx.Error)
				return tx.Error
			}
		}

		// このデッキへのお気に入りも解除する。decks は論理削除だがこちらは実削除のため、
		// 残すと参照先の無い行になり、お気に入りの件数にも数えられてしまう。
		if tx := tx.Where("deck_id = ?", id).Delete(&model.UserFavoriteDeck{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		if tx := tx.Where("id = ?", id).Delete(&model.Deck{}); tx.Error != nil {
			logError(ctx, tx.Error)
			return tx.Error
		}

		return nil
	}, &sql.TxOptions{Isolation: sql.LevelDefault})
}
