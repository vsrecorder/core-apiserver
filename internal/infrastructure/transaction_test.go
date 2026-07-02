package infrastructure

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/vsrecorder/core-apiserver/internal/domain/repository"
)

func setup4TransactionInfrastructure() (*gorm.DB, repository.TransactionManager, sqlmock.Sqlmock, error) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, nil, err
	}

	db, err := gorm.Open(
		postgres.New(postgres.Config{
			Conn: mockDB,
		}),
		&gorm.Config{},
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return db, NewTransactionManager(db), mock, nil
}

func TestTransactionInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"commit-on-success":             test_TransactionInfrastructure_CommitOnSuccess,
		"rollback-on-error":             test_TransactionInfrastructure_RollbackOnError,
		"propagates-tx-bound-db-to-ctx": test_TransactionInfrastructure_PropagatesTxBoundDbToCtx,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TransactionInfrastructure_CommitOnSuccess(t *testing.T) {
	_, tm, mock, err := setup4TransactionInfrastructure()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectCommit()

	err = tm.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func test_TransactionInfrastructure_RollbackOnError(t *testing.T) {
	_, tm, mock, err := setup4TransactionInfrastructure()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectRollback()

	wantErr := errors.New("boom")
	err = tm.Do(context.Background(), func(ctx context.Context) error {
		return wantErr
	})

	require.Equal(t, wantErr, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// dbFromContext が実際に tx を伝播しているかを検証する。
// もし伝播に失敗して呼び出し先が repo 自身の db (=非tx) を使ってしまうと、
// gorm が単体statementを別の暗黙トランザクションとして扱おうとし、
// ここで登録していない2回目の BEGIN を要求するため sqlmock が失敗する。
func test_TransactionInfrastructure_PropagatesTxBoundDbToCtx(t *testing.T) {
	db, tm, mock, err := setup4TransactionInfrastructure()
	require.NoError(t, err)

	userRepo := NewUser(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "users" SET "deleted_at"=$1 WHERE id = $2 AND "users"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = tm.Do(context.Background(), func(ctx context.Context) error {
		return userRepo.Delete(ctx, "01HD7Y3K8D6FDHMHTZ2GT41TN2")
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
