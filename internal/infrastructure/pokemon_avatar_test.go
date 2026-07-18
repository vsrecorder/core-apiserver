package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

var pokemonAvatarColumns = []string{"id", "title", "image_url", "detail", "created_at", "updated_at"}

func TestPokemonAvatarInfrastructure(t *testing.T) {
	t.Run("正常系_現在のアバター画像を除外してランダムに1件返す", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPokemonAvatar(db)

		now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.Local)
		currentImageURL := "https://example.com/current.png"

		mock.ExpectQuery(`SELECT \* FROM "pokemon_avatars" WHERE image_url <> \$1 ORDER BY RANDOM\(\)`).
			WithArgs(currentImageURL, 1).
			WillReturnRows(sqlmock.NewRows(pokemonAvatarColumns).AddRow(
				25, "ピカチュウ", "https://example.com/pikachu.png", "ねずみポケモン", now, now,
			))

		ret, err := r.FindRandomExcludingImageURL(context.Background(), currentImageURL)

		require.NoError(t, err)
		require.Equal(t, 25, ret.ID)
		require.Equal(t, "ピカチュウ", ret.Title)
		require.Equal(t, "https://example.com/pikachu.png", ret.ImageURL)
		require.Equal(t, "ねずみポケモン", ret.Detail)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("異常系_候補が無ければErrRecordNotFoundへ変換する", func(t *testing.T) {
		db, mock := setupSqlmockDB(t)
		r := NewPokemonAvatar(db)

		mock.ExpectQuery(`SELECT \* FROM "pokemon_avatars"`).
			WithArgs("https://example.com/current.png", 1).
			WillReturnRows(sqlmock.NewRows(pokemonAvatarColumns))

		ret, err := r.FindRandomExcludingImageURL(context.Background(), "https://example.com/current.png")

		require.Equal(t, apperror.ErrRecordNotFound, err)
		require.Nil(t, ret)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
