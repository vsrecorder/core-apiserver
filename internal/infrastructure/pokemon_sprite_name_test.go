package infrastructure

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestPokemonSpriteNameInfrastructure(t *testing.T) {
	t.Run("正常系_指定IDの正式名をidをキーに返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPokemonSpriteName(db)

		mock.ExpectQuery(`SELECT \* FROM "pokemon_sprites" WHERE id IN \(\$1,\$2\)`).
			WithArgs("rayquaza-mega", "ho-oh").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
				AddRow("rayquaza-mega", "メガレックウザ").
				AddRow("ho-oh", "ホウオウ"))

		got, err := r.FindNamesByIds(context.Background(), []string{"rayquaza-mega", "ho-oh"})

		require.NoError(t, err)
		require.Equal(t, map[string]string{"rayquaza-mega": "メガレックウザ", "ho-oh": "ホウオウ"}, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("正常系_idsが空ならクエリを発行せず空を返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPokemonSpriteName(db)

		got, err := r.FindNamesByIds(context.Background(), nil)

		require.NoError(t, err)
		require.Empty(t, got)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
