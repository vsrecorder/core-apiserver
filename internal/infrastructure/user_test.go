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

func setupMock4UserInfrastructure() (*gorm.DB, sqlmock.Sqlmock, error) {
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

func setup4UserInfrastructure() (repository.UserInterface, sqlmock.Sqlmock, error) {
	db, mock, err := setupMock4UserInfrastructure()

	if err != nil {
		return nil, nil, err
	}

	r := NewUser(db)

	return r, mock, err
}

func TestUserInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById": test_UserInfrastructure_FindById,
		"Save":     test_UserInfrastructure_Save,
		"Delete":   test_UserInfrastructure_Delete,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_UserInfrastructure_FindById(t *testing.T) {
	r, mock, err := setup4UserInfrastructure()
	require.NoError(t, err)

	datetime := time.Now().UTC().Truncate(0)
	name := "test"
	imageURL := "http://example.com/image.png"

	rows := sqlmock.NewRows([]string{
		"id",
		"created_at",
		"updated_at",
		"deleted_at",
		"name",
		"image_url",
	}).AddRow(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		datetime,
		datetime,
		gorm.DeletedAt{},
		name,
		imageURL,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "users" WHERE id = $1 AND "users"."deleted_at" IS NULL ORDER BY "users"."id" LIMIT $2`,
	)).WithArgs(
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		1,
	).WillReturnRows(rows)

	user, err := r.FindById(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2")

	require.NoError(t, err)
	require.Equal(t, "01HD7Y3K8D6FDHMHTZ2GT41TN2", user.ID)
	require.Equal(t, name, user.Name)
	require.Equal(t, imageURL, user.ImageURL)
}

func test_UserInfrastructure_Save(t *testing.T) {
	r, mock, err := setup4UserInfrastructure()
	require.NoError(t, err)

	{
		datetime := time.Now().UTC().Truncate(0)
		name := "test"
		imageURL := "http://example.com/image.png"

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(
			`UPDATE "users" SET "created_at"=$1,"updated_at"=$2,"deleted_at"=$3,`+
				`"name"=$4,"image_url"=$5 `+
				`WHERE "users"."deleted_at" IS NULL AND "id" = $6`,
		)).WithArgs(
			datetime,
			AnyTime{},
			gorm.DeletedAt{},
			name,
			imageURL,
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
		).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		user := entity.NewUser(
			"01HD7Y3K8D6FDHMHTZ2GT41TN2",
			datetime,
			name,
			imageURL,
		)

		require.NoError(t, r.Save(context.Background(), user))
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

func test_UserInfrastructure_Delete(t *testing.T) {
	r, mock, err := setup4UserInfrastructure()
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE "users" SET "deleted_at"=$1 WHERE id = $2 AND "users"."deleted_at" IS NULL`,
	)).WithArgs(
		AnyTime{},
		"01HD7Y3K8D6FDHMHTZ2GT41TN2",
	).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	require.NoError(t, r.Delete(context.Background(), "01HD7Y3K8D6FDHMHTZ2GT41TN2"))

	require.NoError(t, mock.ExpectationsWereMet())
}
