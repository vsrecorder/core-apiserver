package infrastructure

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/entity"
)

var regulationColumns = []string{"id", "name"}

func TestRegulationInfrastructure(t *testing.T) {
	t.Run("Find", func(t *testing.T) {
		t.Run("正常系_ID昇順で全レギュレーションを返す", func(t *testing.T) {
			db, mock := setupSqlmockDB(t)
			r := NewRegulation(db)

			mock.ExpectQuery(regexp.QuoteMeta(
				`SELECT * FROM "regulations" ORDER BY id ASC`,
			)).WillReturnRows(sqlmock.NewRows(regulationColumns).AddRow(
				entity.RegulationIdStandard, "スタンダード",
			).AddRow(
				entity.RegulationIdExtra, "エクストラ",
			).AddRow(
				entity.RegulationIdHallOfFame, "殿堂",
			))

			ret, err := r.Find(context.Background())

			require.NoError(t, err)
			require.Len(t, ret, 3)
			require.Equal(t, entity.RegulationIdStandard, ret[0].ID)
			require.Equal(t, "スタンダード", ret[0].Name)
			require.Equal(t, entity.RegulationIdHallOfFame, ret[2].ID)
			require.Equal(t, "殿堂", ret[2].Name)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	})
}
