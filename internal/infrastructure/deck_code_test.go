package infrastructure

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setupMock4DeckCodeInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()

	if err != nil {
		return nil, nil, err
	}

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: mockDB,
		}),
		&gorm.Config{},
	)

	return db, mock, err
}

func setup4DeckCodeInfrastructure() (repository.DeckCodeInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4DeckCodeInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewDeckCode(db)

	return r, mock, err
}

func TestDeckCodeInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"DeleteByUserId":                test_DeckCodeInfrastructure_DeleteByUserId,
		"ReplacePokemonSprites":         test_DeckCodeInfrastructure_ReplacePokemonSprites,
		"ReplacePokemonSprites_未設定に戻す":  test_DeckCodeInfrastructure_ReplacePokemonSprites_Empty,
		"ReplacePokemonSprites_存在しないID": test_DeckCodeInfrastructure_ReplacePokemonSprites_InvalidReference,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

// 退会時の一括削除。作成者が本人であるデッキコードを、件数によらず1文で削除する。
// (本人のデッキに紐づくものは Deck.DeleteByUserId 側で消える)
func test_DeckCodeInfrastructure_DeleteByUserId(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"

	mock.ExpectBegin()

	// 付与タグは、デッキコードを論理削除する前に消す
	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "deck_code_tags" WHERE deck_code_id IN (SELECT "id" FROM "deck_codes" WHERE user_id = $1 AND "deck_codes"."deleted_at" IS NULL)`,
	)).WithArgs(
		uid,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	// スプライトも同じく、デッキコードを論理削除する前に消す
	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "deck_code_pokemon_sprites" WHERE deck_code_id IN (SELECT "id" FROM "deck_codes" WHERE user_id = $1 AND "deck_codes"."deleted_at" IS NULL)`,
	)).WithArgs(
		uid,
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "deck_codes" SET "deleted_at"=$1 WHERE user_id = $2 AND "deck_codes"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		uid,
	).WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectCommit()

	err = r.DeleteByUserId(context.Background(), uid)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// バージョンのスプライトは「全削除 → 再INSERT」で置き換える。position が指定されていれば
// その枠に、未指定(0)なら配列の並び順(1, 2)で採番する。
func test_DeckCodeInfrastructure_ReplacePokemonSprites(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1"

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "deck_code_pokemon_sprites" WHERE deck_code_id = $1`,
	)).WithArgs(
		deckCodeId,
	).WillReturnResult(sqlmock.NewResult(0, 2))

	// 1枠目は空で2枠目だけ指定された場合も、position:2 のまま保存する
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO "deck_code_pokemon_sprites" ("deck_code_id","position","pokemon_sprite_id") VALUES ($1,$2,$3)`,
	)).WithArgs(
		deckCodeId, 2, "0887",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit()

	err = r.ReplacePokemonSprites(context.Background(), deckCodeId, []*entity.PokemonSprite{
		entity.NewPokemonSpriteWithPosition("0887", 2),
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// 空を渡すと行を消すだけで、未設定(親デッキのスプライトへフォールバック)に戻る。
func test_DeckCodeInfrastructure_ReplacePokemonSprites_Empty(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1"

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "deck_code_pokemon_sprites" WHERE deck_code_id = $1`,
	)).WithArgs(
		deckCodeId,
	).WillReturnResult(sqlmock.NewResult(0, 2))

	mock.ExpectCommit()

	err = r.ReplacePokemonSprites(context.Background(), deckCodeId, []*entity.PokemonSprite{})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// マスタに無いスプライトIDは外部キーで弾かれる。クライアント起因として 400 にできるよう
// apperror.ErrInvalidReference へ変換し、置き換え全体をロールバックする。
func test_DeckCodeInfrastructure_ReplacePokemonSprites_InvalidReference(t *testing.T) {
	r, mock, err := setup4DeckCodeInfrastructure()
	require.NoError(t, err)

	deckCodeId := "01HD7Y3K8D6FDHMHTZ2GT41TC1"

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`DELETE FROM "deck_code_pokemon_sprites" WHERE deck_code_id = $1`,
	)).WithArgs(
		deckCodeId,
	).WillReturnResult(sqlmock.NewResult(0, 0))

	// position 未指定(0)は配列の並び順で 1 を採番する
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO "deck_code_pokemon_sprites" ("deck_code_id","position","pokemon_sprite_id") VALUES ($1,$2,$3)`,
	)).WithArgs(
		deckCodeId, 1, "9999",
	).WillReturnError(&pgconn.PgError{Code: "23503"})

	mock.ExpectRollback()

	err = r.ReplacePokemonSprites(context.Background(), deckCodeId, []*entity.PokemonSprite{
		entity.NewPokemonSprite("9999"),
	})

	require.ErrorIs(t, err, apperror.ErrInvalidReference)
	require.NoError(t, mock.ExpectationsWereMet())
}
