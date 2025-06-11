package infrastructure

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMock4DeckInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4DeckInfrastructure() (repository.DeckInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4DeckInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewDeck(db)

	return r, mock, err
}

func TestDeckInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"Find":                 test_DeckInfrastructure_Find,
		"FindOnCursor":         test_DeckInfrastructure_FindOnCursor,
		"FindById":             test_DeckInfrastructure_FindById,
		"FindByUserId":         test_DeckInfrastructure_FindByUserId,
		"FindByUserIdOnCursor": test_DeckInfrastructure_FindByUserIdOnCursor,
		"Save":                 test_DeckInfrastructure_Save,
		"Delete":               test_DeckInfrastructure_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_DeckInfrastructure_Find(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	{
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		limit := 10
		offset := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (private_code_flg = false AND code IS NOT NULL AND archived_at IS NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		)).WithArgs(
			limit,
			offset,
		).WillReturnRows(rows)

		decks, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", decks[0].ID)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}

}

func test_DeckInfrastructure_FindOnCursor(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	{
		cursor := time.Now().UTC().Truncate(0)
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		limit := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (created_at < $1 AND private_code_flg = false AND code IS NOT NULL AND archived_at IS NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2`,
		)).WithArgs(
			cursor,
			limit,
		).WillReturnRows(rows)

		decks, err := r.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", decks[0].ID)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}

}

func test_DeckInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().UTC().Truncate(0)
	archivedAt := sql.NullTime{}

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"archived_at",
		"user_id",
		"name",
		"code",
		"private_code_flg",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		archivedAt,
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "decks" WHERE id = $1 AND "decks"."deleted_at" IS NULL ORDER BY "decks"."id" LIMIT $2`,
	)).WithArgs(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		1,
	).WillReturnRows(rows)

	deck, err := r.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2")

	require.NoError(t, err)
	require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", deck.ID)
}

func test_DeckInfrastructure_FindByUserId(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	{
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		archivedFlg := false
		limit := 10
		offset := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (user_id = $1 AND archived_at IS NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		)).WithArgs(
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			limit,
			offset,
		).WillReturnRows(rows)

		decks, err := r.FindByUserId(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", decks[0].UserId)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}

	{
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		archivedAt.Time = time.Now().UTC().Truncate(0)
		archivedAt.Valid = true
		archivedFlg := true
		limit := 10
		offset := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (user_id = $1 AND archived_at IS NOT NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		)).WithArgs(
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			limit,
			offset,
		).WillReturnRows(rows)

		decks, err := r.FindByUserId(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", archivedFlg, limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", decks[0].UserId)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}
}

func test_DeckInfrastructure_FindByUserIdOnCursor(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	{
		cursor := time.Now().UTC().Truncate(0)
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		archivedFlg := false
		limit := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (created_at < $1 AND user_id = $2 AND archived_at IS NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3`,
		)).WithArgs(
			cursor,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			limit,
		).WillReturnRows(rows)

		decks, err := r.FindByUserIdOnCursor(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", decks[0].UserId)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}

	{
		cursor := time.Now().UTC().Truncate(0)
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		archivedAt.Time = time.Now().UTC().Truncate(0)
		archivedAt.Valid = true
		archivedFlg := true
		limit := 10

		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"archived_at",
			"user_id",
			"name",
			"code",
			"private_code_flg",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "decks" WHERE (created_at < $1 AND user_id = $2 AND archived_at IS NOT NULL) AND "decks"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3`,
		)).WithArgs(
			cursor,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			limit,
		).WillReturnRows(rows)

		decks, err := r.FindByUserIdOnCursor(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", archivedFlg, limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(decks))
		require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", decks[0].UserId)
		require.Equal(t, archivedAt.Time, decks[0].ArchivedAt)
	}
}

func test_DeckInfrastructure_Save(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	{
		datetime := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "decks" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"archived_at"=$4,`+
				`"user_id"=$5,"name"=$6,"code"=$7,"private_code_flg"=$8 `+
				`WHERE "decks"."deleted_at" IS NULL AND "id" = $9`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			time.Time{},
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	}

	{
		datetime := time.Now().UTC().Truncate(0)
		now := time.Now().UTC().Truncate(0)
		archivedAt := sql.NullTime{}
		archivedAt.Time = now
		archivedAt.Valid = true

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "decks" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"archived_at"=$4,`+
				`"user_id"=$5,"name"=$6,"code"=$7,"private_code_flg"=$8 `+
				`WHERE "decks"."deleted_at" IS NULL AND "id" = $9`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			archivedAt,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		deck := entity.NewDeck(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			now,
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
		)

		require.NoError(t, r.Save(context.Background(), deck))
		require.NoError(t, mock.ExpectationsWereMet())
	}

}

func test_DeckInfrastructure_Delete(t *testing.T) {
	r, mock, err := setup4DeckInfrastructure()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "decks" SET "deleted_at"=$1 WHERE id = $2 AND "decks"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	require.NoError(t, r.Delete(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2"))

	require.NoError(t, mock.ExpectationsWereMet())
}
