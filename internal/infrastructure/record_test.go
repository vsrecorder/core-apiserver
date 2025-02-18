package infrastructure

import (
	"context"
	"database/sql/driver"
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

// GORM側で更新される カラム updated_at の値をテストでPASSするための構造体
type AnyTime struct{}

func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func setupMock() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup() (repository.RecordInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock()

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
		"Find":                  test_Find,
		"FindById":              test_FindById,
		"FindByUserId":          test_FindByUserId,
		"FindByOfficialEventId": test_FindByOfficialEventId,
		"FindByTonamelEventId":  test_FindByTonamelEventId,
		"FindByDeckId":          test_FindByDeckId,
		"Save":                  test_Save,
		"Delete":                test_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_Find(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()
	limit := 10
	offset := 10

	{
		rows := sqlmock.NewRows([]string{
			"id",
			"created_at",
			"update_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
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
			false,
			"",
			"",
		)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE private_flg = $1 AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		)).WithArgs(
			false,
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
			"update_at",
			"deleted_at",
			"official_event_id",
			"tonamel_event_id",
			"friend_id",
			"user_id",
			"deck_id",
			"private_flg",
			"tcg_meister_url",
			"memo",
		})

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE private_flg = $1 AND "records"."deleted_at" IS NULL ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		)).WithArgs(
			false,
			limit,
			offset,
		).WillReturnRows(rows)

		records, err := r.Find(context.Background(), limit, offset)

		require.NoError(t, err)
		require.Equal(t, 0, len(records))
	}

}

func test_FindById(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"update_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
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

func test_FindByUserId(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"update_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
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

func test_FindByOfficialEventId(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"update_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
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

func test_FindByTonamelEventId(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"update_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
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

func test_FindByDeckId(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now()
	limit := 10
	offset := 10

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"update_at",
		"deleted_at",
		"official_event_id",
		"tonamel_event_id",
		"friend_id",
		"user_id",
		"deck_id",
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

func test_Save(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	datetime := time.Now().Truncate(0)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "records" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,"official_event_id"=$4,"tonamel_event_id"=$5,`+
			`"friend_id"=$6,"user_id"=$7,"deck_id"=$8,"private_flg"=$9,"tcg_meister_url"=$10,"memo"=$11 `+
			`WHERE "records"."deleted_at" IS NULL AND "id" = $12`,
	)).WithArgs(
		datetime,
		AnyTime{},
		gorm.DeletedAt{},
		236790,
		"",
		"",
		"CeQ0Oa9g9uRThL11lj4l45VAg8p1",
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

func test_Delete(t *testing.T) {
	r, mock, err := setup()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "records" SET "deleted_at"=$1 WHERE id = $2 AND "records"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 0))
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
