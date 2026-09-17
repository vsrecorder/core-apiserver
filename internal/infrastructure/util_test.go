package infrastructure

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"

	"github.com/vsrecorder/core-apiserver/internal/domain/apperror"
)

func TestWrapForeignKeyViolation(t *testing.T) {
	t.Run("正常系_外部キー違反はErrInvalidReferenceになる", func(t *testing.T) {
		err := fmt.Errorf("save: %w", &pgconn.PgError{Code: "23503", Message: "foreign key violation"})

		require.ErrorIs(t, wrapForeignKeyViolation(err), apperror.ErrInvalidReference)
	})

	t.Run("正常系_それ以外のDBエラーはそのまま返す", func(t *testing.T) {
		unique := &pgconn.PgError{Code: "23505"}
		require.Same(t, unique, wrapForeignKeyViolation(unique))

		other := errors.New("db down")
		require.Same(t, other, wrapForeignKeyViolation(other))
	})

	t.Run("正常系_nilはnilのまま", func(t *testing.T) {
		require.NoError(t, wrapForeignKeyViolation(nil))
	})
}
