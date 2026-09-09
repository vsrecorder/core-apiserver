package infrastructure

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setup4OfficialEventEnvironmentInfrastructure(t *testing.T) (repository.OfficialEventEnvironmentInterface, sqlmock.Sqlmock) {
	t.Helper()

	db, mock := setupSqlmockDB(t)

	return NewOfficialEventEnvironment(db), mock
}

func TestOfficialEventEnvironmentInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindEnvironmentIdByOfficialEventId":          test_OfficialEventEnvironmentInfrastructure_FindEnvironmentIdByOfficialEventId,
		"FindEnvironmentIdByOfficialEventId_NotFound": test_OfficialEventEnvironmentInfrastructure_FindEnvironmentIdByOfficialEventId_NotFound,
		"FindAll": test_OfficialEventEnvironmentInfrastructure_FindAll,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_OfficialEventEnvironmentInfrastructure_FindEnvironmentIdByOfficialEventId(t *testing.T) {
	r, mock := setup4OfficialEventEnvironmentInfrastructure(t)

	officialEventId := uint(1113193)
	environmentId := "m6"

	rows := sqlmock.NewRows([]string{
		"official_event_id",
		"environment_id",
	}).AddRow(
		officialEventId,
		environmentId,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "official_event_environments" WHERE official_event_id = $1 ORDER BY "official_event_environments"."official_event_id" LIMIT $2`,
	)).WithArgs(
		officialEventId,
		1,
	).WillReturnRows(rows)

	ret, err := r.FindEnvironmentIdByOfficialEventId(context.Background(), officialEventId)

	require.NoError(t, err)
	require.Equal(t, environmentId, ret)
}

// 例外登録が無いのは通常の状態なので、呼び出し側が開催日からの判定へフォールバック
// できるよう apperror.ErrRecordNotFound を返す。
func test_OfficialEventEnvironmentInfrastructure_FindEnvironmentIdByOfficialEventId_NotFound(t *testing.T) {
	r, mock := setup4OfficialEventEnvironmentInfrastructure(t)

	officialEventId := uint(9999999)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "official_event_environments" WHERE official_event_id = $1 ORDER BY "official_event_environments"."official_event_id" LIMIT $2`,
	)).WithArgs(
		officialEventId,
		1,
	).WillReturnRows(sqlmock.NewRows([]string{"official_event_id", "environment_id"}))

	_, err := r.FindEnvironmentIdByOfficialEventId(context.Background(), officialEventId)

	require.ErrorIs(t, err, apperror.ErrRecordNotFound)
}

func test_OfficialEventEnvironmentInfrastructure_FindAll(t *testing.T) {
	r, mock := setup4OfficialEventEnvironmentInfrastructure(t)

	rows := sqlmock.NewRows([]string{
		"official_event_id",
		"environment_id",
	}).AddRow(
		uint(1113193), "m6",
	).AddRow(
		uint(1113194), "m6",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "official_event_environments" ORDER BY official_event_id ASC`,
	)).WillReturnRows(rows)

	overrides, err := r.FindAll(context.Background())

	require.NoError(t, err)
	require.Equal(t, map[uint]string{1113193: "m6", 1113194: "m6"}, overrides)
}
