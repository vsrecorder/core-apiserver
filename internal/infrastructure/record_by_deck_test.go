package infrastructure

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// デッキ・デッキコードで絞る一覧は、必ず user_id も条件に含めること。
// deck_id / deck_code_id は公開情報から誰でも知り得るため、それだけで絞ると他人の記録が返る。

func newRecordRowsForDeckTests(datetime time.Time, uid string, deckId string, deckCodeId string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
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
		uid,
		deckId,
		deckCodeId,
		false,
		"",
		"",
	)
}

func test_RecordInfrastructure_FindByDeckIdOnCursor(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	deckId := "01JHAKSVXZ4XW91TDQ8EDP1N8P"
	cursorEventDate := time.Now().Local()
	cursorCreatedAt := time.Now().Local()
	limit := 10
	eventType := ""

	// event_date あり区間のカーソル
	{
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE (user_id = $1 AND deck_id = $2 AND ((event_date < $3 AND event_date IS NOT NULL) OR (event_date = $4 AND created_at < $5) OR event_date IS NULL)) AND "records"."deleted_at" IS NULL ORDER BY event_date DESC NULLS LAST, created_at DESC LIMIT $6`,
		)).WithArgs(
			uid,
			deckId,
			cursorEventDate,
			cursorEventDate,
			cursorCreatedAt,
			limit,
		).WillReturnRows(newRecordRowsForDeckTests(time.Now().Local(), uid, deckId, ""))

		expectRecordTagsQuery(mock)

		records, err := r.FindByDeckIdOnCursor(context.Background(), uid, deckId, limit, cursorEventDate, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, 1, len(records))
		require.Equal(t, uid, records[0].UserId)
		require.Equal(t, deckId, records[0].DeckId)
	}

	// event_date なし区間のカーソル(cursorEventDate がゼロ)
	{
		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "records" WHERE (user_id = $1 AND deck_id = $2 AND (event_date IS NULL AND created_at < $3)) AND "records"."deleted_at" IS NULL ORDER BY event_date DESC NULLS LAST, created_at DESC LIMIT $4`,
		)).WithArgs(
			uid,
			deckId,
			cursorCreatedAt,
			limit,
		).WillReturnRows(sqlmock.NewRows([]string{"id"}))

		records, err := r.FindByDeckIdOnCursor(context.Background(), uid, deckId, limit, time.Time{}, cursorCreatedAt, eventType)

		require.NoError(t, err)
		require.Equal(t, 0, len(records))
	}

	require.NoError(t, mock.ExpectationsWereMet())
}

func test_RecordInfrastructure_FindByDeckCodeId(t *testing.T) {
	r, mock, err := setup4RecordInfrastructure()
	require.NoError(t, err)

	uid := "CeQ0Oa9g9uRThL11lj4l45VAg8p1"
	deckCodeId := "01JHAKSVXZ4XW91TDQ8EDP1N8C"
	limit := 1
	offset := 10 // 0 だと GORM が OFFSET 句を省くため、句が出る値にする

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "records" WHERE (user_id = $1 AND deck_code_id = $2) AND "records"."deleted_at" IS NULL ORDER BY event_date DESC NULLS LAST, created_at DESC LIMIT $3 OFFSET $4`,
	)).WithArgs(
		uid,
		deckCodeId,
		limit,
		offset,
	).WillReturnRows(newRecordRowsForDeckTests(time.Now().Local(), uid, "", deckCodeId))

	expectRecordTagsQuery(mock)

	records, err := r.FindByDeckCodeId(context.Background(), uid, deckCodeId, limit, offset)

	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, uid, records[0].UserId)
	require.Equal(t, deckCodeId, records[0].DeckCodeId)
	require.NoError(t, mock.ExpectationsWereMet())
}
