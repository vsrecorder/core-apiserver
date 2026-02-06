package infrastructure

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTonamelEventInfrastructure(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
	){
		"FindById": test_TonamelEventInfrastructure_FindById,
	} {
		t.Run(scenario, func(t *testing.T) {
			fn(t)
		})
	}
}

func test_TonamelEventInfrastructure_FindById(t *testing.T) {
	logger := slog.Default()
	r := NewTonamelEvent(logger)

	t.Run("正常系_#01", func(t *testing.T) {

		id := "OakZc"
		title := "第23回 ACEカップ～FINAL～ - Tonamel"

		res, err := r.FindById(context.Background(), id)

		require.NoError(t, err)
		require.Equal(t, res.ID, id)
		require.Equal(t, res.Title, title)
		require.NotEmpty(t, res.Image)
	})

	t.Run("異常系_#01", func(t *testing.T) {
		id := "ERROR"

		_, err := r.FindById(context.Background(), id)

		require.Equal(t, err, gorm.ErrRecordNotFound)
	})
}
