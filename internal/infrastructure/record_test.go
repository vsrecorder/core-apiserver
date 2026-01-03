package infrastructure

import (
	"context"
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

func setupMock4RecordInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4RecordInfrastructure() (repository.RecordInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4RecordInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewRecord(db)

	return r, mock, err
}

func TestRecordInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"Find":                  test_RecordInfrastructure_Find,
		"FindOnCursor":          test_RecordInfrastructure_FindOnCursor,
		"FindById":              test_RecordInfrastructure_FindById,
		"FindByUserId":          test_RecordInfrastructure_FindByUserId,
		"FindByUserIdOnCursor":  test_RecordInfrastructure_FindByUserIdOnCursor,
		"FindByOfficialEventId": test_RecordInfrastructure_FindByOfficialEventId,
		"FindByTonamelEventId":  test_RecordInfrastructure_FindByTonamelEventId,
		"FindByDeckId":          test_RecordInfrastructure_FindByDeckId,
		"Save":                  test_RecordInfrastructure_Save,
		"Delete":                test_RecordInfrastructure_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_RecordInfrastructure_Find(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()
	limit := 10
	offset := 10

	{
		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
			"deck_code_id",
			"private_flg",
			"tcg_meister_url",
			"memo",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			236790,
			"",
			"",
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
			"",
			"",
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE private_flg = false AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		)).WithArgs(
			limit,
			offset,
		).WillReturnRows(rows)

		records, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, 1, len(records))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", records[0].ID)
	}

	{
		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
			"deck_code_id",
			"private_flg",
			"tcg_meister_url",
			"memo",
		})

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE private_flg = false AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		)).WithArgs(
			limit,
			offset,
		).WillReturnRows(rows)

		records, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, 0, len(records))
	}

}

func test_RecordInfrastructure_FindOnCursor(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	cursor := time.Now().Local()
	datetime := time.Now().Local()
	limit := 10

	{
		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
			"deck_code_id",
			"private_flg",
			"tcg_meister_url",
			"memo",
		}).AddRow(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			datetime,
			gorm.DeletedAt{},
			236790,
			"",
			"",
			"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
			"",
			"",
			false,
			"",
			"",
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE (created_at < $1 AND private_flg = false) AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`,
		)).WithArgs(
			cursor,
			limit,
		).WillReturnRows(rows)

		records, err := r.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 1, len(records))
		require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", records[0].ID)
	}

	{
		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"updated_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
			"deck_code_id",
			"private_flg",
			"tcg_meister_url",
			"memo",
		})

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE (created_at < $1 AND private_flg = false) AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT`,
		)).WithArgs(
			cursor,
			limit,
		).WillReturnRows(rows)

		records, err := r.FindOnCursor(context.Background(), limit, cursor)

		require.NoError(t, err)
		require.Equal(t, 0, len(records))
	}

}

func test_RecordInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE id = $1 AND "records"."deleted_at" IS NULL ORDER BY "records"."id" LIMIT $2`,
	)).WithArgs(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		1,
	).WillReturnRows(rows)

	record, err := r.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2")

	require.NoError(t, err)
	require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", record.ID)
}

func test_RecordInfrastructure_FindByUserId(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE user_id = $1 AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
	)).WithArgs(
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		limit,
		offset,
	).WillReturnRows(rows)

	records, err := r.FindByUserId(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", limit, offset)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", records[0].UserId)
}

func test_RecordInfrastructure_FindByUserIdOnCursor(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	cursor := time.Now().Local()
	datetime := time.Now().Local()
	limit := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE (created_at < $1 AND user_id = $2) AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3`,
	)).WithArgs(
		cursor,
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		limit,
	).WillReturnRows(rows)

	records, err := r.FindByUserIdOnCursor(context.Background(), "CeQ0Oa9g9uRThL11lj4l45VAg8p1", limit, cursor)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "CeQ0Oa9g9uRThL11lj4l45VAg8p1", records[0].UserId)
}

func test_RecordInfrastructure_FindByOfficialEventId(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE (official_event_id = $1 AND private_flg = $2) AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
	)).WithArgs(
		236790,
		false,
		limit,
		offset,
	).WillReturnRows(rows)

	records, err := r.FindByOfficialEventId(context.Background(), 236790, limit, offset)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, (uint)(236790), records[0].OfficialEventId)
}

func test_RecordInfrastructure_FindByTonamelEventId(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		nil,
		"YFUVY",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE (tonamel_event_id = $1 AND private_flg = $2) AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
	)).WithArgs(
		"YFUVY",
		false,
		limit,
		offset,
	).WillReturnRows(rows)

	records, err := r.FindByTonamelEventId(context.Background(), "YFUVY", limit, offset)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "YFUVY", records[0].TonamelEventId)
}

func test_RecordInfrastructure_FindByDeckId(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
		"deck_code_id",
		"private_flg",
		"tcg_meister_url",
		"memo",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"01JHAKSVXZ4XW91TDQ8EDP1N8P",
		"",
		false,
		"",
		"",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE deck_id = $1 AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
	)).WithArgs(
		"01JHAKSVXZ4XW91TDQ8EDP1N8P",
		limit,
		offset,
	).WillReturnRows(rows)

	records, err := r.FindByDeckId(context.Background(), "01JHAKSVXZ4XW91TDQ8EDP1N8P", limit, offset)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "01JHAKSVXZ4XW91TDQ8EDP1N8P", records[0].DeckId)
}

func test_RecordInfrastructure_Save(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().Local()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "records" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"official_event_id"=$4,"tonamel_event_id"=$5,`+
			`"friend_id"=$6,"user_id"=$7,"deck_id"=$8,"deck_code_id"=$9,"private_flg"=$10,"tcg_meister_url"=$11,"memo"=$12 `+
			`WHERE "records"."deleted_at" IS NULL AND "id" = $13`,
	)).WithArgs(
		datetime,
		AnyTime{},
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		"",
		"",
		false,
		"",
		"",
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	record := &entity.Record{
		ID:              "01HD7Y3K8D6FDHMHTZ2GT41TN2",
		CreatedAt:       datetime,
		OfficialEventId: 236790,
		TonamelEventId:  "",
		FriendId:        "",
		UserId:          "CeQ0Oa9g9uRThL11lj4l45VAg8p1",
		DeckId:          "",
		DeckCodeId:      "",
		PrivateFlg:      false,
		TCGMeisterURL:   "",
		Memo:            "",
	}

	{
		err := r.Save(context.Background(), record)
		require.NoError(t, err)
	}

	{
		err := mock.ExpectationsWereMet()
		require.NoError(t, err)
	}
}

func test_RecordInfrastructure_Delete(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "records" SET "deleted_at"=$1 WHERE id = $2 AND "records"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	{
		err := r.Delete(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2")
		require.NoError(t, err)
	}

	{
		err := mock.ExpectationsWereMet()
		require.NoError(t, err)
	}
}
